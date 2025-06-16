package server

import (
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
	var req types.ValidatedQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if err := s.validator.ValidateSchema(req.Schema); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid schema", err.Error())
		return
	}

	response, err := s.llmClient.SendStructuredQuery(r.Context(), req.Messages, req.Schema)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "LLM error", err.Error())
		return
	}

	if err := s.validator.ValidateResponse(req.Schema, response); err != nil {
		validationErr := types.ValidationError{
			Error:    "Schema validation failed",
			Details:  err.Error(),
			Response: response,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(validationErr)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) writeError(w http.ResponseWriter, status int, message, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error":   message,
		"details": details,
	})
	log.Printf("Error: %s - %s", message, details)
}
