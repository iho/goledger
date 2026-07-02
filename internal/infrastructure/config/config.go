package config

import (
	"fmt"
	"strconv"
	"time"

	"github.com/caarlos0/env/v10"
)

// Config holds all application configuration.
type Config struct {
	// Database
	DatabaseURL      string        `env:"DATABASE_URL"       envDefault:"postgres://ledger:ledger@localhost:5432/ledger?sslmode=disable"`
	DatabaseMaxConns int           `env:"DATABASE_MAX_CONNS" envDefault:"25"`
	DatabaseMinConns int           `env:"DATABASE_MIN_CONNS" envDefault:"5"`
	DatabaseTimeout  time.Duration `env:"DATABASE_TIMEOUT"   envDefault:"30s"`

	// Redis
	RedisURL string `env:"REDIS_URL" envDefault:"redis://localhost:6379"`

	// HTTP Server
	HTTPPort            string        `env:"HTTP_PORT"             envDefault:"8080"`
	HTTPReadTimeout     time.Duration `env:"HTTP_READ_TIMEOUT"     envDefault:"30s"`
	HTTPWriteTimeout    time.Duration `env:"HTTP_WRITE_TIMEOUT"    envDefault:"30s"`
	HTTPIdleTimeout     time.Duration `env:"HTTP_IDLE_TIMEOUT"     envDefault:"60s"`
	HTTPShutdownTimeout time.Duration `env:"HTTP_SHUTDOWN_TIMEOUT" envDefault:"10s"`

	// Logging
	LogLevel  string `env:"LOG_LEVEL"  envDefault:"info"`
	LogFormat string `env:"LOG_FORMAT" envDefault:"json"`

	// Idempotency
	IdempotencyTTL time.Duration `env:"IDEMPOTENCY_TTL" envDefault:"24h"`

	// Authentication (optional - leave empty to disable)
	JWTSecret     string        `env:"JWT_SECRET"       envDefault:""`
	JWTExpiration time.Duration `env:"JWT_EXPIRATION"   envDefault:"24h"`
	AuthEnabled   bool          `env:"AUTH_ENABLED"     envDefault:"false"`
}

// Load loads configuration from environment variables and validates it.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks the loaded configuration for values that would otherwise
// fail confusingly (or silently misbehave) at runtime rather than at
// startup.
func (c *Config) Validate() error {
	if c.AuthEnabled && c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET must be set when AUTH_ENABLED is true")
	}

	if _, err := strconv.ParseUint(c.HTTPPort, 10, 16); err != nil {
		return fmt.Errorf("HTTP_PORT must be a valid port number: %w", err)
	}

	if c.DatabaseMaxConns <= 0 {
		return fmt.Errorf("DATABASE_MAX_CONNS must be positive, got %d", c.DatabaseMaxConns)
	}

	if c.DatabaseMinConns < 0 {
		return fmt.Errorf("DATABASE_MIN_CONNS must not be negative, got %d", c.DatabaseMinConns)
	}

	if c.DatabaseMinConns > c.DatabaseMaxConns {
		return fmt.Errorf("DATABASE_MIN_CONNS (%d) must not exceed DATABASE_MAX_CONNS (%d)", c.DatabaseMinConns, c.DatabaseMaxConns)
	}

	return nil
}
