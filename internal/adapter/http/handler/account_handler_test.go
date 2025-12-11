package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/iho/goledger/internal/adapter/http/dto"
	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/usecase"
)

type accountServiceStub struct {
	createFn func(ctx context.Context, input usecase.CreateAccountInput) (*domain.Account, error)
	getFn    func(ctx context.Context, id string) (*domain.Account, error)
	listFn   func(ctx context.Context, input usecase.ListAccountsInput) ([]*domain.Account, error)
}

func (s *accountServiceStub) CreateAccount(ctx context.Context, input usecase.CreateAccountInput) (*domain.Account, error) {
	return s.createFn(ctx, input)
}

func (s *accountServiceStub) GetAccount(ctx context.Context, id string) (*domain.Account, error) {
	return s.getFn(ctx, id)
}

func (s *accountServiceStub) ListAccounts(ctx context.Context, input usecase.ListAccountsInput) ([]*domain.Account, error) {
	return s.listFn(ctx, input)
}

func TestAccountHandler_Create_Success(t *testing.T) {
	account := &domain.Account{
		ID:                   "acc-1",
		Name:                 "test",
		Currency:             "USD",
		AllowNegativeBalance: true,
	}

	var captured usecase.CreateAccountInput
	handler := NewAccountHandler(&accountServiceStub{
		createFn: func(ctx context.Context, input usecase.CreateAccountInput) (*domain.Account, error) {
			captured = input
			return account, nil
		},
		getFn:  func(ctx context.Context, id string) (*domain.Account, error) { return nil, nil },
		listFn: func(ctx context.Context, input usecase.ListAccountsInput) ([]*domain.Account, error) { return nil, nil },
	})

	body, _ := json.Marshal(dto.CreateAccountRequest{
		Name:                 "test",
		Currency:             "USD",
		AllowNegativeBalance: true,
	})

	req := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	if captured.Name != "test" || captured.Currency != "USD" || !captured.AllowNegativeBalance {
		t.Fatalf("expected input to match request, got %+v", captured)
	}

	var resp dto.AccountResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ID != "acc-1" {
		t.Fatalf("expected account ID acc-1, got %s", resp.ID)
	}
}

func TestAccountHandler_Create_InvalidJSON(t *testing.T) {
	handler := NewAccountHandler(&accountServiceStub{
		createFn: func(ctx context.Context, input usecase.CreateAccountInput) (*domain.Account, error) {
			t.Fatal("CreateAccount should not be called for invalid payload")
			return nil, nil
		},
		getFn:  func(ctx context.Context, id string) (*domain.Account, error) { return nil, nil },
		listFn: func(ctx context.Context, input usecase.ListAccountsInput) ([]*domain.Account, error) { return nil, nil },
	})

	req := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewBufferString("{invalid json"))
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestAccountHandler_Create_ServiceError(t *testing.T) {
	handler := NewAccountHandler(&accountServiceStub{
		createFn: func(ctx context.Context, input usecase.CreateAccountInput) (*domain.Account, error) {
			return nil, errors.New("db error")
		},
		getFn:  func(ctx context.Context, id string) (*domain.Account, error) { return nil, nil },
		listFn: func(ctx context.Context, input usecase.ListAccountsInput) ([]*domain.Account, error) { return nil, nil },
	})

	body, _ := json.Marshal(dto.CreateAccountRequest{Name: "test", Currency: "USD"})
	req := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestAccountHandler_Get(t *testing.T) {
	account := &domain.Account{ID: "acc-1", Name: "test"}
	handler := NewAccountHandler(&accountServiceStub{
		getFn: func(ctx context.Context, id string) (*domain.Account, error) {
			if id != "acc-1" {
				t.Fatalf("expected id acc-1, got %s", id)
			}
			return account, nil
		},
		createFn: func(ctx context.Context, input usecase.CreateAccountInput) (*domain.Account, error) { return nil, nil },
		listFn:   func(ctx context.Context, input usecase.ListAccountsInput) ([]*domain.Account, error) { return nil, nil },
	})

	req := httptest.NewRequest(http.MethodGet, "/accounts/acc-1", nil)
	req = req.WithContext(context.Background())
	req = setChiURLParam(req, "id", "acc-1")
	rec := httptest.NewRecorder()

	handler.Get(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestAccountHandler_Get_NotFound(t *testing.T) {
	handler := NewAccountHandler(&accountServiceStub{
		getFn: func(ctx context.Context, id string) (*domain.Account, error) {
			return nil, domain.ErrAccountNotFound
		},
		createFn: func(ctx context.Context, input usecase.CreateAccountInput) (*domain.Account, error) { return nil, nil },
		listFn:   func(ctx context.Context, input usecase.ListAccountsInput) ([]*domain.Account, error) { return nil, nil },
	})

	req := httptest.NewRequest(http.MethodGet, "/accounts/acc-1", nil)
	req = setChiURLParam(req, "id", "acc-1")
	rec := httptest.NewRecorder()

	handler.Get(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestAccountHandler_List(t *testing.T) {
	handler := NewAccountHandler(&accountServiceStub{
		listFn: func(ctx context.Context, input usecase.ListAccountsInput) ([]*domain.Account, error) {
			if input.Limit != 5 || input.Offset != 2 {
				t.Fatalf("expected limit=5 offset=2, got %+v", input)
			}
			return []*domain.Account{{ID: "acc-1"}, {ID: "acc-2"}}, nil
		},
		createFn: func(ctx context.Context, input usecase.CreateAccountInput) (*domain.Account, error) { return nil, nil },
		getFn:    func(ctx context.Context, id string) (*domain.Account, error) { return nil, nil },
	})

	req := httptest.NewRequest(http.MethodGet, "/accounts?limit=5&offset=2", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp dto.ListAccountsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Accounts) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(resp.Accounts))
	}
}

func setChiURLParam(r *http.Request, key, value string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, &chi.Context{
		URLParams: chi.RouteParams{
			Keys:   []string{key},
			Values: []string{value},
		},
	}))
}
