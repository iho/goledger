package dto

import (
	"time"

	"github.com/iho/goledger/internal/domain"
)

// AccountResponse represents an account in API responses.
type AccountResponse struct {
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	Currency             string    `json:"currency"`
	Balance              string    `json:"balance"`
	Version              int64     `json:"version"`
	AllowNegativeBalance bool      `json:"allow_negative_balance"`
	AllowPositiveBalance bool      `json:"allow_positive_balance"`
}

// AccountFromDomain converts domain account to response.
func AccountFromDomain(a *domain.Account) *AccountResponse {
	return &AccountResponse{
		ID:                   a.ID,
		Name:                 a.Name,
		Currency:             a.Currency,
		Balance:              a.Balance.String(),
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
	CreatedAt          time.Time      `json:"created_at"`
	EventAt            time.Time      `json:"event_at"`
	Metadata           map[string]any `json:"metadata,omitempty"`
	ID                 string         `json:"id"`
	FromAccountID      string         `json:"from_account_id"`
	ToAccountID        string         `json:"to_account_id"`
	Amount             string         `json:"amount"`
	ReversedTransferID *string        `json:"reversed_transfer_id,omitempty"`
}

// TransferFromDomain converts domain transfer to response.
func TransferFromDomain(t *domain.Transfer) *TransferResponse {
	return &TransferResponse{
		ID:                 t.ID,
		FromAccountID:      t.FromAccountID,
		ToAccountID:        t.ToAccountID,
		Amount:             t.Amount.String(),
		CreatedAt:          t.CreatedAt,
		EventAt:            t.EventAt,
		Metadata:           t.Metadata,
		ReversedTransferID: t.ReversedTransferID,
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
	CreatedAt              time.Time `json:"created_at"`
	ID                     string    `json:"id"`
	AccountID              string    `json:"account_id"`
	TransferID             string    `json:"transfer_id"`
	Amount                 string    `json:"amount"`
	AccountPreviousBalance string    `json:"account_previous_balance"`
	AccountCurrentBalance  string    `json:"account_current_balance"`
	AccountVersion         int64     `json:"account_version"`
}

// EntryFromDomain converts domain entry to response.
func EntryFromDomain(e *domain.Entry) *EntryResponse {
	return &EntryResponse{
		ID:                     e.ID,
		AccountID:              e.AccountID,
		TransferID:             e.TransferID,
		Amount:                 e.Amount.String(),
		AccountPreviousBalance: e.AccountPreviousBalance.String(),
		AccountCurrentBalance:  e.AccountCurrentBalance.String(),
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

// ListAccountsResponse represents a list of accounts.
type ListAccountsResponse struct {
	Accounts []*AccountResponse `json:"accounts"`
	Total    int64              `json:"total"`
}

// ListTransfersResponse represents a list of transfers.
type ListTransfersResponse struct {
	Transfers []*TransferResponse `json:"transfers"`
}

// ListTransfersCursorResponse represents a keyset-paginated page of
// transfers. NextCursor is empty when there are no more results.
type ListTransfersCursorResponse struct {
	NextCursor string              `json:"next_cursor,omitempty"`
	Transfers  []*TransferResponse `json:"transfers"`
}

// ListEntriesResponse represents a list of entries.
type ListEntriesResponse struct {
	Entries []*EntryResponse `json:"entries"`
}

// AuditLogResponse represents an audit log entry in API responses.
type AuditLogResponse struct {
	CreatedAt    time.Time      `json:"created_at"`
	BeforeState  map[string]any `json:"before_state,omitempty"`
	AfterState   map[string]any `json:"after_state,omitempty"`
	ID           string         `json:"id"`
	UserID       string         `json:"user_id"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resource_type"`
	ResourceID   string         `json:"resource_id"`
	IPAddress    string         `json:"ip_address,omitempty"`
	UserAgent    string         `json:"user_agent,omitempty"`
	RequestID    string         `json:"request_id,omitempty"`
	Status       string         `json:"status"`
	ErrorMessage string         `json:"error_message,omitempty"`
	// PrevHash/Hash/ChainSeq support independent, offline tamper-evidence
	// verification of the audit trail - see verify_audit_log_chain() in
	// migration 000012.
	PrevHash string `json:"prev_hash,omitempty"`
	Hash     string `json:"hash,omitempty"`
	ChainSeq int64  `json:"chain_seq,omitempty"`
}

// AuditLogFromDomain converts a domain audit log to a response.
func AuditLogFromDomain(a *domain.AuditLog) *AuditLogResponse {
	return &AuditLogResponse{
		ID:           a.ID,
		UserID:       a.UserID,
		Action:       a.Action,
		ResourceType: a.ResourceType,
		ResourceID:   a.ResourceID,
		IPAddress:    a.IPAddress,
		UserAgent:    a.UserAgent,
		RequestID:    a.RequestID,
		BeforeState:  a.BeforeState,
		AfterState:   a.AfterState,
		Status:       a.Status,
		ErrorMessage: a.ErrorMessage,
		CreatedAt:    a.CreatedAt,
		PrevHash:     a.PrevHash,
		Hash:         a.Hash,
		ChainSeq:     a.ChainSeq,
	}
}

// AuditLogsFromDomain converts domain audit logs to responses.
func AuditLogsFromDomain(logs []*domain.AuditLog) []*AuditLogResponse {
	result := make([]*AuditLogResponse, len(logs))
	for i, l := range logs {
		result[i] = AuditLogFromDomain(l)
	}

	return result
}

// ErrorResponse represents an error in API responses.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
