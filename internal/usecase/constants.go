package usecase

import "time"

const (
	// DefaultTransactionTimeout is the maximum duration for a database transaction
	// This prevents long-running transactions from blocking tables
	DefaultTransactionTimeout = 10 * time.Second

	// MaxTransferAmount is the maximum amount allowed for a single transfer (in decimal string)
	MaxTransferAmount = "1000000000" // 1 billion

	// IdempotencyKeyTTL is how long idempotency keys are cached
	IdempotencyKeyTTL = 24 * time.Hour
)
