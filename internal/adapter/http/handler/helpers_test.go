package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/iho/goledger/internal/adapter/http/dto"
	"github.com/iho/goledger/internal/domain"
)

func TestParseIntQuery(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/accounts?limit=50", nil)
	if got := parseIntQuery(req, "limit", 10); got != 50 {
		t.Fatalf("expected limit=50, got %d", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/accounts?limit=invalid", nil)
	if got := parseIntQuery(req, "limit", 10); got != 10 {
		t.Fatalf("expected fallback to default, got %d", got)
	}

	req.URL = &url.URL{RawQuery: ""}
	if got := parseIntQuery(req, "limit", 25); got != 25 {
		t.Fatalf("expected default when missing, got %d", got)
	}
}

func TestMapDomainError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{"account not found", domain.ErrAccountNotFound, http.StatusNotFound},
		{"transfer not found", domain.ErrTransferNotFound, http.StatusNotFound},
		{"negative balance", domain.ErrNegativeBalanceNotAllowed, http.StatusBadRequest},
		{"invalid amount", domain.ErrInvalidAmount, http.StatusBadRequest},
		{"currency mismatch", domain.ErrCurrencyMismatch, http.StatusBadRequest},
		{"unknown error", errors.New("boom"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := mapDomainError(tt.err); got != tt.expected {
				t.Fatalf("expected %d, got %d", tt.expected, got)
			}
		})
	}
}

func TestWriteJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	payload := map[string]string{"status": "ok"}

	writeJSON(rr, http.StatusCreated, payload)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rr.Code)
	}

	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected content-type application/json, got %s", ct)
	}

	var decoded map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if decoded["status"] != "ok" {
		t.Fatalf("expected payload to round-trip, got %+v", decoded)
	}
}

func TestWriteError(t *testing.T) {
	rr := httptest.NewRecorder()

	writeError(rr, http.StatusBadRequest, "bad request", "detail")

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}

	var resp dto.ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if resp.Error != "bad request" {
		t.Fatalf("expected error message to propagate, got %+v", resp)
	}
}
