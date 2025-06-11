package builders

import (
	"fmt"
	"reflect"

	"github.com/ha1tch/queryfy"
)

// ArraySchema validates array/slice values.
type ArraySchema struct {
	queryfy.BaseSchema
	elementSchema queryfy.Schema
	minItems      *int
	maxItems      *int
	uniqueItems   bool
	validators    []queryfy.ValidatorFunc
}

// Array creates a new array schema builder.
func Array() *ArraySchema {
	return &ArraySchema{
		BaseSchema: queryfy.BaseSchema{
			SchemaType: queryfy.TypeArray,
		},
	}
}

// Required marks the field as required.
func (s *ArraySchema) Required() *ArraySchema {
	s.SetRequired(true)
	return s
}

// Optional marks the field as optional (default).
func (s *ArraySchema) Optional() *ArraySchema {
	s.SetRequired(false)
	return s
}

// Nullable allows the field to be null.
func (s *ArraySchema) Nullable() *ArraySchema {
	s.SetNullable(true)
	return s
}

// Of sets the schema for array elements.
func (s *ArraySchema) Of(schema queryfy.Schema) *ArraySchema {
	s.elementSchema = schema
	return s
}

// MinItems sets the minimum number of items.
func (s *ArraySchema) MinItems(min int) *ArraySchema {
	s.minItems = &min
	return s
}

// MaxItems sets the maximum number of items.
func (s *ArraySchema) MaxItems(max int) *ArraySchema {
	s.maxItems = &max
	return s
}

// Length sets both minimum and maximum items to the same value.
func (s *ArraySchema) Length(length int) *ArraySchema {
	s.minItems = &length
	s.maxItems = &length
	return s
}

// UniqueItems requires all items to be unique.
func (s *ArraySchema) UniqueItems() *ArraySchema {
	s.uniqueItems = true
	return s
}

// Custom adds a custom validator function.
func (s *ArraySchema) Custom(fn queryfy.ValidatorFunc) *ArraySchema {
	s.validators = append(s.validators, fn)
	return s
}

// Validate implements the Schema interface.
func (s *ArraySchema) Validate(value interface{}, ctx *queryfy.ValidationContext) error {
	if !s.CheckRequired(value, ctx) {
		return nil
	}

	// Type validation
	if !queryfy.ValidateValue(value, queryfy.TypeArray, ctx) {
		return nil
	}

	// Convert to slice for validation
	slice := reflect.ValueOf(value)
	length := slice.Len()

	// Length validation
	if s.minItems != nil && length < *s.minItems {
		ctx.AddError(fmt.Sprintf("must have at least %d items, got %d", *s.minItems, length), value)
	}

	if s.maxItems != nil && length > *s.maxItems {
		ctx.AddError(fmt.Sprintf("must have at most %d items, got %d", *s.maxItems, length), value)
	}

	// Unique items validation
	if s.uniqueItems && length > 1 {
		seen := make(map[string]bool)
		for i := 0; i < length; i++ {
			// Simple string representation for uniqueness check
			// In production, this would need better handling
			key := fmt.Sprintf("%v", slice.Index(i).Interface())
			if seen[key] {
				ctx.AddError("items must be unique", value)
				break
			}
			seen[key] = true
		}
	}

	// Element validation
	if s.elementSchema != nil {
		for i := 0; i < length; i++ {
			ctx.WithIndex(i, func() {
				s.elementSchema.Validate(slice.Index(i).Interface(), ctx)
			})
		}
	}

	// Custom validators
	for _, validator := range s.validators {
		if err := validator(value); err != nil {
			ctx.AddError(err.Error(), value)
		}
	}

	return nil
}

// Type implements the Schema interface.
func (s *ArraySchema) Type() queryfy.SchemaType {
	return queryfy.TypeArray
}
