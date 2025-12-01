package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/iho/goledger/internal/adapter/http/dto"
	"github.com/iho/goledger/internal/domain"
)

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data) // Error handled by http.ResponseWriter
}

// writeError writes an error response.
func writeError(w http.ResponseWriter, status int, message string, detail string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(dto.ErrorResponse{
		Error: message,
	}) // Error handled by http.ResponseWriter
}

// mapDomainError maps domain errors to HTTP status codes.
func mapDomainError(err error) int {
	switch {
	case errors.Is(err, domain.ErrAccountNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrTransferNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrNegativeBalanceNotAllowed):
		return http.StatusBadRequest
	case errors.Is(err, domain.ErrPositiveBalanceNotAllowed):
		return http.StatusBadRequest
	case errors.Is(err, domain.ErrSameAccount):
		return http.StatusBadRequest
	case errors.Is(err, domain.ErrInvalidAmount):
		return http.StatusBadRequest
	case errors.Is(err, domain.ErrCurrencyMismatch):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// parseIntQuery parses an integer query parameter with a default value.
func parseIntQuery(r *http.Request, key string, defaultValue int) int {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultValue
	}

	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}

	return i
}
