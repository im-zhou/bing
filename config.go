package main

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"go.yaml.in/yaml/v3"
)

type Config struct {
	Listen      string `yaml:"listen"`
	RootURL     string `yaml:"root_url"`
	BingURL     string `yaml:"bing_url"`
	Market      string `yaml:"market"`
	Timezone    string `yaml:"timezone"`
	SyncCron    string `yaml:"sync_cron"`
	SyncOnStart bool   `yaml:"sync_on_start"`
	MySQL       struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Database string `yaml:"database"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		MaxOpen  int    `yaml:"max_open"`
		MaxIdle  int    `yaml:"max_idle"`
	} `yaml:"mysql"`
	SMTP struct {
		Address  string `yaml:"address"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		From     string `yaml:"from"`
		To       string `yaml:"to"`
	} `yaml:"smtp"`
	Weibo struct {
		ClientID      string `yaml:"client_id"`
		ClientSecret  string `yaml:"client_secret"`
		SessionSecret string `yaml:"session_secret"`
	} `yaml:"weibo"`
}

func loadConfig(path string) (Config, error) {
	var c Config
	c.Listen, c.BingURL, c.Market, c.Timezone, c.SyncCron = ":12345", "https://www.bing.com", "zh-CN", "Asia/Shanghai", "0 1 * * *"
	c.SyncOnStart = true
	c.MySQL.Host, c.MySQL.Port, c.MySQL.MaxOpen, c.MySQL.MaxIdle = "127.0.0.1", 3306, 10, 5
	f, err := os.Open(path)
	if err != nil {
		return c, err
	}
	defer f.Close()
	d := yaml.NewDecoder(io.LimitReader(f, 1<<20))
	d.KnownFields(true)
	if err = d.Decode(&c); err != nil {
		return c, err
	}
	if d.Decode(new(any)) != io.EOF {
		return c, fmt.Errorf("config must contain one YAML document")
	}
	if password, ok := os.LookupEnv("BING_MYSQL_PASSWORD"); ok {
		c.MySQL.Password = password
	}
	if c.Listen == "" || c.MySQL.Database == "" || c.MySQL.User == "" || c.Market == "" {
		return c, fmt.Errorf("listen, mysql.database, mysql.user and market are required")
	}
	if c.MySQL.Port < 1 || c.MySQL.Port > 65535 || c.MySQL.MaxOpen < 1 || c.MySQL.MaxIdle < 0 || c.MySQL.MaxIdle > c.MySQL.MaxOpen {
		return c, fmt.Errorf("invalid mysql connection settings")
	}
	for _, raw := range []string{c.BingURL, c.RootURL} {
		if raw == "" {
			continue
		}
		u, e := url.Parse(raw)
		if e != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
			return c, fmt.Errorf("invalid base URL: %q", raw)
		}
	}
	if c.BingURL == "" {
		return c, fmt.Errorf("bing_url is required")
	}
	if _, err = time.LoadLocation(c.Timezone); err != nil {
		return c, err
	}
	if c.Weibo.ClientID != "" && (c.Weibo.ClientSecret == "" || len(c.Weibo.SessionSecret) < 32 || c.RootURL == "") {
		return c, fmt.Errorf("weibo needs client_secret, root_url and session_secret (32+ characters)")
	}
	c.BingURL, c.RootURL = strings.TrimRight(c.BingURL, "/"), strings.TrimRight(c.RootURL, "/")
	return c, nil
}

func (c Config) dsn() string {
	m := mysql.NewConfig()
	m.User, m.Passwd, m.Net, m.Addr, m.DBName = c.MySQL.User, c.MySQL.Password, "tcp", net.JoinHostPort(c.MySQL.Host, strconv.Itoa(c.MySQL.Port)), c.MySQL.Database
	m.Timeout, m.ReadTimeout, m.WriteTimeout = 5*time.Second, 10*time.Second, 10*time.Second
	m.Params = map[string]string{"charset": "utf8mb4"}
	return m.FormatDSN()
}
