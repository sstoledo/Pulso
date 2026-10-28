package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultShutdownTimeout = 10 * time.Second

type Config struct {
	Environment            string
	APIAddress             string
	DatabaseURL            string
	WebOrigin              string
	CookieSecure           bool
	ShutdownTimeout        time.Duration
	WorkerConcurrency      int
	WorkerBatchSize        int
	WorkerPollInterval     time.Duration
	CheckAllowPrivateHosts []string
	SMTPHost               string
	SMTPPort               int
	SMTPFrom               string
}

func Load() (Config, error) {
	cfg := Config{
		Environment:            valueOrDefault("PULSO_ENV", "development"),
		APIAddress:             valueOrDefault("PULSO_API_ADDR", ":8080"),
		DatabaseURL:            os.Getenv("PULSO_DATABASE_URL"),
		WebOrigin:              valueOrDefault("PULSO_WEB_ORIGIN", "http://localhost:5173"),
		ShutdownTimeout:        defaultShutdownTimeout,
		WorkerConcurrency:      10,
		WorkerBatchSize:        20,
		WorkerPollInterval:     time.Second,
		CheckAllowPrivateHosts: parseHostList(os.Getenv("PULSO_CHECK_ALLOW_PRIVATE_HOSTS")),
		SMTPHost:               valueOrDefault("PULSO_SMTP_HOST", "mailpit"),
		SMTPPort:               1025,
		SMTPFrom:               valueOrDefault("PULSO_SMTP_FROM", "pulso@localhost"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("PULSO_DATABASE_URL is required")
	}

	if raw := os.Getenv("PULSO_SMTP_PORT"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 65535 {
			return Config{}, fmt.Errorf("PULSO_SMTP_PORT must be between 1 and 65535: %q", raw)
		}
		cfg.SMTPPort = value
	}

	if strings.EqualFold(cfg.Environment, "production") && len(cfg.CheckAllowPrivateHosts) > 0 {
		return Config{}, errors.New("PULSO_CHECK_ALLOW_PRIVATE_HOSTS is not allowed in production")
	}

	if raw := os.Getenv("PULSO_WORKER_CONCURRENCY"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 100 {
			return Config{}, fmt.Errorf("PULSO_WORKER_CONCURRENCY must be between 1 and 100: %q", raw)
		}
		cfg.WorkerConcurrency = value
	}
	if raw := os.Getenv("PULSO_WORKER_BATCH_SIZE"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 100 {
			return Config{}, fmt.Errorf("PULSO_WORKER_BATCH_SIZE must be between 1 and 100: %q", raw)
		}
		cfg.WorkerBatchSize = value
	}
	if raw := os.Getenv("PULSO_WORKER_POLL_INTERVAL"); raw != "" {
		value, err := time.ParseDuration(raw)
		if err != nil || value <= 0 {
			return Config{}, fmt.Errorf("PULSO_WORKER_POLL_INTERVAL must be a positive duration: %q", raw)
		}
		cfg.WorkerPollInterval = value
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

func parseHostList(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	hosts := make([]string, 0, len(parts))
	for _, part := range parts {
		if host := strings.TrimSpace(part); host != "" {
			hosts = append(hosts, host)
		}
	}
	return hosts
}
