package middleware_test

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/iho/goledger/internal/adapter/grpc/middleware"
	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/infrastructure/auth"
)

func TestAuthInterceptor(t *testing.T) {
	t.Parallel()

	jwtManager := auth.NewJWTManager("secret", time.Hour)
	interceptor := middleware.AuthInterceptor(jwtManager)

	t.Run("missing metadata", func(t *testing.T) {
		_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, req interface{}) (interface{}, error) {
			t.Fatal("handler should not be called")
			return nil, nil
		})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("expected unauthenticated, got %v", err)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer invalid"))
		_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, req interface{}) (interface{}, error) {
			t.Fatal("handler should not be called")
			return nil, nil
		})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("expected unauthenticated for invalid token, got %v", err)
		}
	})

	t.Run("valid token injects user", func(t *testing.T) {
		user := &domain.User{
			ID:    "user-1",
			Email: "user@example.com",
			Role:  domain.RoleAdmin,
		}
		token, err := jwtManager.Generate(user)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))

		called := false
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			called = true
			got, ok := middleware.GetUserFromContext(ctx)
			if !ok {
				t.Fatal("expected user in context")
			}
			if got.ID != user.ID || got.Role != user.Role {
				t.Fatalf("unexpected user %+v", got)
			}
			return "ok", nil
		}

		resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !called || resp != "ok" {
			t.Fatalf("expected handler to execute, resp=%v called=%v", resp, called)
		}
	})
}

func TestRequireRoleInterceptor(t *testing.T) {
	t.Parallel()

	adminCtx := context.WithValue(context.Background(), middleware.UserContextKey, &domain.User{
		ID:   "admin",
		Role: domain.RoleAdmin,
	})
	operatorCtx := context.WithValue(context.Background(), middleware.UserContextKey, &domain.User{
		ID:   "operator",
		Role: domain.RoleOperator,
	})

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "allowed", nil
	}

	t.Run("requires admin", func(t *testing.T) {
		interceptor := middleware.RequireRoleInterceptor(domain.RoleAdmin)

		if _, err := interceptor(adminCtx, nil, &grpc.UnaryServerInfo{}, handler); err != nil {
			t.Fatalf("admin should pass: %v", err)
		}

		_, err := interceptor(operatorCtx, nil, &grpc.UnaryServerInfo{}, handler)
		if status.Code(err) != codes.PermissionDenied {
			t.Fatalf("expected permission denied, got %v", err)
		}
	})

	t.Run("operator access", func(t *testing.T) {
		interceptor := middleware.RequireRoleInterceptor(domain.RoleOperator)
		if _, err := interceptor(operatorCtx, nil, &grpc.UnaryServerInfo{}, handler); err != nil {
			t.Fatalf("operator should pass: %v", err)
		}
	})

	t.Run("missing user", func(t *testing.T) {
		interceptor := middleware.RequireRoleInterceptor(domain.RoleViewer)
		_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler)
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("expected unauthenticated, got %v", err)
		}
	})
}

func TestChainUnaryServer(t *testing.T) {
	t.Parallel()

	var sequence []string

	first := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		sequence = append(sequence, "first")
		return handler(ctx, req)
	}

	second := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		sequence = append(sequence, "second")
		return handler(ctx, req)
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		sequence = append(sequence, "handler")
		return "done", nil
	}

	chain := middleware.ChainUnaryServer(first, second)
	resp, err := chain(context.Background(), nil, &grpc.UnaryServerInfo{}, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp != "done" {
		t.Fatalf("expected handler response, got %v", resp)
	}

	expected := []string{"first", "second", "handler"}
	if len(sequence) != len(expected) {
		t.Fatalf("unexpected sequence length: %v", sequence)
	}
	for i, step := range expected {
		if sequence[i] != step {
			t.Fatalf("expected %s at position %d, got %s", step, i, sequence[i])
		}
	}
}
