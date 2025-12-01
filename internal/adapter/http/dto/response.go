package dto

import (
	"time"

	"github.com/iho/goledger/internal/domain"
	"github.com/shopspring/decimal"
)

// AccountResponse represents an account in API responses.
type AccountResponse struct {
	ID                   string          `json:"id"`
	Name                 string          `json:"name"`
	Currency             string          `json:"currency"`
	Balance              decimal.Decimal `json:"balance"`
	Version              int64           `json:"version"`
	AllowNegativeBalance bool            `json:"allow_negative_balance"`
	AllowPositiveBalance bool            `json:"allow_positive_balance"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

// AccountFromDomain converts domain account to response.
func AccountFromDomain(a *domain.Account) *AccountResponse {
	return &AccountResponse{
		ID:                   a.ID,
		Name:                 a.Name,
		Currency:             a.Currency,
		Balance:              a.Balance,
		Version:              a.Version,
		AllowNegativeBalance: a.AllowNegativeBalance,
		AllowPositiveBalance: a.AllowPositiveBalance,
		CreatedAt:            a.CreatedAt,
		UpdatedAt:            a.UpdatedAt,
	}
}

// AccountsFromDomain converts domain accounts to responses.
func AccountsFromDomain(accounts []*domain.Account) []*AccountResponse {
	result := make([]*AccountResponse, len(accounts))
	for i, a := range accounts {
		result[i] = AccountFromDomain(a)
	}
	return result
}

// TransferResponse represents a transfer in API responses.
type TransferResponse struct {
	ID            string          `json:"id"`
	FromAccountID string          `json:"from_account_id"`
	ToAccountID   string          `json:"to_account_id"`
	Amount        decimal.Decimal `json:"amount"`
	CreatedAt     time.Time       `json:"created_at"`
	EventAt       time.Time       `json:"event_at"`
	Metadata      map[string]any  `json:"metadata,omitempty"`
}

// TransferFromDomain converts domain transfer to response.
func TransferFromDomain(t *domain.Transfer) *TransferResponse {
	return &TransferResponse{
		ID:            t.ID,
		FromAccountID: t.FromAccountID,
		ToAccountID:   t.ToAccountID,
		Amount:        t.Amount,
		CreatedAt:     t.CreatedAt,
		EventAt:       t.EventAt,
		Metadata:      t.Metadata,
	}
}

// TransfersFromDomain converts domain transfers to responses.
func TransfersFromDomain(transfers []*domain.Transfer) []*TransferResponse {
	result := make([]*TransferResponse, len(transfers))
	for i, t := range transfers {
		result[i] = TransferFromDomain(t)
	}
	return result
}

// EntryResponse represents an entry in API responses.
type EntryResponse struct {
	ID                     string          `json:"id"`
	AccountID              string          `json:"account_id"`
	TransferID             string          `json:"transfer_id"`
	Amount                 decimal.Decimal `json:"amount"`
	AccountPreviousBalance decimal.Decimal `json:"account_previous_balance"`
	AccountCurrentBalance  decimal.Decimal `json:"account_current_balance"`
	AccountVersion         int64           `json:"account_version"`
	CreatedAt              time.Time       `json:"created_at"`
}

// EntryFromDomain converts domain entry to response.
func EntryFromDomain(e *domain.Entry) *EntryResponse {
	return &EntryResponse{
		ID:                     e.ID,
		AccountID:              e.AccountID,
		TransferID:             e.TransferID,
		Amount:                 e.Amount,
		AccountPreviousBalance: e.AccountPreviousBalance,
		AccountCurrentBalance:  e.AccountCurrentBalance,
		AccountVersion:         e.AccountVersion,
		CreatedAt:              e.CreatedAt,
	}
}

// EntriesFromDomain converts domain entries to responses.
func EntriesFromDomain(entries []*domain.Entry) []*EntryResponse {
	result := make([]*EntryResponse, len(entries))
	for i, e := range entries {
		result[i] = EntryFromDomain(e)
	}
	return result
}

// ErrorResponse represents an error in API responses.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
