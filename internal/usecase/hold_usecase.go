package usecase

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/infrastructure/metrics"
)

type HoldUseCase struct {
	txManager    TransactionManager
	accountRepo  AccountRepository
	holdRepo     HoldRepository
	transferRepo TransferRepository
	entryRepo    EntryRepository
	outboxRepo   OutboxRepository
	auditRepo    AuditRepository
	idGen        IDGenerator
	metrics      *metrics.Metrics
}

func NewHoldUseCase(
	txManager TransactionManager,
	accountRepo AccountRepository,
	holdRepo HoldRepository,
	transferRepo TransferRepository,
	entryRepo EntryRepository,
	outboxRepo OutboxRepository,
	auditRepo AuditRepository,
	idGen IDGenerator,
	metrics *metrics.Metrics,
) *HoldUseCase {
	return &HoldUseCase{
		txManager:    txManager,
		accountRepo:  accountRepo,
		holdRepo:     holdRepo,
		transferRepo: transferRepo,
		entryRepo:    entryRepo,
		outboxRepo:   outboxRepo,
		auditRepo:    auditRepo,
		idGen:        idGen,
		metrics:      metrics,
	}
}

func (uc *HoldUseCase) HoldFunds(ctx context.Context, accountID string, amount decimal.Decimal) (*domain.Hold, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, domain.ErrInvalidAmount
	}

	// Add transaction timeout
	txCtx, cancel := context.WithTimeout(ctx, DefaultTransactionTimeout)
	defer cancel()

	tx, err := uc.txManager.Begin(txCtx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(txCtx) }()

	// Lock account
	account, err := uc.accountRepo.GetByIDForUpdate(txCtx, tx, accountID)
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

	if err := uc.holdRepo.Create(txCtx, tx, hold); err != nil {
		return nil, err
	}

	newEncumbered := account.EncumberedBalance.Add(amount)
	if err := uc.accountRepo.UpdateEncumberedBalance(txCtx, tx, accountID, newEncumbered, now); err != nil {
		return nil, err
	}

	// Emit hold created event
	event := &domain.OutboxEvent{
		ID:            uc.idGen.Generate(),
		AggregateID:   hold.ID,
		AggregateType: domain.AggregateTypeHold,
		EventType:     domain.EventTypeHoldCreated,
		Payload: map[string]any{
			"hold_id":    hold.ID,
			"account_id": hold.AccountID,
			"amount":     hold.Amount.String(),
			"currency":   account.Currency,
		},
		CreatedAt: now,
		Published: false,
	}
	if err := uc.outboxRepo.Create(txCtx, tx, event); err != nil {
		return nil, err
	}

	// Audit logging
	if uc.auditRepo != nil {
		userID := "system"
		if user, ok := domain.UserFromContext(ctx); ok {
			userID = user.ID
		}

		auditLog := &domain.AuditLog{
			ID:           uc.idGen.Generate(),
			UserID:       userID,
			Action:       string(domain.AuditActionHoldCreate),
			ResourceType: "hold",
			ResourceID:   hold.ID,
			AfterState:   domain.MarshalState(hold),
			Status:       string(domain.AuditStatusSuccess),
			CreatedAt:    time.Now(),
		}
		if err := uc.auditRepo.CreateTx(txCtx, tx, auditLog); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(txCtx); err != nil {
		return nil, err
	}

	if uc.metrics != nil {
		uc.metrics.HoldsCreated.Inc()
		uc.metrics.HoldDuration.Observe(time.Since(now).Seconds())
	}

	return hold, nil
}

