
package usecase

import (
	"context"
	"sort"
	"time"

	"github.com/shopspring/decimal"

	"github.com/iho/goledger/internal/domain"
)

// TransferUseCase handles transfer business logic.
type TransferUseCase struct {
	txManager    TransactionManager
	accountRepo  AccountRepository
	transferRepo TransferRepository
	entryRepo    EntryRepository
	idGen        IDGenerator
}

// NewTransferUseCase creates a new TransferUseCase.
func NewTransferUseCase(
	txManager TransactionManager,
	accountRepo AccountRepository,
	transferRepo TransferRepository,
	entryRepo EntryRepository,
	idGen IDGenerator,
) *TransferUseCase {
	return &TransferUseCase{
		txManager:    txManager,
		accountRepo:  accountRepo,
		transferRepo: transferRepo,
		entryRepo:    entryRepo,
		idGen:        idGen,
	}
}

// CreateTransferInput represents input for creating a transfer.
type CreateTransferInput struct {
	EventAt       *time.Time
	Metadata      map[string]any
	FromAccountID string
	ToAccountID   string
	Amount        decimal.Decimal
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
	// 0. Validate inputs before starting transaction
	for _, ti := range input.Transfers {
		if ti.FromAccountID == ti.ToAccountID {
			return nil, domain.ErrSameAccount
		}

		if ti.Amount.LessThanOrEqual(decimal.Zero) {
			return nil, domain.ErrInvalidAmount
		}
	}

	// 1. Collect and sort unique account IDs (DEADLOCK PREVENTION)
	accountIDs := uc.collectUniqueAccountIDs(input.Transfers)
	sort.Strings(accountIDs)

	// 2. Begin transaction
	tx, err := uc.txManager.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// 3. Lock accounts in sorted order
	accounts, err := uc.accountRepo.GetByIDsForUpdate(ctx, tx, accountIDs)
	if err != nil {
		return nil, err
	}

	if len(accounts) != len(accountIDs) {
		return nil, domain.ErrAccountNotFound
	}

	accountMap := uc.buildAccountMap(accounts)

	// 4. Process each transfer
	now := time.Now().UTC()

	eventAt := now
	if input.EventAt != nil {
		eventAt = *input.EventAt
	}

	var transfers []*domain.Transfer
	for _, ti := range input.Transfers {
		metadata := input.Metadata
		if ti.Metadata != nil {
			metadata = ti.Metadata
		}

		transfer, err := uc.processTransfer(ctx, tx, accountMap, ti, now, eventAt, metadata)
		if err != nil {
			return nil, err
		}

		transfers = append(transfers, transfer)
	}

	// 5. Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return transfers, nil
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
		ID:            uc.idGen.Generate(),
		FromAccountID: input.FromAccountID,
		ToAccountID:   input.ToAccountID,
		Amount:        input.Amount,
		CreatedAt:     now,
		EventAt:       eventAt,
		Metadata:      metadata,
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
