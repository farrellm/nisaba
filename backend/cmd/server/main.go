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
	"github.com/farrellm/nisaba/internal/config"
	"github.com/farrellm/nisaba/internal/db"
	"github.com/farrellm/nisaba/internal/handler"
	"github.com/farrellm/nisaba/internal/mode"
	"github.com/farrellm/nisaba/internal/store"
)

func main() {
	cfg := config.Load()
	mode.TemplatesBaseDir = cfg.ModeTemplatesDir

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
	// Mark the cookie Secure in production (HTTPS); SESSION_SECURE=true enables it.
	sess := auth.NewSessions(cfg.SessionSecret, os.Getenv("SESSION_SECURE") == "true")

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
		r.Route("/labels", func(r chi.Router) {
			r.Get("/", handler.ListLabels(st, sess))
			r.Put("/", handler.RenameLabel(st, sess))
			r.Delete("/", handler.DeleteLabel(st, sess))
		})
		r.Get("/attribute-values", handler.ListAttributeValues(st, sess))

		r.Route("/anansi/documents", func(r chi.Router) {
			r.Get("/", handler.ListReflexDocuments(rs, sess))
			r.Get("/{id}", handler.GetReflexDocument(rs, sess))
		})
		r.Route("/charlotte/documents", func(r chi.Router) {
			r.Get("/", handler.ListCharlotteDocuments(cs, sess))
			r.Get("/{id}", handler.GetCharlotteDocument(cs, sess))
		})
		r.Get("/public/documents/{id}/attributes/{key}", handler.PublicDocumentAttribute(st))
		redditAuth := handler.NewRedditAuth(cfg.RedditClientID, cfg.RedditClientSecret, cfg.RedditUsername, cfg.RedditPassword)
		r.Get("/reddit/posts", handler.ListRedditPosts(st, sess, redditAuth))
		r.Get("/reddit/post", handler.GetRedditPost(sess, redditAuth))

		r.Route("/documents", func(r chi.Router) {
			r.Get("/", handler.ListDocuments(st, sess))
			r.Post("/", handler.CreateDocument(st, sess))

			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", handler.GetDocument(st, sess))
				r.Put("/", handler.UpdateDocument(st, sess))
				r.Delete("/", handler.DeleteDocument(st, sess))
				r.Post("/suggest-labels", handler.SuggestDocumentLabels(st, sess))
				r.Post("/recommend-labels", handler.RecommendDocumentLabels(st, sess))
				r.Post("/reddit-submit", handler.SubmitRedditPost(st, sess, redditAuth))

				r.Route("/blocks", func(r chi.Router) {
					r.Post("/", handler.CreateBlock(st, sess))
					r.Put("/{blockId}", handler.UpdateBlock(st, sess))
					r.Delete("/{blockId}", handler.DeleteBlock(st, sess))
					r.Post("/{blockId}/copy", handler.CopyBlock(st, sess))
					r.Post("/{blockId}/run", handler.RunBlock(st, sess))
					r.Post("/{blockId}/run/stream", handler.RunBlockStream(st, sess))
					r.Put("/{blockId}/responses/{responseId}", handler.UpdateResponse(st, sess))
					r.Post("/{blockId}/responses/{responseId}/reparse", handler.ReparseResponse(st, sess))
				})
			})
		})
	})

	slog.Info("server listening", "addr", cfg.Addr)
	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
