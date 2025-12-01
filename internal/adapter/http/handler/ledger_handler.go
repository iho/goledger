package handler

import (
	"net/http"

	"github.com/iho/goledger/internal/usecase"
)

// LedgerHandler handles ledger-wide operations.
type LedgerHandler struct {
	ledgerUC *usecase.LedgerUseCase
}

// NewLedgerHandler creates a new LedgerHandler.
func NewLedgerHandler(ledgerUC *usecase.LedgerUseCase) *LedgerHandler {
	return &LedgerHandler{ledgerUC: ledgerUC}
}

// CheckConsistency checks if the ledger is consistent.
func (h *LedgerHandler) CheckConsistency(w http.ResponseWriter, r *http.Request) {
	consistent, err := h.ledgerUC.CheckConsistency(r.Context())
	if err != nil {
		if err == usecase.ErrInconsistentLedger {
			writeJSON(w, http.StatusConflict, map[string]any{
				"status":     "inconsistent",
				"consistent": false,
				"message":    err.Error(),
			})
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to check consistency", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":     "consistent",
		"consistent": consistent,
	})
}
