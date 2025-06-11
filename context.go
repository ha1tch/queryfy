package queryfy

import (
	"fmt"
	"strings"
)

// ValidationContext maintains state during validation.
// It tracks the current path and accumulates errors.
type ValidationContext struct {
	path   []string
	errors []FieldError
	mode   ValidationMode
}

// NewValidationContext creates a new validation context.
func NewValidationContext(mode ValidationMode) *ValidationContext {
	return &ValidationContext{
		path:   make([]string, 0, 8), // Pre-allocate for typical nesting depth
		errors: make([]FieldError, 0),
		mode:   mode,
	}
}

// Mode returns the validation mode.
func (c *ValidationContext) Mode() ValidationMode {
	return c.mode
}

// PushPath adds a path segment to the current path.
func (c *ValidationContext) PushPath(segment string) {
	c.path = append(c.path, segment)
}

// PushIndex adds an array index to the current path.
func (c *ValidationContext) PushIndex(index int) {
	c.path = append(c.path, fmt.Sprintf("[%d]", index))
}

// PopPath removes the last path segment.
func (c *ValidationContext) PopPath() {
	if len(c.path) > 0 {
		c.path = c.path[:len(c.path)-1]
	}
}

// CurrentPath returns the current field path as a string.
func (c *ValidationContext) CurrentPath() string {
	if len(c.path) == 0 {
		return ""
	}

	var result strings.Builder
	for i, segment := range c.path {
		if i > 0 && !strings.HasPrefix(segment, "[") {
			result.WriteString(".")
		}
		result.WriteString(segment)
	}
	return result.String()
}

// AddError adds an error at the current path.
func (c *ValidationContext) AddError(message string, value interface{}) {
	c.errors = append(c.errors, FieldError{
		Path:    c.CurrentPath(),
		Message: message,
		Value:   value,
	})
}

// AddFieldError adds a pre-constructed field error.
func (c *ValidationContext) AddFieldError(err FieldError) {
	if err.Path == "" {
		err.Path = c.CurrentPath()
	}
	c.errors = append(c.errors, err)
}

// HasErrors returns true if any errors have been added.
func (c *ValidationContext) HasErrors() bool {
	return len(c.errors) > 0
}

// Error returns a ValidationError if there are any errors, nil otherwise.
func (c *ValidationContext) Error() error {
	if !c.HasErrors() {
		return nil
	}
	return &ValidationError{Errors: c.errors}
}

// Errors returns all accumulated errors.
func (c *ValidationContext) Errors() []FieldError {
	return c.errors
}

// WithPath executes a function with a path segment pushed onto the context.
// The path is automatically popped when the function returns.
func (c *ValidationContext) WithPath(segment string, fn func()) {
	c.PushPath(segment)
	defer c.PopPath()
	fn()
}

// WithIndex executes a function with an array index pushed onto the context.
// The index is automatically popped when the function returns.
func (c *ValidationContext) WithIndex(index int, fn func()) {
	c.PushIndex(index)
	defer c.PopPath()
	fn()
}
