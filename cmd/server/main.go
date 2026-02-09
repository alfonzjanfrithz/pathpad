package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pathpad/internal/api"
	"pathpad/internal/config"
	"pathpad/internal/sse"
	"pathpad/internal/storage"
	"pathpad/web"
)

func main() {
	cfg := config.Load()

	log.Printf("[startup] Pathpad server starting on port %s", cfg.Port)
	log.Printf("[startup] DB path: %s", cfg.DBPath)

	// Initialize SQLite store.
	store, err := storage.NewSQLiteStore(cfg.DBPath)
	if err != nil {
		log.Fatalf("[startup] Failed to initialize database: %v", err)
	}
	defer store.Close()

	// Initialize cache.
	cache := storage.NewCache(cfg.CacheTTL)

	// Initialize SSE broadcaster.
	broadcaster := sse.NewBroadcaster(cfg.SSEMaxClients, cfg.SSEKeepalive)

	log.Printf("[startup] Database initialized successfully")

	// Build router with all routes, middleware, and embedded static files.
	router := api.NewRouter(cfg, store, cache, broadcaster, web.StaticFiles)

	// Create HTTP server.
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second, // longer for SSE
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine.
	go func() {
		log.Printf("[startup] Listening on http://0.0.0.0:%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[startup] Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("[shutdown] Received signal: %v", sig)

	// Give in-flight requests 10 seconds to complete.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("[shutdown] Server forced to shutdown: %v", err)
	}

	log.Println("[shutdown] Server stopped")
}
