package query

import (
	"fmt"
	"reflect"
)

// Execute executes a query against data and returns the result.
func Execute(data interface{}, queryStr string) (interface{}, error) {
	if queryStr == "" {
		return data, nil
	}

	// Parse the query
	path, err := PathFromQuery(queryStr)
	if err != nil {
		return nil, fmt.Errorf("invalid query: %w", err)
	}

	// Execute the path
	return ExecutePath(data, path)
}

// ExecutePath executes a path against data.
func ExecutePath(data interface{}, path []interface{}) (interface{}, error) {
	current := data

	for i, segment := range path {
		if current == nil {
			return nil, fmt.Errorf("cannot access %v on nil value", segment)
		}

		switch seg := segment.(type) {
		case string:
			// Field access
			next, err := getField(current, seg)
			if err != nil {
				return nil, fmt.Errorf("at %s: %w", formatPath(path[:i+1]), err)
			}
			current = next

		case int:
			// Array index access
			next, err := getIndex(current, seg)
			if err != nil {
				return nil, fmt.Errorf("at %s: %w", formatPath(path[:i+1]), err)
			}
			current = next

		default:
			return nil, fmt.Errorf("unexpected path segment type: %T", segment)
		}
	}

	return current, nil
}

// getField gets a field from an object.
func getField(obj interface{}, field string) (interface{}, error) {
	// Handle map[string]interface{} directly (most common case)
	if m, ok := obj.(map[string]interface{}); ok {
		value, exists := m[field]
		if !exists {
			return nil, fmt.Errorf("field %q not found", field)
		}
		return value, nil
	}

	// Use reflection for other types
	rv := reflect.ValueOf(obj)

	// Dereference pointers
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil, fmt.Errorf("cannot access field %q on nil pointer", field)
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Map:
		// Handle other map types
		if rv.Type().Key().Kind() != reflect.String {
			return nil, fmt.Errorf("cannot access field %q on non-string keyed map", field)
		}

		key := reflect.ValueOf(field)
		value := rv.MapIndex(key)
		if !value.IsValid() {
			return nil, fmt.Errorf("field %q not found", field)
		}
		return value.Interface(), nil

	case reflect.Struct:
		// Handle struct field access
		fieldValue := rv.FieldByName(field)
		if !fieldValue.IsValid() {
			return nil, fmt.Errorf("field %q not found in struct", field)
		}
		return fieldValue.Interface(), nil

	default:
		return nil, fmt.Errorf("cannot access field %q on %v", field, rv.Kind())
	}
}

// getIndex gets an element from an array/slice.
func getIndex(arr interface{}, index int) (interface{}, error) {
	// Handle []interface{} directly (most common case)
	if a, ok := arr.([]interface{}); ok {
		if index < 0 || index >= len(a) {
			return nil, fmt.Errorf("index %d out of bounds (length %d)", index, len(a))
		}
		return a[index], nil
	}

	// Use reflection for other types
	rv := reflect.ValueOf(arr)

	// Dereference pointers
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil, fmt.Errorf("cannot index nil pointer")
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		if index < 0 || index >= rv.Len() {
			return nil, fmt.Errorf("index %d out of bounds (length %d)", index, rv.Len())
		}
		return rv.Index(index).Interface(), nil

	default:
		return nil, fmt.Errorf("cannot index %v", rv.Kind())
	}
}

// formatPath formats a path for error messages.
func formatPath(path []interface{}) string {
	if len(path) == 0 {
		return "<root>"
	}

	result := ""
	for _, segment := range path {
		switch seg := segment.(type) {
		case string:
			if result == "" {
				result = seg
			} else {
				result += "." + seg
			}
		case int:
			result += fmt.Sprintf("[%d]", seg)
		}
	}

	return result
}
