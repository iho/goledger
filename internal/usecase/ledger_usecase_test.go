package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
)

func TestLedgerUseCase_CheckConsistency(t *testing.T) {
	tests := []struct {
		name        string
		repo        *fakeLedgerRepository
		want        bool
		expectedErr error
	}{
		{
			name: "happy path balanced ledger",
			repo: &fakeLedgerRepository{
				totalBalance: decimal.Zero,
				totalAmount:  decimal.Zero,
			},
			want: true,
		},
		{
			name: "repo error surfaces",
			repo: &fakeLedgerRepository{
				err: errors.New("db down"),
			},
			want:        false,
			expectedErr: errors.New("db down"),
		},
		{
			name: "non-zero balance",
			repo: &fakeLedgerRepository{
				totalBalance: decimal.NewFromInt(10),
				totalAmount:  decimal.Zero,
			},
			want:        false,
			expectedErr: ErrInconsistentLedger,
		},
		{
			name: "non-zero amount",
			repo: &fakeLedgerRepository{
				totalBalance: decimal.Zero,
				totalAmount:  decimal.NewFromInt(1),
			},
			want:        false,
			expectedErr: ErrInconsistentLedger,
		},
		{
			name: "both balance and amount non-zero",
			repo: &fakeLedgerRepository{
				totalBalance: decimal.NewFromInt(1),
				totalAmount:  decimal.NewFromInt(-1),
			},
			want:        false,
			expectedErr: ErrInconsistentLedger,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewLedgerUseCase(tt.repo)
			got, err := uc.CheckConsistency(context.Background())

			if tt.expectedErr != nil {
				if err == nil || err.Error() != tt.expectedErr.Error() {
					t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("CheckConsistency() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLedgerUseCase_RepositoryInvoked(t *testing.T) {
	repo := &fakeLedgerRepository{}
	uc := NewLedgerUseCase(repo)

	if _, err := uc.CheckConsistency(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.calls != 1 {
		t.Fatalf("expected CheckConsistency to call repository once, got %d", repo.calls)
	}
}

type fakeLedgerRepository struct {
	totalBalance decimal.Decimal
	totalAmount  decimal.Decimal
	err          error
	calls        int
}

func (f *fakeLedgerRepository) CheckConsistency(ctx context.Context) (decimal.Decimal, decimal.Decimal, error) {
	f.calls++
	return f.totalBalance, f.totalAmount, f.err
}
