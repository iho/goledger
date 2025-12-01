package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/iho/goledger/internal/adapter/http/dto"
	"github.com/iho/goledger/internal/usecase"
)

// EntryHandler handles entry-related HTTP requests.
type EntryHandler struct {
	entryUC *usecase.EntryUseCase
}

// NewEntryHandler creates a new EntryHandler.
func NewEntryHandler(entryUC *usecase.EntryUseCase) *EntryHandler {
	return &EntryHandler{entryUC: entryUC}
}

// ListByAccount lists entries for an account.
func (h *EntryHandler) ListByAccount(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "id")
	if accountID == "" {
		writeError(w, http.StatusBadRequest, "missing account ID", "")
		return
	}

	limit := parseIntQuery(r, "limit", 20)
	offset := parseIntQuery(r, "offset", 0)

	entries, err := h.entryUC.GetEntriesByAccount(r.Context(), usecase.GetEntriesByAccountInput{
		AccountID: accountID,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list entries", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, dto.EntriesFromDomain(entries))
}

// ListByTransfer lists entries for a transfer.
func (h *EntryHandler) ListByTransfer(w http.ResponseWriter, r *http.Request) {
	transferID := chi.URLParam(r, "id")
	if transferID == "" {
		writeError(w, http.StatusBadRequest, "missing transfer ID", "")
		return
	}

	entries, err := h.entryUC.GetEntriesByTransfer(r.Context(), transferID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list entries", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, dto.EntriesFromDomain(entries))
}

// GetHistoricalBalance gets the balance at a specific time.
func (h *EntryHandler) GetHistoricalBalance(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "id")
	if accountID == "" {
		writeError(w, http.StatusBadRequest, "missing account ID", "")
		return
	}

	atStr := r.URL.Query().Get("at")
	if atStr == "" {
		writeError(w, http.StatusBadRequest, "missing 'at' parameter", "")
		return
	}

	at, err := time.Parse(time.RFC3339, atStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid 'at' format (use RFC3339)", err.Error())
		return
	}

	balance, err := h.entryUC.GetHistoricalBalance(r.Context(), accountID, at)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get historical balance", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"account_id": accountID,
		"at":         at,
		"balance":    balance,
	})
}
