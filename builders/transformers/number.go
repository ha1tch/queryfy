// number.go - Number transformation functions
package transformers

import (
	"fmt"
	"math"
	"strconv"

	"github.com/ha1tch/queryfy/builders"
)

// ToFloat64 converts value to float64.
func ToFloat64() builders.Transformer {
	return func(value interface{}) (interface{}, error) {
		switch v := value.(type) {
		case float64:
			return v, nil
		case float32:
			return float64(v), nil
		case int:
			return float64(v), nil
		case int64:
			return float64(v), nil
		case int32:
			return float64(v), nil
		case int16:
			return float64(v), nil
		case int8:
			return float64(v), nil
		case uint:
			return float64(v), nil
		case uint64:
			return float64(v), nil
		case uint32:
			return float64(v), nil
		case uint16:
			return float64(v), nil
		case uint8:
			return float64(v), nil
		case string:
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, fmt.Errorf("cannot convert string %q to float64: %w", v, err)
			}
			return f, nil
		default:
			return nil, fmt.Errorf("cannot convert %T to float64", value)
		}
	}
}

// ToInt converts value to int.
func ToInt() builders.Transformer {
	return func(value interface{}) (interface{}, error) {
		switch v := value.(type) {
		case int:
			return v, nil
		case float64:
			return int(v), nil
		case float32:
			return int(v), nil
		case int64:
			return int(v), nil
		case int32:
			return int(v), nil
		case string:
			i, err := strconv.Atoi(v)
			if err != nil {
				// Try parsing as float first
				f, err2 := strconv.ParseFloat(v, 64)
				if err2 != nil {
					return nil, fmt.Errorf("cannot convert string %q to int: %w", v, err)
				}
				return int(f), nil
			}
			return i, nil
		default:
			return nil, fmt.Errorf("cannot convert %T to int", value)
		}
	}
}

// Round rounds a number to specified decimal places.
func Round(decimals int) builders.Transformer {
	multiplier := math.Pow(10, float64(decimals))
	return func(value interface{}) (interface{}, error) {
		f, err := ToFloat64()(value)
		if err != nil {
			return nil, err
		}
		num := f.(float64)
		return math.Round(num*multiplier) / multiplier, nil
	}
}

// Clamp restricts a number to a range.
func Clamp(min, max float64) builders.Transformer {
	return func(value interface{}) (interface{}, error) {
		f, err := ToFloat64()(value)
		if err != nil {
			return nil, err
		}
		num := f.(float64)
		if num < min {
			return min, nil
		}
		if num > max {
			return max, nil
		}
		return num, nil
	}
}

// Percentage converts a decimal to percentage (0.15 -> 15).
func Percentage() builders.Transformer {
	return func(value interface{}) (interface{}, error) {
		f, err := ToFloat64()(value)
		if err != nil {
			return nil, err
		}
		return f.(float64) * 100, nil
	}
}

// FromPercentage converts percentage to decimal (15 -> 0.15).
func FromPercentage() builders.Transformer {
	return func(value interface{}) (interface{}, error) {
		f, err := ToFloat64()(value)
		if err != nil {
			return nil, err
		}
		return f.(float64) / 100, nil
	}
}
