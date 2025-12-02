package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"go.uber.org/mock/gomock"

	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/usecase"
	"github.com/iho/goledger/internal/usecase/mocks"
)

func TestTransferUseCase_CreateTransfer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	accRepo := mocks.NewMockAccountRepository(ctrl)
	txRepo := mocks.NewMockTransferRepository(ctrl)
	entryRepo := mocks.NewMockEntryRepository(ctrl)
	outboxRepo := mocks.NewMockOutboxRepository(ctrl)
	txMgr := mocks.NewMockTransactionManager(ctrl)
	idGen := mocks.NewMockIDGenerator(ctrl)
	mockTx := mocks.NewMockTransaction(ctrl)

	txMgr.EXPECT().Begin(gomock.Any()).Return(mockTx, nil)
	accRepo.EXPECT().GetByIDsForUpdate(gomock.Any(), mockTx, gomock.Any()).Return([]*domain.Account{
		{ID: "acc-1", Balance: decimal.NewFromInt(500), Currency: "USD", AllowNegativeBalance: true, AllowPositiveBalance: true},
		{ID: "acc-2", Balance: decimal.Zero, Currency: "USD", AllowNegativeBalance: false, AllowPositiveBalance: true},
	}, nil)
	idGen.EXPECT().Generate().Return("generated-id").Times(4) // transfer + 2 entries + event
	txRepo.EXPECT().Create(gomock.Any(), mockTx, gomock.Any()).Return(nil)
	entryRepo.EXPECT().Create(gomock.Any(), mockTx, gomock.Any()).Return(nil).Times(2)
	accRepo.EXPECT().UpdateBalance(gomock.Any(), mockTx, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(2)
	outboxRepo.EXPECT().Create(gomock.Any(), mockTx, gomock.Any()).Return(nil)
	mockTx.EXPECT().Commit(gomock.Any()).Return(nil)
	mockTx.EXPECT().Rollback(gomock.Any()).Return(nil).AnyTimes()

	uc := usecase.NewTransferUseCase(txMgr, accRepo, txRepo, entryRepo, outboxRepo, nil, idGen, nil)

	transfer, err := uc.CreateTransfer(context.Background(), usecase.CreateTransferInput{
		FromAccountID: "acc-1",
		ToAccountID:   "acc-2",
		Amount:        decimal.NewFromInt(100),
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if transfer == nil {
		t.Fatal("expected transfer, got nil")
	}
}

func TestTransferUseCase_RejectSameAccount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	accRepo := mocks.NewMockAccountRepository(ctrl)
	txRepo := mocks.NewMockTransferRepository(ctrl)
	entryRepo := mocks.NewMockEntryRepository(ctrl)
	txMgr := mocks.NewMockTransactionManager(ctrl)
	idGen := mocks.NewMockIDGenerator(ctrl)

	uc := usecase.NewTransferUseCase(txMgr, accRepo, txRepo, entryRepo, mocks.NewMockOutboxRepository(ctrl), nil, idGen, nil)
	_, err := uc.CreateTransfer(context.Background(), usecase.CreateTransferInput{
		FromAccountID: "acc-1",
		ToAccountID:   "acc-1",
		Amount:        decimal.NewFromInt(100),
	})

	if !errors.Is(err, domain.ErrSameAccount) {
		t.Errorf("expected ErrSameAccount, got %v", err)
	}
}

func TestTransferUseCase_RejectZeroAmount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	accRepo := mocks.NewMockAccountRepository(ctrl)
	txRepo := mocks.NewMockTransferRepository(ctrl)
	entryRepo := mocks.NewMockEntryRepository(ctrl)
	txMgr := mocks.NewMockTransactionManager(ctrl)
	idGen := mocks.NewMockIDGenerator(ctrl)

	uc := usecase.NewTransferUseCase(txMgr, accRepo, txRepo, entryRepo, mocks.NewMockOutboxRepository(ctrl), nil, idGen, nil)
	_, err := uc.CreateTransfer(context.Background(), usecase.CreateTransferInput{
		FromAccountID: "acc-1",
		ToAccountID:   "acc-2",
		Amount:        decimal.Zero,
	})

	if !errors.Is(err, domain.ErrInvalidAmount) {
		t.Errorf("expected ErrInvalidAmount, got %v", err)
	}
}

func TestTransferUseCase_GetTransfer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	txRepo := mocks.NewMockTransferRepository(ctrl)
	txRepo.EXPECT().GetByID(gomock.Any(), "tx-123").Return(&domain.Transfer{
		ID:            "tx-123",
		FromAccountID: "acc-1",
		ToAccountID:   "acc-2",
		Amount:        decimal.NewFromInt(100),
	}, nil)

	uc := usecase.NewTransferUseCase(nil, nil, txRepo, nil, nil, nil, nil, nil)

	transfer, err := uc.GetTransfer(context.Background(), "tx-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if transfer.ID != "tx-123" {
		t.Errorf("expected ID tx-123, got %s", transfer.ID)
	}
}

