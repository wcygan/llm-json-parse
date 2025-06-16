package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/wcygan/llm-json-parse/pkg/types"
)

// MockLLMServer creates a test HTTP server that mimics llama-server behavior
type MockLLMServer struct {
	server   *httptest.Server
	response *types.LLMResponse
	status   int
}

func NewMockLLMServer() *MockLLMServer {
	mock := &MockLLMServer{
		status: http.StatusOK,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", mock.handleCompletion)

	mock.server = httptest.NewServer(mux)
	return mock
}

func (m *MockLLMServer) URL() string {
	return m.server.URL
}

func (m *MockLLMServer) Close() {
	m.server.Close()
}

func (m *MockLLMServer) SetResponse(content string, status int) {
	m.response = &types.LLMResponse{
		Choices: []types.Choice{
			{
				Message: types.Message{
					Role:    "assistant",
					Content: content,
				},
			},
		},
	}
	m.status = status
}

func (m *MockLLMServer) handleCompletion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(m.status)

	if m.response != nil {
		json.NewEncoder(w).Encode(m.response)
	}
}