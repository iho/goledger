package handler

import (
	"encoding/json"
	"net/http"

	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/infrastructure/auth"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	jwtManager *auth.JWTManager
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(jwtManager *auth.JWTManager) *AuthHandler {
	return &AuthHandler{
		jwtManager: jwtManager,
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token string   `json:"token"`
	User  UserInfo `json:"user"`
}

// UserInfo represents user information
type UserInfo struct {
	ID    string      `json:"id"`
	Email string      `json:"email"`
	Role  domain.Role `json:"role"`
}

// Login handles user login (simplified - no password hashing for demo)
// In production, this would validate against a user database with hashed passwords
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	// DEMO ONLY: Hardcoded users for testing
	// In production, validate against database with bcrypt password hashing
	var user *domain.User
	switch req.Email {
	case "admin@goledger.io":
		if req.Password != "admin123" { // DEMO ONLY - never hardcode passwords!
			writeError(w, http.StatusUnauthorized, "invalid credentials", "")
			return
		}
		user = &domain.User{
			ID:     "user-admin-1",
			Email:  "admin@goledger.io",
			Name:   "Admin User",
			Role:   domain.RoleAdmin,
			Active: true,
		}
	case "operator@goledger.io":
		if req.Password != "operator123" {
			writeError(w, http.StatusUnauthorized, "invalid credentials", "")
			return
		}
		user = &domain.User{
			ID:     "user-operator-1",
			Email:  "operator@goledger.io",
			Name:   "Operator User",
			Role:   domain.RoleOperator,
			Active: true,
		}
	case "viewer@goledger.io":
		if req.Password != "viewer123" {
			writeError(w, http.StatusUnauthorized, "invalid credentials", "")
			return
		}
		user = &domain.User{
			ID:     "user-viewer-1",
			Email:  "viewer@goledger.io",
			Name:   "Viewer User",
			Role:   domain.RoleViewer,
			Active: true,
		}
	default:
		writeError(w, http.StatusUnauthorized, "invalid credentials", "")
		return
	}

	// Generate JWT token
	token, err := h.jwtManager.Generate(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token", err.Error())
		return
	}

	// Return token and user info
	response := LoginResponse{
		Token: token,
		User: UserInfo{
			ID:    user.ID,
			Email: user.Email,
			Role:  user.Role,
		},
	}

	writeJSON(w, http.StatusOK, response)
}

// GetCurrentUser returns the current authenticated user
func (h *AuthHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value("user").(*domain.User)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "")
		return
	}

	userInfo := UserInfo{
		ID:    user.ID,
		Email: user.Email,
		Role:  user.Role,
	}

	writeJSON(w, http.StatusOK, userInfo)
}
