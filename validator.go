package queryfy

import (
	"fmt"
	"reflect"
)

// Validator wraps a schema with configuration options.
type Validator struct {
	schema Schema
	mode   ValidationMode
}

// Strict sets the validator to strict mode.
func (v *Validator) Strict() *Validator {
	v.mode = Strict
	return v
}

// Loose sets the validator to loose mode.
func (v *Validator) Loose() *Validator {
	v.mode = Loose
	return v
}

// Validate validates data against the schema.
func (v *Validator) Validate(data interface{}) error {
	return ValidateWithMode(data, v.schema, v.mode)
}

// ValidateValue is a helper function that validates a single value against its expected type.
// It handles type checking and conversion based on the validation mode.
func ValidateValue(value interface{}, expectedType SchemaType, ctx *ValidationContext) bool {
	if value == nil {
		return true // Nil handling should be done by CheckRequired
	}

	switch expectedType {
	case TypeString:
		return validateString(value, ctx)
	case TypeNumber:
		return validateNumber(value, ctx)
	case TypeBool:
		return validateBool(value, ctx)
	case TypeObject:
		return validateObject(value, ctx)
	case TypeArray:
		return validateArray(value, ctx)
	case TypeAny:
		return true
	default:
		ctx.AddError(fmt.Sprintf("unknown schema type: %s", expectedType), value)
		return false
	}
}

func validateString(value interface{}, ctx *ValidationContext) bool {
	switch v := value.(type) {
	case string:
		return true
	default:
		if ctx.Mode() == Loose {
			// In loose mode, try to convert to string
			if _, ok := ConvertToString(v); ok {
				return true
			}
		}
		ctx.AddError(fmt.Sprintf("expected string, got %T", value), value)
		return false
	}
}

func validateNumber(value interface{}, ctx *ValidationContext) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return true
	default:
		if ctx.Mode() == Loose {
			// In loose mode, try to convert string to number
			if str, ok := value.(string); ok {
				if _, ok := ConvertStringToNumber(str); ok {
					return true
				}
			}
		}
		ctx.AddError(fmt.Sprintf("expected number, got %T", value), value)
		return false
	}
}

func validateBool(value interface{}, ctx *ValidationContext) bool {
	switch value.(type) {
	case bool:
		return true
	default:
		if ctx.Mode() == Loose {
			// In loose mode, accept "true" and "false" strings
			if str, ok := value.(string); ok {
				if str == "true" || str == "false" {
					return true
				}
			}
		}
		ctx.AddError(fmt.Sprintf("expected boolean, got %T", value), value)
		return false
	}
}

func validateObject(value interface{}, ctx *ValidationContext) bool {
	// Use reflection to check if it's a map
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Map {
		// Check if it's a string-keyed map
		if rv.Type().Key().Kind() == reflect.String {
			return true
		}
		ctx.AddError("expected object with string keys", value)
		return false
	}

	ctx.AddError(fmt.Sprintf("expected object, got %T", value), value)
	return false
}

func validateArray(value interface{}, ctx *ValidationContext) bool {
	// Use reflection to check if it's a slice or array
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		return true
	default:
		ctx.AddError(fmt.Sprintf("expected array, got %T", value), value)
		return false
	}
}

// ConvertToString attempts to convert a value to string.
// Returns the string and true if successful, empty string and false otherwise.
func ConvertToString(value interface{}) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case fmt.Stringer:
		return v.String(), true
	default:
		// For simple types, use fmt.Sprint
		switch v := value.(type) {
		case bool, int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64,
			float32, float64:
			return fmt.Sprint(v), true
		}
	}
	return "", false
}

// ConvertStringToNumber attempts to convert a string to a number.
// Returns the number and true if successful, 0 and false otherwise.
func ConvertStringToNumber(str string) (float64, bool) {
	var f float64
	_, err := fmt.Sscanf(str, "%f", &f)
	return f, err == nil
}
