package usecase

import (
	"context"
	"maps"
	"sort"
	"time"

	"github.com/shopspring/decimal"

	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/infrastructure/metrics"
)

// Retrier defines retry behavior for operations.
type Retrier interface {
	Retry(ctx context.Context, operation func() error) error
}

// TransferUseCase handles transfer business logic.
type TransferUseCase struct {
	txManager    TransactionManager
	accountRepo  AccountRepository
	transferRepo TransferRepository
	entryRepo    EntryRepository
	outboxRepo   OutboxRepository
	auditRepo    AuditRepository
	idGen        IDGenerator
	retrier      Retrier
	metrics      *metrics.Metrics
}

// NewTransferUseCase creates a new TransferUseCase.
func NewTransferUseCase(
	txManager TransactionManager,
	accountRepo AccountRepository,
	transferRepo TransferRepository,
	entryRepo EntryRepository,
	outboxRepo OutboxRepository,
	auditRepo AuditRepository,
	idGen IDGenerator,
	m *metrics.Metrics,
) *TransferUseCase {
	return &TransferUseCase{
		txManager:    txManager,
		accountRepo:  accountRepo,
		transferRepo: transferRepo,
		entryRepo:    entryRepo,
		outboxRepo:   outboxRepo,
		auditRepo:    auditRepo,
		idGen:        idGen,
		retrier:      &noopRetrier{},
		metrics:      m,
	}
}

// WithRetrier sets a custom retrier for the use case.
func (uc *TransferUseCase) WithRetrier(r Retrier) *TransferUseCase {
	uc.retrier = r
	return uc
}

// noopRetrier is a no-op retrier that just executes the operation once.
type noopRetrier struct{}

func (r *noopRetrier) Retry(ctx context.Context, operation func() error) error {
	return operation()
}

// CreateTransferInput represents input for creating a transfer.
type CreateTransferInput struct {
	EventAt       *time.Time
	Metadata      map[string]any
	FromAccountID string
	ToAccountID   string
	Amount        decimal.Decimal
	// ReversedTransferID, when set, marks this transfer as a reversal of the
	// referenced transfer. Leave nil for ordinary transfers.
	ReversedTransferID *string
}

// CreateBatchTransferInput represents input for creating multiple transfers atomically.
type CreateBatchTransferInput struct {
	EventAt   *time.Time
	Metadata  map[string]any
	Transfers []CreateTransferInput
}

// CreateTransfer creates a single transfer.
func (uc *TransferUseCase) CreateTransfer(ctx context.Context, input CreateTransferInput) (*domain.Transfer, error) {
	result, err := uc.CreateBatchTransfer(ctx, CreateBatchTransferInput{
		Transfers: []CreateTransferInput{input},
		EventAt:   input.EventAt,
		Metadata:  input.Metadata,
	})
	if err != nil {
		return nil, err
	}

	return result[0], nil
}

// CreateBatchTransfer creates multiple transfers atomically.
func (uc *TransferUseCase) CreateBatchTransfer(ctx context.Context, input CreateBatchTransferInput) ([]*domain.Transfer, error) {
	start := time.Now()

	// 0. Validate inputs before starting transaction
	if err := domain.ValidateMetadata(input.Metadata); err != nil {
		return nil, err
	}

	for _, ti := range input.Transfers {
		if ti.FromAccountID == ti.ToAccountID {
			return nil, domain.ErrSameAccount
		}

		if ti.Amount.LessThanOrEqual(decimal.Zero) {
			return nil, domain.ErrInvalidAmount
		}

		if err := domain.ValidateMetadata(ti.Metadata); err != nil {
			return nil, err
		}
	}

	// 1. Collect and sort unique account IDs (DEADLOCK PREVENTION)
	accountIDs := uc.collectUniqueAccountIDs(input.Transfers)
	sort.Strings(accountIDs)

	// Execute with retry for deadlock/serialization errors
	var transfers []*domain.Transfer
	var currencies []string
	err := uc.retrier.Retry(ctx, func() error {
		var txErr error
		transfers, currencies, txErr = uc.executeTransferTransaction(ctx, input, accountIDs)
		return txErr
	})

	if uc.metrics != nil {
		duration := time.Since(start).Seconds()
		uc.metrics.TransferDuration.Observe(duration)

		if err != nil {
			uc.metrics.TransferErrors.WithLabelValues("create_failed").Inc()
		} else {
			uc.metrics.TransfersCreated.Add(float64(len(transfers)))
			for i, t := range transfers {
				val, _ := t.Amount.Float64()
				uc.metrics.TransferAmount.WithLabelValues(currencies[i]).Observe(val)
			}
		}
	}

	return transfers, err
}

