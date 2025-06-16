package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/wcygan/llm-json-parse/internal/server"
	"github.com/wcygan/llm-json-parse/pkg/types"
	"github.com/wcygan/llm-json-parse/tests/mocks"
	"github.com/wcygan/llm-json-parse/tests/utils"
)

func TestValidatedQueryIntegration(t *testing.T) {
	tests := []struct {
		name           string
		request        types.ValidatedQueryRequest
		mockResponse   *types.ValidatedResponse
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
			mockResponse: func() *types.ValidatedResponse {
				data, _ := json.Marshal(map[string]interface{}{
					"name": "John",
					"age":  25,
				})
				return &types.ValidatedResponse{Data: json.RawMessage(data)}
			}(),
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
			mockResponse: func() *types.ValidatedResponse {
				data, _ := json.Marshal(map[string]interface{}{
					"name": "John",
					// Missing required "age" field
				})
				return &types.ValidatedResponse{Data: json.RawMessage(data)}
			}(),
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
			logger := utils.NewTestLogger(t, tt.name)
			defer logger.Finish()

			logger.LogStep(1, "Setting up mock client and server")

			// Setup mock client
			mockClient := mocks.NewMockLLMClient()
			if tt.mockResponse != nil || tt.mockError != nil {
				mockClient.On("SendStructuredQuery",
					mock.Anything, // Use mock.Anything for context
					tt.request.Messages,
					mock.Anything).Return(tt.mockResponse, tt.mockError) // Use mock.Anything for schema since JSON formatting can vary

				logger.LogMockSetup("Mock LLM client configured", tt.mockResponse)
			}

			// Create server with mock client
			srv := server.NewServer(mockClient)
			mux := http.NewServeMux()
			srv.RegisterRoutes(mux)

			// Create test server
			testServer := httptest.NewServer(mux)
			defer testServer.Close()

			logger.LogStep(2, "Executing HTTP request")

			// Prepare request
			reqBody, err := json.Marshal(tt.request)
			require.NoError(t, err)

			// Log schema if present
			if len(tt.request.Schema) > 0 {
				var schemaObj interface{}
				json.Unmarshal(tt.request.Schema, &schemaObj)
				logger.LogSchema(schemaObj)
			}

			startTime := time.Now()
			logger.LogRequest("POST", testServer.URL+"/v1/validated-query", tt.request)

			// Send request
			resp, err := http.Post(
				testServer.URL+"/v1/validated-query",
				"application/json",
				bytes.NewReader(reqBody),
			)
			duration := time.Since(startTime)
			require.NoError(t, err)
			defer resp.Body.Close()

			var responseBody interface{}
			if resp.StatusCode == http.StatusOK && tt.expectedStatus == http.StatusOK {
				err = json.NewDecoder(resp.Body).Decode(&responseBody)
				require.NoError(t, err)
			} else {
				// For error responses, read as error type
				var errorBody map[string]interface{}
				json.NewDecoder(resp.Body).Decode(&errorBody)
				responseBody = errorBody
			}

			logger.LogResponse(resp.StatusCode, responseBody, duration)

			logger.LogStep(3, "Validating response")

			// Verify response
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			logger.LogValidation(resp.StatusCode == tt.expectedStatus,
				fmt.Sprintf("HTTP status code matches expected %d", tt.expectedStatus))

			if tt.expectedStatus == http.StatusOK {
				logger.LogJSONComparison("Response body validation", tt.expectedBody, responseBody)
				assert.Equal(t, tt.expectedBody, responseBody)
			}

			// Verify mock was called if expected
			if tt.mockResponse != nil || tt.mockError != nil {
				mockClient.AssertExpectations(t)
				logger.LogValidation(true, "Mock expectations satisfied")
			}

			logger.LogTestSummary(resp.StatusCode == tt.expectedStatus, "Server integration test completed")
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
