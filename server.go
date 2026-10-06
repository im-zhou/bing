package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type App struct {
	cfg        Config
	store      *Store
	client     *http.Client
	templates  *template.Template
	loc        *time.Location
	imageSlots chan struct{}
}

type photoView struct {
	Photo
	Slug, Date, Small, Middle, Large, Share, Location string
}
type pageView struct {
	Photos                                        []photoView
	Title, Section, Base, Query, PrevURL, NextURL string
	Current, Prev, Next, Pages                    int
}

func (a *App) view(p Photo) photoView {
	slug := p.Key
	if !photoKeyPattern.MatchString(slug) || slug == "0" {
		slug = strconv.FormatInt(p.ID, 10)
	}
	small, _ := imageURL(a.cfg.BingURL, p, 400, 240)
	middle, _ := imageURL(a.cfg.BingURL, p, 800, 480)
	large, _ := imageURL(a.cfg.BingURL, p, 1920, 1080)
	date := p.EndDate
	if d, e := time.Parse("20060102", date); e == nil {
		date = d.Format("2006-01-02")
	}
	locations := []string{}
	for _, s := range []string{p.Continent, p.Country, p.City} {
		if s != "" {
			locations = append(locations, s)
		}
	}
	share := "https://service.weibo.com/share/share.php?" + url.Values{"url": {a.cfg.RootURL + "/photo/" + slug}, "pic": {large}, "title": {date + " / " + p.Copyright}}.Encode()
	return photoView{Photo: p, Slug: slug, Date: date, Small: small, Middle: middle, Large: large, Share: share, Location: strings.Join(locations, ", ")}
}

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) { a.list(w, r, false) })
	mux.HandleFunc("GET /ranking", func(w http.ResponseWriter, r *http.Request) { a.list(w, r, true) })
	mux.HandleFunc("GET /photo/{photo}", a.photo)
	mux.HandleFunc("GET /photo/{$}", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/", http.StatusFound) })
	for _, path := range []string{"/v1", "/v1/{$}", "/v1/rand", "/v1/blur"} {
		mux.HandleFunc("GET "+path, a.api)
		mux.HandleFunc("POST "+path, a.api)
	}
	static, _ := fs.Sub(assets, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(static))))
	mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = "/images/bing.ico"
		http.FileServer(http.FS(static)).ServeHTTP(w, r)
	})
	mux.HandleFunc("GET /about.html", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/about.html", http.StatusFound)
	})
	mux.HandleFunc("GET /robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, "User-agent: *\nAllow: /\n")
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := a.store.db.PingContext(r.Context()); err != nil {
			a.failure(w, err)
			return
		}
		writeJSON(w, 200, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /weibo", a.weiboLogin)
	mux.HandleFunc("GET /weibo/{$}", a.weiboLogin)
	mux.HandleFunc("GET /weibo/callback", a.weiboCallback)
	mux.HandleFunc("GET /weibo/shorten", a.weiboShorten)
	// Keep the old path usable as a confirmation page; publish only on an explicit POST.
	mux.HandleFunc("GET /weibo/send", a.weiboSendConfirm)
	mux.HandleFunc("POST /weibo/send", a.weiboSend)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { http.Error(w, "页面不存在", http.StatusNotFound) })
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				slog.Error("request panic", "error", e)
				http.Error(w, "服务器错误", 500)
			}
		}()
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func (a *App) list(w http.ResponseWriter, r *http.Request, ranking bool) {
	page, err := positiveInt(r.URL.Query().Get("p"), 1, 1000000)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if utf8.RuneCountInString(query) > 200 {
		http.Error(w, "搜索条件最多 200 个字符", 400)
		return
	}
	photos, count, page, err := a.store.list(r.Context(), strings.ToLower(a.cfg.Market), time.Now().In(a.loc).Format("20060102"), query, page, ranking)
	if err != nil {
		a.failure(w, err)
		return
	}
	v := pageView{Title: "必应每日高清壁纸", Section: "home", Base: "/", Query: query, Current: page, Prev: max(1, page-1), Pages: max(1, (count+11)/12)}
	v.Next = min(v.Pages, page+1)
	if ranking {
		v.Title = "下载榜 | 必应每日高清壁纸"
		v.Section = "ranking"
		v.Base = "/ranking"
	}
	pageURL := func(n int) string {
		values := url.Values{"p": {strconv.Itoa(n)}}
		if query != "" {
			values.Set("q", query)
		}
		return v.Base + "?" + values.Encode()
	}
	v.PrevURL, v.NextURL = pageURL(v.Prev), pageURL(v.Next)
	for _, p := range photos {
		v.Photos = append(v.Photos, a.view(p))
	}
	a.render(w, "index.html", v)
}

