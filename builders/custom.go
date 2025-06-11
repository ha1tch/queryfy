package builders

import (
	"github.com/ha1tch/queryfy"
)

// CustomSchema allows custom validation logic.
type CustomSchema struct {
	queryfy.BaseSchema
	validator queryfy.ValidatorFunc
}

// Custom creates a new custom schema with a validator function.
func Custom(validator queryfy.ValidatorFunc) *CustomSchema {
	return &CustomSchema{
		BaseSchema: queryfy.BaseSchema{
			SchemaType: queryfy.TypeCustom,
		},
		validator: validator,
	}
}

// Required marks the field as required.
func (s *CustomSchema) Required() *CustomSchema {
	s.SetRequired(true)
	return s
}

// Optional marks the field as optional (default).
func (s *CustomSchema) Optional() *CustomSchema {
	s.SetRequired(false)
	return s
}

// Nullable allows the field to be null.
func (s *CustomSchema) Nullable() *CustomSchema {
	s.SetNullable(true)
	return s
}

// Validate implements the Schema interface.
func (s *CustomSchema) Validate(value interface{}, ctx *queryfy.ValidationContext) error {
	if !s.CheckRequired(value, ctx) {
		return nil
	}

	if s.validator != nil {
		if err := s.validator(value); err != nil {
			ctx.AddError(err.Error(), value)
		}
	}

	return nil
}

// Type implements the Schema interface.
func (s *CustomSchema) Type() queryfy.SchemaType {
	return queryfy.TypeCustom
}
