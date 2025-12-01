package logging

import (
	"context"
	"log/slog"
	"os"
)

// ContextKey is the type for context keys
type ContextKey string

const (
	// RequestIDKey is the context key for request IDs
	RequestIDKey ContextKey = "request_id"
	// UserIDKey is the context key for user IDs
	UserIDKey ContextKey = "user_id"
)

// Logger wraps slog.Logger with additional context support
type Logger struct {
	*slog.Logger
}

// New creates a new structured logger
func New(level slog.Level, format string) *Logger {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level: level,
	}

	switch format {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return &Logger{
		Logger: slog.New(handler),
	}
}

// WithContext extracts common fields from context and returns a logger with those fields
func (l *Logger) WithContext(ctx context.Context) *slog.Logger {
	logger := l.Logger

	if requestID, ok := ctx.Value(RequestIDKey).(string); ok && requestID != "" {
		logger = logger.With("request_id", requestID)
	}

	if userID, ok := ctx.Value(UserIDKey).(string); ok && userID != "" {
		logger = logger.With("user_id", userID)
	}

	return logger
}

// InfoCtx logs an info message with context
func (l *Logger) InfoCtx(ctx context.Context, msg string, args ...any) {
	l.WithContext(ctx).Info(msg, args...)
}

// ErrorCtx logs an error message with context
func (l *Logger) ErrorCtx(ctx context.Context, msg string, args ...any) {
	l.WithContext(ctx).Error(msg, args...)
}

// WarnCtx logs a warning message with context
func (l *Logger) WarnCtx(ctx context.Context, msg string, args ...any) {
	l.WithContext(ctx).Warn(msg, args...)
}

// DebugCtx logs a debug message with context
func (l *Logger) DebugCtx(ctx context.Context, msg string, args ...any) {
	l.WithContext(ctx).Debug(msg, args...)
}

// ParseLevel parses a log level string
func ParseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
