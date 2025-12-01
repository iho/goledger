package usecase_test

import (
	"context"
	"testing"

	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/usecase"
	"github.com/iho/goledger/internal/usecase/mocks"
)

func TestAccountUseCase_CreateAccount(t *testing.T) {
	tests := []struct {
		name        string
		input       usecase.CreateAccountInput
		setupMocks  func(*mocks.MockAccountRepository, *mocks.MockIDGenerator)
		expectError bool
	}{
		{
			name: "successful account creation",
			input: usecase.CreateAccountInput{
				Name:                 "test-account",
				Currency:             "USD",
				AllowNegativeBalance: true,
				AllowPositiveBalance: true,
			},
			setupMocks: func(repo *mocks.MockAccountRepository, idGen *mocks.MockIDGenerator) {
				idGen.GenerateFunc = func() string { return "test-id-123" }
				repo.CreateFunc = func(ctx context.Context, account *domain.Account) error {
					return nil
				}
			},
			expectError: false,
		},
		{
			name: "create with repository error",
			input: usecase.CreateAccountInput{
				Name:     "test-account",
				Currency: "USD",
			},
			setupMocks: func(repo *mocks.MockAccountRepository, idGen *mocks.MockIDGenerator) {
				idGen.GenerateFunc = func() string { return "test-id-123" }
				repo.CreateFunc = func(ctx context.Context, account *domain.Account) error {
					return domain.ErrAccountNotFound // simulate error
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockAccountRepository()
			idGen := mocks.NewMockIDGenerator()
			tt.setupMocks(repo, idGen)

			uc := usecase.NewAccountUseCase(repo, idGen)
			account, err := uc.CreateAccount(context.Background(), tt.input)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if account == nil {
					t.Error("expected account, got nil")
				}
				if account != nil && account.Name != tt.input.Name {
					t.Errorf("expected name %q, got %q", tt.input.Name, account.Name)
				}
			}
		})
	}
}

func TestAccountUseCase_GetAccount(t *testing.T) {
	tests := []struct {
		name        string
		accountID   string
		setupMocks  func(*mocks.MockAccountRepository)
		expectError bool
	}{
		{
			name:      "get existing account",
			accountID: "test-id-123",
			setupMocks: func(repo *mocks.MockAccountRepository) {
				repo.GetByIDFunc = func(ctx context.Context, id string) (*domain.Account, error) {
					return &domain.Account{ID: id, Name: "test"}, nil
				}
			},
			expectError: false,
		},
		{
			name:      "get non-existent account",
			accountID: "non-existent",
			setupMocks: func(repo *mocks.MockAccountRepository) {
				repo.GetByIDFunc = func(ctx context.Context, id string) (*domain.Account, error) {
					return nil, domain.ErrAccountNotFound
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockAccountRepository()
			idGen := mocks.NewMockIDGenerator()
			tt.setupMocks(repo)

			uc := usecase.NewAccountUseCase(repo, idGen)
			account, err := uc.GetAccount(context.Background(), tt.accountID)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if account == nil {
					t.Error("expected account, got nil")
				}
			}
		})
	}
}

func TestAccountUseCase_ListAccounts(t *testing.T) {
	repo := mocks.NewMockAccountRepository()
	idGen := mocks.NewMockIDGenerator()

	// Pre-populate with accounts
	repo.Create(context.Background(), &domain.Account{ID: "1", Name: "acc1"})
	repo.Create(context.Background(), &domain.Account{ID: "2", Name: "acc2"})

	uc := usecase.NewAccountUseCase(repo, idGen)

	accounts, err := uc.ListAccounts(context.Background(), usecase.ListAccountsInput{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(accounts) != 2 {
		t.Errorf("expected 2 accounts, got %d", len(accounts))
	}
}
