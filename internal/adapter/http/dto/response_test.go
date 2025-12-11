package dto

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/iho/goledger/internal/domain"
)

func TestAccountFromDomain(t *testing.T) {
	now := time.Now()
	account := &domain.Account{
		ID:                   "acc-1",
		Name:                 "Main",
		Currency:             "USD",
		Balance:              decimal.RequireFromString("123.45"),
		Version:              2,
		AllowNegativeBalance: true,
		AllowPositiveBalance: false,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	resp := AccountFromDomain(account)
	if resp.ID != account.ID || resp.Balance != "123.45" || resp.Version != 2 {
		t.Fatalf("unexpected account response: %+v", resp)
	}

	list := AccountsFromDomain([]*domain.Account{account})
	if len(list) != 1 || list[0].ID != account.ID {
		t.Fatalf("AccountsFromDomain returned %+v", list)
	}
}

func TestTransferFromDomain(t *testing.T) {
	now := time.Now()
	reversed := "rev-1"
	transfer := &domain.Transfer{
		ID:                 "tr-1",
		FromAccountID:      "A",
		ToAccountID:        "B",
		Amount:             decimal.RequireFromString("10"),
		CreatedAt:          now,
		EventAt:            now,
		Metadata:           map[string]any{"key": "value"},
		ReversedTransferID: &reversed,
	}

	resp := TransferFromDomain(transfer)
	if resp.ID != transfer.ID || resp.Amount != "10" || resp.ReversedTransferID == nil {
		t.Fatalf("unexpected transfer response: %+v", resp)
	}

	list := TransfersFromDomain([]*domain.Transfer{transfer})
	if len(list) != 1 || list[0].ID != transfer.ID {
		t.Fatalf("TransfersFromDomain returned %+v", list)
	}
}

func TestEntryFromDomain(t *testing.T) {
	entry := &domain.Entry{
		ID:                     "entry-1",
		AccountID:              "acc",
		TransferID:             "tr",
		Amount:                 decimal.RequireFromString("5"),
		AccountPreviousBalance: decimal.RequireFromString("10"),
		AccountCurrentBalance:  decimal.RequireFromString("15"),
		AccountVersion:         3,
		CreatedAt:              time.Now(),
	}

	resp := EntryFromDomain(entry)
	if resp.AccountID != entry.AccountID || resp.AccountVersion != entry.AccountVersion {
		t.Fatalf("unexpected entry response: %+v", resp)
	}

	list := EntriesFromDomain([]*domain.Entry{entry})
	if len(list) != 1 || list[0].ID != entry.ID {
		t.Fatalf("EntriesFromDomain returned %+v", list)
	}
}
