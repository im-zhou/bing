package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func (a *App) signature(value string) string {
	m := hmac.New(sha256.New, []byte(a.cfg.Weibo.SessionSecret))
	_, _ = m.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}
func (a *App) signedCookie(w http.ResponseWriter, name, value string, age int) {
	payload := base64.RawURLEncoding.EncodeToString([]byte(value)) + "." + strconv.FormatInt(time.Now().Add(time.Duration(age)*time.Second).Unix(), 10)
	http.SetCookie(w, &http.Cookie{Name: name, Value: payload + "." + a.signature(payload), Path: "/weibo", HttpOnly: true, Secure: strings.HasPrefix(a.cfg.RootURL, "https://"), SameSite: http.SameSiteLaxMode, MaxAge: age})
}
func (a *App) cookieValue(r *http.Request, name string) (string, error) {
	c, err := r.Cookie(name)
	if err != nil {
		return "", err
	}
	parts := strings.Split(c.Value, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid cookie")
	}
	if !hmac.Equal([]byte(parts[2]), []byte(a.signature(parts[0]+"."+parts[1]))) {
		return "", fmt.Errorf("invalid signature")
	}
	expires, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || expires <= time.Now().Unix() {
		return "", fmt.Errorf("expired cookie")
	}
	b, err := base64.RawURLEncoding.DecodeString(parts[0])
	return string(b), err
}
func (a *App) weiboEnabled(w http.ResponseWriter) bool {
	if a.cfg.Weibo.ClientID == "" {
		http.Error(w, "微博功能未配置", http.StatusServiceUnavailable)
		return false
	}
	return true
}
func (a *App) weiboLogin(w http.ResponseWriter, r *http.Request) {
	if !a.weiboEnabled(w) {
		return
	}
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		a.failure(w, err)
		return
	}
	state := base64.RawURLEncoding.EncodeToString(b[:])
	a.signedCookie(w, "bing_oauth_state", state, 600)
	u := "https://api.weibo.com/oauth2/authorize?" + url.Values{"client_id": {a.cfg.Weibo.ClientID}, "redirect_uri": {a.cfg.RootURL + "/weibo/callback"}, "state": {state}}.Encode()
	http.Redirect(w, r, u, 302)
}

func (a *App) weiboPost(r *http.Request, path string, form url.Values, dst any) error {
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, "https://api.weibo.com"+path, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("Weibo HTTP %d", res.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	var upstream struct {
		Error string `json:"error"`
		Code  int    `json:"error_code"`
	}
	if err = json.Unmarshal(b, &upstream); err != nil {
		return err
	}
	if upstream.Code != 0 || upstream.Error != "" {
		return fmt.Errorf("Weibo error %d", upstream.Code)
	}
	return json.Unmarshal(b, dst)
}

func (a *App) weiboCallback(w http.ResponseWriter, r *http.Request) {
	if !a.weiboEnabled(w) {
		return
	}
	state, err := a.cookieValue(r, "bing_oauth_state")
	if err != nil || !hmac.Equal([]byte(state), []byte(r.URL.Query().Get("state"))) || r.URL.Query().Get("code") == "" {
		http.Error(w, "无效 OAuth 回调", 400)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "bing_oauth_state", Value: "", Path: "/weibo", MaxAge: -1, HttpOnly: true})
	var token struct {
		AccessToken string `json:"access_token"`
		Expires     int64  `json:"expires_in"`
		UID         string `json:"uid"`
	}
	err = a.weiboPost(r, "/oauth2/access_token", url.Values{"client_id": {a.cfg.Weibo.ClientID}, "client_secret": {a.cfg.Weibo.ClientSecret}, "grant_type": {"authorization_code"}, "code": {r.URL.Query().Get("code")}, "redirect_uri": {a.cfg.RootURL + "/weibo/callback"}}, &token)
	if err != nil {
		a.failure(w, err)
		return
	}
	if token.AccessToken == "" || token.UID == "" || token.Expires < 1 || token.Expires > 2147483647 {
		http.Error(w, "无效微博令牌", 502)
		return
	}
	// Existing bing_session.insertdate is INT: Unix seconds fit, JavaScript milliseconds do not.
	conn, err := a.store.db.Conn(r.Context())
	if err != nil {
		a.failure(w, err)
		return
	}
	defer conn.Close()
	var locked int
	if err = conn.QueryRowContext(r.Context(), `SELECT GET_LOCK(CONCAT('bing-oauth:',DATABASE()),10)`).Scan(&locked); err != nil || locked != 1 {
		http.Error(w, "授权保存繁忙", 503)
		return
	}
	defer releaseLock(conn, "bing-oauth:")
	var id int64
	err = conn.QueryRowContext(r.Context(), `SELECT id FROM bing_session WHERE uid=? ORDER BY id DESC LIMIT 1`, token.UID).Scan(&id)
	if err == sql.ErrNoRows {
		_, err = conn.ExecContext(r.Context(), `INSERT INTO bing_session (token,expires_in,insertdate,uid) VALUES (?,?,?,?)`, token.AccessToken, token.Expires, time.Now().Unix(), token.UID)
	} else if err == nil {
		_, err = conn.ExecContext(r.Context(), `UPDATE bing_session SET token=?,expires_in=?,insertdate=? WHERE uid=?`, token.AccessToken, token.Expires, time.Now().Unix(), token.UID)
	}
	if err != nil {
		a.failure(w, err)
		return
	}
	a.signedCookie(w, "bing_weibo", token.UID, min(int(token.Expires), 1800))
	http.Redirect(w, r, "/", 302)
}

