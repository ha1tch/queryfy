package builders

import (
	"fmt"

	"github.com/ha1tch/queryfy"
)

// AndSchema validates that all schemas pass.
type AndSchema struct {
	queryfy.BaseSchema
	schemas []queryfy.Schema
}

// And creates a new AND schema that requires all sub-schemas to pass.
func And(schemas ...queryfy.Schema) *AndSchema {
	return &AndSchema{
		BaseSchema: queryfy.BaseSchema{
			SchemaType: queryfy.TypeComposite,
		},
		schemas: schemas,
	}
}

// Required marks the field as required.
func (s *AndSchema) Required() *AndSchema {
	s.SetRequired(true)
	return s
}

// Optional marks the field as optional (default).
func (s *AndSchema) Optional() *AndSchema {
	s.SetRequired(false)
	return s
}

// Nullable allows the field to be null.
func (s *AndSchema) Nullable() *AndSchema {
	s.SetNullable(true)
	return s
}

// Validate implements the Schema interface.
func (s *AndSchema) Validate(value interface{}, ctx *queryfy.ValidationContext) error {
	if !s.CheckRequired(value, ctx) {
		return nil
	}

	// All schemas must pass
	for i, schema := range s.schemas {
		if err := schema.Validate(value, ctx); err != nil {
			// The sub-schema will have added its errors to the context
			// We don't need to do anything else
			_ = i // Just to use the variable
		}
	}

	return nil
}

// Type implements the Schema interface.
func (s *AndSchema) Type() queryfy.SchemaType {
	return queryfy.TypeComposite
}

// OrSchema validates that at least one schema passes.
type OrSchema struct {
	queryfy.BaseSchema
	schemas []queryfy.Schema
}

// Or creates a new OR schema that requires at least one sub-schema to pass.
func Or(schemas ...queryfy.Schema) *OrSchema {
	return &OrSchema{
		BaseSchema: queryfy.BaseSchema{
			SchemaType: queryfy.TypeComposite,
		},
		schemas: schemas,
	}
}

// Required marks the field as required.
func (s *OrSchema) Required() *OrSchema {
	s.SetRequired(true)
	return s
}

// Optional marks the field as optional (default).
func (s *OrSchema) Optional() *OrSchema {
	s.SetRequired(false)
	return s
}

// Nullable allows the field to be null.
func (s *OrSchema) Nullable() *OrSchema {
	s.SetNullable(true)
	return s
}

// Validate implements the Schema interface.
func (s *OrSchema) Validate(value interface{}, ctx *queryfy.ValidationContext) error {
	if !s.CheckRequired(value, ctx) {
		return nil
	}

	if len(s.schemas) == 0 {
		return nil
	}

	// Try each schema, collecting errors
	originalErrorCount := len(ctx.Errors())
	allFailed := true

	for _, schema := range s.schemas {
		// Create a temporary context to test this schema
		tempCtx := queryfy.NewValidationContext(ctx.Mode())

		// Copy the current path
		for i := 0; i < len(ctx.Errors()); i++ {
			// This is a bit hacky, but we need to preserve the path
			// In a real implementation, we'd have a method to copy the path
		}

		if err := schema.Validate(value, tempCtx); err == nil && !tempCtx.HasErrors() {
			// At least one schema passed
			allFailed = false
			break
		}
	}

	if allFailed {
		// Remove any errors added during testing
		for len(ctx.Errors()) > originalErrorCount {
			// In a real implementation, we'd have a method to pop errors
		}
		ctx.AddError("none of the validators passed", value)
	}

	return nil
}

// Type implements the Schema interface.
func (s *OrSchema) Type() queryfy.SchemaType {
	return queryfy.TypeComposite
}

// NotSchema inverts the result of another schema.
type NotSchema struct {
	queryfy.BaseSchema
	schema queryfy.Schema
}

// Not creates a new NOT schema that inverts another schema's validation.
func Not(schema queryfy.Schema) *NotSchema {
	return &NotSchema{
		BaseSchema: queryfy.BaseSchema{
			SchemaType: queryfy.TypeComposite,
		},
		schema: schema,
	}
}

// Required marks the field as required.
func (s *NotSchema) Required() *NotSchema {
	s.SetRequired(true)
	return s
}

// Optional marks the field as optional (default).
func (s *NotSchema) Optional() *NotSchema {
	s.SetRequired(false)
	return s
}

// Nullable allows the field to be null.
func (s *NotSchema) Nullable() *NotSchema {
	s.SetNullable(true)
	return s
}

// Validate implements the Schema interface.
func (s *NotSchema) Validate(value interface{}, ctx *queryfy.ValidationContext) error {
	if !s.CheckRequired(value, ctx) {
		return nil
	}

	// Create a temporary context to test the schema
	tempCtx := queryfy.NewValidationContext(ctx.Mode())

	if err := s.schema.Validate(value, tempCtx); err == nil && !tempCtx.HasErrors() {
		// Schema passed, but NOT means it should fail
		ctx.AddError("value must not match the validation", value)
	}

	return nil
}

// Type implements the Schema interface.
func (s *NotSchema) Type() queryfy.SchemaType {
	return queryfy.TypeComposite
}

// Helper method to create error message for composite schemas
func compositeErrorMessage(schemas []queryfy.Schema) string {
	if len(schemas) == 0 {
		return "no schemas defined"
	}
	if len(schemas) == 1 {
		return fmt.Sprintf("must match %s schema", schemas[0].Type())
	}
	return fmt.Sprintf("must match one of %d schemas", len(schemas))
}
