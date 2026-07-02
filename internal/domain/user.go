package domain

import (
	"context"
	"errors"
	"time"
)

// User represents a system user
type User struct {
	ID             string
	Email          string
	Name           string
	HashedPassword string // bcrypt hashed password
	Role           Role
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Active         bool
}

// contextKey is the type for context keys
type contextKey string

const (
	// UserContextKey is the context key for the authenticated user
	UserContextKey contextKey = "user"

	// RequestMetaContextKey is the context key for per-request metadata
	// (request ID, client IP, user agent) used for audit trail attribution.
	RequestMetaContextKey contextKey = "requestMeta"
)

// UserFromContext extracts the authenticated user from context
func UserFromContext(ctx context.Context) (*User, bool) {
	user, ok := ctx.Value(UserContextKey).(*User)
	return user, ok
}

// RequestMeta carries request-scoped attribution data (request ID, client
// IP, user agent) from the transport layer down to use cases, so audit rows
// can be traced back to the originating HTTP/gRPC request.
type RequestMeta struct {
	RequestID string
	IPAddress string
	UserAgent string
}

// RequestMetaFromContext extracts request metadata from context.
func RequestMetaFromContext(ctx context.Context) (RequestMeta, bool) {
	meta, ok := ctx.Value(RequestMetaContextKey).(RequestMeta)
	return meta, ok
}

// ContextWithRequestMeta returns a new context carrying the given request metadata.
func ContextWithRequestMeta(ctx context.Context, meta RequestMeta) context.Context {
	return context.WithValue(ctx, RequestMetaContextKey, meta)
}

// Role represents a user's access level
type Role string

const (
	// RoleAdmin has full access to all operations
	RoleAdmin Role = "admin"

	// RoleOperator can create and view transfers, but cannot manage accounts
	RoleOperator Role = "operator"

	// RoleViewer can only view resources, no mutations
	RoleViewer Role = "viewer"
)

// Valid roles
var validRoles = map[Role]bool{
	RoleAdmin:    true,
	RoleOperator: true,
	RoleViewer:   true,
}

// IsValid checks if the role is a valid role
func (r Role) IsValid() bool {
	return validRoles[r]
}

// CanCreate checks if the role can create resources
func (r Role) CanCreate() bool {
	return r == RoleAdmin || r == RoleOperator
}

// CanDelete checks if the role can delete resources
func (r Role) CanDelete() bool {
	return r == RoleAdmin
}

// CanManageAccounts checks if the role can manage accounts
func (r Role) CanManageAccounts() bool {
	return r == RoleAdmin
}

// CanViewAll checks if the role can view all resources
func (r Role) CanViewAll() bool {
	// All authenticated users can view
	return r.IsValid()
}

// Authentication errors
var (
	ErrUnauthorized     = errors.New("unauthorized")
	ErrInvalidToken     = errors.New("invalid token")
	ErrExpiredToken     = errors.New("token has expired")
	ErrInsufficientRole = errors.New("insufficient role for this operation")
)
