// ocnews-backend: servicio Go con la News REST API v1.3 + fetcher de feeds.
// Arranque: config fail-fast → SQLite + migraciones → bootstrap admin →
// HTTP con auth Basic. Shutdown gracioso con SIGINT/SIGTERM.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gnacho/ocnews/backend/internal/api"
	"github.com/gnacho/ocnews/backend/internal/auth"
	"github.com/gnacho/ocnews/backend/internal/config"
	"github.com/gnacho/ocnews/backend/internal/feed"
	"github.com/gnacho/ocnews/backend/internal/favicon"
	"github.com/gnacho/ocnews/backend/internal/refresher"
	"github.com/gnacho/ocnews/backend/internal/scheduler"
	"github.com/gnacho/ocnews/backend/internal/store"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := newLogger(cfg.LogLevel)
	slog.SetDefault(log)

	st, err := store.Open(cfg.DBPath())
	if err != nil {
		return err
	}
	defer st.Close()

	if cfg.AuthUser != "" && cfg.AuthPass != "" {
		hash, err := auth.HashPassword(cfg.AuthPass)
		if err != nil {
			return err
		}
		created, err := st.BootstrapUser(cfg.AuthUser, hash)
		if err != nil {
			return err
		}
		if created {
			log.Info("usuario admin bootstrap", "username", cfg.AuthUser)
		}
	} else if cfg.AuthUser == "" || cfg.AuthPass == "" {
		var count int
		if err := st.BootstrapCount(&count); err != nil {
			return err
		}
		if count == 0 {
			return errors.New("sin usuarios en BD: define AUTH_USER y AUTH_PASS para el bootstrap del primer admin")
		}
	}

	favicons, err := favicon.NewCache(cfg.FaviconsDir(), log)
	if err != nil {
		return err
	}
	fetcher := feed.NewHTTPFetcher(cfg.FetchTimeout)
	refresh := refresher.New(st, fetcher, log, cfg.FeedInterval, cfg.MaxGap)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.NewServer(st, fetcher, refresh, favicons, cfg.Retention, log).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// scheduler: refresco periódico + retención; se drena al cancelar ctx
	sched := scheduler.New(st, refresh, favicons, log, 30*time.Second, 4, cfg.Retention)
	go func() {
		sched.Run(ctx)
	}()
	log.Info("scheduler arrancado", "interval", cfg.FeedInterval, "max_gap", cfg.MaxGap,
		"retencion", cfg.Retention.String())

	errCh := make(chan error, 1)
	go func() {
		log.Info("ocnews-backend escuchando", "addr", cfg.Addr, "db", cfg.DBPath(), "api", api.Base)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		log.Info("apagando (señal)…")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func newLogger(level string) *slog.Logger {
	var lv slog.Level
	switch level {
	case "debug":
		lv = slog.LevelDebug
	case "warn":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		lv = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lv}))
}
