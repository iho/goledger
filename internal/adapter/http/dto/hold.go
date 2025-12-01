package dto

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/iho/goledger/internal/domain"
)

type CreateHoldRequest struct {
	AccountID string `json:"account_id"`
	Amount    string `json:"amount"`
}

type HoldResponse struct {
	ID        string          `json:"id"`
	AccountID string          `json:"account_id"`
	Amount    decimal.Decimal `json:"amount"` // Using decimal directly for response as JSON number/string
	Status    string          `json:"status"`
	ExpiresAt *time.Time      `json:"expires_at,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type CaptureHoldRequest struct {
	ToAccountID string `json:"to_account_id"`
}

func HoldFromDomain(h *domain.Hold) HoldResponse {
	return HoldResponse{
		ID:        h.ID,
		AccountID: h.AccountID,
		Amount:    h.Amount,
		Status:    string(h.Status),
		ExpiresAt: h.ExpiresAt,
		CreatedAt: h.CreatedAt,
		UpdatedAt: h.UpdatedAt,
	}
}
