package usecase_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/usecase"
)

type stubAccountRepository struct {
	getByIDFn func(ctx context.Context, id string) (*domain.Account, error)
	listFn    func(ctx context.Context, limit, offset int) ([]*domain.Account, error)
}

func (s *stubAccountRepository) Create(context.Context, *domain.Account) error { return nil }
func (s *stubAccountRepository) CreateTx(context.Context, usecase.Transaction, *domain.Account) error {
	return nil
}
func (s *stubAccountRepository) GetByID(ctx context.Context, id string) (*domain.Account, error) {
	return s.getByIDFn(ctx, id)
}
func (s *stubAccountRepository) GetByIDForUpdate(context.Context, usecase.Transaction, string) (*domain.Account, error) {
	return nil, errors.New("not implemented")
}
func (s *stubAccountRepository) GetByIDsForUpdate(context.Context, usecase.Transaction, []string) ([]*domain.Account, error) {
	return nil, errors.New("not implemented")
}
func (s *stubAccountRepository) UpdateBalance(context.Context, usecase.Transaction, string, decimal.Decimal, time.Time) error {
	return errors.New("not implemented")
}
func (s *stubAccountRepository) UpdateEncumberedBalance(context.Context, usecase.Transaction, string, decimal.Decimal, time.Time) error {
	return errors.New("not implemented")
}
func (s *stubAccountRepository) UpdateBalanceAndEncumbered(context.Context, usecase.Transaction, string, decimal.Decimal, decimal.Decimal, time.Time) error {
	return errors.New("not implemented")
}
func (s *stubAccountRepository) List(ctx context.Context, limit, offset int) ([]*domain.Account, error) {
	return s.listFn(ctx, limit, offset)
}

type stubEntryRepository struct {
	sumFn     func(ctx context.Context, accountID string) (decimal.Decimal, error)
	orderedFn func(ctx context.Context, accountID string) ([]*domain.Entry, error)
}

func (s *stubEntryRepository) Create(context.Context, usecase.Transaction, *domain.Entry) error {
	return nil
}
func (s *stubEntryRepository) GetByTransfer(context.Context, string) ([]*domain.Entry, error) {
	return nil, nil
}
func (s *stubEntryRepository) GetByAccount(context.Context, string, int, int) ([]*domain.Entry, error) {
	return nil, nil
}
func (s *stubEntryRepository) GetBalanceAtTime(context.Context, string, time.Time) (decimal.Decimal, error) {
	return decimal.Zero, nil
}
func (s *stubEntryRepository) SumAmountsByAccount(ctx context.Context, accountID string) (decimal.Decimal, error) {
	if s.sumFn != nil {
		return s.sumFn(ctx, accountID)
	}
	return decimal.Zero, nil
}
func (s *stubEntryRepository) GetAllByAccountOrdered(ctx context.Context, accountID string) ([]*domain.Entry, error) {
	if s.orderedFn != nil {
		return s.orderedFn(ctx, accountID)
	}
	return nil, nil
}

type stubLedgerRepository struct {
	checkFn      func(ctx context.Context) (decimal.Decimal, decimal.Decimal, error)
	byCurrencyFn func(ctx context.Context) ([]usecase.CurrencyConsistency, error)
}

func (s *stubLedgerRepository) CheckConsistency(ctx context.Context) (totalBalance, totalAmount decimal.Decimal, err error) {
	return s.checkFn(ctx)
}

func (s *stubLedgerRepository) CheckConsistencyByCurrency(ctx context.Context) ([]usecase.CurrencyConsistency, error) {
	if s.byCurrencyFn != nil {
		return s.byCurrencyFn(ctx)
	}
	totalBalance, totalAmount, err := s.checkFn(ctx)
	if err != nil {
		return nil, err
	}
	return []usecase.CurrencyConsistency{{Currency: "USD", TotalBalance: totalBalance, TotalEntries: totalAmount}}, nil
}

