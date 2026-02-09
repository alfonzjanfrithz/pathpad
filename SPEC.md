# Dontpad Clone - Implementation Spec

## Overview

### What We're Building

An open-source, self-hostable alternative to dontpad.com with hierarchical path-based navigation (similar to HashiCorp Vault's path system). Users can create nested pads like `/mypage/hello` and `/mypage`, with navigation between parent and child pads. The primary goal is a **simple snippet-sharing platform** — real-time collaboration is deferred to a future phase.

### Core Features (MVP)

- **Hierarchical Paths**: Nested pad structure (`/parent/child/grandchild`)
- **Implicit Pads**: Every URL is a valid pad — empty editor if no content exists yet
- **Navigation UI**: Breadcrumbs (parsed from URL), child pad links
- **Real-time Notifications**: SSE-based update notifications (not collaborative editing)
- **Auto-save**: Changes saved automatically as you type (debounced)
- **REST API**: Vault-style prefixed API for CLI integration
- **No Auth**: Public access by URL path
- **Self-hostable**: Single binary, minimal memory footprint

### Technology Stack

**Backend:**
- Go (single binary, ~10-50MB memory)
- SQLite (file-based, zero config)
- Chi router (lightweight HTTP router)
- Native Go HTTP for SSE (no external deps)

**Frontend:**
- Vanilla JavaScript (minimal bundle)
- Web App Manifest (installable PWA shell)

**Storage:**
- SQLite database (single file)
- In-memory cache for active pads

### Architecture

```
Browser
    │
    ├─ HTTP GET/PUT → REST API (/api/pad/...)
    ├─ SSE Stream   → Update notifications
    └─ Static Files → HTML/CSS/JS
    │
Go Server
    │
    ├─ HTTP Router (Chi)    → API handlers
    ├─ SSE Broadcaster      → Event distribution
    ├─ In-Memory Cache      → Fast reads
    ├─ SQLite               → Persistent storage
    └─ Graceful Shutdown    → Clean connection drain
```

### Key Design Decisions

1. **SSE over WebSockets**: Simpler, works through corporate firewalls, native browser reconnect
2. **SQLite**: Free, file-based, perfect for self-hosting, no separate database server
3. **Vault-style API paths**: Operation keyword before user path (`/api/pad/content/*path`) eliminates ambiguity
4. **Implicit pads**: Every URL is valid — no 404 for pads, DB row only created on first write
5. **No FK constraints**: `parent_path` is derived from the path string, not a foreign key — no ancestor auto-creation needed
6. **Last-write-wins**: Simple conflict resolution — real-time collaborative editing deferred to future phase
7. **REST API**: CLI-friendly endpoints for future tooling

---

## Path Design

### Path Rules

- **Allowed characters**: lowercase alphanumeric, `-`, `_` (no spaces, no special chars)
- **Separator**: `/` for hierarchy levels
- **Case**: All paths are lowercased on input (normalize)
- **Trailing slashes**: Stripped on input (`/mypage/` → `/mypage`)
- **Double slashes**: Collapsed (`/mypage//hello` → `/mypage/hello`)
- **Max depth**: 10 levels
- **Max segment length**: 64 characters
- **Max total path length**: 512 characters
- **Root path**: Empty string `""` (the homepage pad)

### Reserved Paths

The following prefixes are **blocked** from being created as pads:
- `api` — API endpoints
- `static` — Static file serving
- `healthz` — Health check
- `manifest.json`, `sw.js`, `favicon.ico` — PWA/browser files

Path validation must reject these at the API layer with a `400 Bad Request`.

### Parent Path Derivation

`parent_path` is computed from the path string, never stored as a foreign key:

| Path | Parent Path |
|------|-------------|
| `""` (root) | `""` (self) |
| `mypage` | `""` |
| `mypage/hello` | `mypage` |
| `mypage/hello/world` | `mypage/hello` |

---

## Implementation Breakdown

### Phase 1: Backend Foundation

#### 1.1 Project Setup
- Initialize Go module (`go mod init dontpad`)
- Create directory structure:
  ```
  cmd/server/main.go
  internal/api/handlers.go
  internal/api/routes.go
  internal/api/middleware.go
  internal/storage/sqlite.go
  internal/storage/cache.go
  internal/sse/broadcaster.go
  internal/models/pad.go
  internal/config/config.go
  web/static/ (frontend files)
  ```
- Add dependencies: `github.com/go-chi/chi/v5`, `github.com/mattn/go-sqlite3`

#### 1.2 Database Schema

```sql
-- Schema version tracking for future migrations
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS pads (
    path TEXT PRIMARY KEY,                  -- Full path: 'mypage/hello', '' for root
    content TEXT NOT NULL DEFAULT '',
    parent_path TEXT NOT NULL DEFAULT '',   -- Derived from path, NOT a foreign key
    updated_at INTEGER NOT NULL,            -- Unix timestamp
    created_at INTEGER NOT NULL             -- Unix timestamp
);

CREATE INDEX idx_parent_path ON pads(parent_path);
CREATE INDEX idx_updated_at ON pads(updated_at);
```

**Key behaviors:**
- No foreign key on `parent_path` — it's a denormalized field derived from the path at insert time
- A pad row is only created when content is first written (PUT with non-empty content)
- GET on a non-existent path returns `200` with empty content (implicit pad)
- `parent_path` is computed by the application: `strings.Join(segments[:len(segments)-1], "/")`

**Storage Operations:**
- `GetPad(path string) (*Pad, error)` — Get pad by path (returns empty pad if not in DB)
- `SavePad(path, content string) error` — Upsert pad content (INSERT OR REPLACE)
- `DeletePad(path string) (int64, error)` — Delete a pad and all descendants, returns count deleted
- `GetChildren(parentPath string) ([]Pad, error)` — List child pads that have content
- `PathExists(path string) bool` — Check if pad has content in DB

#### 1.3 In-Memory Cache
- Cache active pads with TTL (5 minutes default)
- Fast reads for frequently accessed pads
- Invalidate on writes
- Thread-safe: `sync.RWMutex` protecting a `map[string]*CacheEntry`
- `CacheEntry` = `{ Pad, ExpiresAt }`

### Phase 2: REST API

#### 2.1 API Endpoints (Vault-Style Prefix)

All API endpoints use the pattern `/api/pad/{operation}/*path` where the operation keyword comes **before** the user's path to avoid ambiguity.

**Pad Operations:**
- `GET /api/pad/content/*path` — Get pad content
  - Response: `{"path": "...", "content": "...", "updated_at": 123, "created_at": 123}`
  - Always returns 200 (empty content for non-existent pads, `updated_at: 0`)
  - Example: `GET /api/pad/content/mypage/hello`

- `PUT /api/pad/content/*path` — Create or update pad content
  - Body: `{"content": "..."}`
  - Upserts: creates if doesn't exist, updates if it does
  - Computes and stores `parent_path` from the path
  - Response: `{"path": "...", "content": "...", "updated_at": 123, "created_at": 123}`
  - Triggers SSE broadcast (`type: "update"`) to other connected clients

- `DELETE /api/pad/content/*path` — Delete a pad and all its descendants
  - Deletes the pad at the given path **and** all pads whose `parent_path` starts with it (recursive children)
  - Response: `{"deleted": 3}` (count of deleted pads)
  - Returns 200 even if pad didn't exist (idempotent, `{"deleted": 0}`)
  - Triggers SSE broadcast (`type: "delete"`) to connected clients on the deleted pad
  - Invalidates cache for the deleted pad and its descendants

**Navigation:**
- `GET /api/pad/children/*path` — List direct children with content
  - Response: `{"children": [{"path": "...", "updated_at": 123}, ...]}`
  - Only returns pads that have been written to (exist in DB)
  - Children are sorted alphabetically by path

**Real-time:**
- `GET /api/pad/events/*path` — SSE event stream
  - Update event: `data: {"type": "update", "content": "...", "updated_at": 123, "client_id": "..."}\n\n`
  - Delete event: `data: {"type": "delete", "path": "...", "client_id": "..."}\n\n`
  - Keeps connection open
  - Includes `client_id` of the sender so clients can ignore self-echoed events
  - Sends `:keepalive\n\n` comment every 30 seconds to prevent proxy timeouts

**System:**
- `GET /healthz` — Health check
  - Response: `{"status": "ok", "db": "ok"}` (pings SQLite)
  - Returns 200 if healthy, 503 if DB unreachable

#### 2.2 Error Handling
- Standard HTTP status codes: 200, 400, 404 (only for system routes), 413, 429, 500
- JSON error responses: `{"error": "message"}`
- Input validation: path format, reserved paths, content size
- Content size limit: configurable, default 1MB

#### 2.3 Middleware Stack
- **Logging**: Structured request logging (method, path, status, duration)
- **Recovery**: Panic recovery → 500 response
- **CORS**: Configurable allowed origins (default: `*`)
- **Rate Limiting**: Per-IP, configurable (default: 100 req/min)
- **Content-Type**: Enforce `application/json` on PUT requests
- **Path Normalization**: Lowercase, strip trailing slashes, collapse double slashes

### Phase 3: SSE Real-time Notifications

#### 3.1 SSE Broadcaster

Simplified for MVP — notifications only, not collaborative editing.

- `map[string]map[string]chan SSEEvent` — pad path → client ID → event channel
- Each client gets a unique ID (UUID v4) assigned on connection
- On pad update via PUT:
  - Broadcast event to all connected clients for that pad
  - Include `client_id` of the writer so the sender can filter self-events
- Auto-cleanup disconnected clients (detect via context cancellation)
- Connection limit per pad: configurable (default: 50)

#### 3.2 Event Flow
1. Client generates a random `client_id` (UUID) on page load
2. Client connects to `GET /api/pad/events/mypage?client_id=abc123`
3. Server registers the client channel in the broadcaster
4. On pad update, server broadcasts `{"type": "update", "content": "...", "client_id": "writer-id"}`
5. On pad delete, server broadcasts `{"type": "delete", "path": "...", "client_id": "deleter-id"}`
6. Client receives event — if `client_id` matches own ID, ignore (self-echo); otherwise update editor (or redirect to parent on delete)
7. Server sends `:keepalive\n\n` every 30 seconds
8. On disconnect, server removes client from broadcaster (context.Done)

#### 3.3 Known Limitations (MVP)
- **Last-write-wins**: No conflict resolution, no merge
- **Full content broadcast**: Entire pad content sent on each update (no diffs/patches)
- **No cursor sync**: No awareness of other users' cursor positions
- These will be addressed in a future collaboration phase (OT/CRDT)

### Phase 4: Frontend Application

#### 4.1 Core UI Components

**Main App (`index.html`):**
- `<textarea>` for editing (simple, no XSS risk unlike contenteditable)
- Breadcrumb navigation (computed from URL path)
- Children list panel (pads that have content)
- Delete button (with confirmation prompt — deletes pad and all descendants)
- Save status indicator ("Saving...", "Saved", "Error")
- Connection status indicator (SSE connected/disconnected)

**Styling (`styles.css`):**
- Minimal, clean design
- Responsive layout (mobile-friendly)
- System font stack (no web font loading)

#### 4.2 JavaScript Application (`app.js`)

**State:**
- `currentPath` — current pad path (from URL)
- `clientId` — random UUID generated on page load
- `sseConnection` — current EventSource instance
- `saveTimeout` — debounce timer for auto-save
- `lastSavedContent` — to detect actual changes

**Core Functions:**
- `loadPad(path)` — Fetch pad from `GET /api/pad/content/{path}`
- `savePad(path, content)` — Update pad via `PUT /api/pad/content/{path}`
- `deletePad(path)` — Delete pad via `DELETE /api/pad/content/{path}`, then navigate to parent
- `navigateTo(path)` — Update URL (History API pushState), load pad, reconnect SSE
- `connectSSE(path)` — Connect to `GET /api/pad/events/{path}?client_id=...`
- `renderBreadcrumbs(path)` — Parse path string into clickable breadcrumb links
- `loadChildren(path)` — Fetch from `GET /api/pad/children/{path}` and render list

**Auto-save:**
- Listen to `input` event on textarea
- Debounce with 500ms delay
- Only save if content actually changed (compare with `lastSavedContent`)
- Show "Saving..." → "Saved" indicator
- On save error, show "Error saving" with retry

**SSE Handling:**
- On `message` event: parse JSON, check `client_id`, skip if self-echo
- On `type: "update"`: replace textarea content with new content
- On `type: "delete"`: show notification and navigate to parent pad
- On `error` event: EventSource auto-reconnects (browser built-in)
- Show connection status indicator

**SPA Navigation:**
- Use History API (`pushState` / `popState`)
- On `popstate` event: load the pad for the new URL
- All internal links use `navigateTo()` instead of full page reload

#### 4.3 HTML Serving & Routing
- `GET /` → Serve `index.html`
- `GET /*path` (non-API, non-static) → Serve `index.html` (SPA catch-all)
- `GET /static/*` → Serve static files (CSS, JS, icons)
- `GET /api/*` → API handlers
- JavaScript reads the path from `window.location.pathname` to determine current pad

### Phase 5: PWA Shell (Minimal)

For MVP, implement a **minimal PWA shell** — installable with basic caching, but **no offline editing** (deferred to future phase).

#### 5.1 Web App Manifest (`manifest.json`)
```json
{
  "name": "Dontpad",
  "short_name": "Dontpad",
  "start_url": "/",
  "display": "standalone",
  "theme_color": "#ffffff",
  "background_color": "#ffffff",
  "icons": [
    { "src": "/static/icons/icon-192.png", "sizes": "192x192", "type": "image/png" },
    { "src": "/static/icons/icon-512.png", "sizes": "512x512", "type": "image/png" }
  ]
}
```

#### 5.2 Service Worker (`sw.js`) — Minimal
- Cache app shell on install (HTML, CSS, JS)
- Network-first strategy for API calls (no offline fallback for MVP)
- Serve cached app shell if network unavailable (shows "offline" state)

#### 5.3 Deferred to Future Phase
- IndexedDB for offline pad storage
- Background sync for pending edits
- Offline editing queue
- Conflict resolution UI

### Phase 6: Configuration

#### 6.1 Config Options
| Variable | Default | Description |
|----------|---------|-------------|
| `DONTPAD_PORT` | `8080` | Server port |
| `DONTPAD_DB_PATH` | `./dontpad.db` | SQLite database file path |
| `DONTPAD_MAX_CONTENT_SIZE` | `1048576` (1MB) | Max pad content size in bytes |
| `DONTPAD_CACHE_TTL` | `300` (5 min) | In-memory cache TTL in seconds |
| `DONTPAD_RATE_LIMIT` | `100` | Max requests per minute per IP |
| `DONTPAD_CORS_ORIGINS` | `*` | Allowed CORS origins |
| `DONTPAD_SSE_MAX_CLIENTS` | `50` | Max SSE connections per pad |
| `DONTPAD_SSE_KEEPALIVE` | `30` | SSE keepalive interval in seconds |
| `DONTPAD_LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |

#### 6.2 Configuration Method
- **Environment variables only** for MVP (simplest, 12-factor app compliant)
- Config file and CLI flags deferred to future phase

### Phase 7: Server Lifecycle

#### 7.1 Startup
1. Load configuration from environment variables
2. Initialize SQLite database (create file + tables if not exist)
3. Run schema migrations (check `schema_version` table)
4. Initialize in-memory cache
5. Initialize SSE broadcaster
6. Register routes and middleware
7. Start HTTP server
8. Log "Server started on :PORT"

#### 7.2 Graceful Shutdown
1. Listen for `SIGTERM` and `SIGINT`
2. Stop accepting new connections
3. Close all SSE client connections (signal via context cancellation)
4. Wait for in-flight HTTP requests to complete (with timeout, e.g., 10s)
5. Close SQLite database connection
6. Log "Server stopped"

### Phase 8: Build & Deployment

#### 8.1 Build
- Embed static files using Go `embed` package (single binary)
- Build: `go build -o dontpad cmd/server/main.go`
- Cross-compile: `GOOS=linux GOARCH=amd64 go build ...`

#### 8.2 Deployment Options
- **Direct**: Run binary, SQLite file created automatically
- **Docker**: Lightweight container with volume for DB file
- **Systemd**: Service file for Linux servers

#### 8.3 Dockerfile
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o dontpad cmd/server/main.go

FROM alpine:latest
COPY --from=builder /app/dontpad /usr/local/bin/
VOLUME /data
ENV DONTPAD_DB_PATH=/data/dontpad.db
EXPOSE 8080
CMD ["dontpad"]
```

---

## File Structure

```
dontpad/
├── cmd/
│   └── server/
│       └── main.go                 # Entry point, config, graceful shutdown
├── internal/
│   ├── api/
│   │   ├── handlers.go            # HTTP handlers (GetPad, SavePad, GetChildren, Health)
│   │   ├── routes.go              # Route definitions with Chi
│   │   └── middleware.go          # Logging, recovery, rate limiting, path normalization
│   ├── storage/
│   │   ├── sqlite.go              # SQLite operations, migrations
│   │   └── cache.go               # In-memory cache with TTL
│   ├── sse/
│   │   └── broadcaster.go         # SSE event broadcasting, keepalive, cleanup
│   ├── models/
│   │   └── pad.go                 # Pad struct, path validation, path helpers
│   └── config/
│       └── config.go              # Env var parsing, defaults, validation
├── web/
│   └── static/
│       ├── index.html             # Main SPA shell
│       ├── app.js                 # Frontend logic (editor, SSE, navigation)
│       ├── styles.css             # Styling
│       ├── sw.js                  # Service worker (minimal cache)
│       ├── manifest.json          # PWA manifest
│       └── icons/
│           ├── icon-192.png
│           └── icon-512.png
├── go.mod
├── go.sum
├── Dockerfile
├── .gitignore
└── SPEC.md
```

---

## Implementation Order

1. **Phase 1: Backend Foundation** — Go project, SQLite schema + migrations, storage layer, cache
2. **Phase 2: REST API** — Vault-style endpoints, middleware, error handling, health check
3. **Phase 3: SSE** — Broadcaster, keepalive, client ID filtering
4. **Phase 4: Frontend** — SPA shell, textarea editor, breadcrumbs, children list, auto-save, SSE client
5. **Phase 5: PWA Shell** — Manifest, minimal service worker
6. **Phase 6: Configuration** — Env var parsing, defaults
7. **Phase 7: Server Lifecycle** — Graceful shutdown, startup logging
8. **Phase 8: Build & Deploy** — Embed static files, Dockerfile, build scripts

---

## Testing Strategy

**Unit Tests:**
- Storage: CRUD operations, path derivation, upsert behavior
- Cache: TTL expiry, invalidation, concurrent access
- Models: Path validation, normalization, reserved path rejection
- Config: Env var parsing, defaults

**Integration Tests:**
- API: Full request/response cycle for all endpoints
- SSE: Connection, event delivery, keepalive, client cleanup
- SPA routing: Catch-all serves index.html, API routes work

**Manual Testing:**
- Browser: SPA navigation, auto-save, SSE reconnect
- Multi-tab: Update in one tab, see notification in another
- Mobile: Responsive layout, touch interactions

---

## Future Enhancements (Out of Scope for MVP)

**Phase 2 — Collaboration:**
- OT/CRDT for real-time collaborative editing
- Cursor position sync / user presence
- Diff-based SSE events (patches instead of full content)
- Conflict resolution UI

**Phase 2 — Offline:**
- IndexedDB for offline pad storage
- Background sync for pending edits
- Offline editing queue with sync indicator

**Future:**
- CLI client using REST API
- Optional authentication / private pads
- Markdown rendering toggle
- Syntax highlighting
- Export functionality (JSON, text, PDF)
- Pad history / versioning with undo
- Full-text search across pads
- Config file and CLI flag support
- Metrics / observability endpoint
