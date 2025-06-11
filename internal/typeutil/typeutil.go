// Package typeutil provides type checking and conversion utilities.
package typeutil

import (
	"fmt"
	"reflect"
	"strconv"
)

// IsString checks if a value is a string.
func IsString(v interface{}) bool {
	_, ok := v.(string)
	return ok
}

// IsNumber checks if a value is a numeric type.
func IsNumber(v interface{}) bool {
	switch v.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return true
	default:
		return false
	}
}

// IsBool checks if a value is a boolean.
func IsBool(v interface{}) bool {
	_, ok := v.(bool)
	return ok
}

// IsMap checks if a value is a map with string keys.
func IsMap(v interface{}) bool {
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Map && rv.Type().Key().Kind() == reflect.String
}

// IsSlice checks if a value is a slice or array.
func IsSlice(v interface{}) bool {
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array
}

// ToString converts a value to string if possible.
func ToString(v interface{}) (string, bool) {
	switch v := v.(type) {
	case string:
		return v, true
	case fmt.Stringer:
		return v.String(), true
	case bool:
		return strconv.FormatBool(v), true
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v), true
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v), true
	case float32, float64:
		return fmt.Sprintf("%g", v), true
	default:
		return "", false
	}
}

// ToFloat64 converts a value to float64 if possible.
func ToFloat64(v interface{}) (float64, bool) {
	switch v := v.(type) {
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
	case string:
		f, err := strconv.ParseFloat(v, 64)
		return f, err == nil
	default:
		return 0, false
	}
}
