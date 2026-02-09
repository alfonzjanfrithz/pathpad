# Dontpad

An open-source, self-hostable alternative to [dontpad.com](https://dontpad.com) with hierarchical path-based navigation. Users can create nested pads like `/mypage/hello` and `/mypage`, with navigation between parent and child pads.

## Features

- **Hierarchical Paths** — Nested pad structure (`/parent/child/grandchild`)
- **Implicit Pads** — Every URL is a valid pad; empty editor if no content exists
- **Real-time Notifications** — SSE-based update notifications
- **Auto-save** — Changes saved automatically as you type (debounced)
- **REST API** — Vault-style prefixed API for CLI integration
- **No Auth** — Public access by URL path
- **Self-hostable** — Single binary, minimal memory footprint (~10-50MB)

## Quick Start

### Prerequisites

- **Go 1.21+** (with CGO enabled)
- **GCC** (required by the SQLite driver)

### Build & Run

```bash
# Build
CGO_ENABLED=1 go build -o dontpad ./cmd/server/

# Run (creates dontpad.db in current directory)
./dontpad
```

The server starts on `http://localhost:8080` by default.

### Docker

```bash
docker build -t dontpad .
docker run -p 8080:8080 -v dontpad-data:/data dontpad
```

## Configuration

All configuration is via environment variables:

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

Example:

```bash
DONTPAD_PORT=3000 DONTPAD_DB_PATH=/var/lib/dontpad/data.db ./dontpad
```

## REST API

All pad endpoints use the Vault-style prefix pattern: `/api/pad/{operation}/*path`.

### Get Pad Content

```
GET /api/pad/content/{path}
```

Returns 200 for all paths. Non-existent pads return empty content with `updated_at: 0`.

```bash
curl http://localhost:8080/api/pad/content/mypage
```

```json
{
  "path": "mypage",
  "content": "hello world",
  "updated_at": 1770622520,
  "created_at": 1770622520
}
```

### Create / Update Pad

```
PUT /api/pad/content/{path}
```

Upserts pad content. Pass `client_id` query param to identify the writer for SSE event filtering.

```bash
curl -X PUT -H "Content-Type: application/json" \
  -d '{"content":"hello world"}' \
  "http://localhost:8080/api/pad/content/mypage?client_id=my-uuid"
```

### Delete Pad (and Descendants)

```
DELETE /api/pad/content/{path}
```

Deletes the pad and all child pads recursively. Idempotent — returns `{"deleted": 0}` if pad didn't exist.

```bash
curl -X DELETE "http://localhost:8080/api/pad/content/mypage?client_id=my-uuid"
```

```json
{"deleted": 2}
```

### List Children

```
GET /api/pad/children/{path}
```

Returns direct child pads that have content, sorted alphabetically.

```bash
curl http://localhost:8080/api/pad/children/mypage
```

```json
{
  "children": [
    {"path": "mypage/child", "updated_at": 1770622520}
  ]
}
```

### SSE Event Stream

```
GET /api/pad/events/{path}?client_id={uuid}
```

Server-Sent Events stream for real-time notifications. `client_id` is required.

```bash
curl -N "http://localhost:8080/api/pad/events/mypage?client_id=my-uuid"
```

Events:

```
data: {"type":"update","content":"new content","client_id":"writer-uuid"}

data: {"type":"delete","path":"mypage","client_id":"deleter-uuid"}

:keepalive
```

Clients should ignore events where `client_id` matches their own (self-echo filtering).

### Health Check

```
GET /healthz
```

```json
{"status": "ok", "db": "ok"}
```

Returns 200 if healthy, 503 if the database is unreachable.

## Path Rules

- **Allowed characters**: lowercase alphanumeric, `-`, `_`
- **Case**: All paths lowercased on input
- **Max depth**: 10 levels
- **Max segment length**: 64 characters
- **Max total path length**: 512 characters
- **Reserved prefixes**: `api`, `static`, `healthz`, `manifest.json`, `sw.js`, `favicon.ico`

## Architecture

```
Browser
    |
    +-- HTTP GET/PUT/DELETE --> REST API (/api/pad/...)
    +-- SSE Stream          --> Update notifications
    +-- Static Files        --> HTML/CSS/JS
    |
Go Server
    |
    +-- HTTP Router (Chi)   --> API handlers
    +-- SSE Broadcaster     --> Event distribution
    +-- In-Memory Cache     --> Fast reads (5 min TTL)
    +-- SQLite (WAL mode)   --> Persistent storage
    +-- Graceful Shutdown   --> Clean connection drain
```

## Project Structure

```
dontpad/
├── cmd/server/main.go          # Entry point, config, graceful shutdown
├── internal/
│   ├── api/
│   │   ├── handlers.go         # HTTP handlers (CRUD, children, health, SSE)
│   │   ├── routes.go           # Chi route definitions
│   │   └── middleware.go       # Logging, recovery, CORS, rate limiting
│   ├── storage/
│   │   ├── sqlite.go           # SQLite operations + migrations
│   │   └── cache.go            # In-memory cache with TTL
│   ├── sse/
│   │   └── broadcaster.go      # SSE event broadcasting + keepalive
│   ├── models/
│   │   └── pad.go              # Pad struct, path validation/normalization
│   └── config/
│       └── config.go           # Env var parsing with defaults
├── web/static/                  # Frontend files (Phase 4)
├── go.mod
├── go.sum
├── Dockerfile
├── SPEC.md
└── README.md
```

## License

MIT
