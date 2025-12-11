package redis

import (
	"context"
	"testing"
	"time"
)

func TestCacheSetAndGet(t *testing.T) {
	client, mr := newTestRedisClient(t)
	defer mr.Close()
	defer client.Close()

	cache := NewCache(client)
	ctx := context.Background()

	if err := cache.Set(ctx, "foo", "bar", time.Minute); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	val, err := cache.Get(ctx, "foo")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}

	if val != "bar" {
		t.Fatalf("expected bar, got %s", val)
	}
}

func TestCacheSetNX(t *testing.T) {
	client, mr := newTestRedisClient(t)
	defer mr.Close()
	defer client.Close()

	cache := NewCache(client)
	ctx := context.Background()

	set, err := cache.SetNX(ctx, "key", "first", time.Minute)
	if err != nil || !set {
		t.Fatalf("expected first SetNX to succeed, got set=%v err=%v", set, err)
	}

	set, err = cache.SetNX(ctx, "key", "second", time.Minute)
	if err != nil {
		t.Fatalf("SetNX failed: %v", err)
	}
	if set {
		t.Fatalf("expected second SetNX to fail because key exists")
	}
}

func TestCacheDelete(t *testing.T) {
	client, mr := newTestRedisClient(t)
	defer mr.Close()
	defer client.Close()

	cache := NewCache(client)
	ctx := context.Background()

	if err := cache.Set(ctx, "foo", "bar", time.Minute); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	if err := cache.Delete(ctx, "foo"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	if _, err := cache.Get(ctx, "foo"); err == nil {
		t.Fatalf("expected error getting deleted key")
	}
}
