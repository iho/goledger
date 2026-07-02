package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/iho/goledger/internal/infrastructure/postgres/generated"
	"github.com/iho/goledger/internal/usecase"
)

// LedgerRepository implements usecase.LedgerRepository.
type LedgerRepository struct {
	pool *pgxpool.Pool
}

// NewLedgerRepository creates a new LedgerRepository.
func NewLedgerRepository(pool *pgxpool.Pool) *LedgerRepository {
	return &LedgerRepository{pool: pool}
}

// CheckConsistency checks the consistency of the ledger.
func (r *LedgerRepository) CheckConsistency(ctx context.Context) (totalDebits, totalCredits decimal.Decimal, err error) {
	q := generated.New(r.pool)
	result, err := q.CheckLedgerConsistency(ctx)
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}

	totalBalance, err := toDecimal(result.TotalAccountBalance)
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}

	totalAmount, err := toDecimal(result.TotalEntryAmount)
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}

	return totalBalance, totalAmount, nil
}

// CheckConsistencyByCurrency checks ledger consistency grouped by currency.
func (r *LedgerRepository) CheckConsistencyByCurrency(ctx context.Context) ([]usecase.CurrencyConsistency, error) {
	q := generated.New(r.pool)
	rows, err := q.CheckLedgerConsistencyByCurrency(ctx)
	if err != nil {
		return nil, err
	}

	results := make([]usecase.CurrencyConsistency, 0, len(rows))
	for _, row := range rows {
		totalBalance, err := toDecimal(row.TotalAccountBalance)
		if err != nil {
			return nil, err
		}

		totalEntries, err := toDecimal(row.TotalEntryAmount)
		if err != nil {
			return nil, err
		}

		results = append(results, usecase.CurrencyConsistency{
			Currency:     row.Currency,
			TotalBalance: totalBalance,
			TotalEntries: totalEntries,
		})
	}

	return results, nil
}

func toDecimal(n pgtype.Numeric) (decimal.Decimal, error) {
	if !n.Valid {
		return decimal.Zero, nil
	}

	d, err := decimal.NewFromString(n.Int.String())
	if err != nil {
		return decimal.Zero, err
	}

	if n.Exp != 0 {
		d = d.Shift(n.Exp)
	}

	return d, nil
}
