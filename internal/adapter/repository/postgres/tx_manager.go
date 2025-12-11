package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iho/goledger/internal/usecase"
)

// TxManager implements usecase.TransactionManager.
type pgxPool interface {
	Begin(context.Context) (pgx.Tx, error)
}

type TxManager struct {
	pool pgxPool
}

// NewTxManager creates a new TxManager.
func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return newTxManagerWithPool(pool)
}

func newTxManagerWithPool(pool pgxPool) *TxManager {
	return &TxManager{pool: pool}
}

// Begin starts a new transaction.
func (m *TxManager) Begin(ctx context.Context) (usecase.Transaction, error) {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}

	return &Tx{tx: tx}, nil
}

// Tx wraps a pgx transaction.
type Tx struct {
	tx pgx.Tx
}

// Commit commits the transaction.
func (t *Tx) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

// Rollback rolls back the transaction.
func (t *Tx) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

// PgxTx returns the underlying pgx.Tx.
func (t *Tx) PgxTx() pgx.Tx {
	return t.tx
}
