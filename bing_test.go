package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
)

func TestLegacyImageURLs(t *testing.T) {
	for _, p := range []Photo{
		{URLBase: "/th?id=OHR.Test_ZH-CN123", Key: "legacy-slug", URL: "https://old-oss.example/OHR.Test_ZH-CN123_1920x1080.jpg"},
		{Key: "OHR.Test_ZH-CN123", URL: "https://old-oss.example/file.jpg"},
		{FetchURL: "https://www4.bing.com/th?id=OHR.Test_ZH-CN123_1920x1080.jpg&pid=hp"},
	} {
		u, e := imageURL("https://www.bing.com", p, 800, 480)
		if e != nil {
			t.Fatal(e)
		}
		parsed, _ := url.Parse(u)
		if parsed.Host != "www.bing.com" || parsed.Query().Get("id") != "OHR.Test_ZH-CN123_800x480.jpg" {
			t.Fatalf("unexpected URL %s", u)
		}
	}
	if _, e := imageURL("https://www.bing.com", Photo{URLBase: "https://attacker.invalid/file"}, 800, 480); e == nil {
		t.Fatal("invalid identifier accepted")
	}
}

func TestParamsAndJSONPValidation(t *testing.T) {
	r := httptest.NewRequest("POST", "/v1?w=640", strings.NewReader(`{"w":800,"h":"480","d":0}`))
	r.Header.Set("Content-Type", "application/json")
	q, e := params(httptest.NewRecorder(), r)
	if e != nil || q.Get("w") != "640" || q.Get("h") != "480" || q.Get("d") != "0" {
		t.Fatalf("%v %v", q, e)
	}
	a := &App{}
	for _, path := range []string{"/v1?callback=alert(1)", "/v1?w=0", "/v1?w=3840&h=3840", "/v1?d=-1"} {
		w := httptest.NewRecorder()
		a.api(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 400 {
			t.Fatalf("%s: %d", path, w.Code)
		}
	}
	r = httptest.NewRequest("POST", "/v1", strings.NewReader(`{"w":640} {}`))
	r.Header.Set("Content-Type", "application/json")
	if _, e = params(httptest.NewRecorder(), r); e == nil {
		t.Fatal("trailing JSON accepted")
	}
}

func TestResizeAndBlur(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 20, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 20; x++ {
			src.SetRGBA(x, y, color.RGBA{100, 150, 200, 255})
		}
	}
	dst := boxBlur(resizeCover(src, 13, 17), 10)
	if dst.Bounds().Dx() != 13 || dst.Bounds().Dy() != 17 {
		t.Fatal(dst.Bounds())
	}
	if got := dst.RGBAAt(6, 8); got != (color.RGBA{100, 150, 200, 255}) {
		t.Fatal(got)
	}
	impulse := image.NewRGBA(image.Rect(0, 0, 9, 9))
	impulse.SetRGBA(4, 4, color.RGBA{255, 255, 255, 255})
	blurred := boxBlur(impulse, 1)
	if blurred.RGBAAt(4, 4).R >= 255 || blurred.RGBAAt(4, 3).R == 0 {
		t.Fatal("blur did not spread impulse")
	}
}

func TestOAuthCookieIntegrity(t *testing.T) {
	a := &App{}
	a.cfg.RootURL = "https://bing.example"
	a.cfg.Weibo.SessionSecret = strings.Repeat("x", 32)
	w := httptest.NewRecorder()
	a.signedCookie(w, "bing_weibo", "12345", 1800)
	c := w.Result().Cookies()[0]
	if !c.Secure || !c.HttpOnly {
		t.Fatal("session cookie must be secure and HttpOnly")
	}
	r := httptest.NewRequest("GET", "/weibo", nil)
	r.AddCookie(c)
	if uid, e := a.cookieValue(r, "bing_weibo"); e != nil || uid != "12345" {
		t.Fatal(uid, e)
	}
	c.Value = "x" + c.Value
	r = httptest.NewRequest("GET", "/weibo", nil)
	r.AddCookie(c)
	if _, e := a.cookieValue(r, "bing_weibo"); e == nil {
		t.Fatal("tampered cookie accepted")
	}
}

