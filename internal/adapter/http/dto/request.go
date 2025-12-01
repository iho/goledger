package dto

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/iho/goledger/internal/usecase"
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
	EventAt       *time.Time     `json:"event_at,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	FromAccountID string         `json:"from_account_id"`
	ToAccountID   string         `json:"to_account_id"`
	Amount        string         `json:"amount"`
}

// ToUseCaseInput converts to use case input.
func (r *CreateTransferRequest) ToUseCaseInput() (usecase.CreateTransferInput, error) {
	amount, err := decimal.NewFromString(r.Amount)
	if err != nil {
		return usecase.CreateTransferInput{}, err
	}

	return usecase.CreateTransferInput{
		FromAccountID: r.FromAccountID,
		ToAccountID:   r.ToAccountID,
		Amount:        amount,
		EventAt:       r.EventAt,
		Metadata:      r.Metadata,
	}, nil
}

// CreateBatchTransferRequest represents a request to create multiple transfers.
type CreateBatchTransferRequest struct {
	EventAt   *time.Time     `json:"event_at,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Transfers []TransferItem `json:"transfers"`
}

// TransferItem represents a single transfer in a batch.
type TransferItem struct {
	FromAccountID string `json:"from_account_id"`
	ToAccountID   string `json:"to_account_id"`
	Amount        string `json:"amount"`
}

// ToUseCaseInput converts to use case input.
func (r *CreateBatchTransferRequest) ToUseCaseInput() (usecase.CreateBatchTransferInput, error) {
	transfers := make([]usecase.CreateTransferInput, len(r.Transfers))
	for i, t := range r.Transfers {
		amount, err := decimal.NewFromString(t.Amount)
		if err != nil {
			return usecase.CreateBatchTransferInput{}, err
		}

		transfers[i] = usecase.CreateTransferInput{
			FromAccountID: t.FromAccountID,
			ToAccountID:   t.ToAccountID,
			Amount:        amount,
		}
	}

	return usecase.CreateBatchTransferInput{
		Transfers: transfers,
		EventAt:   r.EventAt,
		Metadata:  r.Metadata,
	}, nil
}

// ReverseTransferRequest represents a request to reverse a transfer.
type ReverseTransferRequest struct {
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ToUseCaseInput converts to use case input.
func (r *ReverseTransferRequest) ToUseCaseInput(transferID string) usecase.ReverseTransferInput {
	return usecase.ReverseTransferInput{
		TransferID: transferID,
		Metadata:   r.Metadata,
	}
}

// PaginationRequest represents pagination parameters.
type PaginationRequest struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}
