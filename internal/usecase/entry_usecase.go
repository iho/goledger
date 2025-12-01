package usecase

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"github.com/iho/goledger/internal/domain"
)

// EntryUseCase handles entry business logic.
type EntryUseCase struct {
	entryRepo EntryRepository
}

// NewEntryUseCase creates a new EntryUseCase.
func NewEntryUseCase(entryRepo EntryRepository) *EntryUseCase {
	return &EntryUseCase{
		entryRepo: entryRepo,
	}
}

// GetEntriesByAccountInput represents input for listing entries.
type GetEntriesByAccountInput struct {
	AccountID string
	Limit     int
	Offset    int
}

// GetEntriesByAccount lists entries for an account.
func (uc *EntryUseCase) GetEntriesByAccount(ctx context.Context, input GetEntriesByAccountInput) ([]*domain.Entry, error) {
	if input.Limit <= 0 {
		input.Limit = 20
	}

	if input.Limit > 100 {
		input.Limit = 100
	}

	return uc.entryRepo.GetByAccount(ctx, input.AccountID, input.Limit, input.Offset)
}

// GetEntriesByTransfer lists entries for a transfer.
func (uc *EntryUseCase) GetEntriesByTransfer(ctx context.Context, transferID string) ([]*domain.Entry, error) {
	return uc.entryRepo.GetByTransfer(ctx, transferID)
}

// GetHistoricalBalance returns the balance at a specific point in time.
func (uc *EntryUseCase) GetHistoricalBalance(ctx context.Context, accountID string, at time.Time) (decimal.Decimal, error) {
	return uc.entryRepo.GetBalanceAtTime(ctx, accountID, at)
}
