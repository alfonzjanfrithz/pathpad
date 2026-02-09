package config

import (
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Config holds all application configuration.
type Config struct {
	Port            string
	DBPath          string
	MaxContentSize  int64
	CacheTTL        time.Duration
	RateLimit       int
	CORSOrigins     string
	SSEMaxClients   int
	SSEKeepalive    time.Duration
	LogLevel        string
}

// Load reads configuration from environment variables with defaults.
func Load() *Config {
	dbPath := envOrDefault("DONTPAD_DB_PATH", "./dontpad.db")
	// Resolve to absolute path so it works regardless of working directory.
	if abs, err := filepath.Abs(dbPath); err == nil {
		dbPath = abs
	}

	return &Config{
		Port:            envOrDefault("DONTPAD_PORT", "8080"),
		DBPath:          dbPath,
		MaxContentSize:  envOrDefaultInt64("DONTPAD_MAX_CONTENT_SIZE", 1048576),
		CacheTTL:        time.Duration(envOrDefaultInt("DONTPAD_CACHE_TTL", 300)) * time.Second,
		RateLimit:       envOrDefaultInt("DONTPAD_RATE_LIMIT", 100),
		CORSOrigins:     envOrDefault("DONTPAD_CORS_ORIGINS", "*"),
		SSEMaxClients:   envOrDefaultInt("DONTPAD_SSE_MAX_CLIENTS", 50),
		SSEKeepalive:    time.Duration(envOrDefaultInt("DONTPAD_SSE_KEEPALIVE", 30)) * time.Second,
		LogLevel:        envOrDefault("DONTPAD_LOG_LEVEL", "info"),
	}
}

func envOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func envOrDefaultInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func envOrDefaultInt64(key string, defaultVal int64) int64 {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.ParseInt(val, 10, 64); err == nil {
			return i
		}
	}
	return defaultVal
}
