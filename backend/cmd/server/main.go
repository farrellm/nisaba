package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"

	"github.com/farrellm/nisaba/internal/auth"
	"github.com/farrellm/nisaba/internal/config"
	"github.com/farrellm/nisaba/internal/db"
	"github.com/farrellm/nisaba/internal/handler"
	"github.com/farrellm/nisaba/internal/store"
)

func main() {
	cfg := config.Load()

	pool, err := db.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	st := store.New(pool)
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
		})

		r.Get("/modes", handler.ListModes())

		r.Route("/documents", func(r chi.Router) {
			r.Get("/", handler.ListDocuments(st, sess))
			r.Post("/", handler.CreateDocument(st, sess))

			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", handler.GetDocument(st, sess))

				r.Route("/blocks", func(r chi.Router) {
					r.Post("/", handler.CreateBlock(st, sess))
					r.Put("/{blockId}", handler.UpdateBlock(st, sess))
					r.Post("/{blockId}/run", handler.RunBlock(st, sess))
				})
			})
		})
	})

	slog.Info("server listening", "addr", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, r); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
