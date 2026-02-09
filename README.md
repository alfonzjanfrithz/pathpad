# Pathpad

A fast, self-hostable notepad with hierarchical pages. Create nested pages like `/projects/todo` and `/projects/notes`, navigate between them, and collaborate in real time across browser tabs.

No login. No sign-up. Just open a URL and start typing.

## Quick Start

### Using a container (recommended)

```bash
# With Podman
podman run -d --name pathpad -p 8080:8080 -v pathpad-data:/data pathpad

# With Docker
docker run -d --name pathpad -p 8080:8080 -v pathpad-data:/data pathpad
```

Then open **http://localhost:8080** in your browser.

### From a pre-built binary

Download the binary for your platform, then:

```bash
./pathpad
```

This starts the server on port 8080 and creates a `pathpad.db` file in the current directory.

## How It Works

### Pages

Every URL path is a page. Visit `/meeting-notes` and you have a page. Visit `/meeting-notes/monday` and you have a child page under it.

- Pages are created automatically when you visit a URL or type content
- Content is saved automatically as you type (no save button needed)
- Pages can be nested to any depth: `/a/b/c/d/e`

### Navigation

- **Breadcrumbs** at the top show where you are: `root / projects / todo` — each segment is clickable
- **Sidebar** on the left shows child pages under the current page
- **Landing page** at `/` lets you jump to any page by typing its name

### Creating Pages

There are multiple ways to create a new page:

1. **From the sidebar** — type a name in the "new page..." input at the bottom and press Enter
2. **From the command palette** — press `Ctrl+K`, type a path that doesn't exist yet, and select "Create /your-path"
3. **From the URL bar** — navigate directly to any URL like `/my-new-page`

### Deleting Pages

- Click the trash icon in the sidebar footer
- Or press `Ctrl+K` and select "Delete current page"

Deleting a page also removes all its children.

### Real-Time Sync

Open the same page in multiple tabs or on different devices — changes appear instantly everywhere. The green dot in the sidebar indicates a live connection.

## Keyboard Shortcuts

| Shortcut | Action |
|---|---|
| `Ctrl+K` | Open command palette |
| `Ctrl+S` | Force save current page |
| `Ctrl+N` | Create a new page (opens palette with prefix) |
| `Ctrl+\` | Toggle sidebar |
| `Escape` | Close command palette |

### Command Palette

The command palette (`Ctrl+K`) lets you:

- **Jump to any page** — fuzzy search across all your pages
- **Create a new page** — type a path and select "Create /..."
- **Run actions** — go to parent, go to root, toggle sidebar, delete page

Use arrow keys to navigate and Enter to select.

## Configuration

All settings are optional environment variables:

| Variable | Default | Description |
|---|---|---|
| `PATHPAD_PORT` | `8080` | Server port |
| `PATHPAD_DB_PATH` | `./pathpad.db` | Database file location |
| `PATHPAD_MAX_CONTENT_SIZE` | `1048576` | Max page content size (bytes, default 1 MB) |
| `PATHPAD_RATE_LIMIT` | `100` | Max requests per minute per IP |
| `PATHPAD_CORS_ORIGINS` | `*` | Allowed CORS origins |
| `PATHPAD_LOG_LEVEL` | `info` | Log verbosity (debug, info, warn, error) |

### Example

```bash
PATHPAD_PORT=3000 PATHPAD_DB_PATH=/var/lib/pathpad/data.db ./pathpad
```

## Data & Backup

All data is stored in a single SQLite file (`pathpad.db` by default). To back up your data, simply copy this file while the server is stopped — or use SQLite's backup API for live backups.

The database uses WAL mode for good read/write concurrency.

## Container Deployment

### Build the image

```bash
podman build -t pathpad .
```

### Run with persistent storage

```bash
podman run -d \
  --name pathpad \
  -p 8080:8080 \
  -v pathpad-data:/data \
  pathpad
```

### Custom configuration

```bash
podman run -d \
  --name pathpad \
  -p 3000:3000 \
  -v pathpad-data:/data \
  -e PATHPAD_PORT=3000 \
  -e PATHPAD_LOG_LEVEL=debug \
  pathpad
```

### Stop and restart

```bash
podman stop pathpad
podman start pathpad
```

Your data persists in the `pathpad-data` volume across restarts.

## License

MIT
