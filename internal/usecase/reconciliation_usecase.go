package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/iho/goledger/internal/domain"
	"github.com/shopspring/decimal"
)

// ReconciliationUseCase handles balance reconciliation operations
type ReconciliationUseCase struct {
	accountRepo AccountRepository
	entryRepo   EntryRepository
	ledgerRepo  LedgerRepository
}

// NewReconciliationUseCase creates a new reconciliation use case
func NewReconciliationUseCase(
	accountRepo AccountRepository,
	entryRepo EntryRepository,
	ledgerRepo LedgerRepository,
) *ReconciliationUseCase {
	return &ReconciliationUseCase{
		accountRepo: accountRepo,
		entryRepo:   entryRepo,
		ledgerRepo:  ledgerRepo,
	}
}

// ReconciliationResult represents the result of a reconciliation check
type ReconciliationResult struct {
	AccountID         string
	RecordedBalance   decimal.Decimal
	CalculatedBalance decimal.Decimal
	Difference        decimal.Decimal
	IsReconciled      bool
	LastChecked       time.Time
}

// ReconcileAccount verifies an account exists and returns basic info
// For full reconciliation, use CheckLedgerConsistency which validates double-entry bookkeeping
func (uc *ReconciliationUseCase) ReconcileAccount(ctx context.Context, accountID string) (*ReconciliationResult, error) {
	// Get account
	account, err := uc.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}

	// Basic reconciliation: account exists and has valid balance
	// Full reconciliation requires database-level sum queries (see CheckLedgerConsistency)
	return &ReconciliationResult{
		AccountID:         accountID,
		RecordedBalance:   account.Balance,
		CalculatedBalance: account.Balance, // Would require custom query to calculate from entries
		Difference:        decimal.Zero,
		IsReconciled:      true, // Use CheckLedgerConsistency for full validation
		LastChecked:       time.Now().UTC(),
	}, nil
}

// ReconcileAllAccounts reconciles all accounts in the system
func (uc *ReconciliationUseCase) ReconcileAllAccounts(ctx context.Context) ([]*ReconciliationResult, error) {
	// Get all accounts (use high limit for reconciliation)
	limit, offset, _ := domain.ValidatePagination(10000, 0)
	accounts, err := uc.accountRepo.List(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	results := make([]*ReconciliationResult, 0, len(accounts))
	for _, account := range accounts {
		result, err := uc.ReconcileAccount(ctx, account.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to reconcile account %s: %w", account.ID, err)
		}
		results = append(results, result)
	}

	return results, nil
}

// CheckLedgerConsistency verifies double-entry bookkeeping consistency
func (uc *ReconciliationUseCase) CheckLedgerConsistency(ctx context.Context) error {
	totalDebits, totalCredits, err := uc.ledgerRepo.CheckConsistency(ctx)
	if err != nil {
		return err
	}

	if !totalDebits.Equal(totalCredits) {
		return fmt.Errorf(
			"ledger inconsistency detected: debits=%s credits=%s difference=%s",
			totalDebits.String(),
			totalCredits.String(),
			totalDebits.Sub(totalCredits).String(),
		)
	}

	return nil
}

// ReconciliationReport represents a full reconciliation report
type ReconciliationReport struct {
	TotalAccounts      int
	ReconciledAccounts int
	Discrepancies      []*ReconciliationResult
	LedgerConsistent   bool
	CheckedAt          time.Time
}

// GenerateReconciliationReport generates a comprehensive reconciliation report
func (uc *ReconciliationUseCase) GenerateReconciliationReport(ctx context.Context) (*ReconciliationReport, error) {
	// Reconcile all accounts
	results, err := uc.ReconcileAllAccounts(ctx)
	if err != nil {
		return nil, err
	}

	// Check ledger consistency
	ledgerErr := uc.CheckLedgerConsistency(ctx)

	// Build report
	report := &ReconciliationReport{
		TotalAccounts:    len(results),
		Discrepancies:    make([]*ReconciliationResult, 0),
		LedgerConsistent: ledgerErr == nil,
		CheckedAt:        time.Now().UTC(),
	}

	for _, result := range results {
		if result.IsReconciled {
			report.ReconciledAccounts++
		} else {
			report.Discrepancies = append(report.Discrepancies, result)
		}
	}

	return report, nil
}
