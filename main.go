package main

import (
	"context"
	"database/sql"
	"embed"
	"flag"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/robfig/cron/v3"
)

//go:embed static templates
var assets embed.FS

func main() {
	if err := run(); err != nil {
		slog.Error("bing stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "config.yaml", "configuration file")
	syncOnly := flag.Bool("sync-once", false, "synchronize metadata and exit")
	flag.Parse()
	cfg, err := loadConfig(*configPath)
	if err != nil {
		return err
	}
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return err
	}
	db, err := sql.Open("mysql", cfg.dsn())
	if err != nil {
		return err
	}
	defer db.Close()
	db.SetMaxOpenConns(cfg.MySQL.MaxOpen)
	db.SetMaxIdleConns(cfg.MySQL.MaxIdle)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(time.Minute)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	ping, cancel := context.WithTimeout(ctx, 10*time.Second)
	err = db.PingContext(ping)
	cancel()
	if err != nil {
		return fmt.Errorf("connect mysql: %w", err)
	}
	client := newHTTPClient()
	store := &Store{db: db}
	syncer := &Syncer{cfg: cfg, store: store, client: client}
	if *syncOnly {
		job, c := context.WithTimeout(ctx, 2*time.Minute)
		defer c()
		n, e := syncer.Sync(job)
		slog.Info("sync-once completed", "inserted", n)
		return e
	}
	tmpl, err := template.ParseFS(assets, "templates/*.html")
	if err != nil {
		return err
	}
	app := &App{cfg: cfg, store: store, client: client, templates: tmpl, loc: loc, imageSlots: make(chan struct{}, 4)}
	scheduler := cron.New(cron.WithLocation(loc), cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger), cron.Recover(cron.DefaultLogger)))
	if cfg.SyncCron != "" {
		if _, err = scheduler.AddFunc(cfg.SyncCron, func() { syncer.Job(ctx) }); err != nil {
			return fmt.Errorf("sync_cron: %w", err)
		}
	}
	scheduler.Start()
	var startup sync.WaitGroup
	if cfg.SyncOnStart {
		startup.Add(1)
		go func() { defer startup.Done(); syncer.Job(ctx) }()
	}
	srv := &http.Server{Addr: cfg.Listen, Handler: app.routes(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 45 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	errCh := make(chan error, 1)
	go func() { slog.Info("listening", "address", cfg.Listen); errCh <- srv.ListenAndServe() }()
	select {
	case <-ctx.Done():
	case err = <-errCh:
		if err == http.ErrServerClosed {
			err = nil
		}
	}
	stop()
	shutdown, c := context.WithTimeout(context.Background(), 15*time.Second)
	defer c()
	if e := srv.Shutdown(shutdown); e != nil {
		_ = srv.Close()
		if err == nil {
			err = e
		}
	}
	<-scheduler.Stop().Done()
	startup.Wait()
	return err
}

func newHTTPClient() *http.Client {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 100
	t.MaxIdleConnsPerHost = 20
	t.MaxConnsPerHost = 32
	t.ResponseHeaderTimeout = 10 * time.Second
	return &http.Client{Transport: t, Timeout: 30 * time.Second}
}
