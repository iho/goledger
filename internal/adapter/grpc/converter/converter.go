package converter

import (
	"time"

	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/iho/goledger/internal/adapter/grpc/pb/goledger/v1"
	"github.com/iho/goledger/internal/domain"
)

// AccountToPb converts domain.Account to protobuf Account
func AccountToPb(a *domain.Account) *pb.Account {
	if a == nil {
		return nil
	}
	return &pb.Account{
		Id:                   a.ID,
		Name:                 a.Name,
		Currency:             a.Currency,
		Balance:              a.Balance.String(),
		EncumberedBalance:    a.EncumberedBalance.String(),
		Version:              a.Version,
		AllowNegativeBalance: a.AllowNegativeBalance,
		AllowPositiveBalance: a.AllowPositiveBalance,
		CreatedAt:            timestamppb.New(a.CreatedAt),
		UpdatedAt:            timestamppb.New(a.UpdatedAt),
	}
}

// TransferToPb converts domain.Transfer to protobuf Transfer
func TransferToPb(t *domain.Transfer) *pb.Transfer {
	if t == nil {
		return nil
	}

	metadata := make(map[string]string)
	for k, v := range t.Metadata {
		if str, ok := v.(string); ok {
			metadata[k] = str
		}
	}

	pbTransfer := &pb.Transfer{
		Id:            t.ID,
		FromAccountId: t.FromAccountID,
		ToAccountId:   t.ToAccountID,
		Amount:        t.Amount.String(),
		CreatedAt:     timestamppb.New(t.CreatedAt),
		EventAt:       timestamppb.New(t.EventAt),
		Metadata:      metadata,
	}

	if t.ReversedTransferID != nil {
		pbTransfer.ReversedTransferId = t.ReversedTransferID
	}

	return pbTransfer
}

// EntryToPb converts domain.Entry to protobuf Entry
func EntryToPb(e *domain.Entry) *pb.Entry {
	if e == nil {
		return nil
	}
	return &pb.Entry{
		Id:                     e.ID,
		AccountId:              e.AccountID,
		TransferId:             e.TransferID,
		Amount:                 e.Amount.String(),
		AccountPreviousBalance: e.AccountPreviousBalance.String(),
		AccountCurrentBalance:  e.AccountCurrentBalance.String(),
		AccountVersion:         e.AccountVersion,
		CreatedAt:              timestamppb.New(e.CreatedAt),
	}
}

// HoldToPb converts domain.Hold to protobuf Hold
func HoldToPb(h *domain.Hold) *pb.Hold {
	if h == nil {
		return nil
	}

	metadata := make(map[string]string)
	for k, v := range h.Metadata {
		if str, ok := v.(string); ok {
			metadata[k] = str
		}
	}

	pbHold := &pb.Hold{
		Id:        h.ID,
		AccountId: h.AccountID,
		Amount:    h.Amount.String(),
		Status:    string(h.Status),
		CreatedAt: timestamppb.New(h.CreatedAt),
		UpdatedAt: timestamppb.New(h.UpdatedAt),
		Metadata:  metadata,
	}

	if h.ExpiresAt != nil {
		pbHold.ExpiresAt = timestamppb.New(*h.ExpiresAt)
	}

	return pbHold
}

// ParseDecimal parses a decimal string with validation
func ParseDecimal(s string) (decimal.Decimal, error) {
	return decimal.NewFromString(s)
}

// ParseTimestamp converts protobuf timestamp to time.Time
func ParseTimestamp(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.AsTime()
	return &t
}

// MetadataToMap converts protobuf metadata to map[string]any
func MetadataToMap(pbMetadata map[string]string) map[string]any {
	if pbMetadata == nil {
		return nil
	}
	metadata := make(map[string]any, len(pbMetadata))
	for k, v := range pbMetadata {
		metadata[k] = v
	}
	return metadata
}
