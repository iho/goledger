package config_test

import (
	"os"
	"testing"
	"time"

	"github.com/iho/goledger/internal/infrastructure/config"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}

	if cfg.DatabaseURL == "" {
		t.Fatalf("expected default database URL to be set")
	}

	if cfg.JWTSecret != "" {
		t.Fatalf("expected JWT secret default to be empty, got %q", cfg.JWTSecret)
	}

	if cfg.HTTPPort != "8080" {
		t.Fatalf("expected default HTTP port 8080, got %s", cfg.HTTPPort)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("REDIS_URL", "redis://example")
	t.Setenv("HTTP_PORT", "9090")
	t.Setenv("DATABASE_TIMEOUT", "45s")
	t.Setenv("JWT_SECRET", "top-secret")
	t.Setenv("AUTH_ENABLED", "true")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}

	if cfg.DatabaseURL != "postgres://example" {
		t.Fatalf("expected custom database URL, got %s", cfg.DatabaseURL)
	}

	if cfg.RedisURL != "redis://example" {
		t.Fatalf("expected custom redis URL, got %s", cfg.RedisURL)
	}

	if cfg.HTTPPort != "9090" {
		t.Fatalf("expected HTTP port override, got %s", cfg.HTTPPort)
	}

	if cfg.DatabaseTimeout != 45*time.Second {
		t.Fatalf("expected database timeout override, got %s", cfg.DatabaseTimeout)
	}

	if cfg.JWTSecret != "top-secret" || !cfg.AuthEnabled {
		t.Fatalf("expected auth settings to be set, got secret=%s enabled=%v", cfg.JWTSecret, cfg.AuthEnabled)
	}
}

func TestLoadInvalidDuration(t *testing.T) {
	original := os.Getenv("HTTP_READ_TIMEOUT")
	t.Setenv("HTTP_READ_TIMEOUT", "not-a-duration")
	t.Cleanup(func() {
		t.Setenv("HTTP_READ_TIMEOUT", original)
	})

	if _, err := config.Load(); err == nil {
		t.Fatalf("expected error for invalid duration")
	}
}

func TestLoadAuthEnabledWithoutSecret(t *testing.T) {
	t.Setenv("AUTH_ENABLED", "true")
	t.Setenv("JWT_SECRET", "")

	if _, err := config.Load(); err == nil {
		t.Fatalf("expected error when AUTH_ENABLED is true and JWT_SECRET is empty")
	}
}

func TestLoadAuthEnabledWithSecret(t *testing.T) {
	t.Setenv("AUTH_ENABLED", "true")
	t.Setenv("JWT_SECRET", "top-secret")

	if _, err := config.Load(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadInvalidHTTPPort(t *testing.T) {
	t.Setenv("HTTP_PORT", "not-a-port")

	if _, err := config.Load(); err == nil {
		t.Fatalf("expected error for invalid HTTP_PORT")
	}
}

func TestLoadDatabaseMinConnsExceedsMaxConns(t *testing.T) {
	t.Setenv("DATABASE_MIN_CONNS", "50")
	t.Setenv("DATABASE_MAX_CONNS", "25")

	if _, err := config.Load(); err == nil {
		t.Fatalf("expected error when DATABASE_MIN_CONNS exceeds DATABASE_MAX_CONNS")
	}
}

func TestLoadDatabaseMaxConnsNotPositive(t *testing.T) {
	t.Setenv("DATABASE_MAX_CONNS", "0")

	if _, err := config.Load(); err == nil {
		t.Fatalf("expected error when DATABASE_MAX_CONNS is not positive")
	}
}
