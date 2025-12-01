package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/mock/gomock"

	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/usecase"
	"github.com/iho/goledger/internal/usecase/mocks"
)

func TestEntryUseCase_GetEntriesByAccount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	entryRepo := mocks.NewMockEntryRepository(ctrl)
	entryRepo.EXPECT().GetByAccount(gomock.Any(), "acc-1", 10, 0).Return([]*domain.Entry{
		{ID: "e1", AccountID: "acc-1", Amount: decimal.NewFromInt(100)},
		{ID: "e2", AccountID: "acc-1", Amount: decimal.NewFromInt(-50)},
	}, nil)

	uc := usecase.NewEntryUseCase(entryRepo)

	entries, err := uc.GetEntriesByAccount(context.Background(), usecase.GetEntriesByAccountInput{
		AccountID: "acc-1",
		Limit:     10,
		Offset:    0,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}

func TestEntryUseCase_GetEntriesByTransfer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	entryRepo := mocks.NewMockEntryRepository(ctrl)
	entryRepo.EXPECT().GetByTransfer(gomock.Any(), "tx-1").Return([]*domain.Entry{
		{ID: "e1", TransferID: "tx-1", Amount: decimal.NewFromInt(-100)},
		{ID: "e2", TransferID: "tx-1", Amount: decimal.NewFromInt(100)},
	}, nil)

	uc := usecase.NewEntryUseCase(entryRepo)

	entries, err := uc.GetEntriesByTransfer(context.Background(), "tx-1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}

func TestEntryUseCase_GetHistoricalBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	entryRepo := mocks.NewMockEntryRepository(ctrl)
	entryRepo.EXPECT().GetBalanceAtTime(gomock.Any(), "acc-1", gomock.Any()).Return(decimal.NewFromInt(500), nil)

	uc := usecase.NewEntryUseCase(entryRepo)

	balance, err := uc.GetHistoricalBalance(context.Background(), "acc-1", time.Now())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !balance.Equal(decimal.NewFromInt(500)) {
		t.Errorf("expected balance 500, got %s", balance)
	}
}
