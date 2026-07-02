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
	m *metrics.Metrics,
) *AccountUseCase {
	return &AccountUseCase{
		txManager:   txManager,
		accountRepo: accountRepo,
		auditRepo:   auditRepo,
		idGen:       idGen,
		metrics:     m,
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
func (uc *AccountUseCase) CreateAccount(ctx context.Context, input CreateAccountInput) (account *domain.Account, err error) {
	defer func() {
		if err != nil {
			uc.auditFailedAccount(ctx, input, err)
		}
	}()

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

	account = &domain.Account{
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
		userID, requestID, ipAddress, userAgent := auditActor(ctx)

		auditLog := &domain.AuditLog{
			ID:           uc.idGen.Generate(),
			UserID:       userID,
			Action:       string(domain.AuditActionAccountCreate),
			ResourceType: "account",
			ResourceID:   account.ID,
			RequestID:    requestID,
			IPAddress:    ipAddress,
			UserAgent:    userAgent,
			AfterState:   domain.MarshalState(account),
			Status:       string(domain.AuditStatusSuccess),
			CreatedAt:    time.Now().UTC(),
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

// auditFailedAccount records a failure audit row for a rejected account
// creation attempt, outside any database transaction so it survives the
// rollback that rejected it. Best-effort: an audit write failure here never
// masks the original error.
func (uc *AccountUseCase) auditFailedAccount(ctx context.Context, input CreateAccountInput, failErr error) {
	if uc.auditRepo == nil {
		return
	}

	userID, requestID, ipAddress, userAgent := auditActor(ctx)

	auditLog := &domain.AuditLog{
		ID:           uc.idGen.Generate(),
		UserID:       userID,
		Action:       string(domain.AuditActionAccountCreate),
		ResourceType: "account",
		RequestID:    requestID,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		BeforeState: domain.JSON{
			"name":     input.Name,
			"currency": input.Currency,
		},
		Status:       string(domain.AuditStatusFailure),
		ErrorMessage: failErr.Error(),
		CreatedAt:    time.Now().UTC(),
	}
	auditLog.ResourceID = auditLog.ID

	_ = uc.auditRepo.Create(ctx, auditLog)
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
