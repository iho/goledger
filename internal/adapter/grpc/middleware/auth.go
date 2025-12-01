package middleware

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/infrastructure/auth"
)

// ContextKey is the type for context keys
type ContextKey string

const (
	// UserContextKey is the context key for the authenticated user
	UserContextKey ContextKey = "user"

	// AuthorizationHeader is the metadata key for authorization
	AuthorizationHeader = "authorization"
)

// AuthInterceptor creates a gRPC authentication interceptor
func AuthInterceptor(jwtManager *auth.JWTManager) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		// Extract metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		// Extract authorization token
		values := md.Get(AuthorizationHeader)
		if len(values) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization token")
		}

		accessToken := values[0]

		// Remove "Bearer " prefix if present
		if strings.HasPrefix(accessToken, "Bearer ") {
			accessToken = strings.TrimPrefix(accessToken, "Bearer ")
		}

		// Verify token
		claims, err := jwtManager.Verify(accessToken)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
		}

		// Create user from claims
		user := &domain.User{
			ID:    claims.UserID,
			Email: claims.Email,
			Role:  claims.Role,
		}

		// Add user to context
		ctx = context.WithValue(ctx, UserContextKey, user)

		// Call the handler with the new context
		return handler(ctx, req)
	}
}

// RequireRole creates a gRPC interceptor that checks for a specific role
func RequireRoleInterceptor(minRole domain.Role) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		user, ok := GetUserFromContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "unauthorized")
		}

		// Check role permissions
		switch minRole {
		case domain.RoleAdmin:
			if user.Role != domain.RoleAdmin {
				return nil, status.Error(codes.PermissionDenied, "insufficient permissions")
			}
		case domain.RoleOperator:
			if user.Role != domain.RoleAdmin && user.Role != domain.RoleOperator {
				return nil, status.Error(codes.PermissionDenied, "insufficient permissions")
			}
		case domain.RoleViewer:
			// All authenticated users can view
		}

		return handler(ctx, req)
	}
}

// GetUserFromContext extracts the authenticated user from context
func GetUserFromContext(ctx context.Context) (*domain.User, bool) {
	user, ok := ctx.Value(UserContextKey).(*domain.User)
	return user, ok
}

// ChainUnaryServer chains multiple unary interceptors
func ChainUnaryServer(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		chain := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			interceptor := interceptors[i]
			next := chain
			chain = func(ctx context.Context, req interface{}) (interface{}, error) {
				return interceptor(ctx, req, info, next)
			}
		}
		return chain(ctx, req)
	}
}
