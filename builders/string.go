package builders

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ha1tch/queryfy"
)

// StringSchema validates string values.
type StringSchema struct {
	queryfy.BaseSchema
	minLength  *int
	maxLength  *int
	pattern    *regexp.Regexp
	patternStr string
	enum       []string
	validators []queryfy.ValidatorFunc
	formatType string // "email", "url", "uuid", or ""
}

// String creates a new string schema builder.
func String() *StringSchema {
	return &StringSchema{
		BaseSchema: queryfy.BaseSchema{
			SchemaType: queryfy.TypeString,
		},
	}
}

// Required marks the field as required.
func (s *StringSchema) Required() *StringSchema {
	s.SetRequired(true)
	return s
}

// Optional marks the field as optional (default).
func (s *StringSchema) Optional() *StringSchema {
	s.SetRequired(false)
	return s
}

// Nullable allows the field to be null.
func (s *StringSchema) Nullable() *StringSchema {
	s.SetNullable(true)
	return s
}

// MinLength sets the minimum string length.
func (s *StringSchema) MinLength(min int) *StringSchema {
	s.minLength = &min
	return s
}

// MaxLength sets the maximum string length.
func (s *StringSchema) MaxLength(max int) *StringSchema {
	s.maxLength = &max
	return s
}

// Length sets both minimum and maximum length to the same value.
func (s *StringSchema) Length(length int) *StringSchema {
	s.minLength = &length
	s.maxLength = &length
	return s
}

// Pattern sets a regular expression pattern that the string must match.
func (s *StringSchema) Pattern(pattern string) *StringSchema {
	re, err := regexp.Compile(pattern)
	if err != nil {
		// Store the error to be reported during validation
		s.validators = append(s.validators, func(value interface{}) error {
			return fmt.Errorf("invalid regex pattern: %s", err.Error())
		})
	} else {
		s.pattern = re
		s.patternStr = pattern
	}
	return s
}

// Enum restricts the string to one of the specified values.
func (s *StringSchema) Enum(values ...string) *StringSchema {
	s.enum = values
	return s
}

// Email validates that the string is a valid email address.
func (s *StringSchema) Email() *StringSchema {
	s.formatType = "email"
	return s.Pattern(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
}

// URL validates that the string is a valid URL.
func (s *StringSchema) URL() *StringSchema {
	s.formatType = "url"
	return s.Pattern(`^https?://[^\s/$.?#].[^\s]*$`)
}

// UUID validates that the string is a valid UUID.
func (s *StringSchema) UUID() *StringSchema {
	s.formatType = "uuid"
	return s.Pattern(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
}

// Custom adds a custom validator function.
func (s *StringSchema) Custom(fn queryfy.ValidatorFunc) *StringSchema {
	s.validators = append(s.validators, fn)
	return s
}

// Validate implements the Schema interface.
func (s *StringSchema) Validate(value interface{}, ctx *queryfy.ValidationContext) error {
	if !s.CheckRequired(value, ctx) {
		return nil
	}

	// In loose mode, try to convert to string first
	var str string
	var ok bool

	if ctx.Mode() == queryfy.Loose {
		str, ok = queryfy.ConvertToString(value)
		if !ok {
			ctx.AddError(fmt.Sprintf("cannot convert %T to string", value), value)
			return nil
		}
	} else {
		// Strict mode - must be a string
		str, ok = value.(string)
		if !ok {
			ctx.AddError(fmt.Sprintf("expected string, got %T", value), value)
			return nil
		}
	}

	// Length validation
	if s.minLength != nil && len(str) < *s.minLength {
		ctx.AddError(fmt.Sprintf("length must be at least %d, got %d", *s.minLength, len(str)), str)
	}

	if s.maxLength != nil && len(str) > *s.maxLength {
		ctx.AddError(fmt.Sprintf("length must be at most %d, got %d", *s.maxLength, len(str)), str)
	}

	// Pattern validation
	if s.pattern != nil && !s.pattern.MatchString(str) {
		var msg string
		switch s.formatType {
		case "email":
			msg = "must be a valid email address"
		case "url":
			msg = "must be a valid URL"
		case "uuid":
			msg = "must be a valid UUID"
		default:
			msg = fmt.Sprintf("must match pattern %s", s.patternStr)
		}
		ctx.AddError(msg, str)
	}

	// Enum validation
	if len(s.enum) > 0 {
		found := false
		for _, allowed := range s.enum {
			if str == allowed {
				found = true
				break
			}
		}
		if !found {
			ctx.AddError(fmt.Sprintf("must be one of: %s", strings.Join(s.enum, ", ")), str)
		}
	}

	// Custom validators
	for _, validator := range s.validators {
		if err := validator(str); err != nil {
			ctx.AddError(err.Error(), str)
		}
	}

	return nil
}

// Type implements the Schema interface.
func (s *StringSchema) Type() queryfy.SchemaType {
	return queryfy.TypeString
}
