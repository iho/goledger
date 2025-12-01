
package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// Entry represents a single ledger entry (debit or credit).
type Entry struct {
	CreatedAt              time.Time
	ID                     string
	AccountID              string
	TransferID             string
	Amount                 decimal.Decimal
	AccountPreviousBalance decimal.Decimal
	AccountCurrentBalance  decimal.Decimal
	AccountVersion         int64
}
