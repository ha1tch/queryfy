package validators

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/ha1tch/queryfy"
)

// MinValidator validates minimum values for numbers and minimum length for strings/arrays.
type MinValidator struct {
	min      float64
	minInt   int
	isLength bool
}

// NewMinValidator creates a validator for minimum numeric values.
func NewMinValidator(min float64) *MinValidator {
	return &MinValidator{min: min}
}

// NewMinLengthValidator creates a validator for minimum length.
func NewMinLengthValidator(min int) *MinValidator {
	return &MinValidator{minInt: min, isLength: true}
}

// Validate checks if the value meets the minimum requirement.
func (v *MinValidator) Validate(value interface{}) error {
	if v.isLength {
		return v.validateLength(value)
	}
	return v.validateNumeric(value)
}

func (v *MinValidator) validateLength(value interface{}) error {
	length := getLength(value)
	if length < 0 {
		return &queryfy.FieldError{
			Message: fmt.Sprintf("cannot determine length of %T", value),
			Value:   value,
		}
	}

	if length < v.minInt {
		return &queryfy.FieldError{
			Message: fmt.Sprintf("length must be at least %d, got %d", v.minInt, length),
			Value:   value,
		}
	}

	return nil
}

func (v *MinValidator) validateNumeric(value interface{}) error {
	num, ok := toNumber(value)
	if !ok {
		return &queryfy.FieldError{
			Message: fmt.Sprintf("cannot convert %T to number", value),
			Value:   value,
		}
	}

	if num < v.min {
		return &queryfy.FieldError{
			Message: fmt.Sprintf("must be >= %v", v.min),
			Value:   value,
		}
	}

	return nil
}

// MaxValidator validates maximum values for numbers and maximum length for strings/arrays.
type MaxValidator struct {
	max      float64
	maxInt   int
	isLength bool
}

// NewMaxValidator creates a validator for maximum numeric values.
func NewMaxValidator(max float64) *MaxValidator {
	return &MaxValidator{max: max}
}

// NewMaxLengthValidator creates a validator for maximum length.
func NewMaxLengthValidator(max int) *MaxValidator {
	return &MaxValidator{maxInt: max, isLength: true}
}

// Validate checks if the value meets the maximum requirement.
func (v *MaxValidator) Validate(value interface{}) error {
	if v.isLength {
		return v.validateLength(value)
	}
	return v.validateNumeric(value)
}

func (v *MaxValidator) validateLength(value interface{}) error {
	length := getLength(value)
	if length < 0 {
		return &queryfy.FieldError{
			Message: fmt.Sprintf("cannot determine length of %T", value),
			Value:   value,
		}
	}

	if length > v.maxInt {
		return &queryfy.FieldError{
			Message: fmt.Sprintf("length must be at most %d, got %d", v.maxInt, length),
			Value:   value,
		}
	}

	return nil
}

func (v *MaxValidator) validateNumeric(value interface{}) error {
	num, ok := toNumber(value)
	if !ok {
		return &queryfy.FieldError{
			Message: fmt.Sprintf("cannot convert %T to number", value),
			Value:   value,
		}
	}

	if num > v.max {
		return &queryfy.FieldError{
			Message: fmt.Sprintf("must be <= %v", v.max),
			Value:   value,
		}
	}

	return nil
}

// EnumValidator validates that a value is one of the allowed values.
type EnumValidator struct {
	allowed []interface{}
}

// NewEnumValidator creates a validator for enumerated values.
func NewEnumValidator(allowed ...interface{}) *EnumValidator {
	return &EnumValidator{allowed: allowed}
}

// Validate checks if the value is in the allowed list.
func (v *EnumValidator) Validate(value interface{}) error {
	for _, allowed := range v.allowed {
		if reflect.DeepEqual(value, allowed) {
			return nil
		}
	}

	// Create string representation of allowed values
	allowedStrs := make([]string, len(v.allowed))
	for i, a := range v.allowed {
		allowedStrs[i] = fmt.Sprintf("%v", a)
	}

	return &queryfy.FieldError{
		Message: fmt.Sprintf("must be one of: %s", strings.Join(allowedStrs, ", ")),
		Value:   value,
	}
}

// Helper functions

func getLength(value interface{}) int {
	if value == nil {
		return 0
	}

	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.String, reflect.Slice, reflect.Array, reflect.Map:
		return rv.Len()
	default:
		return -1
	}
}

func toNumber(value interface{}) (float64, bool) {
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint()), true
	case reflect.Float32, reflect.Float64:
		return rv.Float(), true
	default:
		return 0, false
	}
}
