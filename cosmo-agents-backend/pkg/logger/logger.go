package logger

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// ContextKey is the type for logger context keys
type ContextKey string

const (
	// RequestIDKey is the context key for request ID
	RequestIDKey ContextKey = "request_id"
	// JobIDKey is the context key for job ID
	JobIDKey ContextKey = "job_id"
)

var (
	// Logger is the global logger instance
	Logger zerolog.Logger
)

// Config holds logger configuration
type Config struct {
	Level       string
	Environment string
	Output      io.Writer
}

// Init initializes the global logger
// Matches Python's core/logger functionality
func Init(cfg Config) {
	// Parse log level
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(level)
	zerolog.TimeFieldFormat = time.RFC3339Nano

	// Set output writer
	output := cfg.Output
	if output == nil {
		output = os.Stdout
	}

	// Pretty print for development
	if cfg.Environment == "development" {
		output = zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: "15:04:05",
		}
	}

	Logger = zerolog.New(output).
		With().
		Timestamp().
		Caller().
		Logger()

	log.Logger = Logger
}

// FromContext extracts logger with context fields
// Equivalent to Python's log_context
func FromContext(ctx context.Context) *zerolog.Logger {
	logger := Logger

	// Add request_id if present
	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		if id, ok := requestID.(string); ok {
			logger = logger.With().Str("request_id", id).Logger()
		}
	}

	// Add job_id if present
	if jobID := ctx.Value(JobIDKey); jobID != nil {
		if id, ok := jobID.(string); ok {
			logger = logger.With().Str("job_id", id).Logger()
		}
	}

	return &logger
}

// WithContext creates a new context with logger fields
func WithContext(ctx context.Context, requestID, jobID string) context.Context {
	if requestID != "" {
		ctx = context.WithValue(ctx, RequestIDKey, requestID)
	}
	if jobID != "" {
		ctx = context.WithValue(ctx, JobIDKey, jobID)
	}
	return ctx
}

// Info logs an info message
func Info(msg string) {
	Logger.Info().Msg(msg)
}

// Error logs an error message and returns event for chaining
func Error(err error) *zerolog.Event {
	return Logger.Error().Err(err)
}

// Debug logs a debug message
func Debug(msg string) {
	Logger.Debug().Msg(msg)
}

// Warn logs a warning message
func Warn(msg string) {
	Logger.Warn().Msg(msg)
}

// Fatal logs a fatal message and exits
func Fatal(msg string) {
	Logger.Fatal().Msg(msg)
}

// With creates a child logger with additional fields
func With() zerolog.Context {
	return Logger.With()
}
