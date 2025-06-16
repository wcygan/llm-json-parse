package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wcygan/llm-json-parse/internal/client"
	"github.com/wcygan/llm-json-parse/internal/server"
	"github.com/wcygan/llm-json-parse/pkg/types"
	"github.com/wcygan/llm-json-parse/tests/utils"
)

// skipIfNoLLM skips the test if no real LLM server is available
func skipIfNoLLM(t *testing.T) {
	llmURL := os.Getenv("LLM_SERVER_URL")
	if llmURL == "" {
		llmURL = "http://localhost:8080"
	}
	
	// Test if LLM server is available
	resp, err := http.Post(llmURL+"/v1/chat/completions", "application/json", 
		bytes.NewReader([]byte(`{"messages":[{"role":"user","content":"test"}]}`)))
	if err != nil {
		t.Skipf("No LLM server available at %s: %v", llmURL, err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Skipf("LLM server at %s returned status %d", llmURL, resp.StatusCode)
	}
}

func TestEndToEndIntegration(t *testing.T) {
	skipIfNoLLM(t)

	llmURL := os.Getenv("LLM_SERVER_URL")
	if llmURL == "" {
		llmURL = "http://localhost:8080"
	}

	// Create our gateway server with real LLM client
	llmClient := client.NewLlamaServerClient(llmURL)
	srv := server.NewServer(llmClient)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	gatewayServer := httptest.NewServer(mux)
	defer gatewayServer.Close()

	// Test successful end-to-end flow
	t.Run("successful_e2e_flow", func(t *testing.T) {
		logger := utils.NewTestLogger(t, "successful_e2e_flow")
		defer logger.Finish()

		logger.LogStep(1, "Setting up test data and schema")
		
		requestBody := types.ValidatedQueryRequest{
			Schema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"name": {"type": "string"},
					"age": {"type": "number"},
					"occupation": {"type": "string"}
				},
				"required": ["name"]
			}`),
			Messages: []types.Message{
				{Role: "user", Content: "Tell me about a person named Alice who is a software engineer"},
			},
		}

		// Log the schema being used
		var schemaObj interface{}
		json.Unmarshal(requestBody.Schema, &schemaObj)
		logger.LogSchema(schemaObj)

		logger.LogStep(2, "Sending request to validation gateway")
		
		reqBody, err := json.Marshal(requestBody)
		require.NoError(t, err)

		startTime := time.Now()
		logger.LogRequest("POST", gatewayServer.URL+"/v1/validated-query", requestBody)

		resp, err := http.Post(
			gatewayServer.URL+"/v1/validated-query",
			"application/json",
			bytes.NewReader(reqBody),
		)
		duration := time.Since(startTime)
		require.NoError(t, err)
		defer resp.Body.Close()

		var responseData map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&responseData)
		require.NoError(t, err)

		logger.LogResponse(resp.StatusCode, responseData, duration)

		logger.LogStep(3, "Validating response structure")
		
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		logger.LogValidation(resp.StatusCode == http.StatusOK, "HTTP status code is 200 OK")

		// Verify the response has required fields and correct types
		name, hasName := responseData["name"]
		assert.True(t, hasName, "Response should have 'name' field")
		logger.LogValidation(hasName, "Response contains required 'name' field")
		
		if hasName {
			assert.IsType(t, "", name, "Name should be a string")
			logger.LogAssertion("Name type validation", "string", fmt.Sprintf("%T", name))
		}

		// Check optional fields if present
		if age, hasAge := responseData["age"]; hasAge {
			assert.IsType(t, float64(0), age, "Age should be a number")
			logger.LogAssertion("Age type validation", "number", fmt.Sprintf("%T", age))
		}

		if occupation, hasOccupation := responseData["occupation"]; hasOccupation {
			assert.IsType(t, "", occupation, "Occupation should be a string")
			logger.LogAssertion("Occupation type validation", "string", fmt.Sprintf("%T", occupation))
		}

		logger.LogTestSummary(true, "Schema validation passed with real LLM response")
	})

	// Test invalid schema handling
	t.Run("invalid_schema_e2e", func(t *testing.T) {
		logger := utils.NewTestLogger(t, "invalid_schema_e2e")
		defer logger.Finish()

		logger.LogStep(1, "Setting up invalid schema")

		requestBody := types.ValidatedQueryRequest{
			Schema: json.RawMessage(`{
				"type": "invalid_type"
			}`),
			Messages: []types.Message{
				{Role: "user", Content: "Test invalid schema"},
			},
		}

		var schemaObj interface{}
		json.Unmarshal(requestBody.Schema, &schemaObj)
		logger.LogSchema(schemaObj)

		logger.LogStep(2, "Sending request with invalid schema")

		reqBody, err := json.Marshal(requestBody)
		require.NoError(t, err)

		startTime := time.Now()
		logger.LogRequest("POST", gatewayServer.URL+"/v1/validated-query", requestBody)

		resp, err := http.Post(
			gatewayServer.URL+"/v1/validated-query",
			"application/json",
			bytes.NewReader(reqBody),
		)
		duration := time.Since(startTime)
		require.NoError(t, err)
		defer resp.Body.Close()

		var errorResponse types.ValidationError
		err = json.NewDecoder(resp.Body).Decode(&errorResponse)
		require.NoError(t, err)

		logger.LogResponse(resp.StatusCode, errorResponse, duration)

		logger.LogStep(3, "Validating error response")
		
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		logger.LogValidation(resp.StatusCode == http.StatusBadRequest, "HTTP status code is 400 Bad Request")

		assert.Equal(t, "Invalid schema", errorResponse.Error)
		logger.LogValidation(errorResponse.Error == "Invalid schema", "Error message indicates invalid schema")

		logger.LogTestSummary(true, "Invalid schema handled correctly")
	})

}

func TestRealWorldSchemas(t *testing.T) {
	skipIfNoLLM(t)

	llmURL := os.Getenv("LLM_SERVER_URL")
	if llmURL == "" {
		llmURL = "http://localhost:8080"
	}

	llmClient := client.NewLlamaServerClient(llmURL)
	srv := server.NewServer(llmClient)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	gatewayServer := httptest.NewServer(mux)
	defer gatewayServer.Close()

	t.Run("simple_product_schema", func(t *testing.T) {

		schema := json.RawMessage(`{
			"type": "object",
			"properties": {
				"name": {"type": "string"},
				"price": {"type": "number"},
				"category": {"type": "string"},
				"available": {"type": "boolean"}
			},
			"required": ["name", "price"]
		}`)

		requestBody := types.ValidatedQueryRequest{
			Schema: schema,
			Messages: []types.Message{
				{Role: "user", Content: "Describe a laptop product that costs $1200"},
			},
		}

		reqBody, err := json.Marshal(requestBody)
		require.NoError(t, err)

		resp, err := http.Post(
			gatewayServer.URL+"/v1/validated-query",
			"application/json",
			bytes.NewReader(reqBody),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		var responseData map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&responseData)
		require.NoError(t, err)

		// Test should pass if schema validation succeeds
		if resp.StatusCode == http.StatusOK {
			// Verify required fields
			assert.Contains(t, responseData, "name")
			assert.Contains(t, responseData, "price")
			
			// Verify types
			assert.IsType(t, "", responseData["name"])
			assert.IsType(t, float64(0), responseData["price"])
		} else {
			t.Logf("LLM response didn't match schema, status: %d", resp.StatusCode)
		}
	})
}

func TestConcurrentRequests(t *testing.T) {
	skipIfNoLLM(t)
	
	logger := utils.NewTestLogger(t, "concurrent_requests")
	defer logger.Finish()

	logger.LogStep(1, "Setting up real LLM and gateway server")
	
	llmURL := os.Getenv("LLM_SERVER_URL")
	if llmURL == "" {
		llmURL = "http://localhost:8080"
	}

	llmClient := client.NewLlamaServerClient(llmURL)
	srv := server.NewServer(llmClient)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	gatewayServer := httptest.NewServer(mux)
	defer gatewayServer.Close()

	requestBody := types.ValidatedQueryRequest{
		Schema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"name": {"type": "string"},
				"value": {"type": "number"}
			},
			"required": ["name", "value"]
		}`),
		Messages: []types.Message{
			{Role: "user", Content: "Generate test data"},
		},
	}

	var schemaObj interface{}
	json.Unmarshal(requestBody.Schema, &schemaObj)
	logger.LogSchema(schemaObj)

	reqBody, err := json.Marshal(requestBody)
	require.NoError(t, err)

	logger.LogStep(2, "Executing concurrent requests")
	
	// Send multiple concurrent requests (fewer with real LLM for performance)
	numRequests := 3
	results := make(chan int, numRequests)

	logger.LogConcurrentTestStart(numRequests)
	
	startTime := time.Now()
	for i := 0; i < numRequests; i++ {
		go func(requestID int) {
			resp, err := http.Post(
				gatewayServer.URL+"/v1/validated-query",
				"application/json",
				bytes.NewReader(reqBody),
			)
			if err != nil {
				results <- 0
				return
			}
			defer resp.Body.Close()
			results <- resp.StatusCode
		}(i)
	}

	logger.LogStep(3, "Collecting results from concurrent requests")
	
	// Wait for all requests to complete (longer timeout for real LLM)
	successCount := 0
	failedCount := 0
	timeout := time.After(30 * time.Second)
	
	for i := 0; i < numRequests; i++ {
		select {
		case status := <-results:
			if status == http.StatusOK || status == http.StatusUnprocessableEntity {
				// Both OK and validation errors are considered successful for concurrency testing
				successCount++
			} else {
				failedCount++
			}
		case <-timeout:
			t.Fatal("Timeout waiting for concurrent requests")
		}
	}
	
	totalDuration := time.Since(startTime)
	logger.LogConcurrentTestResult(successCount, failedCount, totalDuration)

	assert.Equal(t, numRequests, successCount, "All concurrent requests should complete")
	logger.LogValidation(successCount == numRequests, fmt.Sprintf("All %d concurrent requests completed", numRequests))
	
	logger.LogTestSummary(successCount == numRequests, fmt.Sprintf("Concurrent test completed: %d/%d requests successful", successCount, numRequests))
}