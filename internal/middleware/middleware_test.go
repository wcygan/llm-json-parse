package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wcygan/llm-json-parse/internal/logging"
)

func TestRequestLogging(t *testing.T) {
	t.Run("logs_request_and_response", func(t *testing.T) {
		var buf bytes.Buffer
		logger := logging.NewLogger(logging.LogConfig{
			Level:  "info",
			Format: "json",
			Output: &buf,
		})

		handler := RequestLogging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("test response"))
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "test-agent")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		output := buf.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")

		// Should have 2 log lines: request start and request completed
		assert.Len(t, lines, 2)

		// Check request log
		var requestLog map[string]interface{}
		err := json.Unmarshal([]byte(lines[0]), &requestLog)
		require.NoError(t, err)
		assert.Equal(t, "HTTP request started", requestLog["msg"])
		assert.Equal(t, "GET", requestLog["method"])
		assert.Equal(t, "/test", requestLog["path"])
		assert.Equal(t, "test-agent", requestLog["user_agent"])

		// Check response log
		var responseLog map[string]interface{}
		err = json.Unmarshal([]byte(lines[1]), &responseLog)
		require.NoError(t, err)
		assert.Equal(t, "HTTP request completed", responseLog["msg"])
		assert.Equal(t, float64(200), responseLog["status_code"])
		assert.Equal(t, float64(13), responseLog["response_size_bytes"]) // "test response"
	})

	t.Run("adds_request_id_to_context", func(t *testing.T) {
		var buf bytes.Buffer
		logger := logging.NewLogger(logging.LogConfig{
			Level:  "info",
			Format: "json",
			Output: &buf,
		})

		var capturedRequestID string
		handler := RequestLogging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedRequestID = GetRequestID(r.Context())
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.NotEmpty(t, capturedRequestID)
		assert.Equal(t, capturedRequestID, rr.Header().Get("X-Request-ID"))
	})

	t.Run("uses_existing_request_id", func(t *testing.T) {
		var buf bytes.Buffer
		logger := logging.NewLogger(logging.LogConfig{
			Level:  "info",
			Format: "json",
			Output: &buf,
		})

		existingID := "existing-req-123"
		var capturedRequestID string
		handler := RequestLogging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedRequestID = GetRequestID(r.Context())
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Request-ID", existingID)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, existingID, capturedRequestID)
		assert.Equal(t, existingID, rr.Header().Get("X-Request-ID"))
	})
}

func TestRecovery(t *testing.T) {
	t.Run("recovers_from_panic", func(t *testing.T) {
		var buf bytes.Buffer
		logger := logging.NewLogger(logging.LogConfig{
			Level:  "info",
			Format: "json",
			Output: &buf,
		})

		handler := Recovery(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "Internal Server Error")

		output := buf.String()
		assert.Contains(t, output, "Panic recovered in HTTP handler")
		assert.Contains(t, output, "test panic")
	})

	t.Run("continues_normal_execution", func(t *testing.T) {
		var buf bytes.Buffer
		logger := logging.NewLogger(logging.LogConfig{
			Level:  "info",
			Format: "json",
			Output: &buf,
		})

		handler := Recovery(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "success", rr.Body.String())

		// Should not have any error logs
		output := buf.String()
		assert.Empty(t, output)
	})
}

func TestCORS(t *testing.T) {
	t.Run("adds_cors_headers", func(t *testing.T) {
		handler := CORS()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t, rr.Header().Get("Access-Control-Allow-Methods"), "GET")
		assert.Contains(t, rr.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
	})

	t.Run("handles_options_request", func(t *testing.T) {
		handler := CORS()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Should not call next handler for OPTIONS")
		}))

		req := httptest.NewRequest("OPTIONS", "/test", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})
}

func TestRequestTimeout(t *testing.T) {
	t.Run("enforces_timeout", func(t *testing.T) {
		timeout := 50 * time.Millisecond
		handler := RequestTimeout(timeout)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate slow operation
			select {
			case <-time.After(200 * time.Millisecond):
				w.WriteHeader(http.StatusOK)
			case <-r.Context().Done():
				// Context cancelled due to timeout
				return
			}
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()

		start := time.Now()
		handler.ServeHTTP(rr, req)
		duration := time.Since(start)

		// Should complete quickly due to timeout (add some buffer for test execution)
		assert.Less(t, duration, 150*time.Millisecond)
	})

	t.Run("allows_fast_requests", func(t *testing.T) {
		timeout := 100 * time.Millisecond
		handler := RequestTimeout(timeout)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "success", rr.Body.String())
	})
}

