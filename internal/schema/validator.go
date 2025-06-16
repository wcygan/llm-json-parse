package schema

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
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
	cache *SchemaCache
}

func NewValidator() *Validator {
	return &Validator{
		cache: NewSchemaCache(100), // Cache up to 100 compiled schemas
	}
}

func (v *Validator) ValidateResponse(schemaBytes json.RawMessage, response interface{}) error {
	schema, err := v.compileSchema(schemaBytes)
	if err != nil {
		return fmt.Errorf("compile schema: %w", err)
	}

	if err := schema.Validate(response); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}

func (v *Validator) ValidateSchema(schemaBytes json.RawMessage) error {
	_, err := v.compileSchema(schemaBytes)
	if err != nil {
		return fmt.Errorf("invalid schema: %w", err)
	}
	return nil
}

func (v *Validator) compileSchema(schemaBytes json.RawMessage) (*jsonschema.Schema, error) {
	// Generate cache key based on schema content
	hash := sha256.Sum256(schemaBytes)
	cacheKey := fmt.Sprintf("%x", hash[:16]) // Use first 16 bytes for shorter key

	// Check cache first
	if schema, exists := v.cache.Get(cacheKey); exists {
		return schema, nil
	}

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
	if err != nil {
		return nil, fmt.Errorf("compile schema: %w", err)
	}

	// Store in cache for future use
	v.cache.Set(cacheKey, schema)

	return schema, nil
}
