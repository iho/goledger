package dto

import (
	"time"

	"github.com/iho/goledger/internal/usecase"
	"github.com/shopspring/decimal"
)

// CreateAccountRequest represents a request to create an account.
type CreateAccountRequest struct {
	Name                 string `json:"name"`
	Currency             string `json:"currency"`
	AllowNegativeBalance bool   `json:"allow_negative_balance"`
	AllowPositiveBalance bool   `json:"allow_positive_balance"`
}

// ToUseCaseInput converts to use case input.
func (r *CreateAccountRequest) ToUseCaseInput() usecase.CreateAccountInput {
	return usecase.CreateAccountInput{
		Name:                 r.Name,
		Currency:             r.Currency,
		AllowNegativeBalance: r.AllowNegativeBalance,
		AllowPositiveBalance: r.AllowPositiveBalance,
	}
}

// CreateTransferRequest represents a request to create a transfer.
type CreateTransferRequest struct {
	FromAccountID string          `json:"from_account_id"`
	ToAccountID   string          `json:"to_account_id"`
	Amount        decimal.Decimal `json:"amount"`
	EventAt       *time.Time      `json:"event_at,omitempty"`
	Metadata      map[string]any  `json:"metadata,omitempty"`
}

// ToUseCaseInput converts to use case input.
func (r *CreateTransferRequest) ToUseCaseInput() usecase.CreateTransferInput {
	return usecase.CreateTransferInput{
		FromAccountID: r.FromAccountID,
		ToAccountID:   r.ToAccountID,
		Amount:        r.Amount,
		EventAt:       r.EventAt,
		Metadata:      r.Metadata,
	}
}

// CreateBatchTransferRequest represents a request to create multiple transfers.
type CreateBatchTransferRequest struct {
	Transfers []TransferItem `json:"transfers"`
	EventAt   *time.Time     `json:"event_at,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// TransferItem represents a single transfer in a batch.
type TransferItem struct {
	FromAccountID string          `json:"from_account_id"`
	ToAccountID   string          `json:"to_account_id"`
	Amount        decimal.Decimal `json:"amount"`
}

// ToUseCaseInput converts to use case input.
func (r *CreateBatchTransferRequest) ToUseCaseInput() usecase.CreateBatchTransferInput {
	transfers := make([]usecase.CreateTransferInput, len(r.Transfers))
	for i, t := range r.Transfers {
		transfers[i] = usecase.CreateTransferInput{
			FromAccountID: t.FromAccountID,
			ToAccountID:   t.ToAccountID,
			Amount:        t.Amount,
		}
	}
	return usecase.CreateBatchTransferInput{
		Transfers: transfers,
		EventAt:   r.EventAt,
		Metadata:  r.Metadata,
	}
}

// PaginationRequest represents pagination parameters.
type PaginationRequest struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}
