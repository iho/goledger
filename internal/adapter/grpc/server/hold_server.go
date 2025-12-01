package server

import (
	"context"

	"github.com/iho/goledger/internal/adapter/grpc/converter"
	grpcErrors "github.com/iho/goledger/internal/adapter/grpc/errors"
	pb "github.com/iho/goledger/internal/adapter/grpc/pb/goledger/v1"
	"github.com/iho/goledger/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// HoldServer implements the gRPC HoldService
type HoldServer struct {
	pb.UnimplementedHoldServiceServer
	holdUC *usecase.HoldUseCase
}

// NewHoldServer creates a new HoldServer
func NewHoldServer(holdUC *usecase.HoldUseCase) *HoldServer {
	return &HoldServer{
		holdUC: holdUC,
	}
}

// HoldFunds places a hold on an account
func (s *HoldServer) HoldFunds(ctx context.Context, req *pb.HoldFundsRequest) (*pb.HoldFundsResponse, error) {
	amount, err := converter.ParseDecimal(req.Amount)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid amount format")
	}

	hold, err := s.holdUC.HoldFunds(ctx, req.AccountId, amount)
	if err != nil {
		return nil, grpcErrors.MapDomainError(err)
	}

	return &pb.HoldFundsResponse{
		Hold: converter.HoldToPb(hold),
	}, nil
}

// VoidHold cancels a hold
func (s *HoldServer) VoidHold(ctx context.Context, req *pb.VoidHoldRequest) (*pb.VoidHoldResponse, error) {
	if err := s.holdUC.VoidHold(ctx, req.HoldId); err != nil {
		return nil, grpcErrors.MapDomainError(err)
	}

	return &pb.VoidHoldResponse{}, nil
}

// CaptureHold captures a hold as a transfer
func (s *HoldServer) CaptureHold(ctx context.Context, req *pb.CaptureHoldRequest) (*pb.CaptureHoldResponse, error) {
	transfer, err := s.holdUC.CaptureHold(ctx, req.HoldId, req.ToAccountId)
	if err != nil {
		return nil, grpcErrors.MapDomainError(err)
	}

	return &pb.CaptureHoldResponse{
		Transfer: converter.TransferToPb(transfer),
	}, nil
}

// ListHoldsByAccount lists holds for an account
func (s *HoldServer) ListHoldsByAccount(ctx context.Context, req *pb.ListHoldsByAccountRequest) (*pb.ListHoldsByAccountResponse, error) {
	holds, err := s.holdUC.ListHoldsByAccount(ctx, usecase.ListHoldsByAccountInput{
		AccountID: req.AccountId,
		Limit:     int(req.Limit),
		Offset:    int(req.Offset),
	})
	if err != nil {
		return nil, grpcErrors.MapDomainError(err)
	}

	pbHolds := make([]*pb.Hold, len(holds))
	for i, hold := range holds {
		pbHolds[i] = converter.HoldToPb(hold)
	}

	return &pb.ListHoldsByAccountResponse{
		Holds: pbHolds,
	}, nil
}