func TestReconcileAccount(t *testing.T) {
	t.Parallel()

	account := &domain.Account{
		ID:      "acc-1",
		Balance: decimal.NewFromInt(150),
	}

	accountRepo := &stubAccountRepository{
		getByIDFn: func(context.Context, string) (*domain.Account, error) {
			return account, nil
		},
		listFn: func(context.Context, int, int) ([]*domain.Account, error) {
			return []*domain.Account{account}, nil
		},
	}

	uc := usecase.NewReconciliationUseCase(accountRepo, &stubEntryRepository{
		sumFn: func(context.Context, string) (decimal.Decimal, error) {
			return account.Balance, nil
		},
	}, &stubLedgerRepository{
		checkFn: func(context.Context) (decimal.Decimal, decimal.Decimal, error) {
			return decimal.Zero, decimal.Zero, nil
		},
	})

	result, err := uc.ReconcileAccount(context.Background(), "acc-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.RecordedBalance.Equal(account.Balance) {
		t.Fatalf("expected balance %s, got %s", account.Balance, result.RecordedBalance)
	}

	if !result.IsReconciled {
		t.Fatal("expected account to be marked as reconciled")
	}

	if result.LastChecked.IsZero() {
		t.Fatal("expected LastChecked timestamp to be set")
	}
}

func TestReconcileAccount_PropagatesError(t *testing.T) {
	t.Parallel()

	accountRepo := &stubAccountRepository{
		getByIDFn: func(context.Context, string) (*domain.Account, error) {
			return nil, fmt.Errorf("boom")
		},
		listFn: func(context.Context, int, int) ([]*domain.Account, error) {
			return nil, nil
		},
	}

	uc := usecase.NewReconciliationUseCase(accountRepo, &stubEntryRepository{}, &stubLedgerRepository{
		checkFn: func(context.Context) (decimal.Decimal, decimal.Decimal, error) {
			return decimal.Zero, decimal.Zero, nil
		},
	})

	_, err := uc.ReconcileAccount(context.Background(), "missing")
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected propagated error, got %v", err)
	}
}

func TestReconcileAllAccounts(t *testing.T) {
	t.Parallel()

	accounts := []*domain.Account{
		{ID: "acc-1", Balance: decimal.NewFromInt(100)},
		{ID: "acc-2", Balance: decimal.NewFromInt(200)},
	}

	accountRepo := &stubAccountRepository{
		listFn: func(context.Context, int, int) ([]*domain.Account, error) {
			return accounts, nil
		},
		getByIDFn: func(_ context.Context, id string) (*domain.Account, error) {
			for _, a := range accounts {
				if a.ID == id {
					return a, nil
				}
			}
			return nil, fmt.Errorf("not found")
		},
	}

	uc := usecase.NewReconciliationUseCase(accountRepo, &stubEntryRepository{}, &stubLedgerRepository{
		checkFn: func(context.Context) (decimal.Decimal, decimal.Decimal, error) {
			return decimal.Zero, decimal.Zero, nil
		},
	})

	results, err := uc.ReconcileAllAccounts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != len(accounts) {
		t.Fatalf("expected %d results, got %d", len(accounts), len(results))
	}
}

