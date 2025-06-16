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
	"github.com/stretchr/testify/require"
	"github.com/wcygan/llm-json-parse/internal/client"
	"github.com/wcygan/llm-json-parse/internal/server"
	"github.com/wcygan/llm-json-parse/pkg/types"
	"github.com/wcygan/llm-json-parse/tests/utils"
)

func TestEndToEndIntegration(t *testing.T) {
	// Start mock LLM server
	mockLLM := NewMockLLMServer()
	defer mockLLM.Close()

	// Set up valid JSON response
	validResponse := `{
		"name": "Chocolate Chip Cookies",
		"ingredients": ["flour", "sugar", "eggs", "chocolate chips"],
		"prep_time": 15,
		"cook_time": 12
	}`
	mockLLM.SetResponse(validResponse, http.StatusOK)

	// Create our gateway server with real LLM client pointing to mock
	llmClient := client.NewLlamaServerClient(mockLLM.URL())
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
					"ingredients": {
						"type": "array",
						"items": {"type": "string"}
					},
					"prep_time": {"type": "number"},
					"cook_time": {"type": "number"}
				},
				"required": ["name", "ingredients"]
			}`),
			Messages: []types.Message{
				{Role: "user", Content: "Give me a cookie recipe"},
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

		// Verify the response matches our expected structure
		logger.LogAssertion("Recipe name validation", "Chocolate Chip Cookies", responseData["name"])
		assert.Equal(t, "Chocolate Chip Cookies", responseData["name"])

		logger.LogAssertion("Ingredients contain flour", true, func() bool {
			ingredients, ok := responseData["ingredients"].([]interface{})
			if !ok {
				return false
			}
			for _, ingredient := range ingredients {
				if ingredient == "flour" {
					return true
				}
			}
			return false
		}())
		assert.Contains(t, responseData["ingredients"], "flour")

		logger.LogAssertion("Prep time validation", float64(15), responseData["prep_time"])
		assert.Equal(t, float64(15), responseData["prep_time"])

		logger.LogAssertion("Cook time validation", float64(12), responseData["cook_time"])
		assert.Equal(t, float64(12), responseData["cook_time"])

		logger.LogTestSummary(true, "All validations passed")
	})

	// Test validation failure
	t.Run("validation_failure_e2e", func(t *testing.T) {
		logger := utils.NewTestLogger(t, "validation_failure_e2e")
		defer logger.Finish()

		logger.LogStep(1, "Setting up invalid response from mock LLM")
		
		// Set up invalid response (missing required field)
		invalidResponse := `{
			"ingredients": ["flour", "sugar"]
		}`
		mockLLM.SetResponse(invalidResponse, http.StatusOK)
		
		var invalidJSON interface{}
		json.Unmarshal([]byte(invalidResponse), &invalidJSON)
		logger.LogMockSetup("Mock LLM will return invalid response (missing 'name' field)", invalidJSON)

		requestBody := types.ValidatedQueryRequest{
			Schema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"name": {"type": "string"},
					"ingredients": {
						"type": "array",
						"items": {"type": "string"}
					}
				},
				"required": ["name", "ingredients"]
			}`),
			Messages: []types.Message{
				{Role: "user", Content: "Give me a recipe"},
			},
		}

		var schemaObj interface{}
		json.Unmarshal(requestBody.Schema, &schemaObj)
		logger.LogSchema(schemaObj)

		logger.LogStep(2, "Sending request expecting validation failure")

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
		
		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
		logger.LogValidation(resp.StatusCode == http.StatusUnprocessableEntity, "HTTP status code is 422 Unprocessable Entity")

		assert.Equal(t, "Schema validation failed", errorResponse.Error)
		logger.LogValidation(errorResponse.Error == "Schema validation failed", "Error message indicates schema validation failure")

		assert.Contains(t, errorResponse.Details, "missing properties")
		logger.LogValidation(true, "Error details mention missing properties")

		logger.LogTestSummary(true, "Validation failure handled correctly")
	})

	// Test LLM server error handling
	t.Run("llm_server_error_e2e", func(t *testing.T) {
		mockLLM.SetResponse("", http.StatusInternalServerError)

		requestBody := types.ValidatedQueryRequest{
			Schema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"test": {"type": "string"}
				}
			}`),
			Messages: []types.Message{
				{Role: "user", Content: "Test"},
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

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})
}

func TestRealWorldSchemas(t *testing.T) {
	mockLLM := NewMockLLMServer()
	defer mockLLM.Close()

	llmClient := client.NewLlamaServerClient(mockLLM.URL())
	srv := server.NewServer(llmClient)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	gatewayServer := httptest.NewServer(mux)
	defer gatewayServer.Close()

	t.Run("recipe_analysis_schema", func(t *testing.T) {
		response := `{
			"recipe_name": "Chocolate Chip Cookies",
			"ingredients": [
				{"name": "flour", "amount": "2", "unit": "cups"},
				{"name": "sugar", "amount": "1", "unit": "cup"}
			],
			"steps": [
				{"step_number": 1, "instruction": "Mix dry ingredients", "duration_minutes": 5},
				{"step_number": 2, "instruction": "Add wet ingredients", "duration_minutes": 3}
			],
			"cooking_details": {
				"prep_time_minutes": 15,
				"cook_time_minutes": 12,
				"temperature_fahrenheit": 350,
				"servings": 24
			}
		}`
		mockLLM.SetResponse(response, http.StatusOK)

		schema := json.RawMessage(`{
			"type": "object",
			"properties": {
				"recipe_name": {"type": "string"},
				"ingredients": {
					"type": "array",
					"items": {
						"type": "object",
						"properties": {
							"name": {"type": "string"},
							"amount": {"type": "string"},
							"unit": {"type": "string"}
						},
						"required": ["name", "amount"],
						"additionalProperties": false
					}
				},
				"steps": {
					"type": "array",
					"items": {
						"type": "object",
						"properties": {
							"step_number": {"type": "integer"},
							"instruction": {"type": "string"},
							"duration_minutes": {"type": "integer"}
						},
						"required": ["step_number", "instruction"],
						"additionalProperties": false
					}
				},
				"cooking_details": {
					"type": "object",
					"properties": {
						"prep_time_minutes": {"type": "integer"},
						"cook_time_minutes": {"type": "integer"},
						"temperature_fahrenheit": {"type": "integer"},
						"servings": {"type": "integer"}
					},
					"required": ["prep_time_minutes", "cook_time_minutes"],
					"additionalProperties": false
				}
			},
			"required": ["recipe_name", "ingredients", "steps", "cooking_details"],
			"additionalProperties": false
		}`)

		requestBody := types.ValidatedQueryRequest{
			Schema: schema,
			Messages: []types.Message{
				{Role: "user", Content: "Analyze this recipe: chocolate chip cookies"},
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

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var responseData map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&responseData)
		require.NoError(t, err)

		assert.Equal(t, "Chocolate Chip Cookies", responseData["recipe_name"])
		assert.IsType(t, []interface{}{}, responseData["ingredients"])
		assert.IsType(t, []interface{}{}, responseData["steps"])
		assert.IsType(t, map[string]interface{}{}, responseData["cooking_details"])
	})
}

func TestConcurrentRequests(t *testing.T) {
	logger := utils.NewTestLogger(t, "concurrent_requests")
	defer logger.Finish()

	logger.LogStep(1, "Setting up mock LLM and gateway server")
	
	mockLLM := NewMockLLMServer()
	defer mockLLM.Close()

	validResponse := `{"name": "Test Response", "value": 42}`
	mockLLM.SetResponse(validResponse, http.StatusOK)

	var responseJSON interface{}
	json.Unmarshal([]byte(validResponse), &responseJSON)
	logger.LogMockSetup("Mock LLM configured to return", responseJSON)

	llmClient := client.NewLlamaServerClient(mockLLM.URL())
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
	
	// Send multiple concurrent requests
	numRequests := 10
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
	
	// Wait for all requests to complete
	successCount := 0
	failedCount := 0
	timeout := time.After(5 * time.Second)
	
	for i := 0; i < numRequests; i++ {
		select {
		case status := <-results:
			if status == http.StatusOK {
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

	assert.Equal(t, numRequests, successCount, "All concurrent requests should succeed")
	logger.LogValidation(successCount == numRequests, fmt.Sprintf("All %d concurrent requests succeeded", numRequests))
	
	logger.LogTestSummary(successCount == numRequests, fmt.Sprintf("Concurrent test completed: %d/%d requests successful", successCount, numRequests))
}