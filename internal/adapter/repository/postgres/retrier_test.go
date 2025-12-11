package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestRetrierRetriesOnRetryableError(t *testing.T) {
	r := NewRetrier()
	r.maxRetries = 2
	r.initialInterval = 1 * time.Millisecond
	r.maxInterval = 2 * time.Millisecond
	r.maxElapsedTime = 10 * time.Millisecond

	attempts := 0
	err := r.Retry(context.Background(), func() error {
		attempts++
		if attempts < 2 {
			return &pgconn.PgError{Code: pgErrDeadlock}
		}
		return nil
	})

	if err != nil {
		t.Fatalf("expected success after retry, got %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestRetrierStopsOnPermanentError(t *testing.T) {
	r := NewRetrier()
	attempts := 0
	permanentErr := errors.New("permanent")

	err := r.Retry(context.Background(), func() error {
		attempts++
		return permanentErr
	})

	if !errors.Is(err, permanentErr) {
		t.Fatalf("expected permanent error, got %v", err)
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
}

func TestIsRetryableError(t *testing.T) {
	retryableErr := &pgconn.PgError{Code: pgErrDeadlock}
	if !isRetryableError(retryableErr) {
		t.Fatalf("expected deadlock error to be retryable")
	}

	nonRetryable := errors.New("other")
	if isRetryableError(nonRetryable) {
		t.Fatalf("expected generic error to be non-retryable")
	}
}
