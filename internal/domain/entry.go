package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// Entry represents a single ledger entry (debit or credit).
type Entry struct {
	ID                     string
	AccountID              string
	TransferID             string
	Amount                 decimal.Decimal // Negative for debit, positive for credit
	AccountPreviousBalance decimal.Decimal
	AccountCurrentBalance  decimal.Decimal
	AccountVersion         int64
	CreatedAt              time.Time
}
