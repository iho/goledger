package domain

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

var (
	ErrHoldNotFound      = errors.New("hold not found")
	ErrInsufficientFunds = errors.New("insufficient funds for hold")
	ErrHoldNotActive     = errors.New("hold is not active")
)

type HoldStatus string

const (
	HoldStatusActive   HoldStatus = "active"
	HoldStatusVoided   HoldStatus = "voided"
	HoldStatusCaptured HoldStatus = "captured"
)

type Hold struct {
	ID        string
	AccountID string
	Amount    decimal.Decimal
	Status    HoldStatus
	ExpiresAt *time.Time
	Metadata  map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Validate checks if hold is valid.
func (h *Hold) Validate() error {
	if h.Amount.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidAmount
	}
	return nil
}
