package usecase

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"github.com/iho/goledger/internal/domain"
)

type HoldUseCase struct {
	txManager    TransactionManager
	accountRepo  AccountRepository
	holdRepo     HoldRepository
	transferRepo TransferRepository
	entryRepo    EntryRepository
	idGen        IDGenerator
}

func NewHoldUseCase(
	txManager TransactionManager,
	accountRepo AccountRepository,
	holdRepo HoldRepository,
	transferRepo TransferRepository,
	entryRepo EntryRepository,
	idGen IDGenerator,
) *HoldUseCase {
	return &HoldUseCase{
		txManager:    txManager,
		accountRepo:  accountRepo,
		holdRepo:     holdRepo,
		transferRepo: transferRepo,
		entryRepo:    entryRepo,
		idGen:        idGen,
	}
}

func (uc *HoldUseCase) HoldFunds(ctx context.Context, accountID string, amount decimal.Decimal) (*domain.Hold, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, domain.ErrInvalidAmount
	}

	tx, err := uc.txManager.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Lock account
	account, err := uc.accountRepo.GetByIDForUpdate(ctx, tx, accountID)
	if err != nil {
		return nil, err
	}

	// Check available balance
	if err := account.ValidateDebit(amount); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	hold := &domain.Hold{
		ID:        uc.idGen.Generate(),
		AccountID: accountID,
		Amount:    amount,
		Status:    domain.HoldStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := uc.holdRepo.Create(ctx, tx, hold); err != nil {
		return nil, err
	}

	newEncumbered := account.EncumberedBalance.Add(amount)
	if err := uc.accountRepo.UpdateEncumberedBalance(ctx, tx, accountID, newEncumbered, now); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return hold, nil
}

func (uc *HoldUseCase) VoidHold(ctx context.Context, holdID string) error {
	tx, err := uc.txManager.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	hold, err := uc.holdRepo.GetByIDForUpdate(ctx, tx, holdID)
	if err != nil {
		return err
	}

	if hold.Status != domain.HoldStatusActive {
		return domain.ErrHoldNotActive
	}

	account, err := uc.accountRepo.GetByIDForUpdate(ctx, tx, hold.AccountID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	if err := uc.holdRepo.UpdateStatus(ctx, tx, holdID, domain.HoldStatusVoided, now); err != nil {
		return err
	}

	newEncumbered := account.EncumberedBalance.Sub(hold.Amount)
	// Safety check: encumbered balance shouldn't go negative unless data corruption
	if newEncumbered.IsNegative() {
		newEncumbered = decimal.Zero
	}

	if err := uc.accountRepo.UpdateEncumberedBalance(ctx, tx, hold.AccountID, newEncumbered, now); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (uc *HoldUseCase) CaptureHold(ctx context.Context, holdID string, toAccountID string) (*domain.Transfer, error) {
	tx, err := uc.txManager.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	hold, err := uc.holdRepo.GetByIDForUpdate(ctx, tx, holdID)
	if err != nil {
		return nil, err
	}

	if hold.Status != domain.HoldStatusActive {
		return nil, domain.ErrHoldNotActive
	}

	// Lock accounts (From is hold.AccountID)
	// To prevent deadlocks, we should sort IDs, but here one ID is fixed (hold.AccountID).
	// If we always lock From then To, we might deadlock if another TX locks To then From.
	// Ideally we should use `GetByIDsForUpdate` with sorted IDs.
	ids := []string{hold.AccountID, toAccountID}
	// ... sorting logic needed if we want to be safe.
	// But `GetByIDsForUpdate` already sorts them in the repo implementation (if we use the one from `AccountRepository`).
	// Yes, `GetByIDsForUpdate` uses `ORDER BY id`.

	accounts, err := uc.accountRepo.GetByIDsForUpdate(ctx, tx, ids)
	if err != nil {
		return nil, err
	}

	accountMap := make(map[string]*domain.Account)
	for _, acc := range accounts {
		accountMap[acc.ID] = acc
	}

	fromAccount := accountMap[hold.AccountID]
	toAccount := accountMap[toAccountID]

	if fromAccount == nil || toAccount == nil {
		return nil, domain.ErrAccountNotFound
	}

	if fromAccount.Currency != toAccount.Currency {
		return nil, domain.ErrCurrencyMismatch
	}

	// Validate Credit for ToAccount
	if err := toAccount.ValidateCredit(hold.Amount); err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	// Create Transfer
	transfer := &domain.Transfer{
		ID:            uc.idGen.Generate(),
		FromAccountID: hold.AccountID,
		ToAccountID:   toAccountID,
		Amount:        hold.Amount,
		CreatedAt:     now,
		EventAt:       now,
		Metadata:      map[string]any{"hold_id": hold.ID, "type": "capture"},
	}

	if err := uc.transferRepo.Create(ctx, tx, transfer); err != nil {
		return nil, err
	}

	// Create Debit Entry (From)
	// Balance decreases, Encumbered decreases.
	// Note: ApplyDebit only updates Balance in the struct method. We need to handle encumbered manually.
	fromNewBalance := fromAccount.Balance.Sub(hold.Amount)
	fromEntry := &domain.Entry{
		ID:                     uc.idGen.Generate(),
		AccountID:              fromAccount.ID,
		TransferID:             transfer.ID,
		Amount:                 hold.Amount.Neg(),
		AccountPreviousBalance: fromAccount.Balance,
		AccountCurrentBalance:  fromNewBalance,
		AccountVersion:         fromAccount.Version + 1,
		CreatedAt:              now,
	}
	if err := uc.entryRepo.Create(ctx, tx, fromEntry); err != nil {
		return nil, err
	}

	// Update From Account
	fromNewEncumbered := fromAccount.EncumberedBalance.Sub(hold.Amount)
	// Update balance AND encumbered balance.
	// We need to call UpdateBalance and UpdateEncumberedBalance.
	// Or strictly: we should probably add `UpdateAccountState` to repo to do both, but doing two updates in same TX is fine.
	if err := uc.accountRepo.UpdateBalance(ctx, tx, fromAccount.ID, fromNewBalance, now); err != nil {
		return nil, err
	}
	if err := uc.accountRepo.UpdateEncumberedBalance(ctx, tx, fromAccount.ID, fromNewEncumbered, now); err != nil {
		return nil, err
	}

	// Create Credit Entry (To)
	toNewBalance := toAccount.ApplyCredit(hold.Amount)
	toEntry := &domain.Entry{
		ID:                     uc.idGen.Generate(),
		AccountID:              toAccount.ID,
		TransferID:             transfer.ID,
		Amount:                 hold.Amount,
		AccountPreviousBalance: toAccount.Balance,
		AccountCurrentBalance:  toNewBalance,
		AccountVersion:         toAccount.Version + 1,
		CreatedAt:              now,
	}
	if err := uc.entryRepo.Create(ctx, tx, toEntry); err != nil {
		return nil, err
	}

	// Update To Account
	if err := uc.accountRepo.UpdateBalance(ctx, tx, toAccount.ID, toNewBalance, now); err != nil {
		return nil, err
	}

	// Update Hold Status
	if err := uc.holdRepo.UpdateStatus(ctx, tx, hold.ID, domain.HoldStatusCaptured, now); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return transfer, nil
}
