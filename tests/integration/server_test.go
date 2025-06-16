package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/wcygan/llm-json-parse/internal/server"
	"github.com/wcygan/llm-json-parse/pkg/types"
	"github.com/wcygan/llm-json-parse/tests/mocks"
)

func TestValidatedQueryIntegration(t *testing.T) {
	tests := []struct {
		name           string
		request        types.ValidatedQueryRequest
		mockResponse   interface{}
		mockError      error
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name: "successful_validation",
			request: types.ValidatedQueryRequest{
				Schema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"name": {"type": "string"},
						"age": {"type": "number"}
					},
					"required": ["name", "age"]
				}`),
				Messages: []types.Message{
					{Role: "user", Content: "Tell me about John who is 25 years old"},
				},
			},
			mockResponse: map[string]interface{}{
				"name": "John",
				"age":  25,
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"name": "John",
				"age":  float64(25), // JSON numbers are float64
			},
		},
		{
			name: "schema_validation_failure",
			request: types.ValidatedQueryRequest{
				Schema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"name": {"type": "string"},
						"age": {"type": "number"}
					},
					"required": ["name", "age"]
				}`),
				Messages: []types.Message{
					{Role: "user", Content: "Tell me about someone"},
				},
			},
			mockResponse: map[string]interface{}{
				"name": "John",
				// Missing required "age" field
			},
			mockError:      nil,
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "invalid_schema",
			request: types.ValidatedQueryRequest{
				Schema: json.RawMessage(`{
					"type": "invalid_type"
				}`),
				Messages: []types.Message{
					{Role: "user", Content: "Test"},
				},
			},
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock client
			mockClient := mocks.NewMockLLMClient()
			if tt.mockResponse != nil || tt.mockError != nil {
				mockClient.On("SendStructuredQuery", 
					mock.Anything, // Use mock.Anything for context
					tt.request.Messages, 
					mock.Anything).Return(tt.mockResponse, tt.mockError) // Use mock.Anything for schema since JSON formatting can vary
			}

			// Create server with mock client
			srv := server.NewServer(mockClient)
			mux := http.NewServeMux()
			srv.RegisterRoutes(mux)

			// Create test server
			testServer := httptest.NewServer(mux)
			defer testServer.Close()

			// Prepare request
			reqBody, err := json.Marshal(tt.request)
			require.NoError(t, err)

			// Send request
			resp, err := http.Post(
				testServer.URL+"/v1/validated-query",
				"application/json",
				bytes.NewReader(reqBody),
			)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Verify response
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusOK {
				var responseBody interface{}
				err = json.NewDecoder(resp.Body).Decode(&responseBody)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedBody, responseBody)
			}

			// Verify mock was called if expected
			if tt.mockResponse != nil || tt.mockError != nil {
				mockClient.AssertExpectations(t)
			}
		})
	}
}

func TestHealthEndpoint(t *testing.T) {
	mockClient := mocks.NewMockLLMClient()
	srv := server.NewServer(mockClient)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	resp, err := http.Get(testServer.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/plain", resp.Header.Get("Content-Type"))
	
	buf := make([]byte, 100)
	n, _ := resp.Body.Read(buf)
	body := string(buf[:n])
	assert.Equal(t, "OK", body)
}

func TestInvalidJSONRequest(t *testing.T) {
	mockClient := mocks.NewMockLLMClient()
	srv := server.NewServer(mockClient)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	// Send invalid JSON
	resp, err := http.Post(
		testServer.URL+"/v1/validated-query",
		"application/json",
		bytes.NewReader([]byte("invalid json")),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}