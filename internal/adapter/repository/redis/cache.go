package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache implements usecase.Cache using Redis.
type Cache struct {
	client *redis.Client
	prefix string
}

// NewCache creates a new Cache.
func NewCache(client *redis.Client) *Cache {
	return &Cache{
		client: client,
		prefix: "cache:",
	}
}

// Get retrieves a value by key.
func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, c.prefix+key).Result()
}

// Set stores a value with TTL.
func (c *Cache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return c.client.Set(ctx, c.prefix+key, value, ttl).Err()
}

// SetNX sets a value only if it doesn't exist.
func (c *Cache) SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	return c.client.SetNX(ctx, c.prefix+key, value, ttl).Result()
}

// Delete removes a key.
func (c *Cache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, c.prefix+key).Err()
}
