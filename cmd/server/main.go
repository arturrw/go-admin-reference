// Command server runs the GoAdmin API and serves the embedded admin UI.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/arturrw/go-admin-reference/internal/auth"
	"github.com/arturrw/go-admin-reference/internal/config"
	"github.com/arturrw/go-admin-reference/internal/httpapi"
	"github.com/arturrw/go-admin-reference/internal/media"
	"github.com/arturrw/go-admin-reference/internal/reqlog"
	"github.com/arturrw/go-admin-reference/internal/seed"
	"github.com/arturrw/go-admin-reference/internal/store/memory"
	"github.com/arturrw/go-admin-reference/internal/store/postgres"
	"github.com/arturrw/go-admin-reference/web"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()
	logger := newLogger(cfg)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	uploads, err := media.NewStorage(cfg.UploadDir)
	if err != nil {
		return err
	}

	now := time.Now()
	requests := reqlog.New(500)
	requests.Seed(now, 120)

	store, sessions, cleanup, err := openStore(ctx, cfg, logger, now)
	if err != nil {
		return err
	}
	defer cleanup()

	srv := &http.Server{
		Addr: cfg.Addr,
		Handler: httpapi.New(httpapi.Deps{
			Logger:    logger,
			Store:     store,
			Requests:  requests,
			Sessions:  sessions,
			Media:     uploads,
			SPA:       web.Handler(),
			Version:   cfg.Version,
			Env:       cfg.Env,
			StartedAt: now,
			// Seeded accounts all share this password; shown on the dev login page.
			DemoPassword: seed.DemoPassword,
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	errc := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", cfg.Addr, "env", cfg.Env, "version", cfg.Version)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errc <- err
		}
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}

	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

// openStore picks Postgres when DATABASE_URL is set (migrating and seeding a
// fresh database) and falls back to the in-memory store otherwise.
func openStore(ctx context.Context, cfg config.Config, logger *slog.Logger, now time.Time) (httpapi.Store, httpapi.SessionStore, func(), error) {
	if cfg.DatabaseURL == "" {
		logger.Info("using in-memory store (set DATABASE_URL for Postgres)")
		return memory.New(now), auth.NewMemorySessions(cfg.SessionTTL), func() {}, nil
	}
	pool, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, nil, nil, err
	}
	applied, err := postgres.Migrate(ctx, pool)
	if err != nil {
		pool.Close()
		return nil, nil, nil, fmt.Errorf("migrate: %w", err)
	}
	if len(applied) > 0 {
		logger.Info("applied migrations", "files", applied)
	}
	if cfg.Seed {
		seeded, err := postgres.SeedIfEmpty(ctx, pool, now)
		if err != nil {
			pool.Close()
			return nil, nil, nil, fmt.Errorf("seed: %w", err)
		}
		if seeded {
			logger.Info("seeded empty database with demo data")
		}
	}

	sessions := postgres.NewSessions(pool, cfg.SessionTTL)
	// Purge expired sessions in the background.
	go func() {
		t := time.NewTicker(time.Hour)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if n, err := sessions.PurgeExpired(ctx); err == nil && n > 0 {
					logger.Info("purged expired sessions", "count", n)
				}
			}
		}
	}()
	logger.Info("using postgres store")
	return postgres.New(pool), sessions, pool.Close, nil
}

func newLogger(cfg config.Config) *slog.Logger {
	opts := &slog.HandlerOptions{Level: cfg.LogLevel}
	if cfg.IsDev() {
		return slog.New(slog.NewTextHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}
