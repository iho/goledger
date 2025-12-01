package server

import (
	"context"

	"github.com/iho/goledger/internal/adapter/grpc/converter"
	grpcErrors "github.com/iho/goledger/internal/adapter/grpc/errors"
	pb "github.com/iho/goledger/internal/adapter/grpc/pb/goledger/v1"
	"github.com/iho/goledger/internal/usecase"
)

// AccountServer implements the gRPC AccountService
type AccountServer struct {
	pb.UnimplementedAccountServiceServer
	accountUC *usecase.AccountUseCase
}

// NewAccountServer creates a new AccountServer
func NewAccountServer(accountUC *usecase.AccountUseCase) *AccountServer {
	return &AccountServer{
		accountUC: accountUC,
	}
}

// CreateAccount creates a new ledger account
func (s *AccountServer) CreateAccount(ctx context.Context, req *pb.CreateAccountRequest) (*pb.CreateAccountResponse, error) {
	account, err := s.accountUC.CreateAccount(ctx, usecase.CreateAccountInput{
		Name:                 req.Name,
		Currency:             req.Currency,
		AllowNegativeBalance: req.AllowNegativeBalance,
		AllowPositiveBalance: req.AllowPositiveBalance,
	})
	if err != nil {
		return nil, grpcErrors.MapDomainError(err)
	}

	return &pb.CreateAccountResponse{
		Account: converter.AccountToPb(account),
	}, nil
}

// GetAccount retrieves an account by ID
func (s *AccountServer) GetAccount(ctx context.Context, req *pb.GetAccountRequest) (*pb.GetAccountResponse, error) {
	account, err := s.accountUC.GetAccount(ctx, req.Id)
	if err != nil {
		return nil, grpcErrors.MapDomainError(err)
	}

	return &pb.GetAccountResponse{
		Account: converter.AccountToPb(account),
	}, nil
}

// ListAccounts lists accounts with pagination
func (s *AccountServer) ListAccounts(ctx context.Context, req *pb.ListAccountsRequest) (*pb.ListAccountsResponse, error) {
	accounts, err := s.accountUC.ListAccounts(ctx, usecase.ListAccountsInput{
		Limit:  int(req.Limit),
		Offset: int(req.Offset),
	})
	if err != nil {
		return nil, grpcErrors.MapDomainError(err)
	}

	pbAccounts := make([]*pb.Account, len(accounts))
	for i, account := range accounts {
		pbAccounts[i] = converter.AccountToPb(account)
	}

	return &pb.ListAccountsResponse{
		Accounts: pbAccounts,
	}, nil
}
