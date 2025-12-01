package config

import (
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

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{}
	err := env.Parse(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
