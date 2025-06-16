package mocks

import (
	"context"
	"encoding/json"

	"github.com/stretchr/testify/mock"
	"github.com/wcygan/llm-json-parse/pkg/types"
)

type MockLLMClient struct {
	mock.Mock
}

func (m *MockLLMClient) SendStructuredQuery(ctx context.Context, messages []types.Message, schema json.RawMessage) (interface{}, error) {
	args := m.Called(ctx, messages, schema)
	return args.Get(0), args.Error(1)
}

func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{}
}