func TestCheckLedgerConsistency(t *testing.T) {
	t.Parallel()

	debits := decimal.NewFromInt(500)
	credits := decimal.NewFromInt(500)

	okLedger := &stubLedgerRepository{
		checkFn: func(context.Context) (decimal.Decimal, decimal.Decimal, error) {
			return debits, credits, nil
		},
	}

	accountRepo := &stubAccountRepository{
		getByIDFn: func(context.Context, string) (*domain.Account, error) { return nil, errors.New("unused") },
		listFn:    func(context.Context, int, int) ([]*domain.Account, error) { return nil, nil },
	}

	uc := usecase.NewReconciliationUseCase(accountRepo, &stubEntryRepository{}, okLedger)
	if err := uc.CheckLedgerConsistency(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	badLedger := &stubLedgerRepository{
		checkFn: func(context.Context) (decimal.Decimal, decimal.Decimal, error) {
			return decimal.NewFromInt(100), decimal.NewFromInt(50), nil
		},
	}

	uc = usecase.NewReconciliationUseCase(accountRepo, &stubEntryRepository{}, badLedger)
	if err := uc.CheckLedgerConsistency(context.Background()); err == nil {
		t.Fatal("expected error for inconsistent ledger")
	}
}

func TestGenerateReconciliationReport(t *testing.T) {
	t.Parallel()

	accounts := []*domain.Account{
		{ID: "r1", Balance: decimal.NewFromInt(10)},
		{ID: "r2", Balance: decimal.NewFromInt(20)},
	}

	accountRepo := &stubAccountRepository{
		listFn: func(context.Context, int, int) ([]*domain.Account, error) {
			return accounts, nil
		},
		getByIDFn: func(_ context.Context, id string) (*domain.Account, error) {
			for _, a := range accounts {
				if a.ID == id {
					return a, nil
				}
			}
			return nil, fmt.Errorf("missing %s", id)
		},
	}

	ledgerRepo := &stubLedgerRepository{
		checkFn: func(context.Context) (decimal.Decimal, decimal.Decimal, error) {
			return decimal.NewFromInt(30), decimal.NewFromInt(30), nil
		},
	}

	entryRepo := &stubEntryRepository{
		sumFn: func(_ context.Context, accountID string) (decimal.Decimal, error) {
			for _, a := range accounts {
				if a.ID == accountID {
					return a.Balance, nil
				}
			}
			return decimal.Zero, nil
		},
	}

	uc := usecase.NewReconciliationUseCase(accountRepo, entryRepo, ledgerRepo)

	report, err := uc.GenerateReconciliationReport(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.TotalAccounts != len(accounts) {
		t.Fatalf("expected total accounts %d, got %d", len(accounts), report.TotalAccounts)
	}

	if report.ReconciledAccounts != len(accounts) {
		t.Fatalf("expected reconciled accounts %d, got %d", len(accounts), report.ReconciledAccounts)
	}

	if !report.LedgerConsistent {
		t.Fatal("expected ledger to be marked consistent")
	}

	if report.CheckedAt.IsZero() {
		t.Fatal("expected CheckedAt timestamp")
	}
}

func TestReconcileAccount_DetectsMismatch(t *testing.T) {
	t.Parallel()

	account := &domain.Account{ID: "acc-1", Balance: decimal.NewFromInt(150)}

	accountRepo := &stubAccountRepository{
		getByIDFn: func(context.Context, string) (*domain.Account, error) {
			return account, nil
		},
	}

	uc := usecase.NewReconciliationUseCase(accountRepo, &stubEntryRepository{
		sumFn: func(context.Context, string) (decimal.Decimal, error) {
			return decimal.NewFromInt(100), nil // drifted from recorded balance
		},
	}, &stubLedgerRepository{})

	result, err := uc.ReconcileAccount(context.Background(), "acc-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsReconciled {
		t.Fatal("expected mismatch to be detected")
	}

	if !result.Difference.Equal(decimal.NewFromInt(50)) {
		t.Fatalf("expected difference 50, got %s", result.Difference)
	}
}

func TestVerifyEntryChain_Valid(t *testing.T) {
	t.Parallel()

	entries := []*domain.Entry{
		{ID: "e1", AccountVersion: 1, AccountPreviousBalance: decimal.Zero, AccountCurrentBalance: decimal.NewFromInt(100)},
		{ID: "e2", AccountVersion: 2, AccountPreviousBalance: decimal.NewFromInt(100), AccountCurrentBalance: decimal.NewFromInt(150)},
	}

	uc := usecase.NewReconciliationUseCase(&stubAccountRepository{}, &stubEntryRepository{
		orderedFn: func(context.Context, string) ([]*domain.Entry, error) {
			return entries, nil
		},
	}, &stubLedgerRepository{})

	result, err := uc.VerifyEntryChain(context.Background(), "acc-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Valid {
		t.Fatalf("expected valid chain, got breaks: %+v", result.Breaks)
	}
}

func TestVerifyEntryChain_DetectsGapAndBalanceMismatch(t *testing.T) {
	t.Parallel()

	entries := []*domain.Entry{
		{ID: "e1", AccountVersion: 1, AccountPreviousBalance: decimal.Zero, AccountCurrentBalance: decimal.NewFromInt(100)},
		// account_version jumps from 1 to 3 (gap), and previous_balance doesn't match e1's current_balance.
		{ID: "e2", AccountVersion: 3, AccountPreviousBalance: decimal.NewFromInt(999), AccountCurrentBalance: decimal.NewFromInt(150)},
	}

	uc := usecase.NewReconciliationUseCase(&stubAccountRepository{}, &stubEntryRepository{
		orderedFn: func(context.Context, string) ([]*domain.Entry, error) {
			return entries, nil
		},
	}, &stubLedgerRepository{})

	result, err := uc.VerifyEntryChain(context.Background(), "acc-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Valid {
		t.Fatal("expected chain to be marked invalid")
	}

	if len(result.Breaks) != 2 {
		t.Fatalf("expected 2 breaks (balance mismatch + version gap), got %d: %+v", len(result.Breaks), result.Breaks)
	}
}
