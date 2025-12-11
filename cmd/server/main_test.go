package main

import (
	"os"
	"testing"
)

func TestResolveGRPCPort(t *testing.T) {
	orig := os.Getenv("GRPC_PORT")
	defer os.Setenv("GRPC_PORT", orig)

	os.Unsetenv("GRPC_PORT")
	if got := resolveGRPCPort(); got != "50051" {
		t.Fatalf("expected default port 50051, got %s", got)
	}

	os.Setenv("GRPC_PORT", "6000")
	if got := resolveGRPCPort(); got != "6000" {
		t.Fatalf("expected overridden port 6000, got %s", got)
	}
}
