package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"

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
	cfg := config.Load()
	templates := mode.NewTemplates(cfg.ModeTemplatesDir)

	pool, err := db.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Legacy reflex.db, browsed read-only by the "Anansi" pages.
	reflexDB, err := db.OpenSQLite(cfg.ReflexDBPath)
	if err != nil {
		slog.Error("reflex sqlite open failed", "path", cfg.ReflexDBPath, "err", err)
		os.Exit(1)
	}
	defer reflexDB.Close()

	st := store.New(pool)
	rs := store.NewReflexStore(reflexDB)
	// Legacy file-based app, browsed read-only by the "Charlotte" pages via charlotte-cli.
	cs := store.NewCharlotteStore(cfg.CharlotteCLI)
	sess := auth.NewSessions(cfg.SessionSecret, cfg.SessionSecure)
	runner := blockrun.New(st, blockrun.LLM{}, templates)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.New(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: true,
	}).Handler)

	r.Route("/api", func(r chi.Router) {
		r.Get("/healthz", handler.Health(pool))

		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", handler.Register(st, sess))
			r.Post("/login", handler.Login(st, sess))
			r.Post("/logout", handler.Logout(sess))
			r.Get("/me", handler.Me(st, sess))
			r.Put("/me", handler.UpdateMe(st, sess))
		})

		r.Get("/modes", handler.ListModes())
		r.Get("/models", handler.ListModels())
		r.Get("/public/documents/{id}/attributes/{key}", handler.PublicDocumentAttribute(st))

		// Everything below requires a logged-in session; the middleware rejects
		// anonymous requests and puts the caller's user id in the context.
		r.Group(func(r chi.Router) {
			r.Use(handler.RequireUser(sess))

			r.Route("/labels", func(r chi.Router) {
				r.Get("/", handler.ListLabels(st))
				r.Put("/", handler.RenameLabel(st))
				r.Delete("/", handler.DeleteLabel(st))
			})
			r.Get("/attribute-values", handler.ListAttributeValues(st))

			r.Route("/anansi/documents", func(r chi.Router) {
				r.Get("/", handler.ListLegacyDocuments(rs))
				r.Get("/{id}", handler.GetLegacyDocument(rs))
				r.Post("/{id}/import", handler.ImportLegacyDocument(rs, st))
			})
			r.Route("/charlotte/documents", func(r chi.Router) {
				r.Get("/", handler.ListLegacyDocuments(cs))
				r.Get("/{id}", handler.GetLegacyDocument(cs))
				r.Post("/{id}/import", handler.ImportLegacyDocument(cs, st))
			})
			redditClient := reddit.NewClient(cfg.RedditClientID, cfg.RedditClientSecret, cfg.RedditUsername, cfg.RedditPassword)
			r.Get("/reddit/posts", handler.ListRedditPosts(st, redditClient))
			r.Get("/reddit/post", handler.GetRedditPost(redditClient))

			r.Route("/documents", func(r chi.Router) {
				r.Get("/", handler.ListDocuments(st))
				r.Post("/", handler.CreateDocument(st))
				r.Get("/search", handler.SearchDocuments(st))

				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", handler.GetDocument(st))
					r.Put("/", handler.UpdateDocument(st))
					r.Delete("/", handler.DeleteDocument(st))
					r.Post("/suggest-labels", handler.SuggestDocumentLabels(st))
					r.Post("/recommend-labels", handler.RecommendDocumentLabels(st))
					r.Post("/reddit-submit", handler.SubmitRedditPost(st, redditClient))

					r.Route("/blocks", func(r chi.Router) {
						r.Post("/", handler.CreateBlock(st))
						r.Put("/{blockId}", handler.UpdateBlock(st))
						r.Delete("/{blockId}", handler.DeleteBlock(st))
						r.Post("/{blockId}/copy", handler.CopyBlock(st))
						r.Post("/{blockId}/run", handler.RunBlock(st, runner))
						r.Post("/{blockId}/run/stream", handler.RunBlockStream(st, runner))
						r.Put("/{blockId}/responses/{responseId}", handler.UpdateResponse(st, runner))
						r.Post("/{blockId}/responses/{responseId}/reparse", handler.ReparseResponse(st, runner))
					})
				})
			})
		})
	})

	slog.Info("server listening", "addr", cfg.Addr)

	// ReadHeaderTimeout closes the Slowloris vector (slow header trickling)
	// that gosec G114 flags; IdleTimeout reaps idle keep-alive connections.
	// ReadTimeout/WriteTimeout are intentionally left unset: they are absolute
	// deadlines on the whole request, and the block-run and NDJSON streaming
	// endpoints call the LLM (up to maxToolIterations tool round-trips), which
	// routinely runs far longer than any fixed cap.
	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
