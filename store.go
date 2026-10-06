package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"
)

type Photo struct {
	ID                                                              int64
	Title, Attribute, Description, Copyright, CopyrightLink         string
	StartDate, EndDate, FullStartDate, URL, URLBase, Hash, Key      string
	Longitude, Latitude, City, Country, Continent, Market, FetchURL string
	Likes, Views, Downloads                                         int64
}

// Nullable legacy columns are normalized in SELECT; no schema migration is required.
const photoColumns = `id, COALESCE(title,''), COALESCE(attribute,''), COALESCE(description,''),
copyright, copyrightlink, startdate, enddate, fullstartdate, url, urlbase, hsh,
COALESCE(qiniu_url,''), COALESCE(longitude,''), COALESCE(latitude,''), COALESCE(city,''),
COALESCE(country,''), COALESCE(continent,''), COALESCE(mkt,''), COALESCE(fetch_url,''),
COALESCE(likes,0), COALESCE(views,0), COALESCE(downloads,0)`

func scanPhoto(row interface{ Scan(...any) error }) (Photo, error) {
	var p Photo
	err := row.Scan(&p.ID, &p.Title, &p.Attribute, &p.Description, &p.Copyright, &p.CopyrightLink,
		&p.StartDate, &p.EndDate, &p.FullStartDate, &p.URL, &p.URLBase, &p.Hash, &p.Key,
		&p.Longitude, &p.Latitude, &p.City, &p.Country, &p.Continent, &p.Market, &p.FetchURL,
		&p.Likes, &p.Views, &p.Downloads)
	return p, err
}

type Store struct{ db *sql.DB }

const usablePhoto = `(NULLIF(urlbase,'') IS NOT NULL AND urlbase<>'0' OR NULLIF(qiniu_url,'') IS NOT NULL AND qiniu_url<>'0' OR NULLIF(url,'') IS NOT NULL AND url<>'0')`

func (s *Store) list(ctx context.Context, market, today, query string, page int, ranking bool) ([]Photo, int, int, error) {
	where := `LOWER(mkt) LIKE ? AND enddate<=? AND ` + usablePhoto
	args := []any{"%" + market + "%", today}
	if query != "" {
		// Bind values and escape LIKE wildcards so filenames containing '_' are literal.
		where += ` AND (title LIKE ? ESCAPE '!' OR copyright LIKE ? ESCAPE '!' OR description LIKE ? ESCAPE '!' OR qiniu_url LIKE ? ESCAPE '!' OR urlbase LIKE ? ESCAPE '!' OR url LIKE ? ESCAPE '!' OR enddate LIKE ? ESCAPE '!')`
		keyword := likePattern(query)
		filename := likePattern(resolutionSuffix.ReplaceAllString(query, ""))
		args = append(args, keyword, keyword, keyword, filename, filename, filename, likePattern(searchDate(query)))
	}
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM bing WHERE `+where, args...).Scan(&count); err != nil {
		return nil, 0, 0, err
	}
	pages := max(1, (count+11)/12)
	page = max(1, min(page, pages))
	order := "id DESC"
	if ranking {
		order = "downloads DESC, id DESC"
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+photoColumns+` FROM bing WHERE `+where+` ORDER BY `+order+` LIMIT ? OFFSET ?`, append(args, 12, (page-1)*12)...)
	if err != nil {
		return nil, 0, 0, err
	}
	defer rows.Close()
	photos := make([]Photo, 0, 12)
	for rows.Next() {
		p, e := scanPhoto(rows)
		if e != nil {
			return nil, 0, 0, e
		}
		photos = append(photos, p)
	}
	return photos, count, page, rows.Err()
}

func likePattern(value string) string {
	return "%" + strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(value) + "%"
}

func searchDate(value string) string {
	date := strings.NewReplacer("/", "-", ".", "-", "年", "-", "月", "-", "日", "").Replace(value)
	date = strings.TrimSuffix(date, "-")
	for _, layout := range []string{"2006-1-2", "2006-1"} {
		if parsed, err := time.Parse(layout, date); err == nil {
			if layout == "2006-1" {
				return parsed.Format("200601")
			}
			return parsed.Format("20060102")
		}
	}
	return value
}

func (s *Store) find(ctx context.Context, key string) (Photo, error) {
	if id, err := strconv.ParseInt(key, 10, 64); err == nil && id > 0 {
		return scanPhoto(s.db.QueryRowContext(ctx, `SELECT `+photoColumns+` FROM bing WHERE id=? ORDER BY id DESC LIMIT 1`, id))
	}
	return scanPhoto(s.db.QueryRowContext(ctx, `SELECT `+photoColumns+` FROM bing WHERE qiniu_url=? OR urlbase=? OR urlbase=? ORDER BY id DESC LIMIT 1`, key, key, "/th?id="+key))
}

func (s *Store) selectAPI(ctx context.Context, market, today, date string, random bool) (Photo, error) {
	where := `LOWER(mkt) LIKE ? AND enddate<=? AND ` + usablePhoto
	args := []any{"%" + market + "%", today}
	if date != "" {
		where += ` AND enddate=?`
		args = append(args, date)
	}
	if !random {
		return scanPhoto(s.db.QueryRowContext(ctx, `SELECT `+photoColumns+` FROM bing WHERE `+where+` ORDER BY id DESC LIMIT 1`, args...))
	}
	// Choose an actual row offset rather than assuming IDs are contiguous.
	var count int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM bing WHERE `+where, args...).Scan(&count); err != nil {
		return Photo{}, err
	}
	if count == 0 {
		return Photo{}, sql.ErrNoRows
	}
	n, err := rand.Int(rand.Reader, big.NewInt(count))
	if err != nil {
		return Photo{}, err
	}
	return scanPhoto(s.db.QueryRowContext(ctx, `SELECT `+photoColumns+` FROM bing WHERE `+where+` ORDER BY id DESC LIMIT 1 OFFSET ?`, append(args, n.Int64())...))
}

