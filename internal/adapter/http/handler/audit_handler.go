package handler

import (
	"context"
	"encoding/csv"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/iho/goledger/internal/adapter/http/dto"
	"github.com/iho/goledger/internal/domain"
)

// AuditService defines the behavior needed by AuditHandler.
type AuditService interface {
	List(ctx context.Context, filter domain.AuditFilter) ([]*domain.AuditLog, error)
	GetByResourceID(ctx context.Context, resourceType, resourceID string) ([]*domain.AuditLog, error)
}

// AuditHandler serves admin-only audit trail reads for examiners.
type AuditHandler struct {
	auditRepo AuditService
}

// NewAuditHandler creates a new AuditHandler.
func NewAuditHandler(auditRepo AuditService) *AuditHandler {
	return &AuditHandler{auditRepo: auditRepo}
}

// List returns audit logs filtered by query parameters: user_id, action,
// resource_type, resource_id, start_date, end_date (RFC3339), limit, offset.
func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	filter, err := filterFromQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid filter", err.Error())
		return
	}

	logs, err := h.auditRepo.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list audit logs", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, dto.AuditLogsFromDomain(logs))
}

// Export returns audit logs matching the same filters as List, as CSV.
func (h *AuditHandler) Export(w http.ResponseWriter, r *http.Request) {
	filter, err := filterFromQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid filter", err.Error())
		return
	}

	if filter.Limit <= 0 {
		filter.Limit = 10000
	}

	logs, err := h.auditRepo.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list audit logs", err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="audit_logs.csv"`)
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{
		"id", "created_at", "user_id", "action", "resource_type", "resource_id",
		"status", "ip_address", "user_agent", "request_id", "error_message",
	})
	for _, l := range logs {
		_ = cw.Write([]string{
			l.ID, l.CreatedAt.Format(time.RFC3339), l.UserID, l.Action, l.ResourceType, l.ResourceID,
			l.Status, l.IPAddress, l.UserAgent, l.RequestID, l.ErrorMessage,
		})
	}
	cw.Flush()
}

// GetByResource returns the audit trail for a single resource, e.g. one transfer.
func (h *AuditHandler) GetByResource(w http.ResponseWriter, r *http.Request) {
	resourceType := chi.URLParam(r, "type")
	resourceID := chi.URLParam(r, "id")
	if resourceType == "" || resourceID == "" {
		writeError(w, http.StatusBadRequest, "missing resource type or id", "")
		return
	}

	logs, err := h.auditRepo.GetByResourceID(r.Context(), resourceType, resourceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get audit logs", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, dto.AuditLogsFromDomain(logs))
}

// GetByUser returns the audit trail of actions performed by a single user.
func (h *AuditHandler) GetByUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "missing user id", "")
		return
	}

	filter := domain.AuditFilter{
		UserID: userID,
		Limit:  parseIntQuery(r, "limit", 100),
		Offset: parseIntQuery(r, "offset", 0),
	}

	logs, err := h.auditRepo.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get audit logs", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, dto.AuditLogsFromDomain(logs))
}

func filterFromQuery(r *http.Request) (domain.AuditFilter, error) {
	q := r.URL.Query()
	filter := domain.AuditFilter{
		UserID:       q.Get("user_id"),
		Action:       q.Get("action"),
		ResourceType: q.Get("resource_type"),
		ResourceID:   q.Get("resource_id"),
		Limit:        parseIntQuery(r, "limit", 100),
		Offset:       parseIntQuery(r, "offset", 0),
	}

	if v := q.Get("start_date"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return filter, err
		}
		filter.StartDate = &t
	}

	if v := q.Get("end_date"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return filter, err
		}
		filter.EndDate = &t
	}

	return filter, nil
}
