package server

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/iho/goledger/internal/adapter/grpc/converter"
	grpcErrors "github.com/iho/goledger/internal/adapter/grpc/errors"
	pb "github.com/iho/goledger/internal/adapter/grpc/pb/goledger/v1"
	"github.com/iho/goledger/internal/usecase"
)

// TransferServer implements the gRPC TransferService
type TransferServer struct {
	pb.UnimplementedTransferServiceServer
	transferUC *usecase.TransferUseCase
}

// NewTransferServer creates a new TransferServer
func NewTransferServer(transferUC *usecase.TransferUseCase) *TransferServer {
	return &TransferServer{
		transferUC: transferUC,
	}
}

// CreateTransfer creates a single transfer
func (s *TransferServer) CreateTransfer(ctx context.Context, req *pb.CreateTransferRequest) (*pb.CreateTransferResponse, error) {
	amount, err := converter.ParseDecimal(req.Amount)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid amount format")
	}

	input := usecase.CreateTransferInput{
		FromAccountID: req.FromAccountId,
		ToAccountID:   req.ToAccountId,
		Amount:        amount,
		EventAt:       converter.ParseTimestamp(req.EventAt),
		Metadata:      converter.MetadataToMap(req.Metadata),
	}

	transfer, err := s.transferUC.CreateTransfer(ctx, input)
	if err != nil {
		return nil, grpcErrors.MapDomainError(err)
	}

	return &pb.CreateTransferResponse{
		Transfer: converter.TransferToPb(transfer),
	}, nil
}

// CreateBatchTransfer creates multiple transfers atomically
func (s *TransferServer) CreateBatchTransfer(ctx context.Context, req *pb.CreateBatchTransferRequest) (*pb.CreateBatchTransferResponse, error) {
	transfers := make([]usecase.CreateTransferInput, len(req.Transfers))
	for i, t := range req.Transfers {
		amount, err := converter.ParseDecimal(t.Amount)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid amount format at index %d", i)
		}

		transfers[i] = usecase.CreateTransferInput{
			FromAccountID: t.FromAccountId,
			ToAccountID:   t.ToAccountId,
			Amount:        amount,
			EventAt:       converter.ParseTimestamp(t.EventAt),
			Metadata:      converter.MetadataToMap(t.Metadata),
		}
	}

	batchInput := usecase.CreateBatchTransferInput{
		Transfers: transfers,
		EventAt:   converter.ParseTimestamp(req.EventAt),
		Metadata:  converter.MetadataToMap(req.Metadata),
	}

	results, err := s.transferUC.CreateBatchTransfer(ctx, batchInput)
	if err != nil {
		return nil, grpcErrors.MapDomainError(err)
	}

	pbTransfers := make([]*pb.Transfer, len(results))
	for i, transfer := range results {
		pbTransfers[i] = converter.TransferToPb(transfer)
	}

	return &pb.CreateBatchTransferResponse{
		Transfers: pbTransfers,
	}, nil
}

// GetTransfer retrieves a transfer by ID
func (s *TransferServer) GetTransfer(ctx context.Context, req *pb.GetTransferRequest) (*pb.GetTransferResponse, error) {
	transfer, err := s.transferUC.GetTransfer(ctx, req.Id)
	if err != nil {
		return nil, grpcErrors.MapDomainError(err)
	}

	return &pb.GetTransferResponse{
		Transfer: converter.TransferToPb(transfer),
	}, nil
}

// ListTransfersByAccount lists transfers for an account
func (s *TransferServer) ListTransfersByAccount(ctx context.Context, req *pb.ListTransfersByAccountRequest) (*pb.ListTransfersByAccountResponse, error) {
	transfers, err := s.transferUC.ListTransfersByAccount(ctx, usecase.ListTransfersByAccountInput{
		AccountID: req.AccountId,
		Limit:     int(req.Limit),
		Offset:    int(req.Offset),
	})
	if err != nil {
		return nil, grpcErrors.MapDomainError(err)
	}

	pbTransfers := make([]*pb.Transfer, len(transfers))
	for i, transfer := range transfers {
		pbTransfers[i] = converter.TransferToPb(transfer)
	}

	return &pb.ListTransfersByAccountResponse{
		Transfers: pbTransfers,
	}, nil
}

// ReverseTransfer creates a reversal transfer
func (s *TransferServer) ReverseTransfer(ctx context.Context, req *pb.ReverseTransferRequest) (*pb.ReverseTransferResponse, error) {
	transfer, err := s.transferUC.ReverseTransfer(ctx, usecase.ReverseTransferInput{
		TransferID: req.TransferId,
		Metadata:   converter.MetadataToMap(req.Metadata),
	})
	if err != nil {
		return nil, grpcErrors.MapDomainError(err)
	}

	return &pb.ReverseTransferResponse{
		Transfer: converter.TransferToPb(transfer),
	}, nil
}
