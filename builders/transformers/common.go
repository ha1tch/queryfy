// common.go - Common transformation functions
package transformers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ha1tch/queryfy/builders"
)

// ToString converts value to string.
func ToString() builders.Transformer {
	return func(value interface{}) (interface{}, error) {
		switch v := value.(type) {
		case string:
			return v, nil
		case fmt.Stringer:
			return v.String(), nil
		case bool:
			return strconv.FormatBool(v), nil
		case int:
			return strconv.Itoa(v), nil
		case int64:
			return strconv.FormatInt(v, 10), nil
		case float64:
			return strconv.FormatFloat(v, 'f', -1, 64), nil
		case float32:
			return strconv.FormatFloat(float64(v), 'f', -1, 32), nil
		default:
			return fmt.Sprintf("%v", value), nil
		}
	}
}

// ToBoolean converts value to boolean.
func ToBoolean() builders.Transformer {
	return func(value interface{}) (interface{}, error) {
		switch v := value.(type) {
		case bool:
			return v, nil
		case string:
			switch strings.ToLower(strings.TrimSpace(v)) {
			case "true", "yes", "1", "on":
				return true, nil
			case "false", "no", "0", "off":
				return false, nil
			default:
				return nil, fmt.Errorf("cannot convert string %q to boolean", v)
			}
		case int:
			return v != 0, nil
		case float64:
			return v != 0, nil
		default:
			return nil, fmt.Errorf("cannot convert %T to boolean", value)
		}
	}
}

// Default returns a default value if the input is nil or empty.
func Default(defaultValue interface{}) builders.Transformer {
	return func(value interface{}) (interface{}, error) {
		if value == nil {
			return defaultValue, nil
		}

		// Check for empty string
		if str, ok := value.(string); ok && str == "" {
			return defaultValue, nil
		}

		return value, nil
	}
}

// RemoveCurrencySymbols removes common currency symbols.
func RemoveCurrencySymbols() builders.Transformer {
	symbols := []string{"$", "€", "£", "¥", "₹", "₽", "¢", "₩", "₪", "₦", "₨", "₱", "₡", "₵", "₴", "₸"}
	return func(value interface{}) (interface{}, error) {
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", value)
		}

		result := str
		for _, symbol := range symbols {
			result = strings.ReplaceAll(result, symbol, "")
		}

		// Also remove commas used as thousands separators
		result = strings.ReplaceAll(result, ",", "")

		return strings.TrimSpace(result), nil
	}
}

// Chain allows chaining multiple transformers.
func Chain(transformers ...builders.Transformer) builders.Transformer {
	return func(value interface{}) (interface{}, error) {
		result := value
		for i, transformer := range transformers {
			var err error
			result, err = transformer(result)
			if err != nil {
				return nil, fmt.Errorf("transformation %d failed: %w", i+1, err)
			}
		}
		return result, nil
	}
}

// Conditional applies a transformer only if a condition is met.
func Conditional(condition func(interface{}) bool, transformer builders.Transformer) builders.Transformer {
	return func(value interface{}) (interface{}, error) {
		if condition(value) {
			return transformer(value)
		}
		return value, nil
	}
}
