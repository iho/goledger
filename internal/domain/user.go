package domain

import (
	"errors"
	"time"
)

// User represents a system user
type User struct {
	ID        string
	Email     string
	Name      string
	Role      Role
	CreatedAt time.Time
	UpdatedAt time.Time
	Active    bool
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
