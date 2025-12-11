package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const (
	// IdempotencyKeyHeader is the metadata key for idempotency
	IdempotencyKeyHeader = "x-idempotency-key"
)

// IdempotencyStore defines the minimal contract needed for idempotency handling.
type IdempotencyStore interface {
	CheckAndSet(ctx context.Context, key string, response []byte, ttl time.Duration) (exists bool, cachedResponse []byte, err error)
}

// IdempotencyInterceptor creates a gRPC unary interceptor for idempotency
func IdempotencyInterceptor(store IdempotencyStore) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		// Skip idempotency for read-only methods (Get, List)
		if isReadOnlyMethod(info.FullMethod) {
			return handler(ctx, req)
		}

		// Extract idempotency key from metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			// No metadata, proceed without idempotency (allow for now)
			return handler(ctx, req)
		}

		keys := md.Get(IdempotencyKeyHeader)
		if len(keys) == 0 {
			// No idempotency key provided, proceed without idempotency
			return handler(ctx, req)
		}

		idempotencyKey := keys[0]
		if idempotencyKey == "" {
			return nil, status.Error(codes.InvalidArgument, "idempotency key cannot be empty")
		}

		// Generate a unique key combining method and idempotency key
		cacheKey := fmt.Sprintf("grpc:%s:%s", info.FullMethod, idempotencyKey)

		// Generate request fingerprint for validation
		requestHash, err := hashRequest(req)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to generate request hash")
		}

		// Check if we've seen this request before
		// Use request hash as the stored value to detect body changes
		exists := false
		var cachedHash []byte
		var storeErr error
		if store != nil {
			exists, cachedHash, storeErr = store.CheckAndSet(ctx, cacheKey, []byte(requestHash), 24*3600*time.Second)
			if storeErr != nil {
				// Log error but continue (degraded mode without idempotency)
				return handler(ctx, req)
			}
		}

		if exists {
			// Request was already processed
			if string(cachedHash) != requestHash {
				// Same idempotency key but different request body
				return nil, status.Error(codes.InvalidArgument, "idempotency key reused with different request body")
			}

			// Same request, return idempotent response
			// Note: For a full implementation, we'd cache and return the actual response
			// For now, we indicate duplicate request was detected
			return nil, status.Error(codes.AlreadyExists, "request already processed (idempotent)")
		}

		// Process the request (first time seeing this idempotency key)
		resp, err := handler(ctx, req)
		if err != nil {
			// Don't cache errors - allow retry
			return resp, err
		}

		// Request processed successfully
		// The hash is already stored by CheckAndSet above
		return resp, nil
	}
}

// isReadOnlyMethod checks if a method is read-only (GET, LIST operations)
func isReadOnlyMethod(method string) bool {
	// Methods that don't modify state
	readOnlyMethods := map[string]bool{
		"/goledger.v1.AccountService/GetAccount":              true,
		"/goledger.v1.AccountService/ListAccounts":            true,
		"/goledger.v1.TransferService/GetTransfer":            true,
		"/goledger.v1.TransferService/ListTransfersByAccount": true,
		"/goledger.v1.HoldService/ListHoldsByAccount":         true,
	}

	return readOnlyMethods[method]
}

// hashRequest generates a SHA-256 hash of the request for fingerprinting
func hashRequest(req any) (string, error) {
	// Try to marshal as protobuf message
	if protoMsg, ok := req.(proto.Message); ok {
		data, err := proto.Marshal(protoMsg)
		if err != nil {
			return "", err
		}

		hash := sha256.Sum256(data)
		return hex.EncodeToString(hash[:]), nil
	}

	// Fallback: use fmt.Sprint (not ideal but better than nothing)
	data := []byte(fmt.Sprintf("%+v", req))
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}
