package api

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"dontpad/internal/config"
	"dontpad/internal/sse"
	"dontpad/internal/storage"
)

// NewRouter creates and configures the Chi router with all routes and middleware.
func NewRouter(cfg *config.Config, store *storage.SQLiteStore, cache *storage.Cache, broadcaster *sse.Broadcaster, staticFS fs.FS) http.Handler {
	r := chi.NewRouter()

	// Global middleware stack.
	r.Use(Recovery)
	r.Use(RequestLogger)
	r.Use(CORS(cfg.CORSOrigins))
	r.Use(NewRateLimiter(cfg.RateLimit).Middleware)

	// Create handler with dependencies.
	h := &Handler{
		Store:          store,
		Cache:          cache,
		Broadcaster:    broadcaster,
		MaxContentSize: cfg.MaxContentSize,
	}

	// Health check.
	r.Get("/healthz", h.Health)

	// API routes — Vault-style prefix: /api/pad/{operation}/*
	r.Route("/api/pad", func(r chi.Router) {
		// Content CRUD.
		r.Get("/content", h.GetPad)
		r.Get("/content/*", h.GetPad)
		r.Put("/content", h.SavePad)
		r.Put("/content/*", h.SavePad)
		r.Delete("/content", h.DeletePad)
		r.Delete("/content/*", h.DeletePad)

		// Children listing.
		r.Get("/children", h.GetChildren)
		r.Get("/children/*", h.GetChildren)

		// SSE events.
		r.Get("/events", h.Events)
		r.Get("/events/*", h.Events)
	})

	// Static files.
	fileServer := http.FileServer(http.FS(staticFS))
	r.Handle("/static/*", fileServer)

	// Read index.html once for the SPA catch-all.
	indexHTML, err := fs.ReadFile(staticFS, "static/index.html")
	if err != nil {
		panic("failed to read index.html: " + err.Error())
	}

	// SPA catch-all: serve index.html for all other GET requests.
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		// Only serve SPA for GET requests that accept HTML.
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		// Don't serve SPA for requests that look like file paths (have extensions).
		if strings.Contains(r.URL.Path, ".") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(indexHTML)
	})

	return r
}
