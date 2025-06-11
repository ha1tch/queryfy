package queryfy

import (
	"fmt"
	"strings"
)

// ValidationError represents one or more validation failures.
// It contains a slice of FieldError that provides detailed information
// about each validation failure.
type ValidationError struct {
	Errors []FieldError
}

// FieldError represents a validation error for a specific field.
type FieldError struct {
	// Path is the field path where the error occurred (e.g., "user.email" or "items[0].price")
	Path string
	// Message describes what validation failed
	Message string
	// Value is the actual value that failed validation (optional)
	Value interface{}
}

// Error returns a string representation of all validation errors.
func (e *ValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "validation failed"
	}

	if len(e.Errors) == 1 {
		return fmt.Sprintf("validation failed: %s", e.Errors[0])
	}

	var b strings.Builder
	b.WriteString("validation failed:\n")
	for _, err := range e.Errors {
		b.WriteString("  ")
		b.WriteString(err.String())
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

// Add adds a field error to the validation error.
func (e *ValidationError) Add(path, message string, value interface{}) {
	e.Errors = append(e.Errors, FieldError{
		Path:    path,
		Message: message,
		Value:   value,
	})
}

// AddError adds an existing FieldError to the validation error.
func (e *ValidationError) AddError(err FieldError) {
	e.Errors = append(e.Errors, err)
}

// HasErrors returns true if there are any validation errors.
func (e *ValidationError) HasErrors() bool {
	return len(e.Errors) > 0
}

// String returns a string representation of the field error.
func (e FieldError) String() string {
	if e.Path == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

// Error implements the error interface for FieldError.
func (e FieldError) Error() string {
	return e.String()
}

// NewValidationError creates a new ValidationError with the given field errors.
func NewValidationError(errors ...FieldError) *ValidationError {
	return &ValidationError{Errors: errors}
}

// NewFieldError creates a new FieldError.
func NewFieldError(path, message string, value interface{}) FieldError {
	return FieldError{
		Path:    path,
		Message: message,
		Value:   value,
	}
}

// WrapError wraps an error with field path information.
// If the error is already a ValidationError, it prepends the path to all field errors.
// Otherwise, it creates a new ValidationError with a single field error.
func WrapError(err error, path string) error {
	if err == nil {
		return nil
	}

	if validationErr, ok := err.(*ValidationError); ok {
		wrapped := &ValidationError{}
		for _, fieldErr := range validationErr.Errors {
			fieldErr.Path = joinPath(path, fieldErr.Path)
			wrapped.AddError(fieldErr)
		}
		return wrapped
	}

	return NewValidationError(NewFieldError(path, err.Error(), nil))
}

// joinPath joins two path segments with appropriate separators.
func joinPath(base, sub string) string {
	if base == "" {
		return sub
	}
	if sub == "" {
		return base
	}

	// Handle array index notation
	if strings.HasPrefix(sub, "[") {
		return base + sub
	}

	return base + "." + sub
}
