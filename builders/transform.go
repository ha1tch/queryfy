// transform.go - Transformation support for Queryfy
package builders

import (
	"context"
	"fmt"

	"github.com/ha1tch/queryfy"
)

// Transformer is a function that transforms a value.
// It returns the transformed value and any error.
type Transformer func(value interface{}) (interface{}, error)

// TransformSchema adds transformation capabilities to any schema.
type TransformSchema struct {
	queryfy.BaseSchema
	innerSchema     queryfy.Schema
	transformers    []Transformer
	asyncValidators []queryfy.AsyncValidatorFunc
}

// Transform creates a new transform schema wrapping an existing schema.
func Transform(schema queryfy.Schema) *TransformSchema {
	return &TransformSchema{
		BaseSchema: queryfy.BaseSchema{
			SchemaType: queryfy.TypeTransform,
		},
		innerSchema:     schema,
		transformers:    []Transformer{},
		asyncValidators: []queryfy.AsyncValidatorFunc{},
	}
}

// Add adds a transformer to the pipeline.
func (s *TransformSchema) Add(transformer Transformer) *TransformSchema {
	s.transformers = append(s.transformers, transformer)
	return s
}

// Required marks the field as required.
func (s *TransformSchema) Required() *TransformSchema {
	s.SetRequired(true)
	if setter, ok := s.innerSchema.(interface{ SetRequired(bool) }); ok {
		setter.SetRequired(true)
	}
	return s
}

// Optional marks the field as optional.
func (s *TransformSchema) Optional() *TransformSchema {
	s.SetRequired(false)
	if setter, ok := s.innerSchema.(interface{ SetRequired(bool) }); ok {
		setter.SetRequired(false)
	}
	return s
}

// IsRequired returns true if either the transform wrapper or the inner schema
// is marked as required. This ensures that wrapping a Required() schema with
// Transform() preserves the required semantics.
func (s *TransformSchema) IsRequired() bool {
	if s.BaseSchema.IsRequired() {
		return true
	}
	if requirer, ok := s.innerSchema.(interface{ IsRequired() bool }); ok {
		return requirer.IsRequired()
	}
	return false
}

// Validate implements the Schema interface.
func (s *TransformSchema) Validate(value interface{}, ctx *queryfy.ValidationContext) error {
	// Apply transformations first
	transformed := value
	for i, transformer := range s.transformers {
		original := transformed
		result, err := transformer(transformed)
		if err != nil {
			ctx.AddError(fmt.Sprintf("transformation %d failed: %s", i+1, err.Error()), transformed)
			return nil
		}
		// Record the transformation
		ctx.RecordTransformation(original, result, fmt.Sprintf("transform_%d", i+1))
		transformed = result
	}

	// Then validate the transformed value
	return s.innerSchema.Validate(transformed, ctx)
}

// ValidateAndTransform validates and returns the transformed value.
func (s *TransformSchema) ValidateAndTransform(value interface{}, ctx *queryfy.ValidationContext) (interface{}, error) {
	// Apply transformations
	transformed := value
	for i, transformer := range s.transformers {
		original := transformed
		result, err := transformer(transformed)
		if err != nil {
			ctx.AddError(fmt.Sprintf("transformation %d failed: %s", i+1, err.Error()), transformed)
			return value, ctx.Error()
		}
		// Record the transformation
		ctx.RecordTransformation(original, result, fmt.Sprintf("transform_%d", i+1))
		transformed = result
	}

	// Validate the transformed value
	if err := s.innerSchema.Validate(transformed, ctx); err != nil {
		return transformed, err
	}

	return transformed, ctx.Error()
}

// InnerSchema returns the wrapped schema.
func (s *TransformSchema) InnerSchema() queryfy.Schema {
	return s.innerSchema
}

// Meta attaches a key-value metadata pair to the schema.
func (s *TransformSchema) Meta(key string, value interface{}) *TransformSchema {
	s.SetMeta(key, value)
	return s
}

// Type implements the Schema interface.
func (s *TransformSchema) Type() queryfy.SchemaType {
	return s.innerSchema.Type()
}

// AsyncCustom adds an async validator to the schema. Async validators are
// only invoked by ValidateAndTransformAsync; sync methods ignore them.
func (s *TransformSchema) AsyncCustom(fn queryfy.AsyncValidatorFunc) *TransformSchema {
	s.asyncValidators = append(s.asyncValidators, fn)
	return s
}

// HasAsyncValidators returns true if async validators are registered.
func (s *TransformSchema) HasAsyncValidators() bool {
	return len(s.asyncValidators) > 0
}

// ValidateAndTransformAsync runs sync validation and transformations first.
// If sync validation passes, it then runs async validators sequentially
// with the provided context. The async validators receive the transformed
// value, not the original input.
func (s *TransformSchema) ValidateAndTransformAsync(goCtx context.Context, value interface{}, ctx *queryfy.ValidationContext) (interface{}, error) {
	// Run sync validation and transformation first
	transformed, err := s.ValidateAndTransform(value, ctx)
	if err != nil {
		return transformed, err
	}

	// If sync passed, run async validators sequentially
	for _, asyncValidator := range s.asyncValidators {
		// Check context cancellation before each validator
		if goCtx.Err() != nil {
			ctx.AddError(fmt.Sprintf("validation cancelled: %s", goCtx.Err()), transformed)
			return transformed, ctx.Error()
		}

		if err := asyncValidator(goCtx, transformed); err != nil {
			ctx.AddError(err.Error(), transformed)
		}
	}

	return transformed, ctx.Error()
}

// Extension methods for existing builders to support Transform()

// Transform adds transformation capability to StringSchema.
func (s *StringSchema) Transform(transformer Transformer) *TransformSchema {
	ts := Transform(s)
	return ts.Add(transformer)
}

// Transform adds transformation capability to NumberSchema.
func (s *NumberSchema) Transform(transformer Transformer) *TransformSchema {
	ts := Transform(s)
	return ts.Add(transformer)
}

// Transform adds transformation capability to BoolSchema.
func (s *BoolSchema) Transform(transformer Transformer) *TransformSchema {
	ts := Transform(s)
	return ts.Add(transformer)
}

// Transform adds transformation capability to ArraySchema.
func (s *ArraySchema) Transform(transformer Transformer) *TransformSchema {
	ts := Transform(s)
	return ts.Add(transformer)
}

// Transform adds transformation capability to ObjectSchema.
func (s *ObjectSchema) Transform(transformer Transformer) *TransformSchema {
	ts := Transform(s)
	return ts.Add(transformer)
}

// Transform adds transformation capability to DateTimeSchema.
func (s *DateTimeSchema) Transform(transformer Transformer) *TransformSchema {
	ts := Transform(s)
	return ts.Add(transformer)
}
