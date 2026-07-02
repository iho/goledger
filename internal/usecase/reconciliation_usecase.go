package usecase

import (
	"context"
	"fmt"
	"strings"
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

// ReconcileAccount verifies that an account's recorded balance matches the
// sum of all its entries. An account's balance always starts at zero and
// only ever changes via entries, so the two should always be equal; a
// mismatch indicates a bug (a balance update without a matching entry, or
// vice versa) or tampering.
func (uc *ReconciliationUseCase) ReconcileAccount(ctx context.Context, accountID string) (*ReconciliationResult, error) {
	account, err := uc.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}

	calculatedBalance, err := uc.entryRepo.SumAmountsByAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}

	difference := account.Balance.Sub(calculatedBalance)

	return &ReconciliationResult{
		AccountID:         accountID,
		RecordedBalance:   account.Balance,
		CalculatedBalance: calculatedBalance,
		Difference:        difference,
		IsReconciled:      difference.IsZero(),
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

// CheckLedgerConsistency verifies double-entry bookkeeping consistency,
// grouped by currency so an error in one currency can't be masked by an
// offsetting error in another.
func (uc *ReconciliationUseCase) CheckLedgerConsistency(ctx context.Context) error {
	results, err := uc.ledgerRepo.CheckConsistencyByCurrency(ctx)
	if err != nil {
		return err
	}

	var mismatches []string
	for _, r := range results {
		if !r.TotalBalance.Equal(r.TotalEntries) {
			mismatches = append(mismatches, fmt.Sprintf(
				"%s: balance=%s entries=%s difference=%s",
				r.Currency, r.TotalBalance.String(), r.TotalEntries.String(), r.TotalBalance.Sub(r.TotalEntries).String(),
			))
		}
	}

	if len(mismatches) > 0 {
		return fmt.Errorf("ledger inconsistency detected: %s", strings.Join(mismatches, "; "))
	}

	return nil
}

// EntryChainBreak describes a single point where an account's entry chain
// fails to link up.
type EntryChainBreak struct {
	EntryID  string
	Sequence int
	Reason   string
}

// EntryChainResult is the result of walking one account's entry chain.
type EntryChainResult struct {
	AccountID string
	Breaks    []EntryChainBreak
	Valid     bool
}

// VerifyEntryChain walks an account's entries in account_version order and
// checks that each entry's account_previous_balance equals the prior
// entry's account_current_balance, and that account_version is contiguous
// starting at 1. This detects gaps or tampering that a balance-sum check
// alone would miss (e.g. two entries swapped, or one deleted and another
// edited to compensate).
func (uc *ReconciliationUseCase) VerifyEntryChain(ctx context.Context, accountID string) (*EntryChainResult, error) {
	entries, err := uc.entryRepo.GetAllByAccountOrdered(ctx, accountID)
	if err != nil {
		return nil, err
	}

	result := &EntryChainResult{AccountID: accountID, Valid: true}

	var prevBalance decimal.Decimal
	var prevVersion int64
	for i, e := range entries {
		if i == 0 {
			if e.AccountVersion != 1 {
				result.Valid = false
				result.Breaks = append(result.Breaks, EntryChainBreak{
					EntryID:  e.ID,
					Sequence: i,
					Reason:   fmt.Sprintf("first entry has account_version %d, expected 1", e.AccountVersion),
				})
			}
		} else {
			if !e.AccountPreviousBalance.Equal(prevBalance) {
				result.Valid = false
				result.Breaks = append(result.Breaks, EntryChainBreak{
					EntryID:  e.ID,
					Sequence: i,
					Reason: fmt.Sprintf(
						"account_previous_balance %s does not match prior entry's account_current_balance %s",
						e.AccountPreviousBalance.String(), prevBalance.String(),
					),
				})
			}

			if e.AccountVersion != prevVersion+1 {
				result.Valid = false
				result.Breaks = append(result.Breaks, EntryChainBreak{
					EntryID:  e.ID,
					Sequence: i,
					Reason:   fmt.Sprintf("account_version %d is not contiguous with prior version %d", e.AccountVersion, prevVersion),
				})
			}
		}

		prevBalance = e.AccountCurrentBalance
		prevVersion = e.AccountVersion
	}

	return result, nil
}

// ReconciliationReport represents a full reconciliation report
type ReconciliationReport struct {
	CheckedAt          time.Time
	Discrepancies      []*ReconciliationResult
	ChainBreaks        []*EntryChainResult
	TotalAccounts      int
	ReconciledAccounts int
	LedgerConsistent   bool
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
		ChainBreaks:      make([]*EntryChainResult, 0),
		LedgerConsistent: ledgerErr == nil,
		CheckedAt:        time.Now().UTC(),
	}

	for _, result := range results {
		if result.IsReconciled {
			report.ReconciledAccounts++
		} else {
			report.Discrepancies = append(report.Discrepancies, result)
		}

		chain, err := uc.VerifyEntryChain(ctx, result.AccountID)
		if err != nil {
			return nil, fmt.Errorf("failed to verify entry chain for account %s: %w", result.AccountID, err)
		}
		if !chain.Valid {
			report.ChainBreaks = append(report.ChainBreaks, chain)
		}
	}

	return report, nil
}