func (uc *HoldUseCase) VoidHold(ctx context.Context, holdID string) error {
	start := time.Now()
	// Add transaction timeout
	txCtx, cancel := context.WithTimeout(ctx, DefaultTransactionTimeout)
	defer cancel()

	tx, err := uc.txManager.Begin(txCtx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(txCtx) }()

	hold, err := uc.holdRepo.GetByIDForUpdate(txCtx, tx, holdID)
	if err != nil {
		return err
	}

	if hold.Status != domain.HoldStatusActive {
		return domain.ErrHoldNotActive
	}

	account, err := uc.accountRepo.GetByIDForUpdate(txCtx, tx, hold.AccountID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	if err := uc.holdRepo.UpdateStatus(txCtx, tx, holdID, domain.HoldStatusVoided, now); err != nil {
		return err
	}

	newEncumbered := account.EncumberedBalance.Sub(hold.Amount)
	// Safety check: encumbered balance shouldn't go negative unless data corruption
	if newEncumbered.IsNegative() {
		newEncumbered = decimal.Zero
	}

	if err := uc.accountRepo.UpdateEncumberedBalance(txCtx, tx, hold.AccountID, newEncumbered, now); err != nil {
		return err
	}

	// Emit hold voided event
	event := &domain.OutboxEvent{
		ID:            uc.idGen.Generate(),
		AggregateID:   hold.ID,
		AggregateType: domain.AggregateTypeHold,
		EventType:     domain.EventTypeHoldVoided,
		Payload: map[string]any{
			"hold_id":    hold.ID,
			"account_id": hold.AccountID,
			"amount":     hold.Amount.String(),
		},
		CreatedAt: now,
		Published: false,
	}
	if err := uc.outboxRepo.Create(txCtx, tx, event); err != nil {
		return err
	}

	if err := tx.Commit(txCtx); err != nil {
		return err
	}

	if uc.metrics != nil {
		uc.metrics.HoldsVoided.Inc()
		uc.metrics.HoldDuration.Observe(time.Since(start).Seconds())
	}

	// Audit logging
	if uc.auditRepo != nil {
		userID := "system"
		if user, ok := domain.UserFromContext(ctx); ok {
			userID = user.ID
		}

		auditLog := &domain.AuditLog{
			ID:           uc.idGen.Generate(),
			UserID:       userID,
			Action:       string(domain.AuditActionHoldVoid),
			ResourceType: "hold",
			ResourceID:   holdID,
			Status:       string(domain.AuditStatusSuccess),
			CreatedAt:    time.Now(),
		}
		_ = uc.auditRepo.Create(ctx, auditLog)
	}

	return nil
}

func (uc *HoldUseCase) CaptureHold(ctx context.Context, holdID, toAccountID string) (*domain.Transfer, error) {
	start := time.Now()
	// Add transaction timeout
	txCtx, cancel := context.WithTimeout(ctx, DefaultTransactionTimeout)
	defer cancel()

	tx, err := uc.txManager.Begin(txCtx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(txCtx) }()

	hold, err := uc.holdRepo.GetByIDForUpdate(txCtx, tx, holdID)
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

	accounts, err := uc.accountRepo.GetByIDsForUpdate(txCtx, tx, ids)
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

	if err := uc.transferRepo.Create(txCtx, tx, transfer); err != nil {
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
	if err := uc.entryRepo.Create(txCtx, tx, fromEntry); err != nil {
		return nil, err
	}

	// Update From Account
	fromNewEncumbered := fromAccount.EncumberedBalance.Sub(hold.Amount)
	// Update balance AND encumbered balance.
	// We need to call UpdateBalance and UpdateEncumberedBalance.
	// Or strictly: we should probably add `UpdateAccountState` to repo to do both, but doing two updates in same TX is fine.
	if err := uc.accountRepo.UpdateBalance(txCtx, tx, fromAccount.ID, fromNewBalance, now); err != nil {
		return nil, err
	}
	if err := uc.accountRepo.UpdateEncumberedBalance(txCtx, tx, fromAccount.ID, fromNewEncumbered, now); err != nil {
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
	if err := uc.entryRepo.Create(txCtx, tx, toEntry); err != nil {
		return nil, err
	}

	// Update To Account
	if err := uc.accountRepo.UpdateBalance(txCtx, tx, toAccount.ID, toNewBalance, now); err != nil {
		return nil, err
	}

	// Update Hold Status
	if err := uc.holdRepo.UpdateStatus(txCtx, tx, hold.ID, domain.HoldStatusCaptured, now); err != nil {
		return nil, err
	}

	// Emit hold captured event
	event := &domain.OutboxEvent{
		ID:            uc.idGen.Generate(),
		AggregateID:   hold.ID,
		AggregateType: domain.AggregateTypeHold,
		EventType:     domain.EventTypeHoldCaptured,
		Payload: map[string]any{
			"hold_id":       hold.ID,
			"transfer_id":   transfer.ID,
			"to_account_id": toAccountID,
			"amount":        hold.Amount.String(),
		},
		CreatedAt: now,
		Published: false,
	}
	if err := uc.outboxRepo.Create(txCtx, tx, event); err != nil {
		return nil, err
	}

	if err := tx.Commit(txCtx); err != nil {
		return nil, err
	}

	if uc.metrics != nil {
		uc.metrics.HoldsCaptured.Inc()
		uc.metrics.HoldDuration.Observe(time.Since(start).Seconds())
	}

	// Audit logging
	if uc.auditRepo != nil {
		userID := "system"
		if user, ok := domain.UserFromContext(ctx); ok {
			userID = user.ID
		}

		auditLog := &domain.AuditLog{
			ID:           uc.idGen.Generate(),
			UserID:       userID,
			Action:       string(domain.AuditActionHoldCapture),
			ResourceType: "hold",
			ResourceID:   holdID,
			AfterState:   domain.MarshalState(transfer),
			Status:       string(domain.AuditStatusSuccess),
			CreatedAt:    time.Now(),
		}
		_ = uc.auditRepo.Create(ctx, auditLog)
	}

	return transfer, nil
}

// ListHoldsByAccountInput represents input for listing holds by account
type ListHoldsByAccountInput struct {
	AccountID string
	Limit     int
	Offset    int
}

// ListHoldsByAccount retrieves holds for a given account
func (uc *HoldUseCase) ListHoldsByAccount(ctx context.Context, input ListHoldsByAccountInput) ([]*domain.Hold, error) {
	return uc.holdRepo.ListByAccount(ctx, input.AccountID, input.Limit, input.Offset)
}
