package usecase

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/infrastructure/metrics"
)

// AccountUseCase handles account business logic.
type AccountUseCase struct {
	txManager   TransactionManager
	accountRepo AccountRepository
	auditRepo   AuditRepository
	idGen       IDGenerator
	metrics     *metrics.Metrics
}

// NewAccountUseCase creates a new AccountUseCase.
func NewAccountUseCase(
	txManager TransactionManager,
	accountRepo AccountRepository,
	auditRepo AuditRepository,
	idGen IDGenerator,
	metrics *metrics.Metrics,
) *AccountUseCase {
	return &AccountUseCase{
		txManager:   txManager,
		accountRepo: accountRepo,
		auditRepo:   auditRepo,
		idGen:       idGen,
		metrics:     metrics,
	}
}

// CreateAccountInput represents input for creating an account.
type CreateAccountInput struct {
	Name                 string
	Currency             string
	AllowNegativeBalance bool
	AllowPositiveBalance bool
}

// CreateAccount creates a new account.
func (uc *AccountUseCase) CreateAccount(ctx context.Context, input CreateAccountInput) (*domain.Account, error) {
	// Validate input
	if err := domain.ValidateAccountName(input.Name); err != nil {
		return nil, err
	}
	if err := domain.ValidateCurrency(input.Currency); err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	// Start transaction
	// Add transaction timeout
	txCtx, cancel := context.WithTimeout(ctx, DefaultTransactionTimeout)
	defer cancel()

	tx, err := uc.txManager.Begin(txCtx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(txCtx) }()

	account := &domain.Account{
		ID:                   uc.idGen.Generate(),
		Name:                 input.Name,
		Currency:             input.Currency,
		Balance:              decimal.Zero,
		Version:              0,
		AllowNegativeBalance: input.AllowNegativeBalance,
		AllowPositiveBalance: input.AllowPositiveBalance,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	// Create account with transaction
	if err := uc.accountRepo.CreateTx(txCtx, tx, account); err != nil {
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
			Action:       string(domain.AuditActionAccountCreate),
			ResourceType: "account",
			ResourceID:   account.ID,
			AfterState:   domain.MarshalState(account),
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
		uc.metrics.AccountsCreated.Inc()
	}

	return account, nil
}

// GetAccount retrieves an account by ID.
func (uc *AccountUseCase) GetAccount(ctx context.Context, id string) (*domain.Account, error) {
	return uc.accountRepo.GetByID(ctx, id)
}

// ListAccountsInput represents input for listing accounts.
type ListAccountsInput struct {
	Limit  int
	Offset int
}

// ListAccounts lists accounts with pagination.
func (uc *AccountUseCase) ListAccounts(ctx context.Context, input ListAccountsInput) ([]*domain.Account, error) {
	if input.Limit <= 0 {
		input.Limit = 20
	}

	if input.Limit > 100 {
		input.Limit = 100
	}

	return uc.accountRepo.List(ctx, input.Limit, input.Offset)
}
