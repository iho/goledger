package logging

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"unknown", slog.LevelInfo},
	}

	for _, tt := range tests {
		if got := ParseLevel(tt.input); got != tt.want {
			t.Fatalf("ParseLevel(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestLoggerWithContext(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, RequestIDKey, "req-1")
	ctx = context.WithValue(ctx, UserIDKey, "user-1")

	output := captureStdout(t, func() {
		base := New(slog.LevelInfo, "json")
		base.InfoCtx(ctx, "test message")
	})

	if !strings.Contains(output, `"request_id":"req-1"`) || !strings.Contains(output, `"user_id":"user-1"`) {
		t.Fatalf("expected context fields in log output, got %q", output)
	}
}

func TestLoggerFormats(t *testing.T) {
	tests := []struct {
		name   string
		format string
	}{
		{name: "json format", format: "json"},
		{name: "text format", format: "text"},
		{name: "default format", format: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStdout(t, func() {
				logger := New(slog.LevelInfo, tt.format)
				logger.Info("formatted output")
			})

			if output == "" {
				t.Fatalf("expected log output, got empty string")
			}
		})
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	fn()

	_ = w.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read stdout: %v", err)
	}

	return buf.String()
}
