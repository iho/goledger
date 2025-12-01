package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/usecase"
	"github.com/iho/goledger/internal/usecase/mocks"
	"github.com/shopspring/decimal"
)

func TestTransferUseCase_CreateTransfer(t *testing.T) {
	tests := []struct {
		name        string
		input       usecase.CreateTransferInput
		setupMocks  func(*mocks.MockAccountRepository, *mocks.MockTransferRepository, *mocks.MockEntryRepository, *mocks.MockTransactionManager)
		expectError bool
		errorType   error
	}{
		{
			name: "successful transfer",
			input: usecase.CreateTransferInput{
				FromAccountID: "acc-1",
				ToAccountID:   "acc-2",
				Amount:        decimal.NewFromInt(100),
			},
			setupMocks: func(accRepo *mocks.MockAccountRepository, txRepo *mocks.MockTransferRepository, entryRepo *mocks.MockEntryRepository, txMgr *mocks.MockTransactionManager) {
				accRepo.GetByIDsForUpdateFunc = func(ctx context.Context, tx usecase.Transaction, ids []string) ([]*domain.Account, error) {
					return []*domain.Account{
						{ID: "acc-1", Balance: decimal.NewFromInt(500), Currency: "USD", AllowNegativeBalance: true, AllowPositiveBalance: true},
						{ID: "acc-2", Balance: decimal.Zero, Currency: "USD", AllowNegativeBalance: false, AllowPositiveBalance: true},
					}, nil
				}
			},
			expectError: false,
		},
		{
			name: "reject same account transfer",
			input: usecase.CreateTransferInput{
				FromAccountID: "acc-1",
				ToAccountID:   "acc-1",
				Amount:        decimal.NewFromInt(100),
			},
			setupMocks: func(accRepo *mocks.MockAccountRepository, txRepo *mocks.MockTransferRepository, entryRepo *mocks.MockEntryRepository, txMgr *mocks.MockTransactionManager) {
				accRepo.GetByIDsForUpdateFunc = func(ctx context.Context, tx usecase.Transaction, ids []string) ([]*domain.Account, error) {
					return []*domain.Account{
						{ID: "acc-1", Balance: decimal.NewFromInt(500), Currency: "USD", AllowNegativeBalance: true, AllowPositiveBalance: true},
					}, nil
				}
			},
			expectError: true,
			errorType:   domain.ErrSameAccount,
		},
		{
			name: "reject negative balance when not allowed",
			input: usecase.CreateTransferInput{
				FromAccountID: "acc-1",
				ToAccountID:   "acc-2",
				Amount:        decimal.NewFromInt(1000),
			},
			setupMocks: func(accRepo *mocks.MockAccountRepository, txRepo *mocks.MockTransferRepository, entryRepo *mocks.MockEntryRepository, txMgr *mocks.MockTransactionManager) {
				accRepo.GetByIDsForUpdateFunc = func(ctx context.Context, tx usecase.Transaction, ids []string) ([]*domain.Account, error) {
					return []*domain.Account{
						{ID: "acc-1", Balance: decimal.NewFromInt(100), Currency: "USD", AllowNegativeBalance: false, AllowPositiveBalance: true},
						{ID: "acc-2", Balance: decimal.Zero, Currency: "USD", AllowNegativeBalance: false, AllowPositiveBalance: true},
					}, nil
				}
			},
			expectError: true,
			errorType:   domain.ErrNegativeBalanceNotAllowed,
		},
		{
			name: "reject currency mismatch",
			input: usecase.CreateTransferInput{
				FromAccountID: "acc-1",
				ToAccountID:   "acc-2",
				Amount:        decimal.NewFromInt(100),
			},
			setupMocks: func(accRepo *mocks.MockAccountRepository, txRepo *mocks.MockTransferRepository, entryRepo *mocks.MockEntryRepository, txMgr *mocks.MockTransactionManager) {
				accRepo.GetByIDsForUpdateFunc = func(ctx context.Context, tx usecase.Transaction, ids []string) ([]*domain.Account, error) {
					return []*domain.Account{
						{ID: "acc-1", Balance: decimal.NewFromInt(500), Currency: "USD", AllowNegativeBalance: true, AllowPositiveBalance: true},
						{ID: "acc-2", Balance: decimal.Zero, Currency: "EUR", AllowNegativeBalance: false, AllowPositiveBalance: true},
					}, nil
				}
			},
			expectError: true,
			errorType:   domain.ErrCurrencyMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accRepo := mocks.NewMockAccountRepository()
			txRepo := mocks.NewMockTransferRepository()
			entryRepo := mocks.NewMockEntryRepository()
			txMgr := mocks.NewMockTransactionManager()
			idGen := mocks.NewMockIDGenerator()

			idGen.GenerateFunc = func() string {
				return "generated-id-" + time.Now().Format("150405.000")
			}

			tt.setupMocks(accRepo, txRepo, entryRepo, txMgr)

			uc := usecase.NewTransferUseCase(txMgr, accRepo, txRepo, entryRepo, idGen)
			transfer, err := uc.CreateTransfer(context.Background(), tt.input)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if tt.errorType != nil && err != tt.errorType {
					t.Errorf("expected error %v, got %v", tt.errorType, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if transfer == nil {
					t.Error("expected transfer, got nil")
				}
			}
		})
	}
}

