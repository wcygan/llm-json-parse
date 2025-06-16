package schema

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaCacheBasicOperations(t *testing.T) {
	cache := NewSchemaCache(3)

	// Test initial empty state
	assert.Equal(t, 0, cache.Size())

	// Test Get on empty cache
	_, exists := cache.Get("nonexistent")
	assert.False(t, exists)

	// Since we can't easily create jsonschema.Schema instances in unit tests,
	// we'll test the cache behavior with the actual validator
}

func TestValidatorCaching(t *testing.T) {
	validator := NewValidator()

	// Define a simple schema
	schemaJSON := json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "number"}
		},
		"required": ["name"]
	}`)

	// First validation - should compile and cache the schema
	testData := map[string]interface{}{
		"name": "John",
		"age":  25,
	}

	err := validator.ValidateResponse(schemaJSON, testData)
	require.NoError(t, err)

	// Verify schema was cached
	assert.Equal(t, 1, validator.cache.Size())

	// Second validation with same schema - should use cached version
	testData2 := map[string]interface{}{
		"name": "Jane",
		"age":  30,
	}

	err = validator.ValidateResponse(schemaJSON, testData2)
	require.NoError(t, err)

	// Cache size should still be 1 (same schema)
	assert.Equal(t, 1, validator.cache.Size())

	// Third validation with different schema - should cache new schema
	schemaJSON2 := json.RawMessage(`{
		"type": "object",
		"properties": {
			"title": {"type": "string"},
			"count": {"type": "number"}
		},
		"required": ["title"]
	}`)

	testData3 := map[string]interface{}{
		"title": "Test",
		"count": 5,
	}

	err = validator.ValidateResponse(schemaJSON2, testData3)
	require.NoError(t, err)

	// Cache size should now be 2
	assert.Equal(t, 2, validator.cache.Size())
}

func TestValidatorCachingWithValidateSchema(t *testing.T) {
	validator := NewValidator()

	schemaJSON := json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		},
		"required": ["name"]
	}`)

	// First call to ValidateSchema should cache the schema
	err := validator.ValidateSchema(schemaJSON)
	require.NoError(t, err)
	assert.Equal(t, 1, validator.cache.Size())

	// Second call should use cached version
	err = validator.ValidateSchema(schemaJSON)
	require.NoError(t, err)
	assert.Equal(t, 1, validator.cache.Size())

	// ValidateResponse with same schema should also use cache
	testData := map[string]interface{}{"name": "Test"}
	err = validator.ValidateResponse(schemaJSON, testData)
	require.NoError(t, err)
	assert.Equal(t, 1, validator.cache.Size())
}

func TestCacheEviction(t *testing.T) {
	// Create validator with small cache size for testing eviction
	validator := &Validator{
		cache: NewSchemaCache(2), // Only cache 2 schemas
	}

	schemas := []json.RawMessage{
		json.RawMessage(`{"type": "object", "properties": {"a": {"type": "string"}}}`),
		json.RawMessage(`{"type": "object", "properties": {"b": {"type": "string"}}}`),
		json.RawMessage(`{"type": "object", "properties": {"c": {"type": "string"}}}`),
	}

	testData := []map[string]interface{}{
		{"a": "test1"},
		{"b": "test2"},
		{"c": "test3"},
	}

	// Add first two schemas
	err := validator.ValidateResponse(schemas[0], testData[0])
	require.NoError(t, err)
	assert.Equal(t, 1, validator.cache.Size())

	err = validator.ValidateResponse(schemas[1], testData[1])
	require.NoError(t, err)
	assert.Equal(t, 2, validator.cache.Size())

	// Adding third schema should trigger cache eviction (simple clear strategy)
	err = validator.ValidateResponse(schemas[2], testData[2])
	require.NoError(t, err)
	
	// After eviction and adding new schema, size should be 1
	assert.Equal(t, 1, validator.cache.Size())
}