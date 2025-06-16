package schema

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
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
	// Parse JSON first to ensure it's valid
	var schemaObj interface{}
	if err := json.Unmarshal(schemaBytes, &schemaObj); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Create a new compiler for each validation to avoid conflicts
	compiler := jsonschema.NewCompiler()

	// Generate unique URL based on schema content
	hash := sha256.Sum256(schemaBytes)
	schemaURL := fmt.Sprintf("https://example.com/schema-%x.json", hash[:8])

	// Add the schema as a resource to the compiler
	if err := compiler.AddResource(schemaURL, strings.NewReader(string(schemaBytes))); err != nil {
		return nil, fmt.Errorf("add schema resource: %w", err)
	}

	// Compile the schema
	schema, err := compiler.Compile(schemaURL)
	if err != nil {
		return nil, fmt.Errorf("compile schema: %w", err)
	}

	return schema, nil
}