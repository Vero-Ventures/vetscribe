// Package config loads VetScribe runtime configuration from environment variables
// with conservative, documented defaults.
package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds validated runtime configuration.
type Config struct {
	// ListenAddr is the host:port the HTTP server binds to.
	ListenAddr string
}

// Load reads configuration from the environment and validates it.
func Load() (Config, error) {
	cfg := Config{
		ListenAddr: envOr("VETSCRIBE_LISTEN_ADDR", ":8080"),
	}

	if strings.TrimSpace(cfg.ListenAddr) == "" {
		return Config{}, fmt.Errorf("VETSCRIBE_LISTEN_ADDR must not be empty")
	}

	return cfg, nil
}

func envOr(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}
