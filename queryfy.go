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

// Compile pre-compiles a schema for better performance when validating
// multiple times. For v0.1.0, this is a no-op that returns the schema as-is.
func Compile(schema Schema) Schema {
	return schema
}

// MustValidate validates data against a schema and panics on error.
// This is useful in initialization code where validation errors are fatal.
func MustValidate(data interface{}, schema Schema) {
	if err := Validate(data, schema); err != nil {
		panic(fmt.Sprintf("validation failed: %v", err))
	}
}
