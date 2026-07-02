package postgres

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolConfig holds configuration for the database pool.
type PoolConfig struct {
	DatabaseURL     string
	MaxConns        int
	MinConns        int
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	QueryTimeout    time.Duration
}

// NewPool creates a new PostgreSQL connection pool.
func NewPool(ctx context.Context, databaseURL string, maxConns, minConns int) (*pgxpool.Pool, error) {
	return NewPoolWithConfig(ctx, PoolConfig{
		DatabaseURL:     databaseURL,
		MaxConns:        maxConns,
		MinConns:        minConns,
		MaxConnLifetime: time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
	})
}

// NewPoolWithConfig creates a new PostgreSQL connection pool with full config.
func NewPoolWithConfig(ctx context.Context, cfg PoolConfig) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	config.MaxConns = toInt32(cfg.MaxConns)
	config.MinConns = toInt32(cfg.MinConns)
	config.MaxConnLifetime = cfg.MaxConnLifetime
	config.MaxConnIdleTime = cfg.MaxConnIdleTime
	config.ConnConfig.Tracer = newOtelQueryTracer()

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

// toInt32 clamps an int to the int32 range before it's used to configure
// the pool, so the conversion can never silently overflow/wrap.
func toInt32(n int) int32 {
	switch {
	case n > math.MaxInt32:
		return math.MaxInt32
	case n < math.MinInt32:
		return math.MinInt32
	default:
		return int32(n)
	}
}
