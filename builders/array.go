package builders

import (
	"context"
	"fmt"
	"reflect"

	"github.com/ha1tch/queryfy"
)

// ArraySchema validates array/slice values.
type ArraySchema struct {
	queryfy.BaseSchema
	elementSchema   queryfy.Schema
	minItems        *int
	maxItems        *int
	uniqueItems     bool
	validators      []queryfy.ValidatorFunc
	asyncValidators []queryfy.AsyncValidatorFunc
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

// Items sets the schema for array elements.
// This is an alias for Of, provided for readability with inline schemas.
func (s *ArraySchema) Items(schema queryfy.Schema) *ArraySchema {
	return s.Of(schema)
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

// ElementSchema returns the schema for array elements, or nil if not set.
func (s *ArraySchema) ElementSchema() queryfy.Schema {
	return s.elementSchema
}

// ItemCountConstraints returns the min and max item count pointers,
// either of which may be nil if not set.
func (s *ArraySchema) ItemCountConstraints() (min, max *int) {
	return s.minItems, s.maxItems
}

// IsUniqueItems reports whether the array requires unique items.
func (s *ArraySchema) IsUniqueItems() bool {
	return s.uniqueItems
}

// Meta attaches a key-value metadata pair to the schema.
func (s *ArraySchema) Meta(key string, value interface{}) *ArraySchema {
	s.SetMeta(key, value)
	return s
}

// Validators returns the custom validator functions.
func (s *ArraySchema) Validators() []queryfy.ValidatorFunc {
	return s.validators
}

// Type implements the Schema interface.
func (s *ArraySchema) Type() queryfy.SchemaType {
	return queryfy.TypeArray
}

// ValidateAndTransform validates the array and returns a new slice with all
// element transformations applied. If the element schema implements
// ValidateAndTransform, each element is transformed; otherwise elements are
// validated normally and their original values are preserved.
func (s *ArraySchema) ValidateAndTransform(value interface{}, ctx *queryfy.ValidationContext) (interface{}, error) {
	if !s.CheckRequired(value, ctx) {
		return value, ctx.Error()
	}

	// Type validation
	if !queryfy.ValidateValue(value, queryfy.TypeArray, ctx) {
		return value, ctx.Error()
	}

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
			key := fmt.Sprintf("%v", slice.Index(i).Interface())
			if seen[key] {
				ctx.AddError("items must be unique", value)
				break
			}
			seen[key] = true
		}
	}

	// Build result slice with transformed elements
	result := make([]interface{}, length)
	for i := 0; i < length; i++ {
		elem := slice.Index(i).Interface()
		ctx.WithIndex(i, func() {
			if s.elementSchema != nil {
				if ts, ok := s.elementSchema.(interface {
					ValidateAndTransform(interface{}, *queryfy.ValidationContext) (interface{}, error)
				}); ok {
					transformed, _ := ts.ValidateAndTransform(elem, ctx)
					result[i] = transformed
				} else {
					s.elementSchema.Validate(elem, ctx)
					result[i] = elem
				}
			} else {
				result[i] = elem
			}
		})
	}

	// Custom validators
	for _, validator := range s.validators {
		if err := validator(value); err != nil {
			ctx.AddError(err.Error(), value)
		}
	}

	return result, ctx.Error()
}

// AsyncCustom adds an async validator to the array schema. Async validators
// are only invoked by ValidateAndTransformAsync; sync methods ignore them.
func (s *ArraySchema) AsyncCustom(fn queryfy.AsyncValidatorFunc) *ArraySchema {
	s.asyncValidators = append(s.asyncValidators, fn)
	return s
}

// HasAsyncValidators returns true if any async validators are registered
// on this schema or on its element schema.
func (s *ArraySchema) HasAsyncValidators() bool {
	if len(s.asyncValidators) > 0 {
		return true
	}
	if s.elementSchema != nil {
		if checker, ok := s.elementSchema.(interface{ HasAsyncValidators() bool }); ok {
			return checker.HasAsyncValidators()
		}
	}
	return false
}

// ValidateAndTransformAsync runs sync validation and transformations first.
// If sync validation passes, it then runs async validators on elements and
// on the array itself, sequentially with the provided context.
func (s *ArraySchema) ValidateAndTransformAsync(goCtx context.Context, value interface{}, ctx *queryfy.ValidationContext) (interface{}, error) {
	// Run sync validation and transformation first
	result, err := s.ValidateAndTransform(value, ctx)
	if err != nil {
		return result, err
	}

	// Check context cancellation before async phase
	if goCtx.Err() != nil {
		ctx.AddError(fmt.Sprintf("validation cancelled: %s", goCtx.Err()), result)
		return result, ctx.Error()
	}

	// Run async validators on elements if element schema supports it
	if s.elementSchema != nil {
		if _, ok := s.elementSchema.(interface {
			ValidateAndTransformAsync(context.Context, interface{}, *queryfy.ValidationContext) (interface{}, error)
		}); ok {
			items := result.([]interface{})
			for i, elem := range items {
				if goCtx.Err() != nil {
					ctx.AddError(fmt.Sprintf("validation cancelled: %s", goCtx.Err()), result)
					return result, ctx.Error()
				}
				ctx.WithIndex(i, func() {
					ts := s.elementSchema.(interface {
						ValidateAndTransformAsync(context.Context, interface{}, *queryfy.ValidationContext) (interface{}, error)
					})
					ts.ValidateAndTransformAsync(goCtx, elem, ctx)
				})
			}
		}
	}

	// Run array-level async validators
	for _, asyncValidator := range s.asyncValidators {
		if goCtx.Err() != nil {
			ctx.AddError(fmt.Sprintf("validation cancelled: %s", goCtx.Err()), result)
			return result, ctx.Error()
		}

		if err := asyncValidator(goCtx, result); err != nil {
			ctx.AddError(err.Error(), result)
		}
	}

	return result, ctx.Error()
}
