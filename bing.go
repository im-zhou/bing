package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

var photoKeyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,99}$`)
var resolutionSuffix = regexp.MustCompile(`_(?:[0-9]+x[0-9]+|UHD)\.(?:jpg|jpeg|png)$`)

// Prefer urlbase, then the historical qiniu_url key. Never contact an old OSS/CDN URL.
func photoKey(p Photo) (string, error) {
	for _, raw := range []string{p.URLBase, p.Key, p.FetchURL, p.URL} {
		key := raw
		if u, e := url.Parse(raw); e == nil && u.Query().Get("id") != "" {
			key = u.Query().Get("id")
		}
		key = resolutionSuffix.ReplaceAllString(key, "")
		if photoKeyPattern.MatchString(key) && key != "0" {
			return key, nil
		}
	}
	return "", fmt.Errorf("invalid Bing image identifier for photo %d", p.ID)
}

func imageURL(base string, p Photo, width, height int) (string, error) {
	key, err := photoKey(p)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(base, "/") + "/th?" + url.Values{"id": {fmt.Sprintf("%s_%dx%d.jpg", key, width, height)}}.Encode(), nil
}

type bingImage struct {
	StartDate     string `json:"startdate"`
	EndDate       string `json:"enddate"`
	FullStartDate string `json:"fullstartdate"`
	URL           string `json:"url"`
	URLBase       string `json:"urlbase"`
	Copyright     string `json:"copyright"`
	CopyrightLink string `json:"copyrightlink"`
	Hash          string `json:"hsh"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	Attribute     string `json:"attribute"`
	Country       string `json:"country"`
	City          string `json:"city"`
	Longitude     string `json:"longitude"`
	Latitude      string `json:"latitude"`
	Continent     string `json:"continent"`
}

type Syncer struct {
	cfg    Config
	store  *Store
	client *http.Client
	mu     sync.Mutex
}

func (s *Syncer) Sync(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	endpoint := s.cfg.BingURL + "/HPImageArchive.aspx?" + url.Values{"format": {"js"}, "idx": {"0"}, "n": {"8"}, "mkt": {s.cfg.Market}}.Encode()
	var payload struct {
		Images []bingImage `json:"images"`
	}
	var err error
	for attempt := range 3 {
		err = s.fetch(ctx, endpoint, &payload)
		if err == nil {
			break
		}
		if attempt == 2 {
			return 0, err
		}
		t := time.NewTimer(time.Duration(attempt+1) * time.Second)
		select {
		case <-ctx.Done():
			t.Stop()
			return 0, ctx.Err()
		case <-t.C:
		}
	}
	if len(payload.Images) == 0 {
		return 0, fmt.Errorf("Bing returned no images")
	}
	added := 0
	// Oldest first keeps descending ID order consistent with descending publication dates.
	for i := len(payload.Images) - 1; i >= 0; i-- {
		b := payload.Images[i]
		if _, e := time.Parse("20060102", b.EndDate); e != nil {
			return added, fmt.Errorf("invalid Bing enddate %q", b.EndDate)
		}
		if _, e := time.Parse("20060102", b.StartDate); e != nil {
			return added, fmt.Errorf("invalid Bing startdate %q", b.StartDate)
		}
		p := Photo{Title: b.Title, Description: b.Description, Attribute: b.Attribute, Copyright: b.Copyright, CopyrightLink: b.CopyrightLink, StartDate: b.StartDate, EndDate: b.EndDate, FullStartDate: b.FullStartDate, URLBase: b.URLBase, Hash: b.Hash, Longitude: b.Longitude, Latitude: b.Latitude, City: b.City, Country: b.Country, Continent: b.Continent, Market: s.cfg.Market}
		p.Key, err = photoKey(p)
		if err != nil {
			return added, err
		}
		p.URL, err = imageURL(s.cfg.BingURL, p, 1920, 1080)
		if err != nil {
			return added, err
		}
		p.FetchURL = p.URL
		ok, e := s.store.insert(ctx, p)
		if e != nil {
			return added, e
		}
		if ok {
			added++
		}
	}
	return added, nil
}

func (s *Syncer) fetch(ctx context.Context, endpoint string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "BingWallpaper-Go/1.0")
	res, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Bing archive HTTP %d", res.StatusCode)
	}
	if err = json.NewDecoder(io.LimitReader(res.Body, 2<<20)).Decode(dst); err != nil {
		return fmt.Errorf("decode Bing archive: %w", err)
	}
	return nil
}

func (s *Syncer) Job(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	n, err := s.Sync(ctx)
	if err != nil {
		slog.Error("Bing sync failed", "inserted", n, "error", err)
	} else {
		slog.Info("Bing sync completed", "inserted", n)
	}
	if err != nil || n > 0 {
		if e := notify(s.cfg, fmt.Sprintf("Bing 同步：新增 %d 条", n), fmt.Sprintf("新增记录：%d\n错误：%v", n, err)); e != nil {
			slog.Error("notification failed", "error", e)
		}
	}
}
