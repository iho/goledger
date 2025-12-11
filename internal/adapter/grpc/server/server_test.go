package server_test

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/iho/goledger/internal/adapter/grpc/pb/goledger/v1"
	"github.com/iho/goledger/internal/adapter/grpc/server"
	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/usecase"
)

// --- Account Server Tests ---

type accountUseCaseStub struct {
	createFn func(ctx context.Context, input usecase.CreateAccountInput) (*domain.Account, error)
	getFn    func(ctx context.Context, id string) (*domain.Account, error)
	listFn   func(ctx context.Context, input usecase.ListAccountsInput) ([]*domain.Account, error)
}

func (s *accountUseCaseStub) CreateAccount(ctx context.Context, input usecase.CreateAccountInput) (*domain.Account, error) {
	return s.createFn(ctx, input)
}
func (s *accountUseCaseStub) GetAccount(ctx context.Context, id string) (*domain.Account, error) {
	return s.getFn(ctx, id)
}
func (s *accountUseCaseStub) ListAccounts(ctx context.Context, input usecase.ListAccountsInput) ([]*domain.Account, error) {
	return s.listFn(ctx, input)
}

func TestAccountServer_CreateAccount_Success(t *testing.T) {
	now := time.Now().UTC()
	expected := &domain.Account{
		ID:        "acc-123",
		Name:      "Main",
		Currency:  "USD",
		Balance:   decimal.NewFromInt(0),
		CreatedAt: now,
		UpdatedAt: now,
	}

	var capturedInput usecase.CreateAccountInput
	accountUC := &accountUseCaseStub{
		createFn: func(ctx context.Context, input usecase.CreateAccountInput) (*domain.Account, error) {
			capturedInput = input
			return expected, nil
		},
	}

	srv := server.NewAccountServer(accountUC)
	resp, err := srv.CreateAccount(context.Background(), &pb.CreateAccountRequest{
		Name:                 "Main",
		Currency:             "USD",
		AllowNegativeBalance: true,
		AllowPositiveBalance: false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedInput.Name != "Main" || !capturedInput.AllowNegativeBalance {
		t.Fatalf("expected input to match request, got %+v", capturedInput)
	}

	if resp.Account == nil || resp.Account.Id != expected.ID {
		t.Fatalf("expected response account to have ID %s, got %+v", expected.ID, resp.Account)
	}
}

func TestAccountServer_GetAccount_ErrorMapping(t *testing.T) {
	accountUC := &accountUseCaseStub{
		getFn: func(ctx context.Context, id string) (*domain.Account, error) {
			return nil, domain.ErrAccountNotFound
		},
		createFn: func(ctx context.Context, input usecase.CreateAccountInput) (*domain.Account, error) { return nil, nil },
		listFn:   func(ctx context.Context, input usecase.ListAccountsInput) ([]*domain.Account, error) { return nil, nil },
	}

	srv := server.NewAccountServer(accountUC)
	_, err := srv.GetAccount(context.Background(), &pb.GetAccountRequest{Id: "missing"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestAccountServer_ListAccounts(t *testing.T) {
	accountUC := &accountUseCaseStub{
		listFn: func(ctx context.Context, input usecase.ListAccountsInput) ([]*domain.Account, error) {
			return []*domain.Account{
				{ID: "1", Name: "A", Currency: "USD"},
				{ID: "2", Name: "B", Currency: "EUR"},
			}, nil
		},
		createFn: func(ctx context.Context, input usecase.CreateAccountInput) (*domain.Account, error) { return nil, nil },
		getFn:    func(ctx context.Context, id string) (*domain.Account, error) { return nil, nil },
	}

	srv := server.NewAccountServer(accountUC)
	resp, err := srv.ListAccounts(context.Background(), &pb.ListAccountsRequest{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Accounts) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(resp.Accounts))
	}
}

// --- Transfer Server Tests ---

type transferUseCaseStub struct {
	createFn      func(ctx context.Context, input usecase.CreateTransferInput) (*domain.Transfer, error)
	createBatchFn func(ctx context.Context, input usecase.CreateBatchTransferInput) ([]*domain.Transfer, error)
	getFn         func(ctx context.Context, id string) (*domain.Transfer, error)
	listFn        func(ctx context.Context, input usecase.ListTransfersByAccountInput) ([]*domain.Transfer, error)
	reverseFn     func(ctx context.Context, input usecase.ReverseTransferInput) (*domain.Transfer, error)
}

func (s *transferUseCaseStub) CreateTransfer(ctx context.Context, input usecase.CreateTransferInput) (*domain.Transfer, error) {
	return s.createFn(ctx, input)
}
func (s *transferUseCaseStub) CreateBatchTransfer(ctx context.Context, input usecase.CreateBatchTransferInput) ([]*domain.Transfer, error) {
	return s.createBatchFn(ctx, input)
}
func (s *transferUseCaseStub) GetTransfer(ctx context.Context, id string) (*domain.Transfer, error) {
	return s.getFn(ctx, id)
}
func (s *transferUseCaseStub) ListTransfersByAccount(ctx context.Context, input usecase.ListTransfersByAccountInput) ([]*domain.Transfer, error) {
	return s.listFn(ctx, input)
}
func (s *transferUseCaseStub) ReverseTransfer(ctx context.Context, input usecase.ReverseTransferInput) (*domain.Transfer, error) {
	return s.reverseFn(ctx, input)
}

func TestTransferServer_CreateTransfer_Success(t *testing.T) {
	transfer := &domain.Transfer{
		ID:            "tx-1",
		FromAccountID: "acc-1",
		ToAccountID:   "acc-2",
		Amount:        decimal.NewFromFloat(42.5),
		CreatedAt:     time.Now().UTC(),
	}

	var captured usecase.CreateTransferInput
	transferUC := &transferUseCaseStub{
		createFn: func(ctx context.Context, input usecase.CreateTransferInput) (*domain.Transfer, error) {
			captured = input
			return transfer, nil
		},
		createBatchFn: func(ctx context.Context, input usecase.CreateBatchTransferInput) ([]*domain.Transfer, error) {
			return nil, nil
		},
		getFn: func(ctx context.Context, id string) (*domain.Transfer, error) { return nil, nil },
		listFn: func(ctx context.Context, input usecase.ListTransfersByAccountInput) ([]*domain.Transfer, error) {
			return nil, nil
		},
		reverseFn: func(ctx context.Context, input usecase.ReverseTransferInput) (*domain.Transfer, error) {
			return nil, nil
		},
	}

	srv := server.NewTransferServer(transferUC)
	resp, err := srv.CreateTransfer(context.Background(), &pb.CreateTransferRequest{
		FromAccountId: "acc-1",
		ToAccountId:   "acc-2",
		Amount:        "42.5",
		Metadata:      map[string]string{"note": "demo"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !captured.Amount.Equal(decimal.RequireFromString("42.5")) || captured.Metadata["note"] != "demo" {
		t.Fatalf("expected input to match request, got %+v", captured)
	}
	if resp.Transfer.Id != transfer.ID {
		t.Fatalf("expected response transfer, got %+v", resp.Transfer)
	}
}

func TestTransferServer_CreateTransfer_InvalidAmount(t *testing.T) {
	transferUC := &transferUseCaseStub{
		createFn: func(ctx context.Context, input usecase.CreateTransferInput) (*domain.Transfer, error) {
			t.Fatal("usecase should not be invoked when amount is invalid")
			return nil, nil
		},
		createBatchFn: func(ctx context.Context, input usecase.CreateBatchTransferInput) ([]*domain.Transfer, error) {
			return nil, nil
		},
		getFn: func(ctx context.Context, id string) (*domain.Transfer, error) { return nil, nil },
		listFn: func(ctx context.Context, input usecase.ListTransfersByAccountInput) ([]*domain.Transfer, error) {
			return nil, nil
		},
		reverseFn: func(ctx context.Context, input usecase.ReverseTransferInput) (*domain.Transfer, error) {
			return nil, nil
		},
	}

	srv := server.NewTransferServer(transferUC)
	_, err := srv.CreateTransfer(context.Background(), &pb.CreateTransferRequest{
		FromAccountId: "acc-1",
		ToAccountId:   "acc-2",
		Amount:        "not-a-number",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

func TestTransferServer_ReverseTransfer_Error(t *testing.T) {
	transferUC := &transferUseCaseStub{
		reverseFn: func(ctx context.Context, input usecase.ReverseTransferInput) (*domain.Transfer, error) {
			return nil, domain.ErrTransferNotFound
		},
		createFn: func(ctx context.Context, input usecase.CreateTransferInput) (*domain.Transfer, error) {
			return nil, nil
		},
		createBatchFn: func(ctx context.Context, input usecase.CreateBatchTransferInput) ([]*domain.Transfer, error) {
			return nil, nil
		},
		getFn: func(ctx context.Context, id string) (*domain.Transfer, error) { return nil, nil },
		listFn: func(ctx context.Context, input usecase.ListTransfersByAccountInput) ([]*domain.Transfer, error) {
			return nil, nil
		},
	}

	srv := server.NewTransferServer(transferUC)
	_, err := srv.ReverseTransfer(context.Background(), &pb.ReverseTransferRequest{TransferId: "missing"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

// --- Hold Server Tests ---

type holdUseCaseStub struct {
	holdFn    func(ctx context.Context, accountID string, amount decimal.Decimal) (*domain.Hold, error)
	voidFn    func(ctx context.Context, holdID string) error
	captureFn func(ctx context.Context, holdID, toAccountID string) (*domain.Transfer, error)
	listFn    func(ctx context.Context, input usecase.ListHoldsByAccountInput) ([]*domain.Hold, error)
}

func (s *holdUseCaseStub) HoldFunds(ctx context.Context, accountID string, amount decimal.Decimal) (*domain.Hold, error) {
	return s.holdFn(ctx, accountID, amount)
}
func (s *holdUseCaseStub) VoidHold(ctx context.Context, holdID string) error {
	return s.voidFn(ctx, holdID)
}
func (s *holdUseCaseStub) CaptureHold(ctx context.Context, holdID, toAccountID string) (*domain.Transfer, error) {
	return s.captureFn(ctx, holdID, toAccountID)
}
func (s *holdUseCaseStub) ListHoldsByAccount(ctx context.Context, input usecase.ListHoldsByAccountInput) ([]*domain.Hold, error) {
	return s.listFn(ctx, input)
}

func TestHoldServer_HoldFunds_InvalidAmount(t *testing.T) {
	holdUC := &holdUseCaseStub{
		holdFn: func(ctx context.Context, accountID string, amount decimal.Decimal) (*domain.Hold, error) {
			t.Fatal("HoldFunds should not be called on invalid input")
			return nil, nil
		},
		voidFn:    func(ctx context.Context, holdID string) error { return nil },
		captureFn: func(ctx context.Context, holdID, toAccountID string) (*domain.Transfer, error) { return nil, nil },
		listFn: func(ctx context.Context, input usecase.ListHoldsByAccountInput) ([]*domain.Hold, error) {
			return nil, nil
		},
	}

	srv := server.NewHoldServer(holdUC)
	_, err := srv.HoldFunds(context.Background(), &pb.HoldFundsRequest{
		AccountId: "acc-1",
		Amount:    "invalid",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

func TestHoldServer_ListHoldsByAccount(t *testing.T) {
	now := time.Now().UTC()
	holdUC := &holdUseCaseStub{
		listFn: func(ctx context.Context, input usecase.ListHoldsByAccountInput) ([]*domain.Hold, error) {
			return []*domain.Hold{
				{ID: "hold-1", AccountID: "acc-1", Amount: decimal.NewFromInt(10), Status: domain.HoldStatusActive, CreatedAt: now, UpdatedAt: now},
			}, nil
		},
		holdFn: func(ctx context.Context, accountID string, amount decimal.Decimal) (*domain.Hold, error) {
			return nil, nil
		},
		voidFn:    func(ctx context.Context, holdID string) error { return nil },
		captureFn: func(ctx context.Context, holdID, toAccountID string) (*domain.Transfer, error) { return nil, nil },
	}

	srv := server.NewHoldServer(holdUC)
	resp, err := srv.ListHoldsByAccount(context.Background(), &pb.ListHoldsByAccountRequest{
		AccountId: "acc-1",
		Limit:     5,
		Offset:    0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Holds) != 1 || resp.Holds[0].Id != "hold-1" {
		t.Fatalf("expected hold to be returned, got %+v", resp.Holds)
	}
}