// executeTransferTransaction runs the transfer in a single transaction. It
// returns the created transfers alongside their account currency (same
// order), for currency-labeled metrics; the currency isn't stored on
// domain.Transfer itself.
func (uc *TransferUseCase) executeTransferTransaction(
	ctx context.Context,
	input CreateBatchTransferInput,
	accountIDs []string,
) ([]*domain.Transfer, []string, error) {
	// Add transaction timeout to prevent long-running transactions
	txCtx, cancel := context.WithTimeout(ctx, DefaultTransactionTimeout)
	defer cancel()

	// Begin transaction
	tx, err := uc.txManager.Begin(txCtx)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback(txCtx) }()

	// 3. Lock accounts in sorted order
	accounts, err := uc.accountRepo.GetByIDsForUpdate(txCtx, tx, accountIDs)
	if err != nil {
		return nil, nil, err
	}

	if len(accounts) != len(accountIDs) {
		return nil, nil, domain.ErrAccountNotFound
	}

	accountMap := uc.buildAccountMap(accounts)

	// 4. Process each transfer
	now := time.Now().UTC()

	eventAt := now
	if input.EventAt != nil {
		eventAt = *input.EventAt
	}

	transfers := make([]*domain.Transfer, 0, len(input.Transfers))
	currencies := make([]string, 0, len(input.Transfers))
	for _, ti := range input.Transfers {
		metadata := input.Metadata
		if ti.Metadata != nil {
			metadata = ti.Metadata
		}

		transfer, err := uc.processTransfer(txCtx, tx, accountMap, ti, now, eventAt, metadata)
		if err != nil {
			return nil, nil, err
		}

		currencies = append(currencies, accountMap[ti.FromAccountID].Currency)

		transfers = append(transfers, transfer)
	}

	// Audit logging
	if uc.auditRepo != nil {
		userID := "system"
		if user, ok := domain.UserFromContext(ctx); ok {
			userID = user.ID
		}

		for _, t := range transfers {
			action := domain.AuditActionTransferCreate
			if t.ReversedTransferID != nil {
				action = domain.AuditActionTransferReverse
			}

			auditLog := &domain.AuditLog{
				ID:           uc.idGen.Generate(),
				UserID:       userID,
				Action:       string(action),
				ResourceType: "transfer",
				ResourceID:   t.ID,
				AfterState:   domain.MarshalState(t),
				Status:       string(domain.AuditStatusSuccess),
				CreatedAt:    time.Now().UTC(),
			}
			if err := uc.auditRepo.CreateTx(txCtx, tx, auditLog); err != nil {
				return nil, nil, err
			}
		}
	}

	// 5. Commit transaction
	if err := tx.Commit(txCtx); err != nil {
		return nil, nil, err
	}

	return transfers, currencies, nil
}

func (uc *TransferUseCase) processTransfer(
	ctx context.Context,
	tx Transaction,
	accountMap map[string]*domain.Account,
	input CreateTransferInput,
	now, eventAt time.Time,
	metadata map[string]any,
) (*domain.Transfer, error) {
	fromAccount := accountMap[input.FromAccountID]
	toAccount := accountMap[input.ToAccountID]

	if fromAccount == nil || toAccount == nil {
		return nil, domain.ErrAccountNotFound
	}

	// Validate currency match
	if fromAccount.Currency != toAccount.Currency {
		return nil, domain.ErrCurrencyMismatch
	}

	// Validate debit
	err := fromAccount.ValidateDebit(input.Amount)
	if err != nil {
		return nil, err
	}

	// Validate credit
	err = toAccount.ValidateCredit(input.Amount)
	if err != nil {
		return nil, err
	}

	// Create transfer
	transfer := &domain.Transfer{
		ID:                 uc.idGen.Generate(),
		FromAccountID:      input.FromAccountID,
		ToAccountID:        input.ToAccountID,
		Amount:             input.Amount,
		CreatedAt:          now,
		EventAt:            eventAt,
		Metadata:           metadata,
		ReversedTransferID: input.ReversedTransferID,
	}

	err = transfer.Validate()
	if err != nil {
		return nil, err
	}

	err = uc.transferRepo.Create(ctx, tx, transfer)
	if err != nil {
		return nil, err
	}

	// Create debit entry (from account)
	fromNewBalance := fromAccount.ApplyDebit(input.Amount)
	fromEntry := &domain.Entry{
		ID:                     uc.idGen.Generate(),
		AccountID:              fromAccount.ID,
		TransferID:             transfer.ID,
		Amount:                 input.Amount.Neg(),
		AccountPreviousBalance: fromAccount.Balance,
		AccountCurrentBalance:  fromNewBalance,
		AccountVersion:         fromAccount.Version + 1,
		CreatedAt:              now,
	}

	err = uc.entryRepo.Create(ctx, tx, fromEntry)
	if err != nil {
		return nil, err
	}

	// Update from account balance
	err = uc.accountRepo.UpdateBalance(ctx, tx, fromAccount.ID, fromNewBalance, now)
	if err != nil {
		return nil, err
	}

	fromAccount.Balance = fromNewBalance
	fromAccount.Version++

	// Create credit entry (to account)
	toNewBalance := toAccount.ApplyCredit(input.Amount)
	toEntry := &domain.Entry{
		ID:                     uc.idGen.Generate(),
		AccountID:              toAccount.ID,
		TransferID:             transfer.ID,
		Amount:                 input.Amount,
		AccountPreviousBalance: toAccount.Balance,
		AccountCurrentBalance:  toNewBalance,
		AccountVersion:         toAccount.Version + 1,
		CreatedAt:              now,
	}

	err = uc.entryRepo.Create(ctx, tx, toEntry)
	if err != nil {
		return nil, err
	}

	// Update to account balance
	err = uc.accountRepo.UpdateBalance(ctx, tx, toAccount.ID, toNewBalance, now)
	if err != nil {
		return nil, err
	}

	toAccount.Balance = toNewBalance
	toAccount.Version++

	// Emit transfer created/reversed event
	event := &domain.OutboxEvent{
		ID:            uc.idGen.Generate(),
		AggregateID:   transfer.ID,
		AggregateType: domain.AggregateTypeTransfer,
		CreatedAt:     now,
		Published:     false,
	}

	if transfer.ReversedTransferID != nil {
		event.EventType = domain.EventTypeTransferReversed
		event.Payload = map[string]any{
			"reversal_transfer_id": transfer.ID,
			"original_transfer_id": *transfer.ReversedTransferID,
			"amount":               transfer.Amount.String(),
			"event_at":             transfer.EventAt.Format(time.RFC3339),
		}
	} else {
		event.EventType = domain.EventTypeTransferCreated
		event.Payload = map[string]any{
			"transfer_id":     transfer.ID,
			"from_account_id": transfer.FromAccountID,
			"to_account_id":   transfer.ToAccountID,
			"amount":          transfer.Amount.String(),
			"event_at":        transfer.EventAt.Format(time.RFC3339),
		}
	}

	if err := uc.outboxRepo.Create(ctx, tx, event); err != nil {
		return nil, err
	}

	return transfer, nil
}