func TestOriginalSchemaIntegration(t *testing.T) {
	dsn := os.Getenv("BING_TEST_DSN")
	if dsn == "" {
		t.Skip("set BING_TEST_DSN to an isolated MySQL/MariaDB admin server")
	}
	cfg, e := mysql.ParseDSN(dsn)
	if e != nil {
		t.Fatal(e)
	}
	cfg.DBName = ""
	cfg.MultiStatements = true
	admin, e := sql.Open("mysql", cfg.FormatDSN())
	if e != nil {
		t.Fatal(e)
	}
	defer admin.Close()
	name := fmt.Sprintf("bing_test_%d", time.Now().UnixNano())
	if _, e = admin.Exec("CREATE DATABASE " + name + " CHARACTER SET utf8"); e != nil {
		t.Fatal(e)
	}
	defer func() { _, _ = admin.Exec("DROP DATABASE " + name) }()
	cfg.DBName = name
	db, e := sql.Open("mysql", cfg.FormatDSN())
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	db.SetMaxOpenConns(16)
	schema, e := os.ReadFile("bing.sql")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(string(schema)); e != nil {
		t.Fatal(e)
	}
	var img bytes.Buffer
	if e = jpeg.Encode(&img, image.NewRGBA(image.Rect(0, 0, 32, 18)), nil); e != nil {
		t.Fatal(e)
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/th" {
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write(img.Bytes())
			return
		}
		if r.URL.Path != "/HPImageArchive.aspx" || r.URL.Query().Get("idx") != "0" || r.URL.Query().Get("n") != "8" {
			http.Error(w, "invalid archive query", 400)
			return
		}
		images := []bingImage{}
		for i := 8; i >= 1; i-- {
			images = append(images, bingImage{StartDate: fmt.Sprintf("202001%02d", i), EndDate: fmt.Sprintf("202001%02d", i+1), FullStartDate: fmt.Sprintf("202001%02d1600", i), URLBase: fmt.Sprintf("/th?id=OHR.Test%d_ZH-CN123", i), Title: `带"引号"的标题`, Copyright: "必应壁纸"})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"images": images})
	}))
	defer upstream.Close()
	appCfg := Config{BingURL: upstream.URL, Market: "zh-CN", RootURL: "https://bing.example"}
	store := &Store{db: db}
	ctx := context.Background()
	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s := &Syncer{cfg: appCfg, store: store, client: upstream.Client()}
			_, err := s.Sync(ctx)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if e = db.QueryRow(`SELECT COUNT(*) FROM bing`).Scan(&count); e != nil || count != 8 {
		t.Fatalf("duplicate sync: count=%d err=%v", count, e)
	}
	var engine string
	if e = db.QueryRow(`SELECT ENGINE FROM information_schema.TABLES WHERE TABLE_SCHEMA=? AND TABLE_NAME='bing'`, name).Scan(&engine); e != nil || engine != "MyISAM" {
		t.Fatalf("schema changed: %s %v", engine, e)
	}
	// Preserve an old permalink and stale OSS URL; all rendering must use Bing instead.
	if _, e = db.Exec(`UPDATE bing SET qiniu_url='old-permalink',url='https://old-oss.invalid/bing.jpg',likes=NULL,views=NULL,downloads=NULL WHERE id=8`); e != nil {
		t.Fatal(e)
	}
	// An ID gap must not make random selection fail.
	if _, e = db.Exec(`DELETE FROM bing WHERE id=3`); e != nil {
		t.Fatal(e)
	}
	loc, _ := time.LoadLocation("Asia/Shanghai")
	tmpl, e := template.ParseFS(assets, "templates/*.html")
	if e != nil {
		t.Fatal(e)
	}
	a := &App{cfg: appCfg, store: store, client: upstream.Client(), loc: loc, templates: tmpl, imageSlots: make(chan struct{}, 4)}
	handler := a.routes()
	request := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		return w
	}
	for _, path := range []string{"/", "/ranking?p=999", "/photo/old-permalink", "/v1?type=json", "/v1/?type=json", "/v1?callback=app.done", "/static/about.html", "/favicon.ico", "/healthz"} {
		w := request(path)
		if w.Code != 200 {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
		}
		if strings.Contains(w.Body.String(), "old-oss") {
			t.Fatalf("stale OSS URL in %s", path)
		}
	}
	for _, tc := range []struct {
		query string
		count int
	}{
		{"20200109", 1}, {"2020-01-09", 1}, {"2020/1/9", 1}, {"2020年1月9日", 1}, {"2020-01", 7},
		{"标题", 7}, {"Test8_ZH-CN123", 1}, {"OHR.Test8_ZH-CN123_1920x1080.jpg", 1}, {"old-permalink", 1},
		{"no-such-image", 0}, {"%", 0}, {"Test8_ZH_CN123", 0}, {"' OR 1=1 --", 0},
	} {
		w := request("/?q=" + url.QueryEscape(tc.query))
		if w.Code != 200 || strings.Count(w.Body.String(), `class="item"`) != tc.count {
			t.Fatalf("search %q: status=%d expected=%d body=%s", tc.query, w.Code, tc.count, w.Body.String())
		}
		if tc.count == 0 && !strings.Contains(w.Body.String(), "没有找到匹配的图片") {
			t.Fatal("missing no-results message")
		}
		if tc.count > 0 && !strings.Contains(w.Body.String(), "q="+url.QueryEscape(tc.query)) {
			t.Fatalf("query missing from pagination: %q", tc.query)
		}
	}
	// Search context survives the detail view's return link, including non-ASCII input.
	if w := request("/photo/old-permalink?force=home_1&q=" + url.QueryEscape("标题")); w.Code != 200 || !strings.Contains(w.Body.String(), "q="+url.QueryEscape("标题")) {
		t.Fatal("detail return lost search")
	}
	if w := request("/?q=" + url.QueryEscape(strings.Repeat("中", 201))); w.Code != 400 {
		t.Fatal("long search accepted")
	}
	if w := request("/photo/old-permalink?force=download"); w.Code != 200 || !strings.HasPrefix(w.Header().Get("Content-Disposition"), "attachment") {
		t.Fatal(w.Code, w.Body.String())
	}
	for i := 0; i < 20; i++ {
		w := request("/v1/rand?type=json")
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	for _, path := range []string{"/v1?type=stream", "/v1?type=stream&w=321&h=199", "/v1/blur?w=64&h=36&r=2"} {
		w := request(path)
		if w.Code != 200 {
			t.Fatal(path, w.Code, w.Body.String())
		}
		decoded, _, e := image.Decode(bytes.NewReader(w.Body.Bytes()))
		if e != nil {
			t.Fatal(e)
		}
		if path != "/v1?type=stream" && decoded.Bounds().Dx() != map[string]int{"/v1?type=stream&w=321&h=199": 321, "/v1/blur?w=64&h=36&r=2": 64}[path] {
			t.Fatal(decoded.Bounds())
		}
	}
	w := request("/photo/8?force=like")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	cookie := w.Result().Cookies()[0]
	r := httptest.NewRequest("GET", "/photo/8?force=like", nil)
	r.AddCookie(cookie)
	handler.ServeHTTP(httptest.NewRecorder(), r)
	var likes, views, downloads int
	if e = db.QueryRow(`SELECT likes,views,downloads FROM bing WHERE id=8`).Scan(&likes, &views, &downloads); e != nil || likes != 1 || views != 2 || downloads != 1 {
		t.Fatalf("counters: %d %d %d %v", likes, views, downloads, e)
	}
	if p, e := store.find(ctx, "old-permalink' OR 1=1 --"); e != sql.ErrNoRows {
		t.Fatalf("SQL injection: %v %v", p, e)
	}
	if _, e = db.Exec(`UPDATE bing SET qiniu_url=NULL WHERE id=8`); e != nil {
		t.Fatal(e)
	}
	if w := request("/photo/8"); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	for i := 1; i <= 16; i++ {
		_, e = store.insert(ctx, Photo{Title: "分页搜索", StartDate: fmt.Sprintf("202002%02d", i), EndDate: fmt.Sprintf("202002%02d", i+1), URLBase: fmt.Sprintf("/th?id=OHR.Page%d", i), Key: fmt.Sprintf("OHR.Page%d", i), Market: "zh-CN"})
		if e != nil {
			t.Fatal(e)
		}
	}
	if w := request("/?q=" + url.QueryEscape("分页搜索") + "&p=2"); w.Code != 200 || strings.Count(w.Body.String(), `class="item"`) != 4 || !strings.Contains(w.Body.String(), "2 / 2") {
		t.Fatal("filtered pagination failed", w.Code, w.Body.String())
	}
	if _, e = db.Exec(`DELETE FROM bing`); e != nil {
		t.Fatal(e)
	}
	if w := request("/"); w.Code != 200 || !strings.Contains(w.Body.String(), "暂无图片") {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := request("/v1?type=json"); w.Code != 404 {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestYAMLConfiguration(t *testing.T) {
	write := func(content string) string {
		p := filepath.Join(t.TempDir(), "config.yaml")
		if err := os.WriteFile(p, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	base := "# YAML comments are supported\nmysql:\n  database: bing\n  user: bing\n  password: 'a#b: c'\nsync_on_start: false\nsync_cron: ''\n"
	c, err := loadConfig(write(base))
	if err != nil {
		t.Fatal(err)
	}
	if c.Listen != ":12345" || c.Market != "zh-CN" || c.MySQL.Password != "a#b: c" || c.SyncOnStart || c.SyncCron != "" {
		t.Fatalf("incorrect YAML configuration: %+v", c)
	}
	for _, suffix := range []string{"unknown_setting: true\n", "---\nlisten: ':9999'\n", "listen: ':12345'\nlisten: ':54321'\n", "mysql: {database: bing, user: bing, unknown_setting: true}\n"} {
		if _, err = loadConfig(write(base + suffix)); err == nil {
			t.Fatalf("invalid configuration accepted: %s", suffix)
		}
	}
	t.Setenv("BING_MYSQL_PASSWORD", "env-password")
	if c, err = loadConfig(write(base)); err != nil || c.MySQL.Password != "env-password" {
		t.Fatal("environment override failed", err)
	}
	if _, err = loadConfig("config.example.yaml"); err != nil {
		t.Fatal("invalid example", err)
	}
}
