package config_test

import (
	"testing"
	"time"

	"pulso/internal/platform/config"
)

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("PULSO_DATABASE_URL", "")

	_, err := config.Load()
	if err == nil || err.Error() != "PULSO_DATABASE_URL is required" {
		t.Fatalf("expected required database URL error, got %v", err)
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("PULSO_DATABASE_URL", "postgres://localhost/pulso")
	t.Setenv("PULSO_ENV", "")
	t.Setenv("PULSO_API_ADDR", "")
	t.Setenv("PULSO_SHUTDOWN_TIMEOUT", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Environment != "development" {
		t.Errorf("Environment = %q, want development", cfg.Environment)
	}
	if cfg.APIAddress != ":8080" {
		t.Errorf("APIAddress = %q, want :8080", cfg.APIAddress)
	}
	if cfg.ShutdownTimeout != 10*time.Second {
		t.Errorf("ShutdownTimeout = %s, want 10s", cfg.ShutdownTimeout)
	}
	if cfg.WebOrigin != "http://localhost:5173" {
		t.Errorf("WebOrigin = %q", cfg.WebOrigin)
	}
	if cfg.CookieSecure {
		t.Fatal("CookieSecure must default to false in development")
	}
	if cfg.WorkerConcurrency != 10 || cfg.WorkerBatchSize != 20 || cfg.WorkerPollInterval != time.Second || len(cfg.CheckAllowPrivateHosts) != 0 {
		t.Fatalf("worker defaults = %#v", cfg)
	}
	if cfg.SMTPHost != "mailpit" || cfg.SMTPPort != 1025 || cfg.SMTPFrom != "pulso@localhost" {
		t.Fatalf("SMTP defaults = %#v", cfg)
	}
}

func TestLoadRejectsPrivateHostExceptionsInProduction(t *testing.T) {
	t.Setenv("PULSO_DATABASE_URL", "postgres://localhost/pulso")
	t.Setenv("PULSO_ENV", "production")
	t.Setenv("PULSO_CHECK_ALLOW_PRIVATE_HOSTS", "127.0.0.1")

	_, err := config.Load()
	if err == nil || err.Error() != "PULSO_CHECK_ALLOW_PRIVATE_HOSTS is not allowed in production" {
		t.Fatalf("Load() error = %v, want production private-host rejection", err)
	}
}

func TestLoadRejectsInvalidWorkerConcurrency(t *testing.T) {
	t.Setenv("PULSO_DATABASE_URL", "postgres://localhost/pulso")
	t.Setenv("PULSO_WORKER_CONCURRENCY", "0")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected invalid worker concurrency error")
	}
}

func TestLoadRejectsInvalidCookieSecure(t *testing.T) {
	t.Setenv("PULSO_DATABASE_URL", "postgres://localhost/pulso")
	t.Setenv("PULSO_COOKIE_SECURE", "sometimes")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected invalid cookie secure error")
	}
}

func TestLoadRejectsInvalidShutdownTimeout(t *testing.T) {
	t.Setenv("PULSO_DATABASE_URL", "postgres://localhost/pulso")
	t.Setenv("PULSO_SHUTDOWN_TIMEOUT", "not-a-duration")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected invalid shutdown timeout error")
	}
}
