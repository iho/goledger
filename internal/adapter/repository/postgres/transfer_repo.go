package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/infrastructure/postgres/generated"
	"github.com/iho/goledger/internal/usecase"
)

// TransferRepository implements usecase.TransferRepository.
type TransferRepository struct {
	pool    *pgxpool.Pool
	queries *generated.Queries
}

// NewTransferRepository creates a new TransferRepository.
func NewTransferRepository(pool *pgxpool.Pool) *TransferRepository {
	return &TransferRepository{
		pool:    pool,
		queries: generated.New(pool),
	}
}

// Create creates a new transfer.
func (r *TransferRepository) Create(ctx context.Context, tx usecase.Transaction, transfer *domain.Transfer) error {
	pgxTx := tx.(*Tx).PgxTx()
	queries := generated.New(pgxTx)

	var metadata []byte
	if transfer.Metadata != nil {
		var err error

		metadata, err = json.Marshal(transfer.Metadata)
		if err != nil {
			return err
		}
	}

	_, err := queries.CreateTransfer(ctx, generated.CreateTransferParams{
		ID:                 transfer.ID,
		FromAccountID:      transfer.FromAccountID,
		ToAccountID:        transfer.ToAccountID,
		Amount:             decimalToNumeric(transfer.Amount),
		CreatedAt:          timeToPgTimestamptz(transfer.CreatedAt),
		EventAt:            timeToPgTimestamptz(transfer.EventAt),
		Metadata:           metadata,
		ReversedTransferID: transfer.ReversedTransferID,
	})

	return err
}

// GetByID retrieves a transfer by ID.
func (r *TransferRepository) GetByID(ctx context.Context, id string) (*domain.Transfer, error) {
	row, err := r.queries.GetTransferByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTransferNotFound
		}

		return nil, err
	}

	return rowToTransfer(row), nil
}

// ListByAccount lists transfers for an account.
func (r *TransferRepository) ListByAccount(ctx context.Context, accountID string, limit, offset int) ([]*domain.Transfer, error) {
	rows, err := r.queries.ListTransfersByAccount(ctx, generated.ListTransfersByAccountParams{
		FromAccountID: accountID,
		Limit:         int32(limit),
		Offset:        int32(offset),
	})
	if err != nil {
		return nil, err
	}

	transfers := make([]*domain.Transfer, 0, len(rows))
	for _, row := range rows {
		transfers = append(transfers, rowToTransfer(row))
	}

	return transfers, nil
}

func rowToTransfer(row generated.Transfer) *domain.Transfer {
	var metadata map[string]any
	if row.Metadata != nil {
		_ = json.Unmarshal(row.Metadata, &metadata)
	}

	return &domain.Transfer{
		ID:                 row.ID,
		FromAccountID:      row.FromAccountID,
		ToAccountID:        row.ToAccountID,
		Amount:             numericToDecimal(row.Amount),
		CreatedAt:          row.CreatedAt.Time,
		EventAt:            row.EventAt.Time,
		Metadata:           metadata,
		ReversedTransferID: row.ReversedTransferID,
	}
}
