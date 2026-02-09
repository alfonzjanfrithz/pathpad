package main

import (
	"fmt"
	"log"

	"dontpad/internal/config"
	"dontpad/internal/storage"
)

func main() {
	cfg := config.Load()

	log.Printf("[startup] Dontpad server starting on port %s", cfg.Port)
	log.Printf("[startup] DB path: %s", cfg.DBPath)

	// Initialize SQLite store.
	store, err := storage.NewSQLiteStore(cfg.DBPath)
	if err != nil {
		log.Fatalf("[startup] Failed to initialize database: %v", err)
	}
	defer store.Close()

	// Initialize cache.
	cache := storage.NewCache(cfg.CacheTTL)
	_ = cache // will be wired into API handlers in Phase 2

	log.Printf("[startup] Database initialized successfully")

	// Verify DB connectivity.
	if err := store.Ping(); err != nil {
		log.Fatalf("[startup] Database ping failed: %v", err)
	}

	fmt.Printf("Phase 1 complete! Server would listen on :%s\n", cfg.Port)
	fmt.Println("Storage and cache layers are ready.")
}
