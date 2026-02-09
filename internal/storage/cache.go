package storage

import (
	"sync"
	"time"

	"dontpad/internal/models"
)

// CacheEntry holds a cached pad with expiration.
type CacheEntry struct {
	Pad       *models.Pad
	ExpiresAt time.Time
}

// Cache provides an in-memory cache for pads with TTL-based expiration.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]*CacheEntry
	ttl     time.Duration
}

// NewCache creates a new cache with the given TTL duration.
func NewCache(ttl time.Duration) *Cache {
	c := &Cache{
		entries: make(map[string]*CacheEntry),
		ttl:     ttl,
	}
	// Start background cleanup goroutine.
	go c.cleanup()
	return c
}

// Get retrieves a pad from cache. Returns nil if not found or expired.
func (c *Cache) Get(path string) *models.Pad {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[path]
	if !ok {
		return nil
	}
	if time.Now().After(entry.ExpiresAt) {
		return nil
	}
	return entry.Pad
}

// Set stores a pad in the cache with the configured TTL.
func (c *Cache) Set(path string, pad *models.Pad) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[path] = &CacheEntry{
		Pad:       pad,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

// Invalidate removes a specific pad from the cache.
func (c *Cache) Invalidate(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, path)
}

// InvalidatePrefix removes all entries whose path starts with the given prefix.
// Used when deleting a pad and its descendants.
func (c *Cache) InvalidatePrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for path := range c.entries {
		if path == prefix || (len(path) > len(prefix) && path[:len(prefix)] == prefix && path[len(prefix)] == '/') {
			delete(c.entries, path)
		}
	}
}

// cleanup periodically removes expired entries. Runs in a background goroutine.
func (c *Cache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for path, entry := range c.entries {
			if now.After(entry.ExpiresAt) {
				delete(c.entries, path)
			}
		}
		c.mu.Unlock()
	}
}
