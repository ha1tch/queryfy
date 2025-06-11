package builders

import (
	"fmt"
	"math"

	"github.com/ha1tch/queryfy"
)

// NumberSchema validates numeric values.
type NumberSchema struct {
	queryfy.BaseSchema
	min        *float64
	max        *float64
	multipleOf *float64
	validators []queryfy.ValidatorFunc
}

// Number creates a new number schema builder.
func Number() *NumberSchema {
	return &NumberSchema{
		BaseSchema: queryfy.BaseSchema{
			SchemaType: queryfy.TypeNumber,
		},
	}
}

// Required marks the field as required.
func (s *NumberSchema) Required() *NumberSchema {
	s.SetRequired(true)
	return s
}

// Optional marks the field as optional (default).
func (s *NumberSchema) Optional() *NumberSchema {
	s.SetRequired(false)
	return s
}

// Nullable allows the field to be null.
func (s *NumberSchema) Nullable() *NumberSchema {
	s.SetNullable(true)
	return s
}

// Min sets the minimum value (inclusive).
func (s *NumberSchema) Min(min float64) *NumberSchema {
	s.min = &min
	return s
}

// Max sets the maximum value (inclusive).
func (s *NumberSchema) Max(max float64) *NumberSchema {
	s.max = &max
	return s
}

// Range sets both minimum and maximum values.
func (s *NumberSchema) Range(min, max float64) *NumberSchema {
	s.min = &min
	s.max = &max
	return s
}

// MultipleOf validates that the number is a multiple of the given value.
func (s *NumberSchema) MultipleOf(value float64) *NumberSchema {
	s.multipleOf = &value
	return s
}

// Integer validates that the number is an integer (no decimal part).
func (s *NumberSchema) Integer() *NumberSchema {
	s.validators = append(s.validators, func(value interface{}) error {
		num := toFloat64(value)
		if num != math.Floor(num) {
			return fmt.Errorf("must be an integer")
		}
		return nil
	})
	return s
}

// Positive validates that the number is positive (> 0).
func (s *NumberSchema) Positive() *NumberSchema {
	zero := 0.0
	s.min = &zero
	s.validators = append(s.validators, func(value interface{}) error {
		if toFloat64(value) <= 0 {
			return fmt.Errorf("must be positive")
		}
		return nil
	})
	return s
}

// Negative validates that the number is negative (< 0).
func (s *NumberSchema) Negative() *NumberSchema {
	zero := 0.0
	s.max = &zero
	s.validators = append(s.validators, func(value interface{}) error {
		if toFloat64(value) >= 0 {
			return fmt.Errorf("must be negative")
		}
		return nil
	})
	return s
}

// Custom adds a custom validator function.
func (s *NumberSchema) Custom(fn queryfy.ValidatorFunc) *NumberSchema {
	s.validators = append(s.validators, fn)
	return s
}

// Validate implements the Schema interface.
func (s *NumberSchema) Validate(value interface{}, ctx *queryfy.ValidationContext) error {
	if !s.CheckRequired(value, ctx) {
		return nil
	}

	// Get numeric value
	num, ok := toFloat64WithMode(value, ctx.Mode())
	if !ok {
		ctx.AddError(fmt.Sprintf("expected number, got %T", value), value)
		return nil
	}

	// Range validation
	if s.min != nil && num < *s.min {
		ctx.AddError(fmt.Sprintf("must be >= %v", *s.min), num)
	}

	if s.max != nil && num > *s.max {
		ctx.AddError(fmt.Sprintf("must be <= %v", *s.max), num)
	}

	// Multiple validation
	if s.multipleOf != nil && *s.multipleOf != 0 {
		if math.Mod(num, *s.multipleOf) != 0 {
			ctx.AddError(fmt.Sprintf("must be a multiple of %v", *s.multipleOf), num)
		}
	}

	// Custom validators - pass the converted number
	for _, validator := range s.validators {
		if err := validator(num); err != nil {
			ctx.AddError(err.Error(), num)
		}
	}

	return nil
}

// Type implements the Schema interface.
func (s *NumberSchema) Type() queryfy.SchemaType {
	return queryfy.TypeNumber
}

// toFloat64 converts various numeric types to float64.
func toFloat64(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int8:
		return float64(v)
	case int16:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case uint:
		return float64(v)
	case uint8:
		return float64(v)
	case uint16:
		return float64(v)
	case uint32:
		return float64(v)
	case uint64:
		return float64(v)
	default:
		return 0
	}
}

// toFloat64WithMode converts value to float64 considering validation mode.
func toFloat64WithMode(value interface{}, mode queryfy.ValidationMode) (float64, bool) {
	// Try direct numeric conversion first
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	}

	// In loose mode, try string conversion
	if mode == queryfy.Loose {
		if str, ok := value.(string); ok {
			return queryfy.ConvertStringToNumber(str)
		}
	}

	return 0, false
}