func (s *Store) increment(ctx context.Context, id int64, column string) (bool, error) {
	switch column {
	case "likes", "views", "downloads":
	default:
		return false, fmt.Errorf("invalid counter")
	}
	r, err := s.db.ExecContext(ctx, `UPDATE bing SET `+column+`=COALESCE(`+column+`,0)+1 WHERE id=?`, id)
	if err != nil {
		return false, err
	}
	n, err := r.RowsAffected()
	return n > 0, err
}

func (s *Store) insert(ctx context.Context, p Photo) (bool, error) {
	// GET_LOCK is held by a dedicated connection. Transactions cannot protect a MyISAM table.
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return false, err
	}
	defer conn.Close()
	var locked sql.NullInt64
	if err = conn.QueryRowContext(ctx, `SELECT GET_LOCK(CONCAT('bing-sync:',DATABASE()),10)`).Scan(&locked); err != nil {
		return false, err
	}
	if !locked.Valid || locked.Int64 != 1 {
		return false, fmt.Errorf("sync lock unavailable")
	}
	defer releaseLock(conn, "bing-sync:")
	var exists int
	err = conn.QueryRowContext(ctx, `SELECT 1 FROM bing WHERE enddate=? AND LOWER(mkt)=LOWER(?) LIMIT 1`, p.EndDate, p.Market).Scan(&exists)
	if err == nil {
		return false, nil
	}
	if err != sql.ErrNoRows {
		return false, err
	}
	_, err = conn.ExecContext(ctx, `INSERT INTO bing (title,attribute,description,copyright,copyrightlink,startdate,enddate,fullstartdate,url,urlbase,hsh,qiniu_url,longitude,latitude,city,country,continent,mkt,fetch_url) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.Title, p.Attribute, p.Description, p.Copyright, p.CopyrightLink, p.StartDate, p.EndDate, p.FullStartDate, p.URL, p.URLBase, p.Hash, p.Key, p.Longitude, p.Latitude, p.City, p.Country, p.Continent, p.Market, p.FetchURL)
	return err == nil, err
}

func releaseLock(conn *sql.Conn, prefix string) {
	cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := conn.ExecContext(cleanup, `DO RELEASE_LOCK(CONCAT(?,DATABASE()))`, prefix); err != nil {
		// Discard the physical connection; a pooled connection must not retain the lock.
		_ = conn.Raw(func(any) error { return driver.ErrBadConn })
	}
}
