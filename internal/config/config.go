// Package config loads runtime configuration from environment variables.
package config

import (
	"log/slog"
	"os"
	"strings"
)

type Config struct {
	Addr     string     // HTTP listen address, e.g. ":8080".
	Env      string     // "development" or "production".
	LogLevel slog.Level // debug | info | warn | error.
	Version  string     // Reported by /api/v1/meta.
}

func Load() Config {
	return Config{
		Addr:     getenv("ADDR", ":8080"),
		Env:      getenv("APP_ENV", "development"),
		LogLevel: parseLevel(getenv("LOG_LEVEL", "info")),
		Version:  getenv("APP_VERSION", "v0.1.0"),
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
