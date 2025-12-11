package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/iho/goledger/internal/adapter/http/dto"
	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/usecase"
)

type transferServiceStub struct {
	createFn      func(ctx context.Context, input usecase.CreateTransferInput) (*domain.Transfer, error)
	createBatchFn func(ctx context.Context, input usecase.CreateBatchTransferInput) ([]*domain.Transfer, error)
	getFn         func(ctx context.Context, id string) (*domain.Transfer, error)
	listFn        func(ctx context.Context, input usecase.ListTransfersByAccountInput) ([]*domain.Transfer, error)
	reverseFn     func(ctx context.Context, input usecase.ReverseTransferInput) (*domain.Transfer, error)
}

func (s *transferServiceStub) CreateTransfer(ctx context.Context, input usecase.CreateTransferInput) (*domain.Transfer, error) {
	return s.createFn(ctx, input)
}

func (s *transferServiceStub) CreateBatchTransfer(ctx context.Context, input usecase.CreateBatchTransferInput) ([]*domain.Transfer, error) {
	return s.createBatchFn(ctx, input)
}

func (s *transferServiceStub) GetTransfer(ctx context.Context, id string) (*domain.Transfer, error) {
	return s.getFn(ctx, id)
}

func (s *transferServiceStub) ListTransfersByAccount(ctx context.Context, input usecase.ListTransfersByAccountInput) ([]*domain.Transfer, error) {
	return s.listFn(ctx, input)
}

func (s *transferServiceStub) ReverseTransfer(ctx context.Context, input usecase.ReverseTransferInput) (*domain.Transfer, error) {
	return s.reverseFn(ctx, input)
}

func TestTransferHandler_Create_Success(t *testing.T) {
	transfer := &domain.Transfer{ID: "tx-1", Amount: decimal.NewFromInt(100)}
	var captured usecase.CreateTransferInput

	handler := NewTransferHandler(&transferServiceStub{
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
	})

	body, _ := json.Marshal(dto.CreateTransferRequest{
		FromAccountID: "acc-1",
		ToAccountID:   "acc-2",
		Amount:        "100",
	})

	req := httptest.NewRequest(http.MethodPost, "/transfers", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	if captured.FromAccountID != "acc-1" || captured.ToAccountID != "acc-2" {
		t.Fatalf("expected input to match request, got %+v", captured)
	}

	var resp dto.TransferResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ID != "tx-1" {
		t.Fatalf("expected transfer ID tx-1, got %s", resp.ID)
	}
}

func TestTransferHandler_Create_InvalidBody(t *testing.T) {
	handler := NewTransferHandler(&transferServiceStub{
		createFn: func(ctx context.Context, input usecase.CreateTransferInput) (*domain.Transfer, error) {
			t.Fatal("CreateTransfer should not be called")
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
	})

	req := httptest.NewRequest(http.MethodPost, "/transfers", bytes.NewBufferString("{bad json"))
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestTransferHandler_Create_InvalidAmount(t *testing.T) {
	handler := NewTransferHandler(&transferServiceStub{
		createFn: func(ctx context.Context, input usecase.CreateTransferInput) (*domain.Transfer, error) {
			t.Fatal("CreateTransfer should not be called on invalid amount")
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
	})

	body, _ := json.Marshal(dto.CreateTransferRequest{FromAccountID: "acc", ToAccountID: "acc2", Amount: "abc"})
	req := httptest.NewRequest(http.MethodPost, "/transfers", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestTransferHandler_Create_ServiceError(t *testing.T) {
	handler := NewTransferHandler(&transferServiceStub{
		createFn: func(ctx context.Context, input usecase.CreateTransferInput) (*domain.Transfer, error) {
			return nil, domain.ErrTransferNotFound
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
	})

	body, _ := json.Marshal(dto.CreateTransferRequest{FromAccountID: "acc", ToAccountID: "acc2", Amount: "10"})
	req := httptest.NewRequest(http.MethodPost, "/transfers", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestTransferHandler_Get(t *testing.T) {
	handler := NewTransferHandler(&transferServiceStub{
		getFn: func(ctx context.Context, id string) (*domain.Transfer, error) {
			return &domain.Transfer{ID: id}, nil
		},
		createFn: func(ctx context.Context, input usecase.CreateTransferInput) (*domain.Transfer, error) {
			return nil, nil
		},
		createBatchFn: func(ctx context.Context, input usecase.CreateBatchTransferInput) ([]*domain.Transfer, error) {
			return nil, nil
		},
		listFn: func(ctx context.Context, input usecase.ListTransfersByAccountInput) ([]*domain.Transfer, error) {
			return nil, nil
		},
		reverseFn: func(ctx context.Context, input usecase.ReverseTransferInput) (*domain.Transfer, error) {
			return nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/transfers/tx-1", nil)
	req = setChiURLParam(req, "id", "tx-1")
	rec := httptest.NewRecorder()

	handler.Get(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestTransferHandler_ListByAccount(t *testing.T) {
	handler := NewTransferHandler(&transferServiceStub{
		listFn: func(ctx context.Context, input usecase.ListTransfersByAccountInput) ([]*domain.Transfer, error) {
			if input.AccountID != "acc-1" || input.Limit != 5 || input.Offset != 1 {
				t.Fatalf("unexpected input %+v", input)
			}
			return []*domain.Transfer{{ID: "tx-1"}}, nil
		},
		createFn: func(ctx context.Context, input usecase.CreateTransferInput) (*domain.Transfer, error) {
			return nil, nil
		},
		createBatchFn: func(ctx context.Context, input usecase.CreateBatchTransferInput) ([]*domain.Transfer, error) {
			return nil, nil
		},
		getFn: func(ctx context.Context, id string) (*domain.Transfer, error) { return nil, nil },
		reverseFn: func(ctx context.Context, input usecase.ReverseTransferInput) (*domain.Transfer, error) {
			return nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/accounts/acc-1/transfers?limit=5&offset=1", nil)
	req = setChiURLParam(req, "id", "acc-1")
	rec := httptest.NewRecorder()

	handler.ListByAccount(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestTransferHandler_Reverse_Error(t *testing.T) {
	handler := NewTransferHandler(&transferServiceStub{
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
	})

	body, _ := json.Marshal(dto.ReverseTransferRequest{Metadata: map[string]any{"reason": "fraud"}})
	req := httptest.NewRequest(http.MethodPost, "/transfers/tx-1/reverse", bytes.NewReader(body))
	req = setChiURLParam(req, "id", "tx-1")
	rec := httptest.NewRecorder()

	handler.Reverse(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
