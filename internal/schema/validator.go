package schema

import (
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

type Validator struct {
	compiler *jsonschema.Compiler
}

func NewValidator() *Validator {
	return &Validator{
		compiler: jsonschema.NewCompiler(),
	}
}

func (v *Validator) ValidateResponse(schemaBytes json.RawMessage, response interface{}) error {
	schema, err := v.compiler.Compile(string(schemaBytes))
	if err != nil {
		return fmt.Errorf("compile schema: %w", err)
	}

	if err := schema.Validate(response); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}

func (v *Validator) ValidateSchema(schemaBytes json.RawMessage) error {
	_, err := v.compiler.Compile(string(schemaBytes))
	if err != nil {
		return fmt.Errorf("invalid schema: %w", err)
	}
	return nil
}