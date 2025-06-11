package builders

import (
	"github.com/ha1tch/queryfy"
)

// BoolSchema validates boolean values.
type BoolSchema struct {
	queryfy.BaseSchema
	validators []queryfy.ValidatorFunc
}

// Bool creates a new boolean schema builder.
func Bool() *BoolSchema {
	return &BoolSchema{
		BaseSchema: queryfy.BaseSchema{
			SchemaType: queryfy.TypeBool,
		},
	}
}

// Required marks the field as required.
func (s *BoolSchema) Required() *BoolSchema {
	s.SetRequired(true)
	return s
}

// Optional marks the field as optional (default).
func (s *BoolSchema) Optional() *BoolSchema {
	s.SetRequired(false)
	return s
}

// Nullable allows the field to be null.
func (s *BoolSchema) Nullable() *BoolSchema {
	s.SetNullable(true)
	return s
}

// Custom adds a custom validator function.
func (s *BoolSchema) Custom(fn queryfy.ValidatorFunc) *BoolSchema {
	s.validators = append(s.validators, fn)
	return s
}

// Validate implements the Schema interface.
func (s *BoolSchema) Validate(value interface{}, ctx *queryfy.ValidationContext) error {
	if !s.CheckRequired(value, ctx) {
		return nil
	}

	// Type validation
	if !queryfy.ValidateValue(value, queryfy.TypeBool, ctx) {
		return nil
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
func (s *BoolSchema) Type() queryfy.SchemaType {
	return queryfy.TypeBool
}
