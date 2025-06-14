// transform.go - Transformation support for Queryfy
package builders

import (
	"fmt"

	"github.com/ha1tch/queryfy"
)

// Transformer is a function that transforms a value.
// It returns the transformed value and any error.
type Transformer func(value interface{}) (interface{}, error)

// TransformSchema adds transformation capabilities to any schema.
type TransformSchema struct {
	queryfy.BaseSchema
	innerSchema  queryfy.Schema
	transformers []Transformer
}

// Transform creates a new transform schema wrapping an existing schema.
func Transform(schema queryfy.Schema) *TransformSchema {
	return &TransformSchema{
		BaseSchema: queryfy.BaseSchema{
			SchemaType: queryfy.TypeTransform,
		},
		innerSchema:  schema,
		transformers: []Transformer{},
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

// Type implements the Schema interface.
func (s *TransformSchema) Type() queryfy.SchemaType {
	return s.innerSchema.Type()
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
