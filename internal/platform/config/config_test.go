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
}

func TestLoadRejectsInvalidShutdownTimeout(t *testing.T) {
	t.Setenv("PULSO_DATABASE_URL", "postgres://localhost/pulso")
	t.Setenv("PULSO_SHUTDOWN_TIMEOUT", "not-a-duration")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected invalid shutdown timeout error")
	}
}
