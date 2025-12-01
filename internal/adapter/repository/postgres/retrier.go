package postgres

import (
	"context"
	"errors"
	"time"

	"log/slog"

	"github.com/cenkalti/backoff/v4"
	"github.com/jackc/pgx/v5/pgconn"
)

// PostgreSQL error codes for retryable errors.
const (
	pgErrDeadlock             = "40P01"
	pgErrSerializationFailure = "40001"
)

// Retrier implements usecase.Retrier with exponential backoff.
type Retrier struct {
	maxRetries      int
	initialInterval time.Duration
	maxInterval     time.Duration
	maxElapsedTime  time.Duration
	logger          *slog.Logger
}

// NewRetrier creates a new PostgreSQL retrier with default settings.
func NewRetrier() *Retrier {
	return &Retrier{
		maxRetries:      3,
		initialInterval: 50 * time.Millisecond,
		maxInterval:     1 * time.Second,
		maxElapsedTime:  10 * time.Second,
		logger:          slog.Default(),
	}
}

// Retry executes an operation with exponential backoff on retryable errors.
func (r *Retrier) Retry(ctx context.Context, operation func() error) error {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = r.initialInterval
	b.MaxInterval = r.maxInterval
	b.MaxElapsedTime = r.maxElapsedTime

	retryCount := 0

	return backoff.Retry(func() error {
		err := operation()
		if err == nil {
			return nil
		}

		if !isRetryableError(err) {
			return backoff.Permanent(err)
		}

		retryCount++
		if retryCount > r.maxRetries {
			return backoff.Permanent(err)
		}

		r.logger.Warn("retryable database error, retrying",
			"error", err,
			"retry", retryCount,
		)

		return err
	}, backoff.WithContext(b, ctx))
}

// isRetryableError checks if a PostgreSQL error should trigger a retry.
func isRetryableError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgErrDeadlock, pgErrSerializationFailure:
			return true
		}
	}
	return false
}
