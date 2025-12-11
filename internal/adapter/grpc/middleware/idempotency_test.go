package middleware_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/iho/goledger/internal/adapter/grpc/middleware"
	pb "github.com/iho/goledger/internal/adapter/grpc/pb/goledger/v1"
)

type fakeIdempotencyStore struct {
	exists bool
	value  []byte
	err    error
	called bool
}

func (f *fakeIdempotencyStore) CheckAndSet(ctx context.Context, key string, response []byte, ttl time.Duration) (bool, []byte, error) {
	f.called = true
	return f.exists, f.value, f.err
}

func TestIdempotencyInterceptor_SkipsReadOnlyMethods(t *testing.T) {
	store := &fakeIdempotencyStore{}
	interceptor := middleware.IdempotencyInterceptor(store)

	info := &grpc.UnaryServerInfo{FullMethod: "/goledger.v1.AccountService/GetAccount"}
	called := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		called = true
		return "ok", nil
	}

	resp, err := interceptor(context.Background(), nil, info, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "ok" || !called {
		t.Fatalf("expected handler to execute")
	}
	if store.called {
		t.Fatalf("expected store not to be called for read-only method")
	}
}

func TestIdempotencyInterceptor_HandleDuplicateRequest(t *testing.T) {
	store := &fakeIdempotencyStore{
		exists: true,
	}
	interceptor := middleware.IdempotencyInterceptor(store)

	info := &grpc.UnaryServerInfo{FullMethod: "/goledger.v1.AccountService/CreateAccount"}
	req := newTransferRequest("duplicate")

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(middleware.IdempotencyKeyHeader, "key-1"))
	store.value = []byte(hashRequestForTest(t, req))

	_, err := interceptor(ctx, req, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		t.Fatal("handler should not execute for duplicate request")
		return nil, nil
	})

	if status.Code(err) != codes.AlreadyExists {
		t.Fatalf("expected AlreadyExists, got %v", err)
	}
}

func TestIdempotencyInterceptor_DetectsBodyMismatch(t *testing.T) {
	store := &fakeIdempotencyStore{
		exists: true,
		value:  []byte("different-hash"),
	}
	interceptor := middleware.IdempotencyInterceptor(store)

	info := &grpc.UnaryServerInfo{FullMethod: "/goledger.v1.TransferService/CreateTransfer"}
	req := newTransferRequest("new")
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(middleware.IdempotencyKeyHeader, "key-1"))

	_, err := interceptor(ctx, req, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		t.Fatal("handler should not execute for conflicting body")
		return nil, nil
	})

	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

func TestIdempotencyInterceptor_FirstRequest(t *testing.T) {
	store := &fakeIdempotencyStore{
		exists: false,
		value:  nil,
	}
	interceptor := middleware.IdempotencyInterceptor(store)

	info := &grpc.UnaryServerInfo{FullMethod: "/goledger.v1.HoldService/CreateHold"}
	req := newTransferRequest("initial")
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(middleware.IdempotencyKeyHeader, "key-1"))

	called := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		called = true
		return "success", nil
	}

	resp, err := interceptor(ctx, req, info, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "success" || !called {
		t.Fatalf("expected handler to run successfully")
	}
}

func TestIdempotencyInterceptor_NoStore(t *testing.T) {
	interceptor := middleware.IdempotencyInterceptor(nil)

	info := &grpc.UnaryServerInfo{FullMethod: "/goledger.v1.TransferService/CreateTransfer"}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(middleware.IdempotencyKeyHeader, "key-1"))

	called := false
	resp, err := interceptor(ctx, newTransferRequest("request"), info, func(ctx context.Context, req interface{}) (interface{}, error) {
		called = true
		return "ok", nil
	})

	if err != nil || resp != "ok" {
		t.Fatalf("expected handler to run in degraded mode, resp=%v err=%v", resp, err)
	}

	if !called {
		t.Fatalf("expected handler to be invoked when store is nil")
	}
}

func TestIdempotencyInterceptor_StoreError(t *testing.T) {
	store := &fakeIdempotencyStore{
		err: errors.New("redis down"),
	}
	interceptor := middleware.IdempotencyInterceptor(store)

	info := &grpc.UnaryServerInfo{FullMethod: "/goledger.v1.TransferService/CreateTransfer"}
	req := newTransferRequest("request")
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(middleware.IdempotencyKeyHeader, "key-1"))

	resp, err := interceptor(ctx, req, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	})

	if err != nil || resp != "ok" {
		t.Fatalf("expected handler to run in degraded mode, resp=%v err=%v", resp, err)
	}
}

func TestIdempotencyInterceptor_EmptyKey(t *testing.T) {
	store := &fakeIdempotencyStore{}
	interceptor := middleware.IdempotencyInterceptor(store)

	info := &grpc.UnaryServerInfo{FullMethod: "/service.Method"}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(middleware.IdempotencyKeyHeader, ""))

	_, err := interceptor(ctx, nil, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		t.Fatal("handler should not run for empty key")
		return nil, nil
	})

	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for empty key, got %v", err)
	}
}

func hashRequestForTest(t *testing.T, req any) string {
	t.Helper()
	if protoMsg, ok := req.(proto.Message); ok {
		data, err := proto.Marshal(protoMsg)
		if err != nil {
			t.Fatalf("failed to marshal proto: %v", err)
		}
		hash := sha256.Sum256(data)
		return hex.EncodeToString(hash[:])
	}
	data := []byte(fmt.Sprintf("%+v", req))
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func newTransferRequest(payload string) *pb.CreateTransferRequest {
	return &pb.CreateTransferRequest{
		FromAccountId: "acc-1",
		ToAccountId:   "acc-2",
		Amount:        payload,
	}
}