func (a *App) weiboToken(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	if !a.weiboEnabled(w) {
		return "", "", false
	}
	uid, err := a.cookieValue(r, "bing_weibo")
	if err != nil {
		http.Redirect(w, r, "/weibo", 302)
		return "", "", false
	}
	var token string
	var inserted, expires int64
	err = a.store.db.QueryRowContext(r.Context(), `SELECT token,insertdate,expires_in FROM bing_session WHERE uid=? ORDER BY id DESC LIMIT 1`, uid).Scan(&token, &inserted, &expires)
	if err == sql.ErrNoRows {
		http.Redirect(w, r, "/weibo", 302)
		return "", "", false
	}
	if err != nil {
		a.failure(w, err)
		return "", "", false
	}
	if inserted+expires <= time.Now().Unix() {
		http.Redirect(w, r, "/weibo", 302)
		return "", "", false
	}
	return token, uid, true
}

func (a *App) weiboSendConfirm(w http.ResponseWriter, r *http.Request) {
	_, uid, ok := a.weiboToken(w, r)
	if !ok {
		return
	}
	a.render(w, "weibo.html", struct{ CSRF string }{a.signature("send:" + uid)})
}

func (a *App) weiboSend(w http.ResponseWriter, r *http.Request) {
	token, uid, ok := a.weiboToken(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := r.ParseForm(); err != nil || !hmac.Equal([]byte(r.PostForm.Get("csrf")), []byte(a.signature("send:"+uid))) {
		http.Error(w, "无效请求", 403)
		return
	}
	// Serialize publishing across instances without relying on MyISAM transactions.
	conn, err := a.store.db.Conn(r.Context())
	if err != nil {
		a.failure(w, err)
		return
	}
	defer conn.Close()
	var locked int
	if err = conn.QueryRowContext(r.Context(), `SELECT GET_LOCK(CONCAT('bing-weibo:',DATABASE()),10)`).Scan(&locked); err != nil || locked != 1 {
		http.Error(w, "微博发送繁忙", 503)
		return
	}
	defer releaseLock(conn, "bing-weibo:")
	p, err := scanPhoto(conn.QueryRowContext(r.Context(), `SELECT `+photoColumns+` FROM bing WHERE enddate=? AND LOWER(mkt)=LOWER(?) AND weibo=0 ORDER BY id DESC LIMIT 1`, time.Now().In(a.loc).Format("20060102"), a.cfg.Market))
	if err == sql.ErrNoRows {
		http.Error(w, "今日没有待发送壁纸", 404)
		return
	}
	if err != nil {
		a.failure(w, err)
		return
	}
	status := []rune("#必应壁纸# " + p.EndDate + " / " + p.Copyright)
	if len(status) > 110 {
		status = status[:110]
	}
	v := a.view(p)
	var result map[string]any
	err = a.weiboPost(r, "/2/statuses/share.json", url.Values{"access_token": {token}, "status": {string(status) + " " + a.cfg.RootURL + "/photo/" + v.Slug}}, &result)
	if err != nil {
		a.failure(w, err)
		return
	}
	if _, err = conn.ExecContext(r.Context(), `UPDATE bing SET weibo=1 WHERE id=?`, p.ID); err != nil {
		a.failure(w, err)
		return
	}
	http.Redirect(w, r, "/", 303)
}

func (a *App) weiboShorten(w http.ResponseWriter, r *http.Request) {
	token, _, ok := a.weiboToken(w, r)
	if !ok {
		return
	}
	raw := r.URL.Query().Get("url")
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		http.Error(w, "无效 URL", 400)
		return
	}
	var result map[string]any
	if err = a.weiboPost(r, "/2/short_url/shorten.json", url.Values{"access_token": {token}, "url_long": {raw}}, &result); err != nil {
		a.failure(w, err)
		return
	}
	writeJSON(w, 200, result)
}
