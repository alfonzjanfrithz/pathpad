.PHONY: all build frontend backend clean dev

all: build

# Build frontend then backend
build: frontend backend

# Build Svelte frontend (outputs to web/static/)
frontend:
	cd web/frontend && npm run build

# Build Go binary (embeds web/static/)
backend:
	CGO_ENABLED=1 go build -o pathpad ./cmd/server/

# Clean build artifacts
clean:
	rm -f pathpad
	rm -rf web/static
	rm -f pathpad.db pathpad.db-wal pathpad.db-shm

# Install frontend dependencies
deps:
	cd web/frontend && npm install
