package errors_test

import (
	"context"
	stdErrors "errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	grpcerrors "github.com/iho/goledger/internal/adapter/grpc/errors"
	"github.com/iho/goledger/internal/domain"
)

func TestMapDomainError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     error
		want    codes.Code
		wantMsg string
	}{
		{"nil error", nil, codes.OK, ""},
		{"account not found", domain.ErrAccountNotFound, codes.NotFound, "account not found"},
		{"transfer not found", domain.ErrTransferNotFound, codes.NotFound, "transfer not found"},
		{"hold not found", domain.ErrHoldNotFound, codes.NotFound, "hold not found"},
		{"invalid amount", domain.ErrInvalidAmount, codes.InvalidArgument, "invalid amount: must be positive"},
		{"same account", domain.ErrSameAccount, codes.InvalidArgument, "cannot transfer to the same account"},
		{"currency mismatch", domain.ErrCurrencyMismatch, codes.InvalidArgument, "currency mismatch between accounts"},
		{"negative balance", domain.ErrNegativeBalanceNotAllowed, codes.FailedPrecondition, "operation would result in negative balance"},
		{"positive balance", domain.ErrPositiveBalanceNotAllowed, codes.FailedPrecondition, "operation would result in positive balance"},
		{"insufficient funds", domain.ErrInsufficientFunds, codes.FailedPrecondition, "insufficient funds"},
		{"hold not active", domain.ErrHoldNotActive, codes.FailedPrecondition, "hold is not active"},
		{"transfer already reversed", domain.ErrTransferAlreadyReversed, codes.FailedPrecondition, "transfer has already been reversed"},
		{"deadline exceeded", context.DeadlineExceeded, codes.DeadlineExceeded, "operation timed out"},
		{"canceled", context.Canceled, codes.Canceled, "operation was canceled"},
		{"unknown error", stdErrors.New("boom"), codes.Internal, "an internal error occurred"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := grpcerrors.MapDomainError(tt.err)

			if tt.err == nil && got != nil {
				t.Fatalf("expected nil, got %v", got)
			}

			if tt.err == nil {
				return
			}

			st, ok := status.FromError(got)
			if !ok {
				t.Fatalf("expected gRPC status error, got %v", got)
			}

			if st.Code() != tt.want {
				t.Fatalf("expected code %s, got %s", tt.want, st.Code())
			}

			if st.Message() != tt.wantMsg {
				t.Fatalf("expected message %q, got %q", tt.wantMsg, st.Message())
			}
		})
	}
}

func TestMapErrorAlias(t *testing.T) {
	err := domain.ErrAccountNotFound

	if grpcerrors.MapError(err).Error() != grpcerrors.MapDomainError(err).Error() {
		t.Fatalf("expected MapError to match MapDomainError")
	}
}
