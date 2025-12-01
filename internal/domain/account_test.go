package domain

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestAccount_ValidateDebit(t *testing.T) {
	tests := []struct {
		name        string
		balance     decimal.Decimal
		debitAmount decimal.Decimal
		allowNeg    bool
		expectError bool
	}{
		{
			name:        "allow negative - debit more than balance",
			balance:     decimal.NewFromInt(100),
			allowNeg:    true,
			debitAmount: decimal.NewFromInt(150),
			expectError: false,
		},
		{
			name:        "disallow negative - debit more than balance",
			balance:     decimal.NewFromInt(100),
			allowNeg:    false,
			debitAmount: decimal.NewFromInt(150),
			expectError: true,
		},
		{
			name:        "disallow negative - debit exact balance",
			balance:     decimal.NewFromInt(100),
			allowNeg:    false,
			debitAmount: decimal.NewFromInt(100),
			expectError: false,
		},
		{
			name:        "disallow negative - debit less than balance",
			balance:     decimal.NewFromInt(100),
			allowNeg:    false,
			debitAmount: decimal.NewFromInt(50),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acc := &Account{
				Balance:              tt.balance,
				AllowNegativeBalance: tt.allowNeg,
			}

			err := acc.ValidateDebit(tt.debitAmount)

			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestAccount_ValidateCredit(t *testing.T) {
	tests := []struct {
		name         string
		balance      decimal.Decimal
		creditAmount decimal.Decimal
		allowPos     bool
		expectError  bool
	}{
		{
			name:         "allow positive - credit to positive balance",
			balance:      decimal.NewFromInt(0),
			allowPos:     true,
			creditAmount: decimal.NewFromInt(100),
			expectError:  false,
		},
		{
			name:         "disallow positive - credit to positive balance",
			balance:      decimal.NewFromInt(0),
			allowPos:     false,
			creditAmount: decimal.NewFromInt(100),
			expectError:  true,
		},
		{
			name:         "disallow positive - credit to zero",
			balance:      decimal.NewFromInt(-100),
			allowPos:     false,
			creditAmount: decimal.NewFromInt(100),
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acc := &Account{
				Balance:              tt.balance,
				AllowPositiveBalance: tt.allowPos,
			}

			err := acc.ValidateCredit(tt.creditAmount)

			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestAccount_ApplyDebit(t *testing.T) {
	acc := &Account{Balance: decimal.NewFromInt(100)}
	newBalance := acc.ApplyDebit(decimal.NewFromInt(30))

	expected := decimal.NewFromInt(70)
	if !newBalance.Equal(expected) {
		t.Errorf("expected balance %s, got %s", expected, newBalance)
	}
}

func TestAccount_ApplyCredit(t *testing.T) {
	acc := &Account{Balance: decimal.NewFromInt(100)}
	newBalance := acc.ApplyCredit(decimal.NewFromInt(30))

	expected := decimal.NewFromInt(130)
	if !newBalance.Equal(expected) {
		t.Errorf("expected balance %s, got %s", expected, newBalance)
	}
}
