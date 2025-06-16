package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/wcygan/llm-json-parse/pkg/types"
)

type LLMClient interface {
	SendStructuredQuery(ctx context.Context, messages []types.Message, schema json.RawMessage) (*types.ValidatedResponse, error)
}

type LlamaServerClient struct {
	baseURL string
	client  *http.Client
}

func NewLlamaServerClient(baseURL string) *LlamaServerClient {
	return &LlamaServerClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *LlamaServerClient) SendStructuredQuery(ctx context.Context, messages []types.Message, schema json.RawMessage) (*types.ValidatedResponse, error) {
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

	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM server returned status %d", resp.StatusCode)
	}

	var llmResponse types.LLMResponse
	if err := json.NewDecoder(resp.Body).Decode(&llmResponse); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(llmResponse.Choices) == 0 {
		return nil, fmt.Errorf("no response choices")
	}

	// Validate that content is valid JSON by attempting to unmarshal/marshal
	var temp interface{}
	content := llmResponse.Choices[0].Message.Content
	if err := json.Unmarshal([]byte(content), &temp); err != nil {
		return nil, fmt.Errorf("LLM response is not valid JSON: %w", err)
	}

	// Return as ValidatedResponse with the raw JSON
	return &types.ValidatedResponse{
		Data: json.RawMessage(content),
	}, nil
}
