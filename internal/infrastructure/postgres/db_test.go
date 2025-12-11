package postgres

import (
	"context"
	"testing"
)

func TestNewPoolWithConfigDefaults(t *testing.T) {
	ctx := context.Background()

	// using invalid URL should return error
	if _, err := NewPoolWithConfig(ctx, PoolConfig{DatabaseURL: "not-a-url"}); err == nil {
		t.Fatalf("expected error when parsing invalid URL")
	}
}

func TestNewPoolWithConfigPingFailure(t *testing.T) {
	ctx := context.Background()
	cfg := PoolConfig{
		DatabaseURL: "postgres://invalid:5432/db",
		MaxConns:    1,
		MinConns:    0,
	}

	_, err := NewPoolWithConfig(ctx, cfg)
	if err == nil {
		t.Fatalf("expected error when pool cannot connect")
	}
}
