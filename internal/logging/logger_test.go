package logging

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	t.Run("json_format_logger", func(t *testing.T) {
		var buf bytes.Buffer
		config := LogConfig{
			Level:  "info",
			Format: "json",
			Output: &buf,
		}

		logger := NewLogger(config)
		logger.Info("test message", "key", "value")

		output := buf.String()
		assert.Contains(t, output, "test message")
		assert.Contains(t, output, "\"key\":\"value\"")

		// Verify it's valid JSON
		var logEntry map[string]interface{}
		err := json.Unmarshal([]byte(output), &logEntry)
		assert.NoError(t, err)
	})

	t.Run("text_format_logger", func(t *testing.T) {
		var buf bytes.Buffer
		config := LogConfig{
			Level:  "debug",
			Format: "text",
			Output: &buf,
		}

		logger := NewLogger(config)
		logger.Debug("debug message", "component", "test")

		output := buf.String()
		assert.Contains(t, output, "debug message")
		assert.Contains(t, output, "component=test")
	})

	t.Run("log_levels", func(t *testing.T) {
		var buf bytes.Buffer
		config := LogConfig{
			Level:  "warn",
			Format: "json",
			Output: &buf,
		}

		logger := NewLogger(config)

		// Debug and info should be filtered out
		logger.Debug("debug message")
		logger.Info("info message")

		// Warn should appear
		logger.Warn("warn message")

		output := buf.String()
		assert.NotContains(t, output, "debug message")
		assert.NotContains(t, output, "info message")
		assert.Contains(t, output, "warn message")
	})
}

func TestLoggerWithContext(t *testing.T) {
	t.Run("with_request_id", func(t *testing.T) {
		var buf bytes.Buffer
		config := LogConfig{
			Level:  "info",
			Format: "json",
			Output: &buf,
		}

		logger := NewLogger(config)
		contextLogger := logger.WithRequestID("req-123")
		contextLogger.Info("test message")

		output := buf.String()
		assert.Contains(t, output, "req-123")
		assert.Contains(t, output, "request_id")
	})

	t.Run("with_component", func(t *testing.T) {
		var buf bytes.Buffer
		config := LogConfig{
			Level:  "info",
			Format: "json",
			Output: &buf,
		}

		logger := NewLogger(config)
		contextLogger := logger.WithComponent("validator")
		contextLogger.Info("validation started")

		output := buf.String()
		assert.Contains(t, output, "validator")
		assert.Contains(t, output, "component")
	})

	t.Run("with_operation", func(t *testing.T) {
		var buf bytes.Buffer
		config := LogConfig{
			Level:  "info",
			Format: "json",
			Output: &buf,
		}

		logger := NewLogger(config)
		contextLogger := logger.WithOperation("schema_validation")
		contextLogger.Info("operation completed")

		output := buf.String()
		assert.Contains(t, output, "schema_validation")
		assert.Contains(t, output, "operation")
	})

	t.Run("with_error", func(t *testing.T) {
		var buf bytes.Buffer
		config := LogConfig{
			Level:  "info",
			Format: "json",
			Output: &buf,
		}

		logger := NewLogger(config)
		err := assert.AnError
		contextLogger := logger.WithError(err)
		contextLogger.Error("operation failed")

		output := buf.String()
		assert.Contains(t, output, err.Error())
		assert.Contains(t, output, "error")
	})

	t.Run("with_duration", func(t *testing.T) {
		var buf bytes.Buffer
		config := LogConfig{
			Level:  "info",
			Format: "json",
			Output: &buf,
		}

		logger := NewLogger(config)
		duration := 150 * time.Millisecond
		contextLogger := logger.WithDuration(duration)
		contextLogger.Info("operation completed")

		output := buf.String()
		assert.Contains(t, output, "150")
		assert.Contains(t, output, "duration_ms")
	})

	t.Run("with_fields", func(t *testing.T) {
		var buf bytes.Buffer
		config := LogConfig{
			Level:  "info",
			Format: "json",
			Output: &buf,
		}

		logger := NewLogger(config)
		fields := map[string]interface{}{
			"user_id":    123,
			"session_id": "sess-456",
			"action":     "login",
		}
		contextLogger := logger.WithFields(fields)
		contextLogger.Info("user action")

		output := buf.String()
		assert.Contains(t, output, "123")
		assert.Contains(t, output, "sess-456")
		assert.Contains(t, output, "login")
	})
}

