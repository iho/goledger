package domain

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestTransfer_Validate(t *testing.T) {
	tests := []struct {
		name        string
		fromID      string
		toID        string
		amount      decimal.Decimal
		expectError error
	}{
		{
			name:        "valid transfer",
			fromID:      "account-1",
			toID:        "account-2",
			amount:      decimal.NewFromInt(100),
			expectError: nil,
		},
		{
			name:        "same account",
			fromID:      "account-1",
			toID:        "account-1",
			amount:      decimal.NewFromInt(100),
			expectError: ErrSameAccount,
		},
		{
			name:        "zero amount",
			fromID:      "account-1",
			toID:        "account-2",
			amount:      decimal.Zero,
			expectError: ErrInvalidAmount,
		},
		{
			name:        "negative amount",
			fromID:      "account-1",
			toID:        "account-2",
			amount:      decimal.NewFromInt(-100),
			expectError: ErrInvalidAmount,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transfer := &Transfer{
				FromAccountID: tt.fromID,
				ToAccountID:   tt.toID,
				Amount:        tt.amount,
			}

			err := transfer.Validate()

			if tt.expectError == nil && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.expectError != nil && err != tt.expectError {
				t.Errorf("expected error %v, got %v", tt.expectError, err)
			}
		})
	}
}
