package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"

	"github.com/wcygan/llm-json-parse/internal/client"
	"github.com/wcygan/llm-json-parse/internal/schema"
	"github.com/wcygan/llm-json-parse/pkg/types"
)

type Server struct {
	llmClient client.LLMClient
	validator *schema.Validator
}

func NewServer(llmClient client.LLMClient) *Server {
	return &Server{
		llmClient: llmClient,
		validator: schema.NewValidator(),
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
	requestID := s.generateRequestID()

	var req types.ValidatedQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, types.ErrorCodeInvalidRequest,
			"Invalid request body", err.Error(), requestID)
		return
	}

	if err := s.validator.ValidateSchema(req.Schema); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema,
			"Invalid JSON schema", err.Error(), requestID)
		return
	}

	response, err := s.llmClient.SendStructuredQuery(r.Context(), req.Messages, req.Schema)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, types.ErrorCodeLLMError,
			"LLM service error", err.Error(), requestID)
		return
	}

	if err := s.validator.ValidateResponse(req.Schema, response); err != nil {
		s.writeValidationError(w, "Schema validation failed", err.Error(), response.Data, requestID)
		return
	}

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
func (s *Server) writeErrorResponse(w http.ResponseWriter, status int, code, message, details string, requestID string) {
	errorResp := types.NewErrorResponse(code, message, details).WithRequestID(requestID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorResp)

	log.Printf("Error [%s]: %s - %s (%s)", requestID, message, details, code)
}

// writeValidationError writes a standardized validation error response
func (s *Server) writeValidationError(w http.ResponseWriter, message, details string, responseData json.RawMessage, requestID string) {
	validationErr := types.NewValidationError(message, details, responseData).
		WithValidationContext("endpoint", "/v1/validated-query")

	if requestID != "" {
		validationErr.RequestID = requestID
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	json.NewEncoder(w).Encode(validationErr)

	log.Printf("Validation Error [%s]: %s - %s", requestID, message, details)
}
