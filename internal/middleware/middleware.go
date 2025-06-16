package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/wcygan/llm-json-parse/internal/logging"
)

// ContextKey represents keys for context values
type ContextKey string

const (
	// ContextKeyRequestID is the context key for request ID
	ContextKeyRequestID ContextKey = "request_id"
	// ContextKeyLogger is the context key for the logger
	ContextKeyLogger ContextKey = "logger"
	// ContextKeyStartTime is the context key for request start time
	ContextKeyStartTime ContextKey = "start_time"
)

// responseWriter wraps http.ResponseWriter to capture response details
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int64
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.size += int64(n)
	return n, err
}

// RequestLogging creates a middleware that logs HTTP requests and responses
func RequestLogging(logger *logging.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Generate request ID if not present
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = generateRequestID()
			}

			// Create request-scoped logger
			requestLogger := logger.
				WithRequestID(requestID).
				WithComponent("http_server")

			// Record start time
			startTime := time.Now()

			// Add context values
			ctx := context.WithValue(r.Context(), ContextKeyRequestID, requestID)
			ctx = context.WithValue(ctx, ContextKeyLogger, requestLogger)
			ctx = context.WithValue(ctx, ContextKeyStartTime, startTime)
			r = r.WithContext(ctx)

			// Log request
			requestLogger.LogRequest(r.Method, r.URL.Path, r.UserAgent(), startTime)

			// Wrap response writer
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     200, // Default status code
			}

			// Add request ID to response headers
			rw.Header().Set("X-Request-ID", requestID)

			// Call next handler
			next.ServeHTTP(rw, r)

			// Calculate duration and log response
			duration := time.Since(startTime)
			requestLogger.LogResponse(rw.statusCode, duration, rw.size)
		})
	}
}

// Recovery creates a middleware that recovers from panics
func Recovery(logger *logging.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Get request-scoped logger if available
					requestLogger := logger
					if ctxLogger, ok := r.Context().Value(ContextKeyLogger).(*logging.Logger); ok {
						requestLogger = ctxLogger
					}

					requestLogger.
						WithComponent("recovery_middleware").
						WithFields(map[string]interface{}{
							"panic_value": err,
							"method":      r.Method,
							"path":        r.URL.Path,
						}).
						Error("Panic recovered in HTTP handler")

					// Return internal server error
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// CORS creates a middleware that handles CORS headers
func CORS() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Request-ID")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequestTimeout creates a middleware that enforces request timeouts
func RequestTimeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

// ContentType creates a middleware that validates content type for specific methods
func ContentType(requiredTypes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only check content type for methods that have a body
			if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
				contentType := r.Header.Get("Content-Type")

				valid := false
				for _, reqType := range requiredTypes {
					if contentType == reqType {
						valid = true
						break
					}
				}

				if !valid {
					// Get request-scoped logger if available
					if ctxLogger, ok := r.Context().Value(ContextKeyLogger).(*logging.Logger); ok {
						ctxLogger.
							WithComponent("content_type_middleware").
							WithFields(map[string]interface{}{
								"received_content_type":  contentType,
								"required_content_types": requiredTypes,
							}).
							Warn("Invalid content type")
					}

					http.Error(w, "Unsupported Media Type", http.StatusUnsupportedMediaType)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetRequestID retrieves request ID from context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(ContextKeyRequestID).(string); ok {
		return requestID
	}
	return ""
}

// GetLogger retrieves logger from context
func GetLogger(ctx context.Context) *logging.Logger {
	if logger, ok := ctx.Value(ContextKeyLogger).(*logging.Logger); ok {
		return logger
	}
	return nil
}

// GetStartTime retrieves request start time from context
func GetStartTime(ctx context.Context) time.Time {
	if startTime, ok := ctx.Value(ContextKeyStartTime).(time.Time); ok {
		return startTime
	}
	return time.Time{}
}

// generateRequestID creates a simple request ID
func generateRequestID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}
