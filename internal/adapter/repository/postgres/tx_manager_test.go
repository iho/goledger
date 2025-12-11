package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestTxManagerBeginSuccess(t *testing.T) {
	mockPool := newMockPool(t)
	mockPool.ExpectBegin()
	mockPool.ExpectCommit()

	manager := newTxManagerWithPool(mockPool)
	tx, err := manager.Begin(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tx == nil {
		t.Fatalf("expected transaction")
	}

	if err := tx.Commit(context.Background()); err != nil {
		t.Fatalf("commit failed: %v", err)
	}

	assertExpectations(t, mockPool)
}

func TestTxManagerBeginError(t *testing.T) {
	mockPool := newMockPool(t)
	mockErr := errors.New("begin failed")
	mockPool.ExpectBegin().WillReturnError(mockErr)

	manager := newTxManagerWithPool(mockPool)
	tx, err := manager.Begin(context.Background())
	if err == nil || !errors.Is(err, mockErr) {
		t.Fatalf("expected begin error, got err=%v tx=%v", err, tx)
	}
}

func TestTxRollback(t *testing.T) {
	mockPool := newMockPool(t)
	mockPool.ExpectBegin()
	mockPool.ExpectRollback()

	manager := newTxManagerWithPool(mockPool)
	tx, err := manager.Begin(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := tx.Rollback(context.Background()); err != nil {
		t.Fatalf("rollback failed: %v", err)
	}

	assertExpectations(t, mockPool)
}

func newMockPool(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func assertExpectations(t *testing.T, pool pgxmock.PgxPoolIface) {
	t.Helper()
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}
