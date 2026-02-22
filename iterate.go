package queryfy

import (
	"fmt"
	"reflect"

	"github.com/ha1tch/queryfy/query"
)

// Each executes a callback for each element matched by a query path.
// The path may include wildcards (e.g., "items[*]"). If no wildcard is
// present, the callback is called once with the single matched value.
//
// The callback receives the index and value of each element. Return a
// non-nil error from the callback to stop iteration early.
func Each(data interface{}, queryStr string, fn func(index int, value interface{}) error) error {
	result, err := query.Execute(data, queryStr)
	if err != nil {
		return fmt.Errorf("query %q: %w", queryStr, err)
	}

	items, ok := toSlice(result)
	if !ok {
		// Single value — call once with index 0
		return fn(0, result)
	}

	for i, item := range items {
		if err := fn(i, item); err != nil {
			return err
		}
	}
	return nil
}

// Collect executes a query path and applies a transform function to each
// matched element, returning the collected results. The path may include
// wildcards.
func Collect(data interface{}, queryStr string, fn func(value interface{}) (interface{}, error)) ([]interface{}, error) {
	result, err := query.Execute(data, queryStr)
	if err != nil {
		return nil, fmt.Errorf("query %q: %w", queryStr, err)
	}

	items, ok := toSlice(result)
	if !ok {
		// Single value
		transformed, err := fn(result)
		if err != nil {
			return nil, err
		}
		return []interface{}{transformed}, nil
	}

	results := make([]interface{}, 0, len(items))
	for _, item := range items {
		transformed, err := fn(item)
		if err != nil {
			return nil, err
		}
		results = append(results, transformed)
	}
	return results, nil
}

// ValidateEach executes a query path, then validates each matched element
// against the given schema. Returns a ValidationError with paths that include
// the element index (e.g., "[0]: field is required").
//
// The mode parameter controls strict/loose validation.
func ValidateEach(data interface{}, queryStr string, schema Schema, mode ValidationMode) error {
	result, err := query.Execute(data, queryStr)
	if err != nil {
		return fmt.Errorf("query %q: %w", queryStr, err)
	}

	items, ok := toSlice(result)
	if !ok {
		// Single value — validate once
		return ValidateWithMode(result, schema, mode)
	}

	ve := &ValidationError{}
	for i, item := range items {
		ctx := NewValidationContext(mode)
		schema.Validate(item, ctx)
		if ctx.HasErrors() {
			prefix := fmt.Sprintf("[%d]", i)
			for _, fieldErr := range ctx.Errors() {
				fieldErr.Path = joinPath(prefix, fieldErr.Path)
				ve.AddError(fieldErr)
			}
		}
	}

	if ve.HasErrors() {
		return ve
	}
	return nil
}

// toSlice attempts to convert a value to []interface{}.
// Returns false if the value is not a slice/array.
func toSlice(value interface{}) ([]interface{}, bool) {
	if value == nil {
		return nil, false
	}

	// Direct type assertion — most common case
	if s, ok := value.([]interface{}); ok {
		return s, true
	}

	// Reflection fallback for typed slices
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		result := make([]interface{}, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			result[i] = rv.Index(i).Interface()
		}
		return result, true
	}

	return nil, false
}
