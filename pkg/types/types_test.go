package types

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorResponse(t *testing.T) {
	t.Run("basic_error_response", func(t *testing.T) {
		err := NewErrorResponse(ErrorCodeInvalidRequest, "Invalid input", "Missing required field")

		assert.Equal(t, "error", err.Error)
		assert.Equal(t, "Invalid input", err.Message)
		assert.Equal(t, ErrorCodeInvalidRequest, err.Code)
		assert.Equal(t, "Missing required field", err.Details)
		assert.NotEmpty(t, err.Timestamp)

		// Verify timestamp is valid RFC3339
		_, parseErr := time.Parse(time.RFC3339, err.Timestamp)
		assert.NoError(t, parseErr)
	})

	t.Run("error_response_with_context", func(t *testing.T) {
		err := NewErrorResponse(ErrorCodeLLMError, "Service unavailable", "Connection timeout").
			WithContext("endpoint", "/v1/validated-query").
			WithContext("retry_after", 30).
			WithRequestID("test-request-123")

		assert.Equal(t, "test-request-123", err.RequestID)
		assert.Equal(t, "/v1/validated-query", err.Context["endpoint"])
		assert.Equal(t, 30, err.Context["retry_after"])
	})

	t.Run("error_response_json_serialization", func(t *testing.T) {
		err := NewErrorResponse(ErrorCodeInvalidSchema, "Schema error", "Type mismatch").
			WithRequestID("req-456")

		jsonData, marshalErr := json.Marshal(err)
		require.NoError(t, marshalErr)

		var unmarshaled ErrorResponse
		unmarshalErr := json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, unmarshalErr)

		assert.Equal(t, err.Error, unmarshaled.Error)
		assert.Equal(t, err.Message, unmarshaled.Message)
		assert.Equal(t, err.Code, unmarshaled.Code)
		assert.Equal(t, err.RequestID, unmarshaled.RequestID)
	})
}

func TestValidationError(t *testing.T) {
	t.Run("basic_validation_error", func(t *testing.T) {
		responseData := json.RawMessage(`{"name": "test"}`)
		err := NewValidationError("Validation failed", "Missing age field", responseData)

		assert.Equal(t, "validation_error", err.Error)
		assert.Equal(t, "Validation failed", err.Message)
		assert.Equal(t, ErrorCodeValidationFailed, err.Code)
		assert.Equal(t, "Missing age field", err.Details)
		assert.Equal(t, responseData, err.Response)
		assert.NotEmpty(t, err.Timestamp)
	})

	t.Run("validation_error_with_context", func(t *testing.T) {
		responseData := json.RawMessage(`{"invalid": "data"}`)
		err := NewValidationError("Schema mismatch", "Type error", responseData).
			WithValidationContext("schema_version", "1.0").
			WithValidationContext("validation_time_ms", 150)

		err.RequestID = "validation-test-789"

		assert.Equal(t, "validation-test-789", err.RequestID)
		assert.Equal(t, "1.0", err.Context["schema_version"])
		assert.Equal(t, 150, err.Context["validation_time_ms"])
	})

	t.Run("validation_error_json_serialization", func(t *testing.T) {
		responseData := json.RawMessage(`{"test":"response"}`)
		err := NewValidationError("Test error", "Test details", responseData)

		jsonData, marshalErr := json.Marshal(err)
		require.NoError(t, marshalErr)

		var unmarshaled ValidationError
		unmarshalErr := json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, unmarshalErr)

		assert.Equal(t, err.Error, unmarshaled.Error)
		assert.Equal(t, err.Message, unmarshaled.Message)
		assert.Equal(t, err.Code, unmarshaled.Code)
		assert.Equal(t, err.Response, unmarshaled.Response)
	})
}

func TestErrorCodes(t *testing.T) {
	t.Run("error_codes_defined", func(t *testing.T) {
		assert.Equal(t, "INVALID_REQUEST", ErrorCodeInvalidRequest)
		assert.Equal(t, "INVALID_SCHEMA", ErrorCodeInvalidSchema)
		assert.Equal(t, "LLM_ERROR", ErrorCodeLLMError)
		assert.Equal(t, "VALIDATION_FAILED", ErrorCodeValidationFailed)
		assert.Equal(t, "INTERNAL_ERROR", ErrorCodeInternalError)
		assert.Equal(t, "TIMEOUT", ErrorCodeTimeout)
		assert.Equal(t, "RATE_LIMITED", ErrorCodeRateLimited)
	})
}
