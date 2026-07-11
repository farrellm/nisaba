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

	"github.com/farrellm/nisaba/internal/auth"
	"github.com/farrellm/nisaba/internal/blockrun"
	"github.com/farrellm/nisaba/internal/config"
	"github.com/farrellm/nisaba/internal/db"
	"github.com/farrellm/nisaba/internal/handler"
	"github.com/farrellm/nisaba/internal/mode"
	"github.com/farrellm/nisaba/internal/reddit"
	"github.com/farrellm/nisaba/internal/store"
)

func main() {
	if err := run(context.Background()); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("database connect: %w", err)
	}
	defer pool.Close()

	// Legacy reflex.db, browsed read-only by the "Anansi" pages.
	reflexDB, err := db.OpenSQLite(cfg.ReflexDBPath)
	if err != nil {
		return fmt.Errorf("reflex sqlite open %q: %w", cfg.ReflexDBPath, err)
	}
	defer reflexDB.Close()

	st := store.New(pool)
	templates := mode.NewTemplates(cfg.ModeTemplatesDir)
	srv := &http.Server{
		Addr: cfg.Addr,
		Handler: handler.Routes(handler.Deps{
			Store:    st,
			Sessions: auth.NewSessions(cfg.SessionSecret, cfg.SessionSecure),
			DB:       pool,
			Reflex:   store.NewReflexStore(reflexDB),
			// Legacy file-based app, browsed read-only by the "Charlotte" pages
			// via charlotte-cli.
			Charlotte: store.NewCharlotteStore(cfg.CharlotteCLI),
			Reddit:    reddit.NewClient(cfg.RedditClientID, cfg.RedditClientSecret, cfg.RedditUsername, cfg.RedditPassword),
			Runner:    blockrun.New(st, blockrun.LLM{}, templates),
			CORS:      cfg.CORSOrigins,
		}),
		// ReadHeaderTimeout closes the Slowloris vector (slow header trickling)
		// that gosec G114 flags; IdleTimeout reaps idle keep-alive connections.
		// ReadTimeout/WriteTimeout are intentionally left unset: they are absolute
		// deadlines on the whole request, and the block-run and NDJSON streaming
		// endpoints call the LLM (up to maxToolIterations tool round-trips), which
		// routinely runs far longer than any fixed cap.
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	slog.Info("server listening", "addr", cfg.Addr)

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		// Give in-flight requests a moment to drain; a second signal
		// terminates the process the hard way.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("shutdown: %w", err)
		}
		return nil
	}
}
