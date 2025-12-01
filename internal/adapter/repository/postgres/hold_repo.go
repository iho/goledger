package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/infrastructure/postgres/generated"
	"github.com/iho/goledger/internal/usecase"
)

// HoldRepository implements usecase.HoldRepository.
type HoldRepository struct {
	pool    *pgxpool.Pool
	queries *generated.Queries
}

// NewHoldRepository creates a new HoldRepository.
func NewHoldRepository(pool *pgxpool.Pool) *HoldRepository {
	return &HoldRepository{
		pool:    pool,
		queries: generated.New(pool),
	}
}

// Create creates a new hold.
func (r *HoldRepository) Create(ctx context.Context, tx usecase.Transaction, hold *domain.Hold) error {
	pgxTx := tx.(*Tx).PgxTx()
	queries := generated.New(pgxTx)

	var expiresAt pgtype.Timestamptz
	if hold.ExpiresAt != nil {
		expiresAt = timeToPgTimestamptz(*hold.ExpiresAt)
	}

	_, err := queries.CreateHold(ctx, generated.CreateHoldParams{
		ID:        hold.ID,
		AccountID: hold.AccountID,
		Amount:    decimalToNumeric(hold.Amount),
		Status:    string(hold.Status),
		ExpiresAt: expiresAt,
		Metadata:  nil, // TODO: map metadata to JSONB
		CreatedAt: timeToPgTimestamptz(hold.CreatedAt),
		UpdatedAt: timeToPgTimestamptz(hold.UpdatedAt),
	})

	return err
}

// GetByID retrieves a hold by ID.
func (r *HoldRepository) GetByID(ctx context.Context, id string) (*domain.Hold, error) {
	row, err := r.queries.GetHoldByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrHoldNotFound
		}
		return nil, err
	}

	return rowToHold(row), nil
}

// GetByIDForUpdate retrieves a hold by ID with a FOR UPDATE lock.
func (r *HoldRepository) GetByIDForUpdate(ctx context.Context, tx usecase.Transaction, id string) (*domain.Hold, error) {
	pgxTx := tx.(*Tx).PgxTx()
	queries := generated.New(pgxTx)

	row, err := queries.GetHoldByIDForUpdate(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrHoldNotFound
		}
		return nil, err
	}

	return rowToHold(row), nil
}

// UpdateStatus updates the status of a hold.
func (r *HoldRepository) UpdateStatus(ctx context.Context, tx usecase.Transaction, id string, status domain.HoldStatus, updatedAt time.Time) error {
	pgxTx := tx.(*Tx).PgxTx()
	queries := generated.New(pgxTx)

	return queries.UpdateHoldStatus(ctx, generated.UpdateHoldStatusParams{
		ID:        id,
		Status:    string(status),
		UpdatedAt: timeToPgTimestamptz(updatedAt),
	})
}

// ListByAccount lists holds for an account.
func (r *HoldRepository) ListByAccount(ctx context.Context, accountID string, limit, offset int) ([]*domain.Hold, error) {
	rows, err := r.queries.ListHoldsByAccount(ctx, generated.ListHoldsByAccountParams{
		AccountID: accountID,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return nil, err
	}

	holds := make([]*domain.Hold, 0, len(rows))
	for _, row := range rows {
		holds = append(holds, rowToHold(row))
	}

	return holds, nil
}

func rowToHold(row generated.Hold) *domain.Hold {
	var expiresAt *time.Time
	if row.ExpiresAt.Valid {
		t := row.ExpiresAt.Time
		expiresAt = &t
	}

	return &domain.Hold{
		ID:        row.ID,
		AccountID: row.AccountID,
		Amount:    numericToDecimal(row.Amount),
		Status:    domain.HoldStatus(row.Status),
		ExpiresAt: expiresAt,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}
