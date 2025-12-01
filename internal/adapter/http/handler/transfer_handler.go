package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/iho/goledger/internal/adapter/http/dto"
	"github.com/iho/goledger/internal/usecase"
)

// TransferHandler handles transfer-related HTTP requests.
type TransferHandler struct {
	transferUC *usecase.TransferUseCase
}

// NewTransferHandler creates a new TransferHandler.
func NewTransferHandler(transferUC *usecase.TransferUseCase) *TransferHandler {
	return &TransferHandler{transferUC: transferUC}
}

// Create creates a new transfer.
func (h *TransferHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	input, err := req.ToUseCaseInput()
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid amount", err.Error())
		return
	}

	transfer, err := h.transferUC.CreateTransfer(r.Context(), input)
	if err != nil {
		status := mapDomainError(err)
		writeError(w, status, "failed to create transfer", err.Error())

		return
	}

	writeJSON(w, http.StatusCreated, dto.TransferFromDomain(transfer))
}

// CreateBatch creates multiple transfers atomically.
func (h *TransferHandler) CreateBatch(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateBatchTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	input, err := req.ToUseCaseInput()
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid amount", err.Error())
		return
	}

	transfers, err := h.transferUC.CreateBatchTransfer(r.Context(), input)
	if err != nil {
		status := mapDomainError(err)
		writeError(w, status, "failed to create transfers", err.Error())

		return
	}

	writeJSON(w, http.StatusCreated, dto.TransfersFromDomain(transfers))
}

// Get retrieves a transfer by ID.
func (h *TransferHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing transfer ID", "")
		return
	}

	transfer, err := h.transferUC.GetTransfer(r.Context(), id)
	if err != nil {
		status := mapDomainError(err)
		writeError(w, status, "failed to get transfer", err.Error())

		return
	}

	writeJSON(w, http.StatusOK, dto.TransferFromDomain(transfer))
}

// ListByAccount lists transfers for an account.
func (h *TransferHandler) ListByAccount(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "id")
	if accountID == "" {
		writeError(w, http.StatusBadRequest, "missing account ID", "")
		return
	}

	limit := parseIntQuery(r, "limit", 20)
	offset := parseIntQuery(r, "offset", 0)

	transfers, err := h.transferUC.ListTransfersByAccount(r.Context(), usecase.ListTransfersByAccountInput{
		AccountID: accountID,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list transfers", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, dto.TransfersFromDomain(transfers))
}

// Reverse creates a reversal transfer.
func (h *TransferHandler) Reverse(w http.ResponseWriter, r *http.Request) {
	transferID := chi.URLParam(r, "id")
	if transferID == "" {
		writeError(w, http.StatusBadRequest, "missing transfer ID", "")
		return
	}

	var req dto.ReverseTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	input := req.ToUseCaseInput(transferID)
	reversalTransfer, err := h.transferUC.ReverseTransfer(r.Context(), input)
	if err != nil {
		status := mapDomainError(err)
		writeError(w, status, "failed to reverse transfer", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, dto.TransferFromDomain(reversalTransfer))
}
