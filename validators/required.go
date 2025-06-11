package validators

import (
	"reflect"

	"github.com/ha1tch/queryfy"
)

// RequiredValidator validates that a value is present and not nil.
type RequiredValidator struct {
	allowZero bool
}

// NewRequiredValidator creates a new required validator.
func NewRequiredValidator() *RequiredValidator {
	return &RequiredValidator{}
}

// AllowZero allows zero values to pass validation.
// By default, zero values (empty string, 0, false) are considered invalid.
func (v *RequiredValidator) AllowZero() *RequiredValidator {
	v.allowZero = true
	return v
}

// Validate checks if the value is present and not nil.
func (v *RequiredValidator) Validate(value interface{}) error {
	if value == nil {
		return &queryfy.FieldError{
			Message: "field is required",
		}
	}

	if !v.allowZero {
		// Check for zero values
		rv := reflect.ValueOf(value)
		if rv.IsZero() {
			return &queryfy.FieldError{
				Message: "field cannot be empty",
				Value:   value,
			}
		}
	}

	return nil
}

// IsRequired checks if a value would be considered "present" for required validation.
// This is useful for other validators that need to check if a value exists.
func IsRequired(value interface{}) bool {
	if value == nil {
		return false
	}

	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.String:
		return rv.Len() > 0
	case reflect.Slice, reflect.Array, reflect.Map:
		return rv.Len() > 0
	case reflect.Ptr, reflect.Interface:
		return !rv.IsNil()
	default:
		return !rv.IsZero()
	}
}

// RequiredIf creates a conditional required validator.
// The field is required only if the condition function returns true.
func RequiredIf(condition func(value interface{}) bool) queryfy.ValidatorFunc {
	return func(value interface{}) error {
		if condition(value) {
			validator := NewRequiredValidator()
			return validator.Validate(value)
		}
		return nil
	}
}
