# Stage 1: Build frontend
FROM docker.io/library/node:20-alpine AS frontend
WORKDIR /app/web/frontend
COPY web/frontend/package.json web/frontend/package-lock.json ./
RUN npm ci
COPY web/frontend/ ./
RUN npm run build

# Stage 2: Build backend
FROM docker.io/library/golang:1.23-alpine AS backend
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/web/static ./web/static
RUN CGO_ENABLED=1 go build -o pathpad ./cmd/server/

# Stage 3: Runtime
FROM docker.io/library/alpine:3.20
RUN apk add --no-cache ca-certificates
RUN adduser -D -h /home/pathpad pathpad
WORKDIR /home/pathpad

COPY --from=backend /app/pathpad /usr/local/bin/pathpad

RUN mkdir -p /data && chown pathpad:pathpad /data
USER pathpad

ENV PATHPAD_DB_PATH=/data/pathpad.db
ENV PATHPAD_PORT=8080
EXPOSE 8080
VOLUME ["/data"]

CMD ["pathpad"]
