package usecase

import (
	"context"
	"time"

	"github.com/iho/goledger/internal/domain"
	"github.com/shopspring/decimal"
)

// AccountUseCase handles account business logic.
type AccountUseCase struct {
	accountRepo AccountRepository
	idGen       IDGenerator
}

// NewAccountUseCase creates a new AccountUseCase.
func NewAccountUseCase(accountRepo AccountRepository, idGen IDGenerator) *AccountUseCase {
	return &AccountUseCase{
		accountRepo: accountRepo,
		idGen:       idGen,
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
	now := time.Now().UTC()

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

	if err := uc.accountRepo.Create(ctx, account); err != nil {
		return nil, err
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
