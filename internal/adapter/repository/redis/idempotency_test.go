package redis

import (
	"context"
	"testing"
	"time"
)

func TestIdempotencyStore_CheckAndSetExisting(t *testing.T) {
	client, mr := newTestRedisClient(t)
	defer mr.Close()
	defer client.Close()

	store := NewIdempotencyStore(client)
	ctx := context.Background()

	if err := client.Set(ctx, store.prefix+"key", "cached", time.Minute).Err(); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	exists, resp, err := store.CheckAndSet(ctx, "key", nil, time.Minute)
	if err != nil {
		t.Fatalf("CheckAndSet failed: %v", err)
	}

	if !exists || string(resp) != "cached" {
		t.Fatalf("expected existing cached response, got exists=%v resp=%s", exists, resp)
	}
}

func TestIdempotencyStore_CheckAndSetLocksNewKey(t *testing.T) {
	client, mr := newTestRedisClient(t)
	defer mr.Close()
	defer client.Close()

	store := NewIdempotencyStore(client)
	ctx := context.Background()

	exists, resp, err := store.CheckAndSet(ctx, "pending", nil, time.Minute)
	if err != nil || exists || resp != nil {
		t.Fatalf("unexpected result: exists=%v resp=%v err=%v", exists, resp, err)
	}

	val, err := client.Get(ctx, store.prefix+"pending").Result()
	if err != nil || val != "processing" {
		t.Fatalf("expected placeholder lock, got val=%s err=%v", val, err)
	}
}

func TestIdempotencyStore_Update(t *testing.T) {
	client, mr := newTestRedisClient(t)
	defer mr.Close()
	defer client.Close()

	store := NewIdempotencyStore(client)
	ctx := context.Background()

	if err := store.Update(ctx, "complete", []byte("done"), time.Minute); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	val, err := client.Get(ctx, store.prefix+"complete").Result()
	if err != nil || val != "done" {
		t.Fatalf("expected stored response, got val=%s err=%v", val, err)
	}
}
