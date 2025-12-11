package converter

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/iho/goledger/internal/domain"
)

func TestAccountToPb(t *testing.T) {
	now := time.Now().UTC().Round(time.Millisecond)
	account := &domain.Account{
		ID:                   "acc-1",
		Name:                 "Primary",
		Currency:             "USD",
		Balance:              decimal.NewFromInt(100),
		EncumberedBalance:    decimal.NewFromInt(5),
		Version:              3,
		AllowNegativeBalance: true,
		AllowPositiveBalance: true,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	got := AccountToPb(account)
	if got == nil {
		t.Fatal("expected protobuf account")
	}

	if got.Id != account.ID || got.Name != account.Name || got.Currency != account.Currency {
		t.Fatalf("unexpected account fields: %+v", got)
	}

	if got.Balance != account.Balance.String() || got.EncumberedBalance != account.EncumberedBalance.String() {
		t.Fatalf("expected balances to match: %+v", got)
	}

	if got.CreatedAt.AsTime() != now || got.UpdatedAt.AsTime() != now {
		t.Fatalf("expected timestamps to match")
	}

	if AccountToPb(nil) != nil {
		t.Fatal("expected nil account to return nil")
	}
}

func TestTransferToPb(t *testing.T) {
	now := time.Now().UTC()
	reversed := "rev-1"
	transfer := &domain.Transfer{
		ID:                 "tx-1",
		FromAccountID:      "acc-1",
		ToAccountID:        "acc-2",
		Amount:             decimal.NewFromFloat(10.5),
		CreatedAt:          now,
		EventAt:            now.Add(time.Minute),
		Metadata:           map[string]any{"note": "test", "ignored": 123},
		ReversedTransferID: &reversed,
	}

	got := TransferToPb(transfer)
	if got == nil {
		t.Fatal("expected protobuf transfer")
	}

	if got.Metadata["note"] != "test" {
		t.Fatalf("expected metadata to include string entries")
	}

	if _, exists := got.Metadata["ignored"]; exists {
		t.Fatalf("expected non-string metadata to be dropped")
	}

	if got.ReversedTransferId != transfer.ReversedTransferID {
		t.Fatalf("expected reversed transfer ID to be set")
	}

	if TransferToPb(nil) != nil {
		t.Fatal("expected nil transfer to return nil")
	}
}

func TestEntryToPb(t *testing.T) {
	now := time.Now().UTC()
	entry := &domain.Entry{
		ID:                     "entry-1",
		AccountID:              "acc-1",
		TransferID:             "tx-1",
		Amount:                 decimal.NewFromInt(-10),
		AccountPreviousBalance: decimal.NewFromInt(100),
		AccountCurrentBalance:  decimal.NewFromInt(90),
		AccountVersion:         4,
		CreatedAt:              now,
	}

	got := EntryToPb(entry)
	if got == nil {
		t.Fatal("expected protobuf entry")
	}

	if got.Amount != "-10" || got.AccountCurrentBalance != "90" {
		t.Fatalf("unexpected balances: %+v", got)
	}

	if got.CreatedAt.AsTime() != now {
		t.Fatalf("expected created at to match")
	}

	if EntryToPb(nil) != nil {
		t.Fatal("expected nil entry to return nil")
	}
}

func TestHoldToPb(t *testing.T) {
	now := time.Now().UTC()
	expiration := now.Add(time.Hour)
	hold := &domain.Hold{
		ID:        "hold-1",
		AccountID: "acc-1",
		Amount:    decimal.NewFromInt(50),
		Status:    domain.HoldStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: &expiration,
		Metadata:  map[string]any{"reason": "reserve"},
	}

	got := HoldToPb(hold)
	if got == nil {
		t.Fatal("expected protobuf hold")
	}

	if got.Metadata["reason"] != "reserve" {
		t.Fatalf("expected metadata to be converted")
	}

	if got.ExpiresAt.AsTime() != expiration {
		t.Fatalf("expected expiration timestamp to match")
	}

	if HoldToPb(nil) != nil {
		t.Fatal("expected nil hold to return nil")
	}
}

func TestParseDecimal(t *testing.T) {
	val, err := ParseDecimal("123.45")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !val.Equal(decimal.RequireFromString("123.45")) {
		t.Fatalf("expected parsed decimal to match, got %s", val)
	}
}

func TestParseTimestamp(t *testing.T) {
	now := time.Now().UTC().Round(time.Millisecond)
	ts := timestamppb.New(now)

	got := ParseTimestamp(ts)
	if got == nil || !got.Equal(now) {
		t.Fatalf("expected timestamp to convert, got %v", got)
	}

	if ParseTimestamp(nil) != nil {
		t.Fatal("expected nil timestamp to return nil")
	}
}

func TestMetadataToMap(t *testing.T) {
	src := map[string]string{"key": "value"}
	got := MetadataToMap(src)
	if got["key"] != "value" {
		t.Fatalf("expected key to be copied, got %v", got)
	}

	if MetadataToMap(nil) != nil {
		t.Fatal("expected nil map to return nil")
	}
}
