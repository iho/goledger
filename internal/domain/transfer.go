package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// Transfer represents a money movement between two accounts.
type Transfer struct {
	CreatedAt          time.Time
	EventAt            time.Time
	Metadata           map[string]any
	ID                 string
	FromAccountID      string
	ToAccountID        string
	Amount             decimal.Decimal
	ReversedTransferID *string
}

// Validate validates transfer request.
func (t *Transfer) Validate() error {
	if t.FromAccountID == t.ToAccountID {
		return ErrSameAccount
	}

	if t.Amount.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidAmount
	}

	return nil
}
