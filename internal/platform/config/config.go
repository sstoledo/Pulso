package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

const defaultShutdownTimeout = 10 * time.Second

type Config struct {
	Environment     string
	APIAddress      string
	DatabaseURL     string
	WebOrigin       string
	CookieSecure    bool
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Environment:     valueOrDefault("PULSO_ENV", "development"),
		APIAddress:      valueOrDefault("PULSO_API_ADDR", ":8080"),
		DatabaseURL:     os.Getenv("PULSO_DATABASE_URL"),
		WebOrigin:       valueOrDefault("PULSO_WEB_ORIGIN", "http://localhost:5173"),
		ShutdownTimeout: defaultShutdownTimeout,
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("PULSO_DATABASE_URL is required")
	}

	if raw := os.Getenv("PULSO_COOKIE_SECURE"); raw != "" {
		secure, err := strconv.ParseBool(raw)
		if err != nil {
			return Config{}, fmt.Errorf("PULSO_COOKIE_SECURE must be a boolean: %q", raw)
		}
		cfg.CookieSecure = secure
	}

	if raw := os.Getenv("PULSO_SHUTDOWN_TIMEOUT"); raw != "" {
		timeout, err := time.ParseDuration(raw)
		if err != nil || timeout <= 0 {
			return Config{}, fmt.Errorf("PULSO_SHUTDOWN_TIMEOUT must be a positive duration: %q", raw)
		}
		cfg.ShutdownTimeout = timeout
	}

	return cfg, nil
}

func valueOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
