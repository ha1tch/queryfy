package queryfy

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

// BaseSchema provides common functionality for all schema types.
// It should be embedded in concrete schema implementations.
type BaseSchema struct {
	SchemaType SchemaType  // Changed from schemaType to SchemaType to make it accessible
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