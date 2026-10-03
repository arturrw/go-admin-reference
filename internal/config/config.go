// Package config loads runtime configuration from environment variables.
package config

import (
	"log/slog"
	"os"
	"strings"
	"time"
)

type Config struct {
	Addr     string     // HTTP listen address, e.g. ":8080".
	Env      string     // "development" or "production".
	LogLevel slog.Level // debug | info | warn | error.
	Version  string     // Reported by /api/v1/meta.

	UploadDir  string        // Where uploaded product images are written.
	SessionTTL time.Duration // Idle lifetime of a login session.
}

func Load() Config {
	return Config{
		Addr:     getenv("ADDR", ":8080"),
		Env:      getenv("APP_ENV", "development"),
		LogLevel: parseLevel(getenv("LOG_LEVEL", "info")),
		Version:  getenv("APP_VERSION", "v0.2.0"),

		UploadDir:  getenv("UPLOAD_DIR", "data/uploads"),
		SessionTTL: parseDuration(getenv("SESSION_TTL", "12h"), 12*time.Hour),
	}
}

func (c Config) IsDev() bool { return c.Env != "production" }

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func parseLevel(s string) slog.Level {
	var l slog.Level
	if err := l.UnmarshalText([]byte(strings.ToUpper(s))); err != nil {
		return slog.LevelInfo
	}
	return l
}

func parseDuration(s string, fallback time.Duration) time.Duration {
	if d, err := time.ParseDuration(s); err == nil && d > 0 {
		return d
	}
	return fallback
}
