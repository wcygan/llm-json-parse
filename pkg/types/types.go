package types

import (
	"encoding/json"
	"time"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ValidatedQueryRequest struct {
	Schema   json.RawMessage `json:"schema"`
	Messages []Message       `json:"messages"`
}

type LLMRequest struct {
	Messages       []Message       `json:"messages"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

type ResponseFormat struct {
	Type       string     `json:"type"`
	JSONSchema JSONSchema `json:"json_schema"`
}

type JSONSchema struct {
	Name   string          `json:"name"`
	Strict bool            `json:"strict"`
	Schema json.RawMessage `json:"schema"`
}

type LLMResponse struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message Message `json:"message"`
}

// ValidatedResponse represents a structured response from LLM validation
type ValidatedResponse struct {
	Data     json.RawMessage   `json:"data"`
	Metadata *ResponseMetadata `json:"metadata,omitempty"`
}

// ResponseMetadata contains optional metadata about the validation
type ResponseMetadata struct {
	SchemaHash     string `json:"schema_hash,omitempty"`
	ValidationTime string `json:"validation_time,omitempty"`
}

// ErrorResponse provides standardized error information across all endpoints
type ErrorResponse struct {
	Error     string                 `json:"error"`
	Message   string                 `json:"message"`
	Code      string                 `json:"code"`
	Details   string                 `json:"details,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Timestamp string                 `json:"timestamp"`
	RequestID string                 `json:"request_id,omitempty"`
}

// ValidationError represents schema validation failures with response data
type ValidationError struct {
	Error     string                 `json:"error"`
	Message   string                 `json:"message"`
	Code      string                 `json:"code"`
	Details   string                 `json:"details"`
	Response  json.RawMessage        `json:"response,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Timestamp string                 `json:"timestamp"`
	RequestID string                 `json:"request_id,omitempty"`
}

// Error codes for consistent error handling
const (
	ErrorCodeInvalidRequest   = "INVALID_REQUEST"
	ErrorCodeInvalidSchema    = "INVALID_SCHEMA"
	ErrorCodeLLMError         = "LLM_ERROR"
	ErrorCodeValidationFailed = "VALIDATION_FAILED"
	ErrorCodeInternalError    = "INTERNAL_ERROR"
	ErrorCodeTimeout          = "TIMEOUT"
	ErrorCodeRateLimited      = "RATE_LIMITED"
)

// NewErrorResponse creates a standardized error response
func NewErrorResponse(code, message, details string) *ErrorResponse {
	return &ErrorResponse{
		Error:     "error",
		Message:   message,
		Code:      code,
		Details:   details,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// WithContext adds context information to an error response
func (e *ErrorResponse) WithContext(key string, value interface{}) *ErrorResponse {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithRequestID adds a request ID to an error response
func (e *ErrorResponse) WithRequestID(requestID string) *ErrorResponse {
	e.RequestID = requestID
	return e
}

// NewValidationError creates a standardized validation error with response data
func NewValidationError(message, details string, responseData json.RawMessage) *ValidationError {
	return &ValidationError{
		Error:     "validation_error",
		Message:   message,
		Code:      ErrorCodeValidationFailed,
		Details:   details,
		Response:  responseData,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// WithValidationContext adds context information to a validation error
func (e *ValidationError) WithValidationContext(key string, value interface{}) *ValidationError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}
