package logger

import (
	"bytes"
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
		if got := parseLevel(tt.input); got != tt.want {
			t.Fatalf("parseLevel(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestNewLoggerFormatsOutput(t *testing.T) {
	tests := []struct {
		name       string
		format     string
		level      string
		assertions func(t *testing.T, output string)
	}{
		{
			name:   "text format includes message field",
			format: "text",
			level:  "info",
			assertions: func(t *testing.T, output string) {
				if !strings.Contains(output, "msg=hello") {
					t.Fatalf("expected text output to contain msg field, got %q", output)
				}
			},
		},
		{
			name:   "json format starts with brace",
			format: "json",
			level:  "debug",
			assertions: func(t *testing.T, output string) {
				if !strings.HasPrefix(strings.TrimSpace(output), "{") {
					t.Fatalf("expected json output to start with '{', got %q", output)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStdout(t, func() {
				log := New(Config{Format: tt.format, Level: tt.level})
				log.Info("hello")
			})

			if output == "" {
				t.Fatalf("expected log output, got empty string")
			}

			tt.assertions(t, output)
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

	fn()

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read stdout: %v", err)
	}

	return buf.String()
}
