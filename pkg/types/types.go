package types

import "encoding/json"

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

type ValidationError struct {
	Error    string          `json:"error"`
	Details  string          `json:"details"`
	Response json.RawMessage `json:"response,omitempty"`
}
