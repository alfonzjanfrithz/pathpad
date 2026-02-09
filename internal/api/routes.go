package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"dontpad/internal/config"
	"dontpad/internal/sse"
	"dontpad/internal/storage"
)

// NewRouter creates and configures the Chi router with all routes and middleware.
func NewRouter(cfg *config.Config, store *storage.SQLiteStore, cache *storage.Cache, broadcaster *sse.Broadcaster) http.Handler {
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

	return r
}
