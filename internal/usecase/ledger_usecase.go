package usecase

import (
	"context"
	"errors"
)

var (
	// ErrInconsistentLedger is returned when the ledger is not balanced.
	ErrInconsistentLedger = errors.New("ledger is inconsistent: debits do not equal credits")
)

// LedgerUseCase handles ledger-wide operations.
type LedgerUseCase struct {
	ledgerRepo LedgerRepository
}

// NewLedgerUseCase creates a new LedgerUseCase.
func NewLedgerUseCase(ledgerRepo LedgerRepository) *LedgerUseCase {
	return &LedgerUseCase{
		ledgerRepo: ledgerRepo,
	}
}

// CheckConsistency verifies that the ledger is balanced.
func (uc *LedgerUseCase) CheckConsistency(ctx context.Context) (bool, error) {
	totalBalance, totalAmount, err := uc.ledgerRepo.CheckConsistency(ctx)
	if err != nil {
		return false, err
	}

	// 1. Total balance of all accounts should be 0 (sum of debits and credits)
	// Note: This assumes a closed system where money is not created/destroyed but moved.
	// If we have external deposits, they should come from a "World" account or similar.
	// For now, we assume strict double-entry where sum(balance) = 0.
	if !totalBalance.IsZero() {
		return false, ErrInconsistentLedger
	}

	// 2. Total amount of entries should be 0
	if !totalAmount.IsZero() {
		return false, ErrInconsistentLedger
	}

	return true, nil
}
