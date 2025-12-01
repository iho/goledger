package domain

import "errors"

var (
	// Account errors
	ErrNegativeBalanceNotAllowed = errors.New("account does not allow negative balance")
	ErrPositiveBalanceNotAllowed = errors.New("account does not allow positive balance")
	ErrAccountNotFound           = errors.New("account not found")

	// Transfer errors
	ErrSameAccount      = errors.New("cannot transfer to same account")
	ErrInvalidAmount    = errors.New("amount must be positive")
	ErrCurrencyMismatch = errors.New("cannot transfer between different currencies")
	ErrTransferNotFound = errors.New("transfer not found")
)
