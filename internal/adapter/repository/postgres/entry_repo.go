
package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/infrastructure/postgres/generated"
	"github.com/iho/goledger/internal/usecase"
)

// EntryRepository implements usecase.EntryRepository.
type EntryRepository struct {
	pool    *pgxpool.Pool
	queries *generated.Queries
}

// NewEntryRepository creates a new EntryRepository.
func NewEntryRepository(pool *pgxpool.Pool) *EntryRepository {
	return &EntryRepository{
		pool:    pool,
		queries: generated.New(pool),
	}
}

// Create creates a new entry.
func (r *EntryRepository) Create(ctx context.Context, tx usecase.Transaction, entry *domain.Entry) error {
	pgxTx := tx.(*Tx).PgxTx()
	queries := generated.New(pgxTx)

	_, err := queries.CreateEntry(ctx, generated.CreateEntryParams{
		ID:                     entry.ID,
		AccountID:              entry.AccountID,
		TransferID:             entry.TransferID,
		Amount:                 decimalToNumeric(entry.Amount),
		AccountPreviousBalance: decimalToNumeric(entry.AccountPreviousBalance),
		AccountCurrentBalance:  decimalToNumeric(entry.AccountCurrentBalance),
		AccountVersion:         entry.AccountVersion,
		CreatedAt:              timeToPgTimestamptz(entry.CreatedAt),
	})

	return err
}

// GetByTransfer retrieves entries by transfer ID.
func (r *EntryRepository) GetByTransfer(ctx context.Context, transferID string) ([]*domain.Entry, error) {
	rows, err := r.queries.GetEntriesByTransfer(ctx, transferID)
	if err != nil {
		return nil, err
	}

	entries := make([]*domain.Entry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, rowToEntry(row))
	}

	return entries, nil
}

// GetByAccount retrieves entries by account ID.
func (r *EntryRepository) GetByAccount(ctx context.Context, accountID string, limit, offset int) ([]*domain.Entry, error) {
	rows, err := r.queries.GetEntriesByAccount(ctx, generated.GetEntriesByAccountParams{
		AccountID: accountID,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return nil, err
	}

	entries := make([]*domain.Entry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, &domain.Entry{
			ID:                     row.ID,
			AccountID:              row.AccountID,
			TransferID:             row.TransferID,
			Amount:                 numericToDecimal(row.Amount),
			AccountPreviousBalance: numericToDecimal(row.AccountPreviousBalance),
			AccountCurrentBalance:  numericToDecimal(row.AccountCurrentBalance),
			AccountVersion:         row.AccountVersion,
			CreatedAt:              row.CreatedAt.Time,
		})
	}

	return entries, nil
}

// GetBalanceAtTime retrieves the balance at a specific time.
func (r *EntryRepository) GetBalanceAtTime(ctx context.Context, accountID string, at time.Time) (decimal.Decimal, error) {
	balance, err := r.queries.GetAccountBalanceAtTime(ctx, generated.GetAccountBalanceAtTimeParams{
		AccountID: accountID,
		CreatedAt: timeToPgTimestamptz(at),
	})
	if err != nil {
		return decimal.Zero, err
	}

	return numericToDecimal(balance), nil
}

func rowToEntry(row generated.Entry) *domain.Entry {
	return &domain.Entry{
		ID:                     row.ID,
		AccountID:              row.AccountID,
		TransferID:             row.TransferID,
		Amount:                 numericToDecimal(row.Amount),
		AccountPreviousBalance: numericToDecimal(row.AccountPreviousBalance),
		AccountCurrentBalance:  numericToDecimal(row.AccountCurrentBalance),
		AccountVersion:         row.AccountVersion,
		CreatedAt:              row.CreatedAt.Time,
	}
}