func TestTransferUseCase_GetTransfer(t *testing.T) {
	txRepo := mocks.NewMockTransferRepository()

	// Add a transfer
	txRepo.Create(context.Background(), nil, &domain.Transfer{
		ID:            "tx-123",
		FromAccountID: "acc-1",
		ToAccountID:   "acc-2",
		Amount:        decimal.NewFromInt(100),
	})

	uc := usecase.NewTransferUseCase(nil, nil, txRepo, nil, nil)

	t.Run("get existing transfer", func(t *testing.T) {
		transfer, err := uc.GetTransfer(context.Background(), "tx-123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if transfer.ID != "tx-123" {
			t.Errorf("expected ID tx-123, got %s", transfer.ID)
		}
	})

	t.Run("get non-existent transfer", func(t *testing.T) {
		_, err := uc.GetTransfer(context.Background(), "non-existent")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestTransferUseCase_CreateBatchTransfer(t *testing.T) {
	accRepo := mocks.NewMockAccountRepository()
	txRepo := mocks.NewMockTransferRepository()
	entryRepo := mocks.NewMockEntryRepository()
	txMgr := mocks.NewMockTransactionManager()
	idGen := mocks.NewMockIDGenerator()

	counter := 0
	idGen.GenerateFunc = func() string {
		counter++
		return "id-" + string(rune('0'+counter))
	}

	accRepo.GetByIDsForUpdateFunc = func(ctx context.Context, tx usecase.Transaction, ids []string) ([]*domain.Account, error) {
		return []*domain.Account{
			{ID: "acc-1", Balance: decimal.NewFromInt(1000), Currency: "USD", AllowNegativeBalance: true, AllowPositiveBalance: true},
			{ID: "acc-2", Balance: decimal.Zero, Currency: "USD", AllowNegativeBalance: true, AllowPositiveBalance: true},
			{ID: "acc-3", Balance: decimal.Zero, Currency: "USD", AllowNegativeBalance: true, AllowPositiveBalance: true},
		}, nil
	}

	uc := usecase.NewTransferUseCase(txMgr, accRepo, txRepo, entryRepo, idGen)

	input := usecase.CreateBatchTransferInput{
		Transfers: []usecase.CreateTransferInput{
			{FromAccountID: "acc-1", ToAccountID: "acc-2", Amount: decimal.NewFromInt(100)},
			{FromAccountID: "acc-1", ToAccountID: "acc-3", Amount: decimal.NewFromInt(200)},
		},
	}

	transfers, err := uc.CreateBatchTransfer(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(transfers) != 2 {
		t.Errorf("expected 2 transfers, got %d", len(transfers))
	}
}
