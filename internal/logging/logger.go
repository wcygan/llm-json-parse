package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
)

// Logger wraps slog.Logger with additional context methods
type Logger struct {
	*slog.Logger
}

// LogConfig represents logging configuration
type LogConfig struct {
	Level  string
	Format string
	Output io.Writer
}

// NewLogger creates a new structured logger based on configuration
func NewLogger(config LogConfig) *Logger {
	level := parseLogLevel(config.Level)

	var handler slog.Handler
	output := config.Output
	if output == nil {
		output = os.Stdout
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	}

	switch strings.ToLower(config.Format) {
	case "json":
		handler = slog.NewJSONHandler(output, opts)
	case "text":
		handler = slog.NewTextHandler(output, opts)
	default:
		handler = slog.NewJSONHandler(output, opts)
	}

	return &Logger{
		Logger: slog.New(handler),
	}
}

// WithRequestID adds request ID to logger context
func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{
		Logger: l.Logger.With("request_id", requestID),
	}
}

// WithComponent adds component name to logger context
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		Logger: l.Logger.With("component", component),
	}
}

// WithOperation adds operation name to logger context
func (l *Logger) WithOperation(operation string) *Logger {
	return &Logger{
		Logger: l.Logger.With("operation", operation),
	}
}

// WithError adds error to logger context
func (l *Logger) WithError(err error) *Logger {
	if err == nil {
		return l
	}
	return &Logger{
		Logger: l.Logger.With("error", err.Error()),
	}
}

// WithDuration adds duration to logger context
func (l *Logger) WithDuration(duration time.Duration) *Logger {
	return &Logger{
		Logger: l.Logger.With("duration_ms", duration.Milliseconds()),
	}
}

// WithFields adds multiple fields to logger context
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	var args []interface{}
	for k, v := range fields {
		args = append(args, k, v)
	}
	return &Logger{
		Logger: l.Logger.With(args...),
	}
}

// LogRequest logs HTTP request information
func (l *Logger) LogRequest(method, path, userAgent string, startTime time.Time) {
	l.Logger.Info("HTTP request started",
		"method", method,
		"path", path,
		"user_agent", userAgent,
		"start_time", startTime.UTC().Format(time.RFC3339),
	)
}

// LogResponse logs HTTP response information
func (l *Logger) LogResponse(statusCode int, duration time.Duration, size int64) {
	level := slog.LevelInfo
	if statusCode >= 400 {
		level = slog.LevelWarn
	}
	if statusCode >= 500 {
		level = slog.LevelError
	}

	l.Logger.Log(context.Background(), level, "HTTP request completed",
		"status_code", statusCode,
		"duration_ms", duration.Milliseconds(),
		"response_size_bytes", size,
	)
}

// LogCacheOperation logs cache operations
func (l *Logger) LogCacheOperation(operation string, hit bool, key string, size int) {
	l.Logger.Debug("Cache operation",
		"operation", operation,
		"cache_hit", hit,
		"key_hash", key,
		"cache_size", size,
	)
}

// LogValidation logs validation operations
func (l *Logger) LogValidation(schemaSize int, responseSize int, duration time.Duration, success bool) {
	level := slog.LevelInfo
	if !success {
		level = slog.LevelWarn
	}

	l.Logger.Log(context.Background(), level, "Schema validation completed",
		"schema_size_bytes", schemaSize,
		"response_size_bytes", responseSize,
		"validation_duration_ms", duration.Milliseconds(),
		"validation_success", success,
	)
}

// LogLLMRequest logs LLM service requests
func (l *Logger) LogLLMRequest(url string, timeout time.Duration, retryAttempt int) {
	l.Logger.Info("LLM request initiated",
		"llm_url", url,
		"timeout_ms", timeout.Milliseconds(),
		"retry_attempt", retryAttempt,
	)
}

// LogLLMResponse logs LLM service responses
func (l *Logger) LogLLMResponse(statusCode int, responseSize int, duration time.Duration, success bool) {
	level := slog.LevelInfo
	if !success {
		level = slog.LevelError
	}

	l.Logger.Log(context.Background(), level, "LLM request completed",
		"llm_status_code", statusCode,
		"llm_response_size_bytes", responseSize,
		"llm_duration_ms", duration.Milliseconds(),
		"llm_success", success,
	)
}

// LogStartup logs application startup information
func (l *Logger) LogStartup(config map[string]interface{}) {
	l.Logger.Info("Application starting",
		"config", config,
		"startup_time", time.Now().UTC().Format(time.RFC3339),
	)
}

// LogShutdown logs application shutdown information
func (l *Logger) LogShutdown(graceful bool, duration time.Duration) {
	l.Logger.Info("Application shutdown",
		"graceful", graceful,
		"shutdown_duration_ms", duration.Milliseconds(),
		"shutdown_time", time.Now().UTC().Format(time.RFC3339),
	)
}

// parseLogLevel converts string log level to slog.Level
func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