func (a *App) photo(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("photo")
	force := r.URL.Query().Get("force")
	if force == "like" {
		a.like(w, r, key)
		return
	}
	if !photoKeyPattern.MatchString(key) {
		http.Error(w, "无效图片标识", 400)
		return
	}
	p, err := a.store.find(r.Context(), key)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		a.failure(w, err)
		return
	}
	if force == "download" {
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": key + "_1920x1080.jpg"}))
		if a.streamImage(w, r, p, 1920, 1080) {
			if _, err = a.store.increment(r.Context(), p.ID, "downloads"); err != nil {
				slog.Error("download counter", "error", err)
			}
		}
		return
	}
	if _, err = a.store.increment(r.Context(), p.ID, "views"); err != nil {
		a.failure(w, err)
		return
	}
	p.Views++
	back := "/"
	backParams := url.Values{}
	parts := strings.Split(force, "_")
	if len(parts) == 2 {
		if n, e := positiveInt(parts[1], 1, 1000000); e == nil {
			if parts[0] == "ranking" {
				back = "/ranking"
			}
			backParams.Set("p", strconv.Itoa(n))
		}
	}
	if q := strings.TrimSpace(r.URL.Query().Get("q")); q != "" && utf8.RuneCountInString(q) <= 200 {
		backParams.Set("q", q)
	}
	if len(backParams) > 0 {
		back += "?" + backParams.Encode()
	}
	a.render(w, "detail.html", struct {
		Doc  photoView
		Back string
	}{a.view(p), back})
}

func (a *App) like(w http.ResponseWriter, r *http.Request, id string) {
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil || n < 1 {
		http.Error(w, "无效图片 ID", 400)
		return
	}
	var liked []string
	if c, e := r.Cookie("likes"); e == nil {
		liked = strings.FieldsFunc(c.Value, func(r rune) bool { return r == '_' || r == ',' })
	}
	for _, existing := range liked {
		if existing == id {
			writeJSON(w, 200, map[string]any{"code": 200, "msg": ""})
			return
		}
	}
	ok, err := a.store.increment(r.Context(), n, "likes")
	if err != nil {
		a.failure(w, err)
		return
	}
	if !ok {
		http.NotFound(w, r)
		return
	}
	if len(liked) > 200 {
		liked = liked[len(liked)-200:]
	}
	liked = append(liked, id)
	http.SetCookie(w, &http.Cookie{Name: "likes", Value: strings.Join(liked, "_"), Path: "/", MaxAge: 730 * 86400, SameSite: http.SameSiteLaxMode, Secure: strings.HasPrefix(a.cfg.RootURL, "https://")})
	writeJSON(w, 200, map[string]any{"code": 200, "msg": ""})
}

