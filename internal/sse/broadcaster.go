package sse

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// Event represents an SSE event sent to clients.
type Event struct {
	Type     string `json:"type"`               // "update" or "delete"
	Content  string `json:"content,omitempty"`   // pad content (for update events)
	Path     string `json:"path,omitempty"`      // pad path (for delete events)
	ClientID string `json:"client_id,omitempty"` // sender's client ID
}

// Broadcaster manages SSE connections and event distribution.
type Broadcaster struct {
	mu            sync.RWMutex
	clients       map[string]map[string]chan Event // pad path -> client ID -> event channel
	maxClients    int
	keepalive     time.Duration
}

// NewBroadcaster creates a new SSE broadcaster.
func NewBroadcaster(maxClientsPerPad int, keepaliveInterval time.Duration) *Broadcaster {
	return &Broadcaster{
		clients:    make(map[string]map[string]chan Event),
		maxClients: maxClientsPerPad,
		keepalive:  keepaliveInterval,
	}
}

// Subscribe registers a client for events on a pad path.
// Returns the event channel and a cleanup function.
func (b *Broadcaster) Subscribe(path, clientID string) (chan Event, func(), error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.clients[path] == nil {
		b.clients[path] = make(map[string]chan Event)
	}

	if len(b.clients[path]) >= b.maxClients {
		return nil, nil, fmt.Errorf("max SSE connections reached for pad %q", path)
	}

	ch := make(chan Event, 16) // buffered to prevent blocking on slow clients
	b.clients[path][clientID] = ch

	log.Printf("[sse] Client %s subscribed to %q (%d clients)", clientID, path, len(b.clients[path]))

	cleanup := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if clients, ok := b.clients[path]; ok {
			delete(clients, clientID)
			if len(clients) == 0 {
				delete(b.clients, path)
			}
			log.Printf("[sse] Client %s unsubscribed from %q", clientID, path)
		}
		close(ch)
	}

	return ch, cleanup, nil
}

// Broadcast sends an event to all clients subscribed to a pad path.
func (b *Broadcaster) Broadcast(path string, event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	clients, ok := b.clients[path]
	if !ok {
		return
	}

	for id, ch := range clients {
		select {
		case ch <- event:
		default:
			// Channel full — slow client, skip to avoid blocking.
			log.Printf("[sse] Dropped event for slow client %s on %q", id, path)
		}
	}
}

// ServeHTTP handles an SSE connection for a given pad path and client ID.
func (b *Broadcaster) ServeHTTP(w http.ResponseWriter, r *http.Request, path, clientID string) {
	// Verify that streaming is supported.
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `{"error":"streaming not supported"}`, http.StatusInternalServerError)
		return
	}

	// Subscribe to events.
	ch, cleanup, err := b.Subscribe(path, clientID)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusTooManyRequests)
		return
	}
	defer cleanup()

	// Set SSE headers.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	// Use request context for cancellation (client disconnect).
	ctx := r.Context()
	keepaliveTicker := time.NewTicker(b.keepalive)
	defer keepaliveTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Client disconnected.
			return

		case event, ok := <-ch:
			if !ok {
				// Channel closed.
				return
			}
			data, err := json.Marshal(event)
			if err != nil {
				log.Printf("[sse] Failed to marshal event: %v", err)
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

		case <-keepaliveTicker.C:
			fmt.Fprintf(w, ":keepalive\n\n")
			flusher.Flush()
		}
	}
}

// ClientCount returns the number of connected clients for a given pad path.
func (b *Broadcaster) ClientCount(path string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients[path])
}
