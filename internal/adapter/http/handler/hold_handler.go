package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"github.com/iho/goledger/internal/adapter/http/dto"
	"github.com/iho/goledger/internal/usecase"
)

type HoldHandler struct {
	holdUC *usecase.HoldUseCase
}

func NewHoldHandler(holdUC *usecase.HoldUseCase) *HoldHandler {
	return &HoldHandler{holdUC: holdUC}
}

func (h *HoldHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateHoldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid amount", err.Error())
		return
	}

	hold, err := h.holdUC.HoldFunds(r.Context(), req.AccountID, amount)
	if err != nil {
		writeError(w, mapDomainError(err), "failed to create hold", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, dto.HoldFromDomain(hold))
}

func (h *HoldHandler) Void(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing hold id", "")
		return
	}

	if err := h.holdUC.VoidHold(r.Context(), id); err != nil {
		writeError(w, mapDomainError(err), "failed to void hold", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *HoldHandler) Capture(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing hold id", "")
		return
	}

	var req dto.CaptureHoldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	transfer, err := h.holdUC.CaptureHold(r.Context(), id, req.ToAccountID)
	if err != nil {
		writeError(w, mapDomainError(err), "failed to capture hold", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, dto.TransferFromDomain(transfer))
}
