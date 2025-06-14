package queryfy

import "fmt"

// Schema represents a validation schema.
// All schema types must implement this interface.
type Schema interface {
	// Validate validates a value against this schema.
	// It should add any validation errors to the context.
	// Returns an error only for unexpected failures (not validation failures).
	Validate(value interface{}, ctx *ValidationContext) error

	// Type returns the schema type.
	Type() SchemaType
}

// Transformer represents a function that can transform values.
// This is defined here to avoid circular dependencies.
type Transformer interface {
	// Transform applies the transformation to a value
	Transform(value interface{}) (interface{}, error)
}

// BaseSchema provides common functionality for all schema types.
// It should be embedded in concrete schema implementations.
type BaseSchema struct {
	SchemaType SchemaType // Changed from schemaType to SchemaType to make it accessible
	required   bool
	nullable   bool
}

// Type returns the schema type.
func (s *BaseSchema) Type() SchemaType {
	return s.SchemaType
}

// IsRequired returns true if the field is required.
func (s *BaseSchema) IsRequired() bool {
	return s.required
}

// IsNullable returns true if the field can be null.
func (s *BaseSchema) IsNullable() bool {
	return s.nullable
}

// SetRequired sets whether the field is required.
func (s *BaseSchema) SetRequired(required bool) {
	s.required = required
}

// SetNullable sets whether the field can be null.
func (s *BaseSchema) SetNullable(nullable bool) {
	s.nullable = nullable
}

// CheckRequired checks if a required field is present and not nil.
// Returns true if validation should continue, false if it should stop.
func (s *BaseSchema) CheckRequired(value interface{}, ctx *ValidationContext) bool {
	if value == nil {
		if s.required {
			ctx.AddError("field is required", nil)
			return false
		}
		if !s.nullable {
			ctx.AddError("field cannot be null", nil)
			return false
		}
		return false // Don't continue validation for nil values
	}
	return true
}

// WithTransform wraps a schema to add transformation capability.
type WithTransform struct {
	Schema
	transformers []TransformerFunc
}

// NewWithTransform creates a new schema with transformation support.
func NewWithTransform(schema Schema) *WithTransform {
	return &WithTransform{
		Schema:       schema,
		transformers: []TransformerFunc{},
	}
}

// AddTransformer adds a transformer to the pipeline.
func (s *WithTransform) AddTransformer(t TransformerFunc) *WithTransform {
	s.transformers = append(s.transformers, t)
	return s
}

// Validate applies transformations then validates.
func (s *WithTransform) Validate(value interface{}, ctx *ValidationContext) error {
	transformed, err := s.transform(value, ctx)
	if err != nil {
		ctx.AddError(fmt.Sprintf("transformation failed: %s", err.Error()), value)
		return nil
	}
	return s.Schema.Validate(transformed, ctx)
}

// ValidateAndTransform validates and returns the transformed value.
func (s *WithTransform) ValidateAndTransform(value interface{}, ctx *ValidationContext) (interface{}, error) {
	transformed, err := s.transform(value, ctx)
	if err != nil {
		ctx.AddError(fmt.Sprintf("transformation failed: %s", err.Error()), value)
		return value, ctx.Error()
	}

	if err := s.Schema.Validate(transformed, ctx); err != nil {
		return transformed, err
	}

	return transformed, ctx.Error()
}

// transform applies all transformations in order.
func (s *WithTransform) transform(value interface{}, ctx *ValidationContext) (interface{}, error) {
	result := value
	for i, transformer := range s.transformers {
		original := result
		var err error
		result, err = transformer(result)
		if err != nil {
			return value, fmt.Errorf("transformation %d: %w", i+1, err)
		}
		// Record the transformation
		ctx.RecordTransformation(original, result, fmt.Sprintf("transform_%d", i+1))
	}
	return result, nil
}

// Type returns the underlying schema type.
func (s *WithTransform) Type() SchemaType {
	return s.Schema.Type()
}
