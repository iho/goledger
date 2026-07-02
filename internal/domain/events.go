package domain

import "time"

// Event types
const (
	EventTypeTransferCreated  = "transfer.created"
	EventTypeTransferReversed = "transfer.reversed"
	EventTypeHoldCreated      = "hold.created"
	EventTypeHoldVoided       = "hold.voided"
	EventTypeHoldCaptured     = "hold.captured"
	EventTypeAccountCreated   = "account.created"
)

// Aggregate types
const (
	AggregateTypeTransfer = "transfer"
	AggregateTypeHold     = "hold"
	AggregateTypeAccount  = "account"
)

// OutboxEvent represents an event to be published
type OutboxEvent struct {
	ID            string
	AggregateID   string
	AggregateType string
	EventType     string
	Payload       map[string]any
	CreatedAt     time.Time
	PublishedAt   *time.Time
	Published     bool
	// EventVersion tags the schema of Payload, so downstream consumers can
	// evolve the payload shape without breaking older readers. Defaults to
	// 1; bump when a payload's fields change in an incompatible way.
	EventVersion int32
	// AggregateSequence is a per-aggregate (aggregate_type+aggregate_id)
	// monotonically increasing counter assigned by the database at insert
	// time, letting consumers detect gaps or out-of-order delivery.
	AggregateSequence int64

	// Attempts, LastError, and DeadLetteredAt track delivery retries. Once
	// DeadLetteredAt is set, the publisher stops polling this event so one
	// poison message can't block the rest of the queue.
	Attempts       int
	LastError      string
	DeadLetteredAt *time.Time
}

// TransferCreatedEvent payload
type TransferCreatedEvent struct {
	TransferID    string `json:"transfer_id"`
	FromAccountID string `json:"from_account_id"`
	ToAccountID   string `json:"to_account_id"`
	Amount        string `json:"amount"`
	Currency      string `json:"currency"`
	EventAt       string `json:"event_at"`
}

// TransferReversedEvent payload
type TransferReversedEvent struct {
	ReversalTransferID string `json:"reversal_transfer_id"`
	OriginalTransferID string `json:"original_transfer_id"`
	Amount             string `json:"amount"`
	Currency           string `json:"currency"`
}

// HoldCreatedEvent payload
type HoldCreatedEvent struct {
	HoldID    string `json:"hold_id"`
	AccountID string `json:"account_id"`
	Amount    string `json:"amount"`
	Currency  string `json:"currency"`
}

// HoldVoidedEvent payload
type HoldVoidedEvent struct {
	HoldID    string `json:"hold_id"`
	AccountID string `json:"account_id"`
	Amount    string `json:"amount"`
}

// HoldCapturedEvent payload
type HoldCapturedEvent struct {
	HoldID      string `json:"hold_id"`
	TransferID  string `json:"transfer_id"`
	ToAccountID string `json:"to_account_id"`
	Amount      string `json:"amount"`
}

// AccountCreatedEvent payload
type AccountCreatedEvent struct {
	AccountID string `json:"account_id"`
	Name      string `json:"name"`
	Currency  string `json:"currency"`
}