func TestContentType(t *testing.T) {
	t.Run("accepts_valid_content_type", func(t *testing.T) {
		handler := ContentType("application/json")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		}))

		req := httptest.NewRequest("POST", "/test", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "success", rr.Body.String())
	})

	t.Run("rejects_invalid_content_type", func(t *testing.T) {
		var buf bytes.Buffer
		logger := logging.NewLogger(logging.LogConfig{
			Level:  "info",
			Format: "json",
			Output: &buf,
		})

		handler := RequestLogging(logger)(ContentType("application/json")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Should not call next handler")
		})))

		req := httptest.NewRequest("POST", "/test", strings.NewReader("data"))
		req.Header.Set("Content-Type", "text/plain")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnsupportedMediaType, rr.Code)
		assert.Contains(t, rr.Body.String(), "Unsupported Media Type")
	})

	t.Run("ignores_get_requests", func(t *testing.T) {
		handler := ContentType("application/json")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		// No Content-Type header set
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "success", rr.Body.String())
	})

	t.Run("accepts_multiple_content_types", func(t *testing.T) {
		handler := ContentType("application/json", "application/xml")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// Test JSON
		req := httptest.NewRequest("POST", "/test", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)

		// Test XML
		req = httptest.NewRequest("POST", "/test", strings.NewReader("<xml/>"))
		req.Header.Set("Content-Type", "application/xml")
		rr = httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	})
}

func TestContextHelpers(t *testing.T) {
	t.Run("get_request_id", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), ContextKeyRequestID, "test-123")
		requestID := GetRequestID(ctx)
		assert.Equal(t, "test-123", requestID)

		// Test with empty context
		emptyRequestID := GetRequestID(context.Background())
		assert.Empty(t, emptyRequestID)
	})

	t.Run("get_logger", func(t *testing.T) {
		logger := logging.NewLogger(logging.LogConfig{Level: "info", Format: "json"})
		ctx := context.WithValue(context.Background(), ContextKeyLogger, logger)

		contextLogger := GetLogger(ctx)
		assert.NotNil(t, contextLogger)

		// Test with empty context
		emptyLogger := GetLogger(context.Background())
		assert.Nil(t, emptyLogger)
	})

	t.Run("get_start_time", func(t *testing.T) {
		startTime := time.Now()
		ctx := context.WithValue(context.Background(), ContextKeyStartTime, startTime)

		contextStartTime := GetStartTime(ctx)
		assert.Equal(t, startTime, contextStartTime)

		// Test with empty context
		emptyStartTime := GetStartTime(context.Background())
		assert.True(t, emptyStartTime.IsZero())
	})
}

func TestGenerateRequestID(t *testing.T) {
	t.Run("generates_unique_ids", func(t *testing.T) {
		id1 := generateRequestID()
		// Small delay to ensure different timestamp
		time.Sleep(1 * time.Millisecond)
		id2 := generateRequestID()

		assert.NotEmpty(t, id1)
		assert.NotEmpty(t, id2)
		assert.NotEqual(t, id1, id2)
	})
}

func TestMiddlewareChaining(t *testing.T) {
	t.Run("middleware_chain_works_correctly", func(t *testing.T) {
		var buf bytes.Buffer
		logger := logging.NewLogger(logging.LogConfig{
			Level:  "info",
			Format: "json",
			Output: &buf,
		})

		finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify all middleware context is available
			requestID := GetRequestID(r.Context())
			contextLogger := GetLogger(r.Context())
			startTime := GetStartTime(r.Context())

			assert.NotEmpty(t, requestID)
			assert.NotNil(t, contextLogger)
			assert.False(t, startTime.IsZero())

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		})

		// Chain multiple middleware
		handler := Recovery(logger)(
			CORS()(
				RequestTimeout(1 * time.Second)(
					ContentType("application/json")(
						RequestLogging(logger)(finalHandler),
					),
				),
			),
		)

		req := httptest.NewRequest("POST", "/test", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "success", rr.Body.String())
		assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"))
		assert.NotEmpty(t, rr.Header().Get("X-Request-ID"))
	})
}