func TestTransferUseCase_RejectNegativeAmount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	uc := usecase.NewTransferUseCase(nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := uc.CreateTransfer(context.Background(), usecase.CreateTransferInput{
		FromAccountID: "acc-1",
		ToAccountID:   "acc-2",
		Amount:        decimal.NewFromInt(-100),
	})

	if !errors.Is(err, domain.ErrInvalidAmount) {
		t.Errorf("expected ErrInvalidAmount, got %v", err)
	}
}

func TestTransferUseCase_CurrencyMismatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	accRepo := mocks.NewMockAccountRepository(ctrl)
	txRepo := mocks.NewMockTransferRepository(ctrl)
	entryRepo := mocks.NewMockEntryRepository(ctrl)
	txMgr := mocks.NewMockTransactionManager(ctrl)
	idGen := mocks.NewMockIDGenerator(ctrl)
	mockTx := mocks.NewMockTransaction(ctrl)

	txMgr.EXPECT().Begin(gomock.Any()).Return(mockTx, nil)
	accRepo.EXPECT().GetByIDsForUpdate(gomock.Any(), mockTx, gomock.Any()).Return([]*domain.Account{
		{ID: "acc-1", Balance: decimal.NewFromInt(500), Currency: "USD", AllowNegativeBalance: true, AllowPositiveBalance: true},
		{ID: "acc-2", Balance: decimal.Zero, Currency: "EUR", AllowNegativeBalance: false, AllowPositiveBalance: true},
	}, nil)
	mockTx.EXPECT().Rollback(gomock.Any()).Return(nil).AnyTimes()

	uc := usecase.NewTransferUseCase(txMgr, accRepo, txRepo, entryRepo, mocks.NewMockOutboxRepository(ctrl), nil, idGen, nil)
	_, err := uc.CreateTransfer(context.Background(), usecase.CreateTransferInput{
		FromAccountID: "acc-1",
		ToAccountID:   "acc-2",
		Amount:        decimal.NewFromInt(100),
	})

	if !errors.Is(err, domain.ErrCurrencyMismatch) {
		t.Errorf("expected ErrCurrencyMismatch, got %v", err)
	}
}

func TestTransferUseCase_InsufficientBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	accRepo := mocks.NewMockAccountRepository(ctrl)
	txRepo := mocks.NewMockTransferRepository(ctrl)
	entryRepo := mocks.NewMockEntryRepository(ctrl)
	txMgr := mocks.NewMockTransactionManager(ctrl)
	idGen := mocks.NewMockIDGenerator(ctrl)
	mockTx := mocks.NewMockTransaction(ctrl)

	txMgr.EXPECT().Begin(gomock.Any()).Return(mockTx, nil)
	accRepo.EXPECT().GetByIDsForUpdate(gomock.Any(), mockTx, gomock.Any()).Return([]*domain.Account{
		{ID: "acc-1", Balance: decimal.NewFromInt(50), Currency: "USD", AllowNegativeBalance: false, AllowPositiveBalance: true},
		{ID: "acc-2", Balance: decimal.Zero, Currency: "USD", AllowNegativeBalance: false, AllowPositiveBalance: true},
	}, nil)
	mockTx.EXPECT().Rollback(gomock.Any()).Return(nil).AnyTimes()

	uc := usecase.NewTransferUseCase(txMgr, accRepo, txRepo, entryRepo, mocks.NewMockOutboxRepository(ctrl), nil, idGen, nil)
	_, err := uc.CreateTransfer(context.Background(), usecase.CreateTransferInput{
		FromAccountID: "acc-1",
		ToAccountID:   "acc-2",
		Amount:        decimal.NewFromInt(100),
	})

	if !errors.Is(err, domain.ErrNegativeBalanceNotAllowed) {
		t.Errorf("expected ErrNegativeBalanceNotAllowed, got %v", err)
	}
}

func TestTransferUseCase_ListByAccount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	txRepo := mocks.NewMockTransferRepository(ctrl)
	txRepo.EXPECT().ListByAccount(gomock.Any(), "acc-1", 10, 0).Return([]*domain.Transfer{
		{ID: "tx-1", FromAccountID: "acc-1", ToAccountID: "acc-2", Amount: decimal.NewFromInt(100)},
		{ID: "tx-2", FromAccountID: "acc-2", ToAccountID: "acc-1", Amount: decimal.NewFromInt(50)},
	}, nil)

	uc := usecase.NewTransferUseCase(nil, nil, txRepo, nil, nil, nil, nil, nil)

	transfers, err := uc.ListTransfersByAccount(context.Background(), usecase.ListTransfersByAccountInput{
		AccountID: "acc-1",
		Limit:     10,
		Offset:    0,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(transfers) != 2 {
		t.Errorf("expected 2 transfers, got %d", len(transfers))
	}
}
