package logger

import (
	"context"
	"log/slog"
	"os"
)

// Logger wraps slog for structured logging
// This allows us to add custom functionality and swap implementations if needed
type Logger struct {
	*slog.Logger
}

// New creates a new logger instance
// In production, you might want to use JSON format for log aggregation tools
func New(level string) *Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	// Use JSON handler for structured logs
	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)

	return &Logger{Logger: logger}
}

// WithContext adds context values to the logger
// This is useful for adding request IDs, user IDs, etc.
func (l *Logger) WithContext(ctx context.Context) *Logger {
	// Extract request ID from context if available
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return &Logger{Logger: l.With("request_id", requestID)}
	}
	return l
}

// WithFields adds additional fields to the logger
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	args := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return &Logger{Logger: l.With(args...)}
}
