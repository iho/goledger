package redis

import (
	"context"
	"fmt"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
)

func TestNewClientSuccess(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	ctx := context.Background()
	client, err := NewClient(ctx, fmt.Sprintf("redis://%s", s.Addr()))
	if err != nil {
		t.Fatalf("expected client, got error: %v", err)
	}
	defer client.Close()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("ping failed: %v", err)
	}
}

func TestNewClientInvalidURL(t *testing.T) {
	_, err := NewClient(context.Background(), "://bad-url")
	if err == nil {
		t.Fatalf("expected error for invalid URL")
	}
}

func TestNewClientPingFailure(t *testing.T) {
	s := miniredis.RunT(t)
	url := fmt.Sprintf("redis://%s", s.Addr())
	s.Close() // close before attempting to connect

	_, err := NewClient(context.Background(), url)
	if err == nil {
		t.Fatalf("expected ping error when server is down")
	}
}
