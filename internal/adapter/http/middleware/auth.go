package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/infrastructure/auth"
)

// ContextKey is the type for context keys
type ContextKey string

const (
	// UserContextKey is the context key for the authenticated user
	UserContextKey ContextKey = "user"
)

// AuthMiddleware creates an authentication middleware
func AuthMiddleware(jwtManager *auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing authorization header", http.StatusUnauthorized)
				return
			}

			// Parse Bearer token
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			// Verify token
			claims, err := jwtManager.Verify(tokenString)
			if err != nil {
				http.Error(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}

			// Create user from claims
			user := &domain.User{
				ID:    claims.UserID,
				Email: claims.Email,
				Role:  claims.Role,
			}

			// Add user to context
			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole creates a middleware that checks for a specific role
func RequireRole(minRole domain.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := r.Context().Value(UserContextKey).(*domain.User)
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// Check role permissions
			switch minRole {
			case domain.RoleAdmin:
				if user.Role != domain.RoleAdmin {
					http.Error(w, "insufficient permissions", http.StatusForbidden)
					return
				}
			case domain.RoleOperator:
				if user.Role != domain.RoleAdmin && user.Role != domain.RoleOperator {
					http.Error(w, "insufficient permissions", http.StatusForbidden)
					return
				}
			case domain.RoleViewer:
				// All authenticated users can view
			}

			next.ServeHTTP(w, r)
		})
	}
}

// OptionalAuth is a middleware that extracts user if present but doesn't require it
func OptionalAuth(jwtManager *auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// No auth provided, continue without user
				next.ServeHTTP(w, r)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				claims, err := jwtManager.Verify(parts[1])
				if err == nil {
					user := &domain.User{
						ID:    claims.UserID,
						Email: claims.Email,
						Role:  claims.Role,
					}
					ctx := context.WithValue(r.Context(), UserContextKey, user)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			// Invalid auth, but don't fail - just continue without user
			next.ServeHTTP(w, r)
		})
	}
}

// GetUserFromContext extracts the authenticated user from context
func GetUserFromContext(ctx context.Context) (*domain.User, bool) {
	user, ok := ctx.Value(UserContextKey).(*domain.User)
	return user, ok
}
