// Package queryfy provides schema validation and querying for dynamic data in Go.
//
// Queryfy is designed to work with map[string]interface{} data structures
// commonly found in JSON APIs, configuration files, and other dynamic contexts.
//
// Basic usage:
//
//	schema := queryfy.Object().
//		Field("id", queryfy.String().Required()).
//		Field("amount", queryfy.Number().Min(0))
//
//	data := map[string]interface{}{
//		"id": "order-123",
//		"amount": 99.99,
//	}
//
//	if err := queryfy.Validate(data, schema); err != nil {
//		// Handle validation error
//	}
//
//	// Query the data
//	amount, _ := queryfy.Query(data, "amount")
package queryfy

import (
	"context"
	"fmt"

	"github.com/ha1tch/queryfy/query"
)

// Validate validates data against a schema.
// Returns a ValidationError containing all validation failures, or nil if valid.
func Validate(data interface{}, schema Schema) error {
	return ValidateWithMode(data, schema, Strict)
}

// ValidateWithMode validates data against a schema with a specific validation mode.
func ValidateWithMode(data interface{}, schema Schema, mode ValidationMode) error {
	ctx := NewValidationContext(mode)
	if err := schema.Validate(data, ctx); err != nil {
		return err
	}
	return ctx.Error()
}

// Query executes a query against the data and returns the result.
// Supports dot notation and array indexing:
//   - "name" - returns the value of field "name"
//   - "user.email" - returns nested field value
//   - "items[0]" - returns first element of array
//   - "items[0].price" - returns field from array element
func Query(data interface{}, queryStr string) (interface{}, error) {
	return query.Execute(data, queryStr)
}

// NewValidator creates a new validator with a schema.
// The validator can be configured with different modes and options.
func NewValidator(schema Schema) *Validator {
	return &Validator{
		schema: schema,
		mode:   Strict,
	}
}

// MustValidate validates data against a schema and panics on error.
// This is useful in initialization code where validation errors are fatal.
func MustValidate(data interface{}, schema Schema) {
	if err := Validate(data, schema); err != nil {
		panic(fmt.Sprintf("validation failed: %v", err))
	}
}

// ValidateAndTransform validates data against a schema and returns
// the transformed result. If the schema does not support transformation,
// it falls back to plain validation and returns the original data.
func ValidateAndTransform(data interface{}, schema Schema, mode ValidationMode) (interface{}, error) {
	ctx := NewValidationContext(mode)
	if ts, ok := schema.(TransformableSchema); ok {
		return ts.ValidateAndTransform(data, ctx)
	}
	// Fall back to plain validation
	schema.Validate(data, ctx)
	return data, ctx.Error()
}

// ValidateAndTransformAsync validates data with async validators and
// returns the transformed result. If the schema has no async validators,
// it falls back to synchronous ValidateAndTransform.
func ValidateAndTransformAsync(goCtx context.Context, data interface{}, schema Schema, mode ValidationMode) (interface{}, error) {
	ctx := NewValidationContext(mode)
	if as, ok := schema.(AsyncTransformableSchema); ok && as.HasAsyncValidators() {
		return as.ValidateAndTransformAsync(goCtx, data, ctx)
	}
	// Fall back to sync
	if ts, ok := schema.(TransformableSchema); ok {
		return ts.ValidateAndTransform(data, ctx)
	}
	schema.Validate(data, ctx)
	return data, ctx.Error()
}
