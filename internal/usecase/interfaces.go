package usecase

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"github.com/iho/goledger/internal/domain"
)

// AccountRepository defines data access for accounts.
type AccountRepository interface {
	Create(ctx context.Context, account *domain.Account) error
	CreateTx(ctx context.Context, tx Transaction, account *domain.Account) error
	GetByID(ctx context.Context, id string) (*domain.Account, error)
	GetByIDForUpdate(ctx context.Context, tx Transaction, id string) (*domain.Account, error)
	GetByIDsForUpdate(ctx context.Context, tx Transaction, ids []string) ([]*domain.Account, error)
	UpdateBalance(ctx context.Context, tx Transaction, id string, balance decimal.Decimal, updatedAt time.Time) error
	UpdateEncumberedBalance(ctx context.Context, tx Transaction, id string, encumberedBalance decimal.Decimal, updatedAt time.Time) error
	List(ctx context.Context, limit, offset int) ([]*domain.Account, error)
}

// TransferRepository defines data access for transfers.
type TransferRepository interface {
	Create(ctx context.Context, tx Transaction, transfer *domain.Transfer) error
	GetByID(ctx context.Context, id string) (*domain.Transfer, error)
	ListByAccount(ctx context.Context, accountID string, limit, offset int) ([]*domain.Transfer, error)
}

// EntryRepository defines data access for entries.
type EntryRepository interface {
	Create(ctx context.Context, tx Transaction, entry *domain.Entry) error
	GetByTransfer(ctx context.Context, transferID string) ([]*domain.Entry, error)
	GetByAccount(ctx context.Context, accountID string, limit, offset int) ([]*domain.Entry, error)
	GetBalanceAtTime(ctx context.Context, accountID string, at time.Time) (decimal.Decimal, error)
}

// LedgerRepository defines data access for ledger-wide operations.
type LedgerRepository interface {
	CheckConsistency(ctx context.Context) (totalBalance, totalAmount decimal.Decimal, err error)
}

// HoldRepository defines data access for holds.
type HoldRepository interface {
	Create(ctx context.Context, tx Transaction, hold *domain.Hold) error
	GetByID(ctx context.Context, id string) (*domain.Hold, error)
	GetByIDForUpdate(ctx context.Context, tx Transaction, id string) (*domain.Hold, error)
	UpdateStatus(ctx context.Context, tx Transaction, id string, status domain.HoldStatus, updatedAt time.Time) error
	ListByAccount(ctx context.Context, accountID string, limit, offset int) ([]*domain.Hold, error)
}

// OutboxRepository defines data access for outbox events.
type OutboxRepository interface {
	Create(ctx context.Context, tx Transaction, event *domain.OutboxEvent) error
	GetUnpublished(ctx context.Context, limit int) ([]*domain.OutboxEvent, error)
	MarkPublished(ctx context.Context, id string, publishedAt time.Time) error
	GetByAggregate(ctx context.Context, aggregateType, aggregateID string, limit, offset int) ([]*domain.OutboxEvent, error)
	DeletePublished(ctx context.Context, before time.Time) error
}

// AuditRepository defines data access for audit logs.
type AuditRepository interface {
	Create(ctx context.Context, log *domain.AuditLog) error
	CreateTx(ctx context.Context, tx Transaction, log *domain.AuditLog) error
	List(ctx context.Context, filter domain.AuditFilter) ([]*domain.AuditLog, error)
	GetByResourceID(ctx context.Context, resourceType, resourceID string) ([]*domain.AuditLog, error)
}

// Transaction represents a database transaction.
type Transaction interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// TransactionManager handles transaction lifecycle.
type TransactionManager interface {
	Begin(ctx context.Context) (Transaction, error)
}

// IDGenerator generates unique IDs.
type IDGenerator interface {
	Generate() string
}

// Cache defines caching operations.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// IdempotencyStore handles idempotency key storage.
type IdempotencyStore interface {
	// CheckAndSet atomically checks if key exists, sets if not.
	// Returns (exists, existingValue, error).
	CheckAndSet(ctx context.Context, key string, response []byte, ttl time.Duration) (bool, []byte, error)
	// Update updates an existing key with the final response.
	Update(ctx context.Context, key string, response []byte, ttl time.Duration) error
}
