package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/infrastructure/auth"
	"github.com/iho/goledger/internal/usecase"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	jwtManager *auth.JWTManager
	userUC     UserAuthenticator
}

// UserAuthenticator interface for user authentication
type UserAuthenticator interface {
	Authenticate(ctx context.Context, input usecase.AuthenticateInput) (*domain.User, error)
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(jwtManager *auth.JWTManager, userUC UserAuthenticator) *AuthHandler {
	return &AuthHandler{
		jwtManager: jwtManager,
		userUC:     userUC,
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

// Login handles user login with real authentication
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	// Authenticate user using UserUseCase
	user, err := h.userUC.Authenticate(context.Background(), usecase.AuthenticateInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
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
