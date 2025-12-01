package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/shopspring/decimal"
)

// Validation errors
var (
	ErrInvalidAccountName = errors.New("invalid account name")
	ErrInvalidCurrency    = errors.New("invalid currency code")
	ErrAmountTooLarge     = errors.New("amount exceeds maximum allowed")
	ErrAmountTooSmall     = errors.New("amount below minimum allowed")
	ErrMetadataTooLarge   = errors.New("metadata size exceeds limit")
	ErrInvalidEmail       = errors.New("invalid email format")
	ErrPasswordTooWeak    = errors.New("password does not meet requirements")
	ErrInvalidIDFormat    = errors.New("invalid ID format")
)

// Validation constants
const (
	MaxAccountNameLength = 255
	MinAccountNameLength = 1
	MaxMetadataSize      = 10240           // 10KB
	MaxTransferAmount    = "1000000000000" // 1 trillion
	MinTransferAmount    = "0.01"
	MinPasswordLength    = 8
	MaxPasswordLength    = 128
)

// Valid currency codes (ISO 4217)
var validCurrencies = map[string]bool{
	"USD": true, "EUR": true, "GBP": true, "JPY": true,
	"CNY": true, "AUD": true, "CAD": true, "CHF": true,
	"SEK": true, "NZD": true, "KRW": true, "SGD": true,
	"NOK": true, "MXN": true, "INR": true, "BRL": true,
	"ZAR": true, "RUB": true, "TRY": true, "HKD": true,
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// ValidateAccountName validates account name
func ValidateAccountName(name string) error {
	name = strings.TrimSpace(name)

	if len(name) < MinAccountNameLength {
		return fmt.Errorf("%w: name cannot be empty", ErrInvalidAccountName)
	}

	if len(name) > MaxAccountNameLength {
		return fmt.Errorf("%w: name exceeds %d characters", ErrInvalidAccountName, MaxAccountNameLength)
	}

	// Check for SQL injection attempts
	dangerous := []string{"--", "/*", "*/", ";", "DROP", "DELETE", "INSERT", "UPDATE"}
	nameUpper := strings.ToUpper(name)
	for _, pattern := range dangerous {
		if strings.Contains(nameUpper, pattern) {
			return fmt.Errorf("%w: contains forbidden characters", ErrInvalidAccountName)
		}
	}

	return nil
}

// ValidateCurrency validates currency code
func ValidateCurrency(currency string) error {
	currency = strings.ToUpper(strings.TrimSpace(currency))

	if !validCurrencies[currency] {
		return fmt.Errorf("%w: %s is not a valid ISO 4217 currency code", ErrInvalidCurrency, currency)
	}

	return nil
}

// ValidateAmount validates transfer/hold amount
func ValidateAmount(amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidAmount
	}

	minAmount, _ := decimal.NewFromString(MinTransferAmount)
	if amount.LessThan(minAmount) {
		return fmt.Errorf("%w: minimum amount is %s", ErrAmountTooSmall, MinTransferAmount)
	}

	maxAmount, _ := decimal.NewFromString(MaxTransferAmount)
	if amount.GreaterThan(maxAmount) {
		return fmt.Errorf("%w: maximum amount is %s", ErrAmountTooLarge, MaxTransferAmount)
	}

	return nil
}

// ValidateMetadata validates metadata size
func ValidateMetadata(metadata map[string]any) error {
	if metadata == nil {
		return nil
	}

	// Estimate size (rough approximation)
	size := 0
	for k, v := range metadata {
		size += len(k)
		size += len(fmt.Sprintf("%v", v))
	}

	if size > MaxMetadataSize {
		return fmt.Errorf("%w: metadata size %d bytes exceeds limit of %d bytes", ErrMetadataTooLarge, size, MaxMetadataSize)
	}

	return nil
}

// ValidateEmail validates email format
func ValidateEmail(email string) error {
	email = strings.TrimSpace(strings.ToLower(email))

	if !emailRegex.MatchString(email) {
		return ErrInvalidEmail
	}

	return nil
}

// ValidatePassword validates password strength
func ValidatePassword(password string) error {
	if len(password) < MinPasswordLength {
		return fmt.Errorf("%w: must be at least %d characters", ErrPasswordTooWeak, MinPasswordLength)
	}

	if len(password) > MaxPasswordLength {
		return fmt.Errorf("%w: must not exceed %d characters", ErrPasswordTooWeak, MaxPasswordLength)
	}

	// Check for at least one uppercase, one lowercase, and one number
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)

	if !hasUpper || !hasLower || !hasNumber {
		return fmt.Errorf("%w: must contain uppercase, lowercase, and numbers", ErrPasswordTooWeak)
	}

	return nil
}

// ValidatePagination validates and limits pagination parameters
func ValidatePagination(limit, offset int) (int, int, error) {
	const MaxPageSize = 1000
	const DefaultPageSize = 50

	if limit <= 0 {
		limit = DefaultPageSize
	}

	if limit > MaxPageSize {
		limit = MaxPageSize
	}

	if offset < 0 {
		offset = 0
	}

	return limit, offset, nil
}
