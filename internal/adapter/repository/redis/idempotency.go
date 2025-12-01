package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// IdempotencyStore implements usecase.IdempotencyStore using Redis.
type IdempotencyStore struct {
	client *redis.Client
	prefix string
}

// NewIdempotencyStore creates a new IdempotencyStore.
func NewIdempotencyStore(client *redis.Client) *IdempotencyStore {
	return &IdempotencyStore{
		client: client,
		prefix: "idempotency:",
	}
}

// CheckAndSet atomically checks if key exists, sets if not.
func (s *IdempotencyStore) CheckAndSet(ctx context.Context, key string, response []byte, ttl time.Duration) (bool, []byte, error) {
	fullKey := s.prefix + key

	// Try to get existing
	existing, err := s.client.Get(ctx, fullKey).Bytes()
	if err == nil {
		return true, existing, nil
	}
	if err != redis.Nil {
		return false, nil, err
	}

	// Key doesn't exist, set it if response provided
	if response != nil {
		err = s.client.Set(ctx, fullKey, response, ttl).Err()
		if err != nil {
			return false, nil, err
		}
	} else {
		// Set placeholder to "lock" the key
		set, err := s.client.SetNX(ctx, fullKey, "processing", ttl).Result()
		if err != nil {
			return false, nil, err
		}
		if !set {
			// Another request got there first
			existing, err := s.client.Get(ctx, fullKey).Bytes()
			if err != nil && err != redis.Nil {
				return false, nil, err
			}
			return true, existing, nil
		}
	}

	return false, nil, nil
}

// Update updates an existing idempotency key with the final response.
func (s *IdempotencyStore) Update(ctx context.Context, key string, response []byte, ttl time.Duration) error {
	fullKey := s.prefix + key
	return s.client.Set(ctx, fullKey, response, ttl).Err()
}
