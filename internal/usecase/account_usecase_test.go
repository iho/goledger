package usecase_test

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/usecase"
	"github.com/iho/goledger/internal/usecase/mocks"
)

func TestAccountUseCase_CreateAccount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockAccountRepository(ctrl)
	idGen := mocks.NewMockIDGenerator(ctrl)

	idGen.EXPECT().Generate().Return("test-id-123")
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

	uc := usecase.NewAccountUseCase(repo, idGen)

	account, err := uc.CreateAccount(context.Background(), usecase.CreateAccountInput{
		Name:                 "test-account",
		Currency:             "USD",
		AllowNegativeBalance: true,
		AllowPositiveBalance: true,
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if account == nil {
		t.Fatal("expected account, got nil")
	}

	if account.Name != "test-account" {
		t.Errorf("expected name test-account, got %s", account.Name)
	}
}

func TestAccountUseCase_GetAccount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockAccountRepository(ctrl)
	idGen := mocks.NewMockIDGenerator(ctrl)

	repo.EXPECT().GetByID(gomock.Any(), "test-id").Return(&domain.Account{
		ID:   "test-id",
		Name: "test",
	}, nil)

	uc := usecase.NewAccountUseCase(repo, idGen)

	account, err := uc.GetAccount(context.Background(), "test-id")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if account == nil {
		t.Fatal("expected account, got nil")
	}

	if account.ID != "test-id" {
		t.Errorf("expected ID test-id, got %s", account.ID)
	}
}

func TestAccountUseCase_GetAccount_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockAccountRepository(ctrl)
	idGen := mocks.NewMockIDGenerator(ctrl)

	repo.EXPECT().GetByID(gomock.Any(), "non-existent").Return(nil, domain.ErrAccountNotFound)

	uc := usecase.NewAccountUseCase(repo, idGen)
	_, err := uc.GetAccount(context.Background(), "non-existent")

	if !errors.Is(err, domain.ErrAccountNotFound) {
		t.Errorf("expected ErrAccountNotFound, got %v", err)
	}
}

func TestAccountUseCase_ListAccounts(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockAccountRepository(ctrl)
	idGen := mocks.NewMockIDGenerator(ctrl)

	repo.EXPECT().List(gomock.Any(), 10, 0).Return([]*domain.Account{
		{ID: "1", Name: "acc1"},
		{ID: "2", Name: "acc2"},
	}, nil)

	uc := usecase.NewAccountUseCase(repo, idGen)

	accounts, err := uc.ListAccounts(context.Background(), usecase.ListAccountsInput{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(accounts) != 2 {
		t.Errorf("expected 2 accounts, got %d", len(accounts))
	}
}

func TestAccountUseCase_CreateAccount_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockAccountRepository(ctrl)
	idGen := mocks.NewMockIDGenerator(ctrl)

	idGen.EXPECT().Generate().Return("test-id-123")
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errors.New("db error"))

	uc := usecase.NewAccountUseCase(repo, idGen)

	_, err := uc.CreateAccount(context.Background(), usecase.CreateAccountInput{
		Name:     "test",
		Currency: "USD",
	})

	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestAccountUseCase_ListAccounts_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockAccountRepository(ctrl)
	idGen := mocks.NewMockIDGenerator(ctrl)

	repo.EXPECT().List(gomock.Any(), 10, 0).Return(nil, errors.New("db error"))

	uc := usecase.NewAccountUseCase(repo, idGen)

	_, err := uc.ListAccounts(context.Background(), usecase.ListAccountsInput{Limit: 10, Offset: 0})

	if err == nil {
		t.Error("expected error, got nil")
	}
}
