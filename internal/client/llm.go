package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/wcygan/llm-json-parse/internal/logging"
	"github.com/wcygan/llm-json-parse/pkg/types"
)

type LLMClient interface {
	SendStructuredQuery(ctx context.Context, messages []types.Message, schema json.RawMessage) (*types.ValidatedResponse, error)
}

type LlamaServerClient struct {
	baseURL string
	client  *http.Client
	logger  *logging.Logger
}

func NewLlamaServerClient(baseURL string) *LlamaServerClient {
	return &LlamaServerClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
		logger:  logging.NewLogger(logging.LogConfig{Level: "info", Format: "json"}),
	}
}

// NewLlamaServerClientWithTimeout creates a new LLM client with custom timeout
func NewLlamaServerClientWithTimeout(baseURL string, timeout time.Duration) *LlamaServerClient {
	return &LlamaServerClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: timeout},
		logger:  logging.NewLogger(logging.LogConfig{Level: "info", Format: "json"}),
	}
}

// NewLlamaServerClientWithLogger creates a new LLM client with custom logger
func NewLlamaServerClientWithLogger(baseURL string, timeout time.Duration, logger *logging.Logger) *LlamaServerClient {
	return &LlamaServerClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: timeout},
		logger:  logger,
	}
}

func (c *LlamaServerClient) SendStructuredQuery(ctx context.Context, messages []types.Message, schema json.RawMessage) (*types.ValidatedResponse, error) {
	start := time.Now()
	logger := c.logger.WithComponent("llm_client").WithOperation("structured_query")

	request := types.LLMRequest{
		Messages: messages,
		ResponseFormat: &types.ResponseFormat{
			Type: "json_schema",
			JSONSchema: types.JSONSchema{
				Name:   "response",
				Strict: true,
				Schema: schema,
			},
		},
	}

	// Marshal request
	marshalStart := time.Now()
	reqBody, err := json.Marshal(request)
	if err != nil {
		logger.WithError(err).Error("Failed to marshal LLM request")
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	marshalDuration := time.Since(marshalStart)

	logger.WithFields(map[string]interface{}{
		"url":                 c.baseURL + "/v1/chat/completions",
		"request_size_bytes":  len(reqBody),
		"schema_size_bytes":   len(schema),
		"message_count":       len(messages),
		"marshal_duration_ms": marshalDuration.Milliseconds(),
	}).Info("Sending structured query to LLM")

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		logger.WithError(err).Error("Failed to create HTTP request")
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Send HTTP request
	httpStart := time.Now()
	resp, err := c.client.Do(httpReq)
	httpDuration := time.Since(httpStart)

	if err != nil {
		logger.WithError(err).
			WithDuration(httpDuration).
			Error("HTTP request to LLM failed")
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.WithFields(map[string]interface{}{
			"status_code":      resp.StatusCode,
			"http_duration_ms": httpDuration.Milliseconds(),
		}).Error("LLM server returned non-200 status")
		return nil, fmt.Errorf("LLM server returned status %d", resp.StatusCode)
	}

	// Decode response
	decodeStart := time.Now()
	var llmResponse types.LLMResponse
	if err := json.NewDecoder(resp.Body).Decode(&llmResponse); err != nil {
		logger.WithError(err).
			WithDuration(time.Since(decodeStart)).
			Error("Failed to decode LLM response")
		return nil, fmt.Errorf("decode response: %w", err)
	}
	decodeDuration := time.Since(decodeStart)

	if len(llmResponse.Choices) == 0 {
		logger.Error("LLM response contains no choices")
		return nil, fmt.Errorf("no response choices")
	}

	// Validate that content is valid JSON
	validateStart := time.Now()
	var temp interface{}
	content := llmResponse.Choices[0].Message.Content
	if err := json.Unmarshal([]byte(content), &temp); err != nil {
		logger.WithError(err).
			WithDuration(time.Since(validateStart)).
			WithFields(map[string]interface{}{
				"content_length": len(content),
			}).Error("LLM response is not valid JSON")
		return nil, fmt.Errorf("LLM response is not valid JSON: %w", err)
	}
	validateDuration := time.Since(validateStart)

	// Success
	totalDuration := time.Since(start)
	logger.WithDuration(totalDuration).
		WithFields(map[string]interface{}{
			"response_size_bytes":  len(content),
			"http_duration_ms":     httpDuration.Milliseconds(),
			"marshal_duration_ms":  marshalDuration.Milliseconds(),
			"decode_duration_ms":   decodeDuration.Milliseconds(),
			"validate_duration_ms": validateDuration.Milliseconds(),
			"llm_success":          true,
		}).Info("LLM structured query completed successfully")

	// Return as ValidatedResponse with the raw JSON
	return &types.ValidatedResponse{
		Data: json.RawMessage(content),
	}, nil
}
