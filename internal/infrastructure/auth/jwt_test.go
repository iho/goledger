package auth_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/infrastructure/auth"
)

func TestJWTManagerGenerateAndVerify(t *testing.T) {
	t.Parallel()

	manager := auth.NewJWTManager("super-secret", time.Minute)

	user := &domain.User{
		ID:    "user-123",
		Email: "user@example.com",
		Role:  domain.RoleAdmin,
	}

	token, err := manager.Generate(user)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	claims, err := manager.Verify(token)
	if err != nil {
		t.Fatalf("expected token to verify, got %v", err)
	}

	if claims.UserID != user.ID || claims.Email != user.Email || claims.Role != user.Role {
		t.Fatalf("expected claims to match user, got %+v", claims)
	}
}

func TestJWTManagerVerifyErrors(t *testing.T) {
	t.Parallel()

	manager := auth.NewJWTManager("secret", time.Minute)

	user := &domain.User{
		ID:    "expired",
		Email: "expired@example.com",
		Role:  domain.RoleViewer,
	}

	expiredClaims := auth.Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Minute)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-2 * time.Minute)),
		},
	}

	expiredToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims).SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("failed to sign expired token: %v", err)
	}

	if _, err := manager.Verify(expiredToken); err != domain.ErrExpiredToken {
		t.Fatalf("expected ErrExpiredToken, got %v", err)
	}

	otherManager := auth.NewJWTManager("other-secret", time.Minute)
	if _, err := otherManager.Verify(expiredToken); err == nil || err == domain.ErrExpiredToken {
		t.Fatalf("expected invalid token error, got %v", err)
	}

	if _, err := manager.Verify("not-a-token"); err == nil {
		t.Fatalf("expected failure for malformed token")
	}
}