// GetTransfer retrieves a transfer by ID.
func (uc *TransferUseCase) GetTransfer(ctx context.Context, id string) (*domain.Transfer, error) {
	return uc.transferRepo.GetByID(ctx, id)
}

// ListTransfersByAccountInput represents input for listing transfers.
type ListTransfersByAccountInput struct {
	AccountID string
	Limit     int
	Offset    int
}

// ListTransfersByAccount lists transfers for an account.
func (uc *TransferUseCase) ListTransfersByAccount(ctx context.Context, input ListTransfersByAccountInput) ([]*domain.Transfer, error) {
	if input.Limit <= 0 {
		input.Limit = 20
	}

	if input.Limit > 100 {
		input.Limit = 100
	}

	return uc.transferRepo.ListByAccount(ctx, input.AccountID, input.Limit, input.Offset)
}

func (uc *TransferUseCase) collectUniqueAccountIDs(transfers []CreateTransferInput) []string {
	seen := make(map[string]bool)

	var ids []string
	for _, t := range transfers {
		if !seen[t.FromAccountID] {
			seen[t.FromAccountID] = true
			ids = append(ids, t.FromAccountID)
		}

		if !seen[t.ToAccountID] {
			seen[t.ToAccountID] = true
			ids = append(ids, t.ToAccountID)
		}
	}

	return ids
}

func (uc *TransferUseCase) buildAccountMap(accounts []*domain.Account) map[string]*domain.Account {
	m := make(map[string]*domain.Account)
	for _, a := range accounts {
		m[a.ID] = a
	}

	return m
}

// ReverseTransferInput represents input for reversing a transfer.
type ReverseTransferInput struct {
	TransferID string
	Metadata   map[string]any
}

// ReverseTransfer creates a reversal transfer that offsets an original transfer.
//
// It delegates to CreateBatchTransfer so the reversal gets the same
// deadlock-safe sorted account locking, outbox event, and audit logging as
// an ordinary transfer. Double-reversal is prevented by a unique partial
// index on transfers.reversed_transfer_id (see migration 000007); a
// violation is translated to domain.ErrTransferAlreadyReversed by the
// transfer repository.
func (uc *TransferUseCase) ReverseTransfer(ctx context.Context, input ReverseTransferInput) (*domain.Transfer, error) {
	originalTransfer, err := uc.transferRepo.GetByID(ctx, input.TransferID)
	if err != nil {
		return nil, err
	}

	metadata := make(map[string]any, len(input.Metadata)+1)
	maps.Copy(metadata, input.Metadata)
	metadata["reversal_of"] = originalTransfer.ID

	now := time.Now().UTC()
	reversedTransferID := originalTransfer.ID

	result, err := uc.CreateBatchTransfer(ctx, CreateBatchTransferInput{
		EventAt: &now,
		Transfers: []CreateTransferInput{
			{
				FromAccountID:      originalTransfer.ToAccountID,   // Swap
				ToAccountID:        originalTransfer.FromAccountID, // Swap
				Amount:             originalTransfer.Amount,
				Metadata:           metadata,
				ReversedTransferID: &reversedTransferID,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return result[0], nil
}