func TestSpecializedLoggingMethods(t *testing.T) {
	t.Run("log_request", func(t *testing.T) {
		var buf bytes.Buffer
		config := LogConfig{
			Level:  "info",
			Format: "json",
			Output: &buf,
		}

		logger := NewLogger(config)
		startTime := time.Now()
		logger.LogRequest("POST", "/v1/validated-query", "curl/7.68.0", startTime)

		output := buf.String()
		assert.Contains(t, output, "POST")
		assert.Contains(t, output, "/v1/validated-query")
		assert.Contains(t, output, "curl/7.68.0")
		assert.Contains(t, output, "HTTP request started")
	})

	t.Run("log_response", func(t *testing.T) {
		var buf bytes.Buffer
		config := LogConfig{
			Level:  "info",
			Format: "json",
			Output: &buf,
		}

		logger := NewLogger(config)
		duration := 100 * time.Millisecond
		logger.LogResponse(200, duration, 1024)

		output := buf.String()
		assert.Contains(t, output, "200")
		assert.Contains(t, output, "100")
		assert.Contains(t, output, "1024")
		assert.Contains(t, output, "HTTP request completed")
	})

	t.Run("log_response_error_levels", func(t *testing.T) {
		var buf bytes.Buffer
		config := LogConfig{
			Level:  "info",
			Format: "json",
			Output: &buf,
		}

		logger := NewLogger(config)

		// Test different status codes result in different log levels
		logger.LogResponse(500, time.Millisecond, 100) // Should be ERROR level
		output := buf.String()
		assert.Contains(t, output, "500")

		buf.Reset()
		logger.LogResponse(404, time.Millisecond, 100) // Should be WARN level
		output = buf.String()
		assert.Contains(t, output, "404")
	})

	t.Run("log_cache_operation", func(t *testing.T) {
		var buf bytes.Buffer
		config := LogConfig{
			Level:  "debug",
			Format: "json",
			Output: &buf,
		}

		logger := NewLogger(config)
		logger.LogCacheOperation("GET", true, "schema-hash-123", 50)

		output := buf.String()
		assert.Contains(t, output, "GET")
		assert.Contains(t, output, "true")
		assert.Contains(t, output, "schema-hash-123")
		assert.Contains(t, output, "50")
		assert.Contains(t, output, "Cache operation")
	})

	t.Run("log_validation", func(t *testing.T) {
		var buf bytes.Buffer
		config := LogConfig{
			Level:  "info",
			Format: "json",
			Output: &buf,
		}

		logger := NewLogger(config)
		duration := 25 * time.Millisecond
		logger.LogValidation(2048, 512, duration, true)

		output := buf.String()
		assert.Contains(t, output, "2048")
		assert.Contains(t, output, "512")
		assert.Contains(t, output, "25")
		assert.Contains(t, output, "true")
		assert.Contains(t, output, "Schema validation completed")
	})

	t.Run("log_llm_request", func(t *testing.T) {
		var buf bytes.Buffer
		config := LogConfig{
			Level:  "info",
			Format: "json",
			Output: &buf,
		}

		logger := NewLogger(config)
		timeout := 30 * time.Second
		logger.LogLLMRequest("http://localhost:8080", timeout, 1)

		output := buf.String()
		assert.Contains(t, output, "http://localhost:8080")
		assert.Contains(t, output, "30000")
		assert.Contains(t, output, "1")
		assert.Contains(t, output, "LLM request initiated")
	})

	t.Run("log_llm_response", func(t *testing.T) {
		var buf bytes.Buffer
		config := LogConfig{
			Level:  "info",
			Format: "json",
			Output: &buf,
		}

		logger := NewLogger(config)
		duration := 500 * time.Millisecond
		logger.LogLLMResponse(200, 1024, duration, true)

		output := buf.String()
		assert.Contains(t, output, "200")
		assert.Contains(t, output, "1024")
		assert.Contains(t, output, "500")
		assert.Contains(t, output, "true")
		assert.Contains(t, output, "LLM request completed")
	})

	t.Run("log_startup", func(t *testing.T) {
		var buf bytes.Buffer
		config := LogConfig{
			Level:  "info",
			Format: "json",
			Output: &buf,
		}

		logger := NewLogger(config)
		appConfig := map[string]interface{}{
			"port":      8081,
			"log_level": "info",
		}
		logger.LogStartup(appConfig)

		output := buf.String()
		assert.Contains(t, output, "8081")
		assert.Contains(t, output, "info")
		assert.Contains(t, output, "Application starting")
	})

	t.Run("log_shutdown", func(t *testing.T) {
		var buf bytes.Buffer
		config := LogConfig{
			Level:  "info",
			Format: "json",
			Output: &buf,
		}

		logger := NewLogger(config)
		duration := 2 * time.Second
		logger.LogShutdown(true, duration)

		output := buf.String()
		assert.Contains(t, output, "true")
		assert.Contains(t, output, "2000")
		assert.Contains(t, output, "Application shutdown")
	})
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"debug", "DEBUG"},
		{"DEBUG", "DEBUG"},
		{"Info", "INFO"},
		{"warn", "WARN"},
		{"WARNING", "WARN"},
		{"error", "ERROR"},
		{"invalid", "INFO"}, // Should default to INFO
		{"", "INFO"},        // Should default to INFO
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			level := parseLogLevel(test.input)
			assert.Equal(t, test.expected, level.String())
		})
	}
}

func TestLoggerChaining(t *testing.T) {
	t.Run("multiple_context_methods", func(t *testing.T) {
		var buf bytes.Buffer
		config := LogConfig{
			Level:  "info",
			Format: "json",
			Output: &buf,
		}

		logger := NewLogger(config)
		contextLogger := logger.
			WithRequestID("req-123").
			WithComponent("validator").
			WithOperation("schema_check").
			WithDuration(50 * time.Millisecond)

		contextLogger.Info("complex operation completed")

		output := buf.String()
		assert.Contains(t, output, "req-123")
		assert.Contains(t, output, "validator")
		assert.Contains(t, output, "schema_check")
		assert.Contains(t, output, "50")

		// Verify it's still valid JSON
		var logEntry map[string]interface{}
		err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
		require.NoError(t, err)
	})
}
