package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// Account represents a ledger account that can hold a balance.
type Account struct {
	ID                   string
	Name                 string
	Currency             string
	Balance              decimal.Decimal
	Version              int64
	AllowNegativeBalance bool
	AllowPositiveBalance bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// ValidateDebit checks if account can be debited by amount.
func (a *Account) ValidateDebit(amount decimal.Decimal) error {
	newBalance := a.Balance.Sub(amount)
	if !a.AllowNegativeBalance && newBalance.IsNegative() {
		return ErrNegativeBalanceNotAllowed
	}
	return nil
}

// ValidateCredit checks if account can be credited by amount.
func (a *Account) ValidateCredit(amount decimal.Decimal) error {
	newBalance := a.Balance.Add(amount)
	if !a.AllowPositiveBalance && newBalance.IsPositive() {
		return ErrPositiveBalanceNotAllowed
	}
	return nil
}

// ApplyDebit returns new balance after debit.
func (a *Account) ApplyDebit(amount decimal.Decimal) decimal.Decimal {
	return a.Balance.Sub(amount)
}

// ApplyCredit returns new balance after credit.
func (a *Account) ApplyCredit(amount decimal.Decimal) decimal.Decimal {
	return a.Balance.Add(amount)
}
