package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/wcygan/llm-json-parse/internal/client"
	"github.com/wcygan/llm-json-parse/internal/logging"
	"github.com/wcygan/llm-json-parse/internal/middleware"
	"github.com/wcygan/llm-json-parse/internal/schema"
	"github.com/wcygan/llm-json-parse/pkg/types"
)

type Server struct {
	llmClient client.LLMClient
	validator *schema.Validator
	logger    *logging.Logger
}

func NewServer(llmClient client.LLMClient) *Server {
	return &Server{
		llmClient: llmClient,
		validator: schema.NewValidator(),
		logger:    logging.NewLogger(logging.LogConfig{Level: "info", Format: "json"}),
	}
}

// NewServerWithCacheSize creates a server with custom schema cache size
func NewServerWithCacheSize(llmClient client.LLMClient, cacheSize int) *Server {
	return &Server{
		llmClient: llmClient,
		validator: schema.NewValidatorWithCacheSize(cacheSize),
		logger:    logging.NewLogger(logging.LogConfig{Level: "info", Format: "json"}),
	}
}

// NewServerWithConfig creates a server with full configuration
func NewServerWithConfig(llmClient client.LLMClient, cacheSize int, logger *logging.Logger) *Server {
	return &Server{
		llmClient: llmClient,
		validator: schema.NewValidatorWithCacheSize(cacheSize),
		logger:    logger,
	}
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/validated-query", s.handleValidatedQuery)
	mux.HandleFunc("GET /health", s.handleHealth)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (s *Server) handleValidatedQuery(w http.ResponseWriter, r *http.Request) {
	// Get request-scoped logger and request ID from middleware
	requestLogger := middleware.GetLogger(r.Context())
	if requestLogger == nil {
		requestLogger = s.logger
	}
	requestID := middleware.GetRequestID(r.Context())
	if requestID == "" {
		requestID = s.generateRequestID()
	}

	requestLogger = requestLogger.WithComponent("validated_query_handler")

	var req types.ValidatedQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		requestLogger.WithError(err).Warn("Failed to decode request body")
		s.writeErrorResponse(w, http.StatusBadRequest, types.ErrorCodeInvalidRequest,
			"Invalid request body", err.Error(), requestID, requestLogger)
		return
	}

	// Validate schema
	schemaValidationStart := time.Now()
	if err := s.validator.ValidateSchema(req.Schema); err != nil {
		requestLogger.WithError(err).WithDuration(time.Since(schemaValidationStart)).Warn("Schema validation failed")
		s.writeErrorResponse(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema,
			"Invalid JSON schema", err.Error(), requestID, requestLogger)
		return
	}
	requestLogger.WithDuration(time.Since(schemaValidationStart)).Debug("Schema validation successful")

	// Send LLM request
	llmRequestStart := time.Now()
	requestLogger.WithOperation("llm_request").Info("Sending structured query to LLM")
	response, err := s.llmClient.SendStructuredQuery(r.Context(), req.Messages, req.Schema)
	llmDuration := time.Since(llmRequestStart)

	if err != nil {
		requestLogger.WithError(err).WithDuration(llmDuration).Error("LLM request failed")
		s.writeErrorResponse(w, http.StatusInternalServerError, types.ErrorCodeLLMError,
			"LLM service error", err.Error(), requestID, requestLogger)
		return
	}
	requestLogger.WithDuration(llmDuration).WithFields(map[string]interface{}{
		"response_size_bytes": len(response.Data),
	}).Info("LLM request successful")

	// Validate response
	responseValidationStart := time.Now()
	if err := s.validator.ValidateResponse(req.Schema, response); err != nil {
		validationDuration := time.Since(responseValidationStart)
		requestLogger.WithError(err).WithDuration(validationDuration).Warn("Response validation failed")
		s.writeValidationError(w, "Schema validation failed", err.Error(), response.Data, requestID, requestLogger)
		return
	}
	validationDuration := time.Since(responseValidationStart)
	requestLogger.WithDuration(validationDuration).Debug("Response validation successful")

	// Success - return validated response
	requestLogger.WithFields(map[string]interface{}{
		"total_duration_ms": time.Since(middleware.GetStartTime(r.Context())).Milliseconds(),
	}).Info("Validated query completed successfully")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response.Data)
}

// generateRequestID creates a unique request identifier
func (s *Server) generateRequestID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// writeErrorResponse writes a standardized error response
func (s *Server) writeErrorResponse(w http.ResponseWriter, status int, code, message, details string, requestID string, logger *logging.Logger) {
	errorResp := types.NewErrorResponse(code, message, details).WithRequestID(requestID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorResp)

	if logger != nil {
		logger.WithFields(map[string]interface{}{
			"error_code":    code,
			"status_code":   status,
			"error_details": details,
		}).Error(message)
	}
}

// writeValidationError writes a standardized validation error response
func (s *Server) writeValidationError(w http.ResponseWriter, message, details string, responseData json.RawMessage, requestID string, logger *logging.Logger) {
	validationErr := types.NewValidationError(message, details, responseData).
		WithValidationContext("endpoint", "/v1/validated-query")

	if requestID != "" {
		validationErr.RequestID = requestID
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	json.NewEncoder(w).Encode(validationErr)

	if logger != nil {
		logger.WithFields(map[string]interface{}{
			"status_code":        http.StatusUnprocessableEntity,
			"validation_details": details,
			"response_size":      len(responseData),
		}).Warn(message)
	}
}
