// Package config loads runtime configuration from the environment.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

type Config struct {
	Env             string
	Addr            string
	DatabaseURL     string
	LogLevel        slog.Level
	MediaHosts      []string
	ShutdownTimeout time.Duration
}

// Load reads configuration, returning an error for any missing or invalid value.
func Load() (Config, error) {
	c := Config{
		Env:             env("FAJR_ENV", "development"),
		Addr:            env("FAJR_ADDR", ":8080"),
		DatabaseURL:     env("FAJR_DATABASE_URL", ""),
		ShutdownTimeout: 15 * time.Second,
	}

	if c.DatabaseURL == "" {
		return Config{}, fmt.Errorf("config: FAJR_DATABASE_URL is required")
	}

	// Extra video hosts an operator trusts, on top of the built-in platforms.
	if hosts := env("FAJR_MEDIA_HOSTS", ""); hosts != "" {
		for _, h := range strings.Split(hosts, ",") {
			if h = strings.TrimSpace(h); h != "" {
				c.MediaHosts = append(c.MediaHosts, h)
			}
		}
	}

	lvl := env("FAJR_LOG_LEVEL", "info")
	if err := c.LogLevel.UnmarshalText([]byte(lvl)); err != nil {
		return Config{}, fmt.Errorf("config: invalid FAJR_LOG_LEVEL %q: %w", lvl, err)
	}

	switch c.Env {
	case "development", "staging", "production":
	default:
		return Config{}, fmt.Errorf("config: invalid FAJR_ENV %q", c.Env)
	}
	return c, nil
}

func (c Config) IsProduction() bool { return c.Env == "production" }

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
