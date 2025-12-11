package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func captureOutput(t *testing.T, fn func()) string {
	t.Helper()

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = origStdout

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read stdout: %v", err)
	}
	return buf.String()
}

func TestTruncate(t *testing.T) {
	if got := truncate("short", 10); got != "short" {
		t.Fatalf("expected short unchanged, got %q", got)
	}

	if got := truncate("longerstring", 6); got != "lon..." {
		t.Fatalf("expected lon..., got %q", got)
	}
}

func TestPrintJSON(t *testing.T) {
	out := captureOutput(t, func() {
		printJSON(struct {
			A int `json:"a"`
		}{A: 1})
	})

	expected := "{\n  \"a\": 1\n}\n"
	if out != expected {
		t.Fatalf("unexpected json output:\n%s", out)
	}
}

func TestHashPasswordCmd(t *testing.T) {
	orig := bcryptGenerate
	bcryptGenerate = func(p []byte, cost int) ([]byte, error) {
		return []byte("hashed-value"), nil
	}
	defer func() { bcryptGenerate = orig }()

	cmd := hashPasswordCmd()
	cmd.SetArgs([]string{"secret"})

	out := captureOutput(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("command failed: %v", err)
		}
	})

	if strings.TrimSpace(out) != "hashed-value" {
		t.Fatalf("expected hashed-value, got %q", out)
	}
}
