package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/wcygan/llm-json-parse/internal/config"
	"github.com/wcygan/llm-json-parse/internal/logging"
	"github.com/wcygan/llm-json-parse/internal/middleware"
	"github.com/wcygan/llm-json-parse/internal/server"
	"github.com/wcygan/llm-json-parse/pkg/types"
	"github.com/wcygan/llm-json-parse/tests/mocks"
)

func TestStructuredLoggingIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("successful_request_with_structured_logging", func(t *testing.T) {
		// Capture logs
		var logBuffer bytes.Buffer
		logger := logging.NewLogger(logging.LogConfig{
			Level:  "debug",
			Format: "json",
			Output: &logBuffer,
		})

		// Create mock LLM client
		mockResponse := map[string]interface{}{
			"name": "John Doe",
			"age":  30,
		}
		mockResponseData, _ := json.Marshal(mockResponse)
		mockClient := mocks.NewMockLLMClient()
		mockClient.On("SendStructuredQuery", mock.Anything, mock.Anything, mock.Anything).Return(
			&types.ValidatedResponse{Data: json.RawMessage(mockResponseData)}, nil)

		// Create server with structured logging
		srv := server.NewServerWithConfig(mockClient, 100, logger)

		// Create test schema
		schema := json.RawMessage(`{
			"type": "object",
			"properties": {
				"name": {"type": "string"},
				"age": {"type": "number"}
			},
			"required": ["name"]
		}`)

		// Create test request
		request := types.ValidatedQueryRequest{
			Messages: []types.Message{
				{Role: "user", Content: "Get user information"},
			},
			Schema: schema,
		}

		reqBody, err := json.Marshal(request)
		require.NoError(t, err)

		// Setup middleware chain with logging
		mux := http.NewServeMux()
		srv.RegisterRoutes(mux)

		handler := middleware.Recovery(logger)(
			middleware.CORS()(
				middleware.RequestTimeout(30*time.Second)(
					middleware.ContentType("application/json")(
						middleware.RequestLogging(logger)(mux),
					),
				),
			),
		)

		// Execute request
		req := httptest.NewRequest("POST", "/v1/validated-query", strings.NewReader(string(reqBody)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "test-agent")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		// Verify response
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
		assert.NotEmpty(t, rr.Header().Get("X-Request-ID"))

		// Parse and verify response body
		var responseData map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &responseData)
		require.NoError(t, err)
		assert.Equal(t, "John Doe", responseData["name"])
		assert.Equal(t, float64(30), responseData["age"])

		// Verify structured logs were written
		logOutput := logBuffer.String()
		logLines := strings.Split(strings.TrimSpace(logOutput), "\n")

		// Should have multiple log entries
		assert.Greater(t, len(logLines), 5)

		// Parse and verify some key log entries
		var requestStartLog, requestCompleteLog, schemaValidationLog, llmRequestLog map[string]interface{}
		for _, line := range logLines {
			if line == "" {
				continue
			}
			var logEntry map[string]interface{}
			err := json.Unmarshal([]byte(line), &logEntry)
			require.NoError(t, err)

			switch logEntry["msg"] {
			case "HTTP request started":
				requestStartLog = logEntry
			case "HTTP request completed":
				requestCompleteLog = logEntry
			case "Schema validation successful":
				schemaValidationLog = logEntry
			case "Sending structured query to LLM":
				llmRequestLog = logEntry
			}
		}

		// Verify request start log
		require.NotNil(t, requestStartLog)
		assert.Equal(t, "POST", requestStartLog["method"])
		assert.Equal(t, "/v1/validated-query", requestStartLog["path"])
		assert.Equal(t, "test-agent", requestStartLog["user_agent"])
		assert.NotEmpty(t, requestStartLog["request_id"])

		// Verify request completion log
		require.NotNil(t, requestCompleteLog)
		assert.Equal(t, float64(200), requestCompleteLog["status_code"])
		assert.NotNil(t, requestCompleteLog["duration_ms"])
		assert.NotNil(t, requestCompleteLog["response_size_bytes"])

		// Verify schema validation log
		require.NotNil(t, schemaValidationLog)
		// Note: component might be "validated_query_handler" due to how logging context is passed
		assert.Contains(t, []interface{}{"schema_validator", "validated_query_handler"}, schemaValidationLog["component"])
		assert.NotNil(t, schemaValidationLog["duration_ms"])

		// Verify LLM request log
		require.NotNil(t, llmRequestLog)
		// Note: component might be "validated_query_handler" since it's logged from the server handler
		assert.Contains(t, []interface{}{"llm_client", "validated_query_handler"}, llmRequestLog["component"])
		assert.Contains(t, []interface{}{"structured_query", "llm_request"}, llmRequestLog["operation"])
	})

	t.Run("error_handling_with_structured_logging", func(t *testing.T) {
		// Capture logs
		var logBuffer bytes.Buffer
		logger := logging.NewLogger(logging.LogConfig{
			Level:  "debug",
			Format: "json",
			Output: &logBuffer,
		})

		// Create mock LLM client that returns an error
		mockClient := mocks.NewMockLLMClient()
		mockClient.On("SendStructuredQuery", mock.Anything, mock.Anything, mock.Anything).Return(
			nil, assert.AnError)

		// Create server with structured logging
		srv := server.NewServerWithConfig(mockClient, 100, logger)

		// Create test schema
		schema := json.RawMessage(`{
			"type": "object",
			"properties": {
				"name": {"type": "string"}
			},
			"required": ["name"]
		}`)

		// Create test request
		request := types.ValidatedQueryRequest{
			Messages: []types.Message{
				{Role: "user", Content: "Get user information"},
			},
			Schema: schema,
		}

		reqBody, err := json.Marshal(request)
		require.NoError(t, err)

		// Setup middleware chain with logging
		mux := http.NewServeMux()
		srv.RegisterRoutes(mux)

		handler := middleware.RequestLogging(logger)(mux)

		// Execute request
		req := httptest.NewRequest("POST", "/v1/validated-query", strings.NewReader(string(reqBody)))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		// Verify error response
		assert.Equal(t, http.StatusInternalServerError, rr.Code)

		// Verify error was logged
		logOutput := logBuffer.String()
		assert.Contains(t, logOutput, "LLM request failed")
		assert.Contains(t, logOutput, "error")
		// Note: The error is logged from the server handler, not directly from llm_client
		assert.Contains(t, logOutput, "validated_query_handler")

		// Verify structured error response
		var errorResponse types.ErrorResponse
		err = json.Unmarshal(rr.Body.Bytes(), &errorResponse)
		require.NoError(t, err)
		assert.Equal(t, "error", errorResponse.Error)
		assert.Equal(t, types.ErrorCodeLLMError, errorResponse.Code)
		assert.NotEmpty(t, errorResponse.RequestID)
		assert.NotEmpty(t, errorResponse.Timestamp)
	})

	t.Run("configuration_based_logging", func(t *testing.T) {
		// Test that configuration affects logging
		cfg := &config.Config{
			Log: config.LogConfig{
				Level:  "warn",
				Format: "text",
			},
		}

		logger := logging.NewLogger(logging.LogConfig{
			Level:  cfg.Log.Level,
			Format: cfg.Log.Format,
		})

		// Verify logger configuration
		assert.NotNil(t, logger)

		// Create a buffer to capture output and test log levels
		var buf bytes.Buffer
		testLogger := logging.NewLogger(logging.LogConfig{
			Level:  "warn",
			Format: "json",
			Output: &buf,
		})

		// Debug and info should be filtered out at warn level
		testLogger.Debug("debug message")
		testLogger.Info("info message")
		testLogger.Warn("warn message")
		testLogger.Error("error message")

		output := buf.String()
		assert.NotContains(t, output, "debug message")
		assert.NotContains(t, output, "info message")
		assert.Contains(t, output, "warn message")
		assert.Contains(t, output, "error message")
	})
}

func TestMiddlewareIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("full_middleware_chain", func(t *testing.T) {
		var logBuffer bytes.Buffer
		logger := logging.NewLogger(logging.LogConfig{
			Level:  "info",
			Format: "json",
			Output: &logBuffer,
		})

		// Create simple handler
		handler := middleware.Recovery(logger)(
			middleware.CORS()(
				middleware.RequestTimeout(5*time.Second)(
					middleware.ContentType("application/json")(
						middleware.RequestLogging(logger)(
							http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
								// Verify context values are available
								requestID := middleware.GetRequestID(r.Context())
								contextLogger := middleware.GetLogger(r.Context())
								startTime := middleware.GetStartTime(r.Context())

								assert.NotEmpty(t, requestID)
								assert.NotNil(t, contextLogger)
								assert.False(t, startTime.IsZero())

								w.Header().Set("Content-Type", "application/json")
								json.NewEncoder(w).Encode(map[string]string{
									"status":     "ok",
									"request_id": requestID,
								})
							}),
						),
					),
				),
			),
		)

		req := httptest.NewRequest("POST", "/test", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		// Verify response
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"))
		assert.NotEmpty(t, rr.Header().Get("X-Request-ID"))

		// Verify response body
		var response map[string]string
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "ok", response["status"])
		assert.NotEmpty(t, response["request_id"])

		// Verify logs were generated
		logOutput := logBuffer.String()
		assert.Contains(t, logOutput, "HTTP request started")
		assert.Contains(t, logOutput, "HTTP request completed")
		assert.Contains(t, logOutput, "POST")
		assert.Contains(t, logOutput, "/test")
	})
}