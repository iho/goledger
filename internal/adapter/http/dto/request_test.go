package dto

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/iho/goledger/internal/usecase"
)

func TestCreateAccountRequest_ToUseCaseInput(t *testing.T) {
	req := &CreateAccountRequest{
		Name:                 "Main",
		Currency:             "USD",
		AllowNegativeBalance: true,
		AllowPositiveBalance: false,
	}

	got := req.ToUseCaseInput()
	want := usecase.CreateAccountInput{
		Name:                 "Main",
		Currency:             "USD",
		AllowNegativeBalance: true,
		AllowPositiveBalance: false,
	}

	if got != want {
		t.Fatalf("ToUseCaseInput() = %+v, want %+v", got, want)
	}
}

func TestCreateTransferRequest_ToUseCaseInput(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		request     *CreateTransferRequest
		want        usecase.CreateTransferInput
		expectError bool
	}{
		{
			name: "valid amount",
			request: &CreateTransferRequest{
				EventAt:       &now,
				Metadata:      map[string]any{"foo": "bar"},
				FromAccountID: "from",
				ToAccountID:   "to",
				Amount:        "12.34",
			},
			want: usecase.CreateTransferInput{
				EventAt:       &now,
				Metadata:      map[string]any{"foo": "bar"},
				FromAccountID: "from",
				ToAccountID:   "to",
				Amount:        decimal.RequireFromString("12.34"),
			},
		},
		{
			name: "invalid amount",
			request: &CreateTransferRequest{
				Amount: "bad",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.request.ToUseCaseInput()

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !transferInputEqual(got, tt.want) {
				t.Fatalf("ToUseCaseInput() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestCreateBatchTransferRequest_ToUseCaseInput(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		request     *CreateBatchTransferRequest
		want        usecase.CreateBatchTransferInput
		expectError bool
	}{
		{
			name: "valid batch",
			request: &CreateBatchTransferRequest{
				EventAt:  &now,
				Metadata: map[string]any{"batch": true},
				Transfers: []TransferItem{
					{FromAccountID: "A", ToAccountID: "B", Amount: "10"},
					{FromAccountID: "B", ToAccountID: "C", Amount: "5.5"},
				},
			},
			want: usecase.CreateBatchTransferInput{
				EventAt:  &now,
				Metadata: map[string]any{"batch": true},
				Transfers: []usecase.CreateTransferInput{
					{FromAccountID: "A", ToAccountID: "B", Amount: decimal.RequireFromString("10")},
					{FromAccountID: "B", ToAccountID: "C", Amount: decimal.RequireFromString("5.5")},
				},
			},
		},
		{
			name: "invalid amount in batch",
			request: &CreateBatchTransferRequest{
				Transfers: []TransferItem{{Amount: "bad"}},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.request.ToUseCaseInput()

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(got.Transfers) != len(tt.want.Transfers) || got.EventAt != tt.want.EventAt {
				t.Fatalf("unexpected output: %+v", got)
			}

			for i := range got.Transfers {
				if !transferInputEqual(got.Transfers[i], tt.want.Transfers[i]) {
					t.Fatalf("transfer %d = %+v, want %+v", i, got.Transfers[i], tt.want.Transfers[i])
				}
			}
		})
	}
}

func TestReverseTransferRequest_ToUseCaseInput(t *testing.T) {
	req := &ReverseTransferRequest{
		Metadata: map[string]any{"reason": "duplicate"},
	}

	want := usecase.ReverseTransferInput{
		TransferID: "transfer-1",
		Metadata:   map[string]any{"reason": "duplicate"},
	}

	if got := req.ToUseCaseInput("transfer-1"); got.TransferID != want.TransferID || got.Metadata["reason"] != "duplicate" {
		t.Fatalf("ToUseCaseInput() = %+v, want %+v", got, want)
	}
}

func transferInputEqual(a, b usecase.CreateTransferInput) bool {
	if a.EventAt != b.EventAt {
		return false
	}
	if a.FromAccountID != b.FromAccountID || a.ToAccountID != b.ToAccountID {
		return false
	}
	if !a.Amount.Equal(b.Amount) {
		return false
	}
	if len(a.Metadata) != len(b.Metadata) {
		return false
	}
	for k, v := range a.Metadata {
		if b.Metadata[k] != v {
			return false
		}
	}
	return true
}
