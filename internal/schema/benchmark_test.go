package schema

import (
	"encoding/json"
	"testing"

	"github.com/wcygan/llm-json-parse/pkg/types"
)

func BenchmarkValidatorWithCache(b *testing.B) {
	validator := NewValidator()

	schemaJSON := json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "number"},
			"email": {"type": "string", "format": "email"}
		},
		"required": ["name", "age"]
	}`)

	testDataJSON, _ := json.Marshal(map[string]interface{}{
		"name":  "John Doe",
		"age":   30,
		"email": "john@example.com",
	})

	response := &types.ValidatedResponse{
		Data: json.RawMessage(testDataJSON),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := validator.ValidateResponse(schemaJSON, response)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkValidatorWithoutCache(b *testing.B) {
	// Create a validator that doesn't use caching by compiling fresh each time
	schemaJSON := json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "number"},
			"email": {"type": "string", "format": "email"}
		},
		"required": ["name", "age"]
	}`)

	testDataJSON, _ := json.Marshal(map[string]interface{}{
		"name":  "John Doe",
		"age":   30,
		"email": "john@example.com",
	})

	response := &types.ValidatedResponse{
		Data: json.RawMessage(testDataJSON),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create a new validator each time to simulate no caching
		validator := &Validator{
			cache: NewSchemaCache(0), // Zero-size cache effectively disables caching
		}
		err := validator.ValidateResponse(schemaJSON, response)
		if err != nil {
			b.Fatal(err)
		}
	}
}
