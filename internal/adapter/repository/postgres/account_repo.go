package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/infrastructure/postgres/generated"
	"github.com/iho/goledger/internal/usecase"
)

// AccountRepository implements usecase.AccountRepository.
type AccountRepository struct {
	pool    *pgxpool.Pool
	queries *generated.Queries
}

// NewAccountRepository creates a new AccountRepository.
func NewAccountRepository(pool *pgxpool.Pool) *AccountRepository {
	return &AccountRepository{
		pool:    pool,
		queries: generated.New(pool),
	}
}

// Create creates a new account.
func (r *AccountRepository) Create(ctx context.Context, account *domain.Account) error {
	_, err := r.queries.CreateAccount(ctx, generated.CreateAccountParams{
		ID:                   account.ID,
		Name:                 account.Name,
		Currency:             account.Currency,
		Balance:              decimalToNumeric(account.Balance),
		Version:              account.Version,
		AllowNegativeBalance: account.AllowNegativeBalance,
		AllowPositiveBalance: account.AllowPositiveBalance,
		CreatedAt:            timeToPgTimestamptz(account.CreatedAt),
		UpdatedAt:            timeToPgTimestamptz(account.UpdatedAt),
	})

	return err
}

// GetByID retrieves an account by ID.
func (r *AccountRepository) GetByID(ctx context.Context, id string) (*domain.Account, error) {
	row, err := r.queries.GetAccountByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAccountNotFound
		}

		return nil, err
	}

	return rowToAccount(row), nil
}

// GetByIDForUpdate retrieves an account by ID with a FOR UPDATE lock.
func (r *AccountRepository) GetByIDForUpdate(ctx context.Context, tx usecase.Transaction, id string) (*domain.Account, error) {
	pgxTx := tx.(*Tx).PgxTx()
	queries := generated.New(pgxTx)

	row, err := queries.GetAccountByIDForUpdate(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAccountNotFound
		}

		return nil, err
	}

	return rowToAccount(row), nil
}

// GetByIDsForUpdate retrieves multiple accounts by IDs with FOR UPDATE locks.
func (r *AccountRepository) GetByIDsForUpdate(ctx context.Context, tx usecase.Transaction, ids []string) ([]*domain.Account, error) {
	pgxTx := tx.(*Tx).PgxTx()
	queries := generated.New(pgxTx)

	rows, err := queries.GetAccountsByIDsForUpdate(ctx, ids)
	if err != nil {
		return nil, err
	}

	accounts := make([]*domain.Account, 0, len(rows))
	for _, row := range rows {
		accounts = append(accounts, rowToAccount(row))
	}

	return accounts, nil
}

// UpdateBalance updates the balance of an account.
func (r *AccountRepository) UpdateBalance(ctx context.Context, tx usecase.Transaction, id string, balance decimal.Decimal, updatedAt time.Time) error {
	pgxTx := tx.(*Tx).PgxTx()
	queries := generated.New(pgxTx)

	return queries.UpdateAccountBalance(ctx, generated.UpdateAccountBalanceParams{
		ID:        id,
		Balance:   decimalToNumeric(balance),
		UpdatedAt: timeToPgTimestamptz(updatedAt),
	})
}

// UpdateEncumberedBalance updates the encumbered balance of an account.
func (r *AccountRepository) UpdateEncumberedBalance(ctx context.Context, tx usecase.Transaction, id string, encumberedBalance decimal.Decimal, updatedAt time.Time) error {
	pgxTx := tx.(*Tx).PgxTx()
	queries := generated.New(pgxTx)

	return queries.UpdateAccountEncumbered(ctx, generated.UpdateAccountEncumberedParams{
		ID:                id,
		EncumberedBalance: decimalToNumeric(encumberedBalance),
		UpdatedAt:         timeToPgTimestamptz(updatedAt),
	})
}

// List lists accounts with pagination.
func (r *AccountRepository) List(ctx context.Context, limit, offset int) ([]*domain.Account, error) {
	rows, err := r.queries.ListAccounts(ctx, generated.ListAccountsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}

	accounts := make([]*domain.Account, 0, len(rows))
	for _, row := range rows {
		accounts = append(accounts, rowToAccount(row))
	}

	return accounts, nil
}

func rowToAccount(row generated.Account) *domain.Account {
	return &domain.Account{
		ID:                   row.ID,
		Name:                 row.Name,
		Currency:             row.Currency,
		Balance:              numericToDecimal(row.Balance),
		EncumberedBalance:    numericToDecimal(row.EncumberedBalance),
		Version:              row.Version,
		AllowNegativeBalance: row.AllowNegativeBalance,
		AllowPositiveBalance: row.AllowPositiveBalance,
		CreatedAt:            row.CreatedAt.Time,
		UpdatedAt:            row.UpdatedAt.Time,
	}
}

// Type conversion helpers.
func decimalToNumeric(d decimal.Decimal) pgtype.Numeric {
	var n pgtype.Numeric

	_ = n.Scan(d.String())

	return n
}

func numericToDecimal(n pgtype.Numeric) decimal.Decimal {
	if !n.Valid {
		return decimal.Zero
	}

	d, _ := decimal.NewFromString(n.Int.String())
	if n.Exp != 0 {
		d = d.Shift(n.Exp)
	}

	return d
}

func timeToPgTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}