var callbackPattern = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*(\.[A-Za-z_$][A-Za-z0-9_$]*)*$`)

func params(w http.ResponseWriter, r *http.Request) (url.Values, error) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	values := url.Values{}
	ct, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if r.Method == http.MethodPost && ct == "application/json" {
		var body map[string]json.RawMessage
		d := json.NewDecoder(r.Body)
		if err := d.Decode(&body); err != nil {
			return nil, fmt.Errorf("无效 JSON")
		}
		if d.Decode(new(any)) != io.EOF {
			return nil, fmt.Errorf("无效 JSON")
		}
		for k, v := range body {
			var s string
			if json.Unmarshal(v, &s) != nil {
				var n json.Number
				if json.Unmarshal(v, &n) != nil {
					return nil, fmt.Errorf("参数 %s 必须为字符串或数字", k)
				}
				s = n.String()
			}
			values.Set(k, s)
		}
	} else if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			return nil, err
		}
		values = r.PostForm
	}
	for k, v := range r.URL.Query() {
		values[k] = v
	}
	return values, nil
}

func positiveInt(raw string, fallback, limit int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	n, e := strconv.Atoi(raw)
	if e != nil || n < 1 || n > limit {
		return 0, fmt.Errorf("参数必须为 1 到 %d 的整数", limit)
	}
	return n, nil
}

func (a *App) api(w http.ResponseWriter, r *http.Request) {
	q, err := params(w, r)
	if err != nil {
		a.apiError(w, 400, err.Error(), "")
		return
	}
	cb := q.Get("callback")
	if len(cb) > 128 || (cb != "" && !callbackPattern.MatchString(cb)) {
		a.apiError(w, 400, "无效 callback", "")
		return
	}
	width, err := positiveInt(q.Get("w"), 1366, 3840)
	if err != nil {
		a.apiError(w, 400, err.Error(), cb)
		return
	}
	height, err := positiveInt(q.Get("h"), 768, 3840)
	if err != nil || width*height > 8294400 {
		a.apiError(w, 400, "无效分辨率，最大 8294400 像素", cb)
		return
	}
	date := ""
	random := r.URL.Path == "/v1/rand"
	if raw := q.Get("d"); raw != "" && !random {
		days, e := strconv.Atoi(raw)
		if e != nil || days < 0 || days > 36500 {
			a.apiError(w, 400, "d 必须为 0 到 36500 的整数", cb)
			return
		}
		date = time.Now().In(a.loc).AddDate(0, 0, -days).Format("20060102")
	}
	p, err := a.store.selectAPI(r.Context(), strings.ToLower(a.cfg.Market), time.Now().In(a.loc).Format("20060102"), date, random)
	if errors.Is(err, sql.ErrNoRows) {
		a.apiError(w, 404, "暂无对应日期的图片", cb)
		return
	}
	if err != nil {
		slog.Error("api query", "error", err)
		a.apiError(w, 500, "数据库查询失败", cb)
		return
	}
	if r.URL.Path != "/v1/blur" && (q.Get("type") == "json" || cb != "") {
		u, e := imageURL(a.cfg.BingURL, p, width, height)
		if e != nil {
			a.apiError(w, 502, e.Error(), cb)
			return
		}
		thumb, _ := imageURL(a.cfg.BingURL, p, 400, 240)
		middle, _ := imageURL(a.cfg.BingURL, p, 800, 480)
		original, _ := imageURL(a.cfg.BingURL, p, 1920, 1080)
		a.apiJSON(w, 200, map[string]any{"data": map[string]string{"enddate": p.EndDate, "url": u, "bmiddle_pic": middle, "original_pic": original, "thumbnail_pic": thumb}, "status": map[string]any{"code": 200, "message": ""}}, cb)
		return
	}
	if r.URL.Path != "/v1/blur" && q.Get("type") == "redirect" {
		u, e := imageURL(a.cfg.BingURL, p, width, height)
		if e != nil {
			a.apiError(w, 502, e.Error(), cb)
			return
		}
		http.Redirect(w, r, u, 302)
		return
	}
	native := nativeResolution(width, height)
	if r.URL.Path == "/v1/blur" || !native {
		radius := 0
		if r.URL.Path == "/v1/blur" {
			radius = 10
			if q.Get("r") != "" {
				n, e := strconv.Atoi(q.Get("r"))
				if e != nil {
					a.apiError(w, 400, "r 必须为整数", cb)
					return
				}
				radius = max(1, min(50, n))
			}
		}
		if width*height > 4000000 {
			a.apiError(w, 400, "本地图片处理最多支持 4000000 像素", cb)
			return
		}
		a.transformedImage(w, r, p, width, height, radius)
		return
	}
	a.streamImage(w, r, p, width, height)
}

func (a *App) apiJSON(w http.ResponseWriter, status int, payload any, callback string) {
	if callback == "" {
		writeJSON(w, status, payload)
		return
	}
	b, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "编码失败", 500)
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.WriteHeader(status)
	_, _ = fmt.Fprintf(w, "%s(%s);", callback, b)
}
func (a *App) apiError(w http.ResponseWriter, status int, message, callback string) {
	w.Header().Del("Content-Disposition")
	a.apiJSON(w, status, map[string]any{"data": map[string]any{}, "status": map[string]any{"code": status, "message": message}}, callback)
}
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
func (a *App) failure(w http.ResponseWriter, err error) {
	slog.Error("request failed", "error", err)
	http.Error(w, "服务器错误，请稍后重试", 500)
}
func (a *App) render(w http.ResponseWriter, name string, data any) {
	var buf bytes.Buffer
	if err := a.templates.ExecuteTemplate(&buf, name, data); err != nil {
		a.failure(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = buf.WriteTo(w)
}

func (a *App) fetchImage(r *http.Request, p Photo, width, height int) (*http.Response, error) {
	u, err := imageURL(a.cfg.BingURL, p, width, height)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "BingWallpaper-Go/1.0")
	res, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		res.Body.Close()
		return nil, fmt.Errorf("Bing 图片 HTTP %d", res.StatusCode)
	}
	if !strings.HasPrefix(res.Header.Get("Content-Type"), "image/") || res.ContentLength > 20<<20 {
		res.Body.Close()
		return nil, fmt.Errorf("Bing 返回无效图片响应")
	}
	return res, nil
}

func (a *App) streamImage(w http.ResponseWriter, r *http.Request, p Photo, width, height int) bool {
	res, err := a.fetchImage(r, p, width, height)
	if err != nil {
		a.apiError(w, 502, err.Error(), "")
		return false
	}
	defer res.Body.Close()
	w.Header().Set("Content-Type", res.Header.Get("Content-Type"))
	w.Header().Set("Cache-Control", "public, max-age=3600")
	if res.ContentLength >= 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(res.ContentLength, 10))
	}
	n, err := io.Copy(w, io.LimitReader(res.Body, (20<<20)+1))
	if err != nil || n > 20<<20 {
		slog.Warn("image stream interrupted", "bytes", n, "error", err)
		return false
	}
	return true
}
