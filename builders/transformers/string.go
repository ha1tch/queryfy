// string.go - String transformation functions
package transformers

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/ha1tch/queryfy/builders"
)

// Trim removes leading and trailing whitespace.
func Trim() builders.Transformer {
	return func(value interface{}) (interface{}, error) {
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", value)
		}
		return strings.TrimSpace(str), nil
	}
}

// Lowercase converts string to lowercase.
func Lowercase() builders.Transformer {
	return func(value interface{}) (interface{}, error) {
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", value)
		}
		return strings.ToLower(str), nil
	}
}

// Uppercase converts string to uppercase.
func Uppercase() builders.Transformer {
	return func(value interface{}) (interface{}, error) {
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", value)
		}
		return strings.ToUpper(str), nil
	}
}

// TitleCase converts string to title case.
func TitleCase() builders.Transformer {
	return func(value interface{}) (interface{}, error) {
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", value)
		}
		return strings.Title(strings.ToLower(str)), nil
	}
}

// RemoveSpaces removes all spaces from string.
func RemoveSpaces() builders.Transformer {
	return func(value interface{}) (interface{}, error) {
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", value)
		}
		return strings.ReplaceAll(str, " ", ""), nil
	}
}

// NormalizeWhitespace converts multiple whitespace to single space.
func NormalizeWhitespace() builders.Transformer {
	whitespaceRegex := regexp.MustCompile(`\s+`)
	return func(value interface{}) (interface{}, error) {
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", value)
		}
		// Trim and normalize internal whitespace
		normalized := whitespaceRegex.ReplaceAllString(str, " ")
		return strings.TrimSpace(normalized), nil
	}
}

// Replace replaces all occurrences of old with new.
func Replace(old, new string) builders.Transformer {
	return func(value interface{}) (interface{}, error) {
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", value)
		}
		return strings.ReplaceAll(str, old, new), nil
	}
}

// RemoveNonAlphanumeric removes all non-alphanumeric characters.
func RemoveNonAlphanumeric() builders.Transformer {
	return func(value interface{}) (interface{}, error) {
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", value)
		}
		return strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				return r
			}
			return -1
		}, str), nil
	}
}

// Truncate truncates string to max length.
func Truncate(maxLength int) builders.Transformer {
	return func(value interface{}) (interface{}, error) {
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", value)
		}
		if len(str) <= maxLength {
			return str, nil
		}
		return str[:maxLength], nil
	}
}

// PadLeft pads string on the left to reach minLength.
func PadLeft(minLength int, padChar rune) builders.Transformer {
	return func(value interface{}) (interface{}, error) {
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", value)
		}
		if len(str) >= minLength {
			return str, nil
		}
		padding := strings.Repeat(string(padChar), minLength-len(str))
		return padding + str, nil
	}
}
