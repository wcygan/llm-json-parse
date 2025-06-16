package integration

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wcygan/llm-json-parse/internal/schema"
	"github.com/wcygan/llm-json-parse/pkg/types"
)

func TestSchemaValidatorIntegration(t *testing.T) {
	validator := schema.NewValidator()

	tests := []struct {
		name        string
		schema      string
		data        interface{}
		expectValid bool
	}{
		{
			name: "simple_object_valid",
			schema: `{
				"type": "object",
				"properties": {
					"name": {"type": "string"},
					"age": {"type": "number"}
				},
				"required": ["name"]
			}`,
			data: map[string]interface{}{
				"name": "John",
				"age":  25,
			},
			expectValid: true,
		},
		{
			name: "simple_object_missing_required",
			schema: `{
				"type": "object",
				"properties": {
					"name": {"type": "string"},
					"age": {"type": "number"}
				},
				"required": ["name"]
			}`,
			data: map[string]interface{}{
				"age": 25,
			},
			expectValid: false,
		},
		{
			name: "nested_object_valid",
			schema: `{
				"type": "object",
				"properties": {
					"user": {
						"type": "object",
						"properties": {
							"name": {"type": "string"},
							"email": {"type": "string"}
						},
						"required": ["name"]
					},
					"preferences": {
						"type": "object",
						"properties": {
							"theme": {"type": "string"},
							"notifications": {"type": "boolean"}
						}
					}
				},
				"required": ["user"]
			}`,
			data: map[string]interface{}{
				"user": map[string]interface{}{
					"name":  "John",
					"email": "john@example.com",
				},
				"preferences": map[string]interface{}{
					"theme":         "dark",
					"notifications": true,
				},
			},
			expectValid: true,
		},
		{
			name: "array_validation_valid",
			schema: `{
				"type": "object",
				"properties": {
					"items": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"id": {"type": "integer"},
								"name": {"type": "string"}
							},
							"required": ["id"]
						}
					}
				}
			}`,
			data: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"id": 1, "name": "Item 1"},
					map[string]interface{}{"id": 2, "name": "Item 2"},
				},
			},
			expectValid: true,
		},
		{
			name: "array_validation_invalid_item",
			schema: `{
				"type": "object",
				"properties": {
					"items": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"id": {"type": "integer"},
								"name": {"type": "string"}
							},
							"required": ["id"]
						}
					}
				}
			}`,
			data: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"id": 1, "name": "Item 1"},
					map[string]interface{}{"name": "Item 2"}, // Missing required "id"
				},
			},
			expectValid: false,
		},
		{
			name: "strict_schema_no_additional_properties",
			schema: `{
				"type": "object",
				"properties": {
					"name": {"type": "string"}
				},
				"additionalProperties": false
			}`,
			data: map[string]interface{}{
				"name": "John",
				"age":  25, // This should cause validation to fail
			},
			expectValid: false,
		},
		{
			name: "type_coercion_number_as_string",
			schema: `{
				"type": "object",
				"properties": {
					"count": {"type": "number"}
				}
			}`,
			data: map[string]interface{}{
				"count": "25", // String instead of number
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test schema validation
			err := validator.ValidateSchema(json.RawMessage(tt.schema))
			require.NoError(t, err, "Schema should be valid")

			// Test data validation
			dataJSON, _ := json.Marshal(tt.data)
			response := &types.ValidatedResponse{
				Data: json.RawMessage(dataJSON),
			}
			err = validator.ValidateResponse(json.RawMessage(tt.schema), response)
			if tt.expectValid {
				assert.NoError(t, err, "Data should be valid according to schema")
			} else {
				assert.Error(t, err, "Data should be invalid according to schema")
			}
		})
	}
}

func TestComplexSchemaValidation(t *testing.T) {
	validator := schema.NewValidator()

	// Recipe analysis schema from the README example
	recipeSchema := `{
		"type": "object",
		"properties": {
			"recipe_name": {
				"type": "string",
				"description": "Name of the recipe"
			},
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
	}`

	validRecipeData := map[string]interface{}{
		"recipe_name": "Chocolate Chip Cookies",
		"ingredients": []interface{}{
			map[string]interface{}{
				"name":   "flour",
				"amount": "2",
				"unit":   "cups",
			},
			map[string]interface{}{
				"name":   "sugar",
				"amount": "1",
				"unit":   "cup",
			},
		},
		"steps": []interface{}{
			map[string]interface{}{
				"step_number":      1,
				"instruction":      "Mix dry ingredients",
				"duration_minutes": 5,
			},
			map[string]interface{}{
				"step_number": 2,
				"instruction": "Add wet ingredients",
			},
		},
		"cooking_details": map[string]interface{}{
			"prep_time_minutes":      15,
			"cook_time_minutes":      12,
			"temperature_fahrenheit": 350,
			"servings":               24,
		},
	}

	t.Run("valid_complex_recipe", func(t *testing.T) {
		err := validator.ValidateSchema(json.RawMessage(recipeSchema))
		require.NoError(t, err)

		validDataJSON, _ := json.Marshal(validRecipeData)
		validResponse := &types.ValidatedResponse{
			Data: json.RawMessage(validDataJSON),
		}
		err = validator.ValidateResponse(json.RawMessage(recipeSchema), validResponse)
		assert.NoError(t, err)
	})

	t.Run("invalid_complex_recipe_missing_required", func(t *testing.T) {
		invalidData := make(map[string]interface{})
		for k, v := range validRecipeData {
			invalidData[k] = v
		}
		delete(invalidData, "recipe_name") // Remove required field

		invalidDataJSON, _ := json.Marshal(invalidData)
		invalidResponse := &types.ValidatedResponse{
			Data: json.RawMessage(invalidDataJSON),
		}
		err := validator.ValidateResponse(json.RawMessage(recipeSchema), invalidResponse)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing properties")
	})

	t.Run("invalid_complex_recipe_wrong_array_item", func(t *testing.T) {
		invalidData := make(map[string]interface{})
		for k, v := range validRecipeData {
			invalidData[k] = v
		}

		// Set invalid ingredient (missing required "amount")
		invalidData["ingredients"] = []interface{}{
			map[string]interface{}{
				"name": "flour",
				"unit": "cups",
				// Missing "amount"
			},
		}

		invalidDataJSON, _ := json.Marshal(invalidData)
		invalidResponse := &types.ValidatedResponse{
			Data: json.RawMessage(invalidDataJSON),
		}
		err := validator.ValidateResponse(json.RawMessage(recipeSchema), invalidResponse)
		assert.Error(t, err)
	})
}

func TestInvalidSchemas(t *testing.T) {
	validator := schema.NewValidator()

	invalidSchemas := []struct {
		name   string
		schema string
	}{
		{
			name:   "invalid_json",
			schema: `{"type": "object"`, // Missing closing brace
		},
		{
			name:   "invalid_type",
			schema: `{"type": "invalid_type"}`,
		},
		{
			name:   "malformed_properties",
			schema: `{"type": "object", "properties": "not an object"}`,
		},
	}

	for _, tt := range invalidSchemas {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateSchema(json.RawMessage(tt.schema))
			assert.Error(t, err, "Invalid schema should return error")
		})
	}
}
