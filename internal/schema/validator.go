package schema

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/wcygan/llm-json-parse/internal/logging"
	"github.com/wcygan/llm-json-parse/pkg/types"
)

// SchemaCache provides thread-safe caching of compiled JSON schemas
type SchemaCache struct {
	mu      sync.RWMutex
	schemas map[string]*jsonschema.Schema
	maxSize int
}

// NewSchemaCache creates a new schema cache with the given maximum size
func NewSchemaCache(maxSize int) *SchemaCache {
	return &SchemaCache{
		schemas: make(map[string]*jsonschema.Schema),
		maxSize: maxSize,
	}
}

// Get retrieves a compiled schema from the cache
func (sc *SchemaCache) Get(key string) (*jsonschema.Schema, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	schema, exists := sc.schemas[key]
	return schema, exists
}

// Set stores a compiled schema in the cache
func (sc *SchemaCache) Set(key string, schema *jsonschema.Schema) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Simple eviction: if at capacity, clear the cache
	// This is simple but effective for most use cases
	if len(sc.schemas) >= sc.maxSize {
		sc.schemas = make(map[string]*jsonschema.Schema)
	}

	sc.schemas[key] = schema
}

// Size returns the current number of cached schemas
func (sc *SchemaCache) Size() int {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return len(sc.schemas)
}

type Validator struct {
	cache  *SchemaCache
	logger *logging.Logger
}

func NewValidator() *Validator {
	return &Validator{
		cache:  NewSchemaCache(100), // Cache up to 100 compiled schemas
		logger: logging.NewLogger(logging.LogConfig{Level: "info", Format: "json"}),
	}
}

// NewValidatorWithCacheSize creates a validator with custom cache size
func NewValidatorWithCacheSize(cacheSize int) *Validator {
	return &Validator{
		cache:  NewSchemaCache(cacheSize),
		logger: logging.NewLogger(logging.LogConfig{Level: "info", Format: "json"}),
	}
}

// NewValidatorWithLogger creates a validator with custom logger
func NewValidatorWithLogger(cacheSize int, logger *logging.Logger) *Validator {
	return &Validator{
		cache:  NewSchemaCache(cacheSize),
		logger: logger,
	}
}

func (v *Validator) ValidateResponse(schemaBytes json.RawMessage, response *types.ValidatedResponse) error {
	start := time.Now()
	schema, err := v.compileSchema(schemaBytes)
	if err != nil {
		v.logger.WithComponent("schema_validator").
			WithError(err).
			WithFields(map[string]interface{}{
				"schema_size_bytes": len(schemaBytes),
			}).
			Error("Schema compilation failed during response validation")
		return fmt.Errorf("compile schema: %w", err)
	}

	// Unmarshal the response data to validate against schema
	parseStart := time.Now()
	var responseData interface{}
	if err := json.Unmarshal(response.Data, &responseData); err != nil {
		v.logger.WithComponent("schema_validator").
			WithError(err).
			WithDuration(time.Since(parseStart)).
			WithFields(map[string]interface{}{
				"response_size_bytes": len(response.Data),
			}).
			Error("Failed to parse response JSON")
		return fmt.Errorf("invalid response JSON: %w", err)
	}
	parseDuration := time.Since(parseStart)

	validateStart := time.Now()
	if err := schema.Validate(responseData); err != nil {
		validateDuration := time.Since(validateStart)
		totalDuration := time.Since(start)

		v.logger.WithComponent("schema_validator").
			WithError(err).
			WithDuration(totalDuration).
			WithFields(map[string]interface{}{
				"response_size_bytes":  len(response.Data),
				"schema_size_bytes":    len(schemaBytes),
				"parse_duration_ms":    parseDuration.Milliseconds(),
				"validate_duration_ms": validateDuration.Milliseconds(),
				"validation_success":   false,
			}).
			Warn("Response validation failed")
		return fmt.Errorf("validation failed: %w", err)
	}

	// Success
	totalDuration := time.Since(start)
	validateDuration := time.Since(validateStart)
	v.logger.WithComponent("schema_validator").
		WithDuration(totalDuration).
		WithFields(map[string]interface{}{
			"response_size_bytes":  len(response.Data),
			"schema_size_bytes":    len(schemaBytes),
			"parse_duration_ms":    parseDuration.Milliseconds(),
			"validate_duration_ms": validateDuration.Milliseconds(),
			"validation_success":   true,
		}).
		Debug("Response validation successful")

	return nil
}

func (v *Validator) ValidateSchema(schemaBytes json.RawMessage) error {
	start := time.Now()
	_, err := v.compileSchema(schemaBytes)
	if err != nil {
		v.logger.WithComponent("schema_validator").
			WithError(err).
			WithDuration(time.Since(start)).
			WithFields(map[string]interface{}{
				"schema_size_bytes": len(schemaBytes),
			}).
			Error("Schema validation failed")
		return fmt.Errorf("invalid schema: %w", err)
	}
	v.logger.WithComponent("schema_validator").
		WithDuration(time.Since(start)).
		WithFields(map[string]interface{}{
			"schema_size_bytes": len(schemaBytes),
		}).
		Debug("Schema validation successful")
	return nil
}

func (v *Validator) compileSchema(schemaBytes json.RawMessage) (*jsonschema.Schema, error) {
	// Generate cache key based on schema content
	hash := sha256.Sum256(schemaBytes)
	cacheKey := fmt.Sprintf("%x", hash[:16]) // Use first 16 bytes for shorter key

	// Check cache first
	if schema, exists := v.cache.Get(cacheKey); exists {
		v.logger.WithComponent("schema_validator").
			WithFields(map[string]interface{}{
				"cache_hit":         true,
				"cache_size":        v.cache.Size(),
				"schema_size_bytes": len(schemaBytes),
			}).
			Debug("Schema retrieved from cache")
		return schema, nil
	}

	// Cache miss - compile schema
	compileStart := time.Now()

	// Parse JSON first to ensure it's valid
	var schemaObj interface{}
	if err := json.Unmarshal(schemaBytes, &schemaObj); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Create a new compiler for each validation to avoid conflicts
	compiler := jsonschema.NewCompiler()

	// Generate unique URL based on schema content
	schemaURL := fmt.Sprintf("https://example.com/schema-%s.json", cacheKey[:8])

	// Add the schema as a resource to the compiler
	if err := compiler.AddResource(schemaURL, strings.NewReader(string(schemaBytes))); err != nil {
		return nil, fmt.Errorf("add schema resource: %w", err)
	}

	// Compile the schema
	schema, err := compiler.Compile(schemaURL)
	compileDuration := time.Since(compileStart)

	if err != nil {
		v.logger.WithComponent("schema_validator").
			WithError(err).
			WithDuration(compileDuration).
			WithFields(map[string]interface{}{
				"cache_hit":         false,
				"schema_size_bytes": len(schemaBytes),
			}).
			Error("Schema compilation failed")
		return nil, fmt.Errorf("compile schema: %w", err)
	}

	// Store in cache for future use
	v.cache.Set(cacheKey, schema)

	v.logger.WithComponent("schema_validator").
		WithDuration(compileDuration).
		WithFields(map[string]interface{}{
			"cache_hit":         false,
			"cache_size":        v.cache.Size(),
			"schema_size_bytes": len(schemaBytes),
		}).
		Debug("Schema compiled and cached")

	return schema, nil
}
