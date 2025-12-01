package errors

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/iho/goledger/internal/domain"
)

// MapDomainError converts domain errors to appropriate gRPC status codes
// This prevents internal error details from being exposed to clients
func MapDomainError(err error) error {
	if err == nil {
		return nil
	}

	// Map known domain errors to specific gRPC codes
	switch {
	// Not Found errors
	case errors.Is(err, domain.ErrAccountNotFound):
		return status.Error(codes.NotFound, "account not found")
	case errors.Is(err, domain.ErrTransferNotFound):
		return status.Error(codes.NotFound, "transfer not found")
	case errors.Is(err, domain.ErrHoldNotFound):
		return status.Error(codes.NotFound, "hold not found")

	// Invalid Argument errors
	case errors.Is(err, domain.ErrInvalidAmount):
		return status.Error(codes.InvalidArgument, "invalid amount: must be positive")
	case errors.Is(err, domain.ErrSameAccount):
		return status.Error(codes.InvalidArgument, "cannot transfer to the same account")
	case errors.Is(err, domain.ErrCurrencyMismatch):
		return status.Error(codes.InvalidArgument, "currency mismatch between accounts")

	// Precondition Failed errors (business logic violations)
	case errors.Is(err, domain.ErrNegativeBalanceNotAllowed):
		return status.Error(codes.FailedPrecondition, "operation would result in negative balance")
	case errors.Is(err, domain.ErrPositiveBalanceNotAllowed):
		return status.Error(codes.FailedPrecondition, "operation would result in positive balance")
	case errors.Is(err, domain.ErrInsufficientFunds):
		return status.Error(codes.FailedPrecondition, "insufficient funds")
	case errors.Is(err, domain.ErrHoldNotActive):
		return status.Error(codes.FailedPrecondition, "hold is not active")

	// Transfer-specific errors
	case errors.Is(err, domain.ErrTransferAlreadyReversed):
		return status.Error(codes.FailedPrecondition, "transfer has already been reversed")

	// Context errors (timeouts, cancellations)
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "operation timed out")
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "operation was canceled")

	// Default: Internal error (don't expose details)
	default:
		// Log the actual error for debugging (you'd want to add logging here)
		// log.Error("internal error", "error", err)
		return status.Error(codes.Internal, "an internal error occurred")
	}
}

// MapError is an alias for MapDomainError for backwards compatibility
func MapError(err error) error {
	return MapDomainError(err)
}
