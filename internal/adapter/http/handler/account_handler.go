package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/iho/goledger/internal/adapter/http/dto"
	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/usecase"
)

// AccountService defines the behavior needed by AccountHandler.
type AccountService interface {
	CreateAccount(ctx context.Context, input usecase.CreateAccountInput) (*domain.Account, error)
	GetAccount(ctx context.Context, id string) (*domain.Account, error)
	ListAccounts(ctx context.Context, input usecase.ListAccountsInput) ([]*domain.Account, error)
}

// AccountHandler handles account-related HTTP requests.
type AccountHandler struct {
	accountUC AccountService
}

// NewAccountHandler creates a new AccountHandler.
func NewAccountHandler(accountUC AccountService) *AccountHandler {
	return &AccountHandler{accountUC: accountUC}
}

// Create creates a new account.
func (h *AccountHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	account, err := h.accountUC.CreateAccount(r.Context(), req.ToUseCaseInput())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create account", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, dto.AccountFromDomain(account))
}

// Get retrieves an account by ID.
func (h *AccountHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing account ID", "")
		return
	}

	account, err := h.accountUC.GetAccount(r.Context(), id)
	if err != nil {
		status := mapDomainError(err)
		writeError(w, status, "failed to get account", err.Error())

		return
	}

	writeJSON(w, http.StatusOK, dto.AccountFromDomain(account))
}

// List lists accounts.
func (h *AccountHandler) List(w http.ResponseWriter, r *http.Request) {
	limit := parseIntQuery(r, "limit", 20)
	offset := parseIntQuery(r, "offset", 0)

	accounts, err := h.accountUC.ListAccounts(r.Context(), usecase.ListAccountsInput{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list accounts", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, dto.ListAccountsResponse{
		Accounts: dto.AccountsFromDomain(accounts),
		Total:    int64(len(accounts)),
	})
}
