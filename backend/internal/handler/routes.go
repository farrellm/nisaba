package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/rs/cors"

	"github.com/farrellm/nisaba/internal/auth"
	"github.com/farrellm/nisaba/internal/blockrun"
	"github.com/farrellm/nisaba/internal/reddit"
	"github.com/farrellm/nisaba/internal/store"
)

// Deps carries everything the route tree needs. Store is the concrete
// Postgres store here; the fan-out to each handler's consumer-side interface
// happens at the constructor calls below.
type Deps struct {
	Store     *store.Store
	Sessions  *auth.Sessions
	DB        pinger // health check
	Reflex    *store.ReflexStore
	Charlotte *store.CharlotteStore
	Reddit    *reddit.Client
	Runner    *blockrun.Service
	CORS      []string
	// WebDir is the built frontend served at "/" with an SPA fallback. Empty
	// (the local-dev default) leaves the tree API-only, since Vite serves the
	// app itself and proxies /api here.
	WebDir string
}

// Routes builds the full /api handler tree with its middleware stack.
func Routes(d Deps) http.Handler {
	st := d.Store
	sess := d.Sessions

	r := chi.NewRouter()
	// The deployed process binds loopback only and is reached solely through
	// `tailscale serve`, which sets X-Forwarded-For, so trusting it here is safe
	// and gives the login limiter below a real client address.
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.New(cors.Options{
		AllowedOrigins:   d.CORS,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: true,
	}).Handler)

	r.Route("/api", func(r chi.Router) {
		// Its own 404, so an unknown /api path can never fall through to the
		// SPA handler below and answer a fetch with index.html.
		r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
			writeError(w, http.StatusNotFound, "Not found")
		})

		r.Get("/healthz", Health(d.DB))

		// No registration endpoint: accounts are created with the server
		// binary's -create-user flag.
		r.Route("/auth", func(r chi.Router) {
			// Login is the one route reachable unauthenticated from the public
			// internet, so it gets a limiter. Funnel requests can share an
			// ingress source address, so this throttles brute force in
			// aggregate rather than strictly per-client; bcrypt (cost 10)
			// remains the per-guess cost.
			r.With(httprate.LimitByIP(10, time.Minute)).Post("/login", Login(st, sess))
			r.Post("/logout", Logout(sess))
			r.Get("/me", Me(st, sess))
			r.Put("/me", UpdateMe(st, sess))
		})

		r.Get("/modes", ListModes())
		r.Get("/models", ListModels())
		r.Get("/public/documents/{id}/attributes/{key}", PublicDocumentAttribute(st))

		// Everything below requires a logged-in session; the middleware rejects
		// anonymous requests and puts the caller's user id in the context.
		r.Group(func(r chi.Router) {
			r.Use(RequireUser(sess))

			r.Route("/labels", func(r chi.Router) {
				r.Get("/", ListLabels(st))
				r.Put("/", RenameLabel(st))
				r.Delete("/", DeleteLabel(st))
			})
			r.Get("/attribute-values", ListAttributeValues(st))

			r.Route("/anansi/documents", func(r chi.Router) {
				r.Get("/", ListLegacyDocuments(d.Reflex))
				r.Get("/{id}", GetLegacyDocument(d.Reflex))
				r.Post("/{id}/import", ImportLegacyDocument(d.Reflex, st))
			})
			r.Route("/charlotte/documents", func(r chi.Router) {
				r.Get("/", ListLegacyDocuments(d.Charlotte))
				r.Get("/{id}", GetLegacyDocument(d.Charlotte))
				r.Post("/{id}/import", ImportLegacyDocument(d.Charlotte, st))
			})
			r.Get("/reddit/posts", ListRedditPosts(st, d.Reddit))
			r.Get("/reddit/post", GetRedditPost(d.Reddit))

			r.Route("/documents", func(r chi.Router) {
				r.Get("/", ListDocuments(st))
				r.Post("/", CreateDocument(st))
				r.Get("/search", SearchDocuments(st))

				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", GetDocument(st))
					r.Put("/", UpdateDocument(st))
					r.Delete("/", DeleteDocument(st))
					r.Post("/suggest-labels", SuggestDocumentLabels(st))
					r.Post("/recommend-labels", RecommendDocumentLabels(st))
					r.Post("/reddit-submit", SubmitRedditPost(st, d.Reddit))

					r.Route("/blocks", func(r chi.Router) {
						r.Post("/", CreateBlock(st))
						r.Put("/{blockId}", UpdateBlock(st))
						r.Delete("/{blockId}", DeleteBlock(st))
						r.Post("/{blockId}/copy", CopyBlock(st))
						r.Post("/{blockId}/run", RunBlock(st, d.Runner))
						r.Post("/{blockId}/run/stream", RunBlockStream(st, d.Runner))
						r.Put("/{blockId}/responses/{responseId}", UpdateResponse(st, d.Runner))
						r.Post("/{blockId}/responses/{responseId}/reparse", ReparseResponse(st, d.Runner))
					})
				})
			})
		})
	})

	if d.WebDir != "" {
		r.Handle("/*", SPA(d.WebDir))
	}

	return r
}
