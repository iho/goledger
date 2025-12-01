package domain

import (
	"encoding/json"
	"time"
)

// AuditLog represents an audit trail entry for compliance and debugging
type AuditLog struct {
	ID           string
	UserID       string // Who performed the action
	Action       string // What action (transfer.create, account.create, etc.)
	ResourceType string // Type of resource (transfer, account, hold)
	ResourceID   string // ID of the resource
	IPAddress    string // Client IP address
	UserAgent    string // Client user agent
	RequestID    string // Request ID for tracing
	BeforeState  JSON   // State before the action
	AfterState   JSON   // State after the action
	Status       string // success, failure, error
	ErrorMessage string // If status=error, the error message
	CreatedAt    time.Time
}

// JSON is a type alias for JSON data
type JSON map[string]any

// AuditAction represents different types of auditable actions
type AuditAction string

const (
	// Account actions
	AuditActionAccountCreate AuditAction = "account.create"
	AuditActionAccountUpdate AuditAction = "account.update"
	AuditActionAccountView   AuditAction = "account.view"

	// Transfer actions
	AuditActionTransferCreate  AuditAction = "transfer.create"
	AuditActionTransferReverse AuditAction = "transfer.reverse"
	AuditActionTransferView    AuditAction = "transfer.view"

	// Hold actions
	AuditActionHoldCreate  AuditAction = "hold.create"
	AuditActionHoldVoid    AuditAction = "hold.void"
	AuditActionHoldCapture AuditAction = "hold.capture"
	AuditActionHoldView    AuditAction = "hold.view"

	// Auth actions
	AuditActionUserLogin  AuditAction = "user.login"
	AuditActionUserLogout AuditAction = "user.logout"
)

// AuditStatus represents the status of an audited action
type AuditStatus string

const (
	AuditStatusSuccess AuditStatus = "success"
	AuditStatusFailure AuditStatus = "failure"
	AuditStatusError   AuditStatus = "error"
)

// MarshalState converts a domain object to JSON for audit logging
func MarshalState(v any) JSON {
	if v == nil {
		return nil
	}

	data, err := json.Marshal(v)
	if err != nil {
		return JSON{"error": "failed to marshal state"}
	}

	var result JSON
	if err := json.Unmarshal(data, &result); err != nil {
		return JSON{"error": "failed to unmarshal state"}
	}

	return result
}

// AuditRepository defines the interface for audit log persistence
type AuditRepository interface {
	Create(log *AuditLog) error
	List(filter AuditFilter) ([]*AuditLog, error)
	GetByResourceID(resourceType, resourceID string) ([]*AuditLog, error)
}

// AuditFilter defines filters for querying audit logs
type AuditFilter struct {
	UserID       string
	Action       string
	ResourceType string
	ResourceID   string
	StartDate    *time.Time
	EndDate      *time.Time
	Limit        int
	Offset       int
}
