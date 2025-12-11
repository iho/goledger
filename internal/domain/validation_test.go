package domain

import (
	"errors"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

func TestValidateAccountName(t *testing.T) {
	t.Parallel()

	t.Run("valid name", func(t *testing.T) {
		if err := ValidateAccountName("Revenue Account"); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("empty name rejected", func(t *testing.T) {
		err := ValidateAccountName("   ")
		if !errors.Is(err, ErrInvalidAccountName) {
			t.Fatalf("expected ErrInvalidAccountName, got %v", err)
		}
	})

	t.Run("name too long", func(t *testing.T) {
		tooLong := strings.Repeat("a", MaxAccountNameLength+1)
		err := ValidateAccountName(tooLong)
		if !errors.Is(err, ErrInvalidAccountName) {
			t.Fatalf("expected ErrInvalidAccountName, got %v", err)
		}
	})

	t.Run("name with dangerous tokens", func(t *testing.T) {
		err := ValidateAccountName("savings; DROP TABLE accounts;")
		if !errors.Is(err, ErrInvalidAccountName) {
			t.Fatalf("expected ErrInvalidAccountName, got %v", err)
		}
	})
}

func TestValidateCurrency(t *testing.T) {
	t.Parallel()

	if err := ValidateCurrency("usd"); err != nil {
		t.Fatalf("expected uppercase conversion to succeed, got %v", err)
	}

	if err := ValidateCurrency("XYZ"); !errors.Is(err, ErrInvalidCurrency) {
		t.Fatalf("expected ErrInvalidCurrency, got %v", err)
	}
}

func TestValidateAmount(t *testing.T) {
	t.Parallel()

	valid := decimal.NewFromFloat(100.25)
	if err := ValidateAmount(valid); err != nil {
		t.Fatalf("expected valid amount, got %v", err)
	}

	if err := ValidateAmount(decimal.Zero); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("expected ErrInvalidAmount for zero, got %v", err)
	}

	if err := ValidateAmount(decimal.NewFromFloat(0.001)); !errors.Is(err, ErrAmountTooSmall) {
		t.Fatalf("expected ErrAmountTooSmall, got %v", err)
	}

	huge := decimal.RequireFromString(MaxTransferAmount).Add(decimal.NewFromInt(1))
	if err := ValidateAmount(huge); !errors.Is(err, ErrAmountTooLarge) {
		t.Fatalf("expected ErrAmountTooLarge, got %v", err)
	}
}

func TestValidateMetadata(t *testing.T) {
	t.Parallel()

	if err := ValidateMetadata(nil); err != nil {
		t.Fatalf("expected nil metadata to be allowed, got %v", err)
	}

	valid := map[string]any{"key": "value", "count": 10}
	if err := ValidateMetadata(valid); err != nil {
		t.Fatalf("expected valid metadata, got %v", err)
	}

	oversized := map[string]any{
		"payload": strings.Repeat("x", MaxMetadataSize),
	}
	if err := ValidateMetadata(oversized); !errors.Is(err, ErrMetadataTooLarge) {
		t.Fatalf("expected ErrMetadataTooLarge, got %v", err)
	}
}

func TestValidateEmail(t *testing.T) {
	t.Parallel()

	if err := ValidateEmail("USER@example.com"); err != nil {
		t.Fatalf("expected valid email, got %v", err)
	}

	if err := ValidateEmail("invalid-email"); !errors.Is(err, ErrInvalidEmail) {
		t.Fatalf("expected ErrInvalidEmail, got %v", err)
	}
}

func TestValidatePassword(t *testing.T) {
	t.Parallel()

	if err := ValidatePassword("StrongPass1"); err != nil {
		t.Fatalf("expected valid password, got %v", err)
	}

	if err := ValidatePassword("short1A"); !errors.Is(err, ErrPasswordTooWeak) {
		t.Fatalf("expected ErrPasswordTooWeak for short password, got %v", err)
	}

	if err := ValidatePassword(strings.Repeat("A", MaxPasswordLength+1)); !errors.Is(err, ErrPasswordTooWeak) {
		t.Fatalf("expected ErrPasswordTooWeak for overly long password, got %v", err)
	}

	if err := ValidatePassword("alllowercase1"); !errors.Is(err, ErrPasswordTooWeak) {
		t.Fatalf("expected ErrPasswordTooWeak for missing upper case, got %v", err)
	}

	if err := ValidatePassword("ALLUPPERCASE1"); !errors.Is(err, ErrPasswordTooWeak) {
		t.Fatalf("expected ErrPasswordTooWeak for missing lower case, got %v", err)
	}

	if err := ValidatePassword("NoDigitsHere"); !errors.Is(err, ErrPasswordTooWeak) {
		t.Fatalf("expected ErrPasswordTooWeak for missing digits, got %v", err)
	}
}
