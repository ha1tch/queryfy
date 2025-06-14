// date.go - Date/time transformation functions
package transformers

import (
	"fmt"
	"time"

	"github.com/ha1tch/queryfy/builders"
)

// ParseDate parses a date string with the given format.
func ParseDate(format string) builders.Transformer {
	return func(value interface{}) (interface{}, error) {
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", value)
		}

		t, err := time.Parse(format, str)
		if err != nil {
			return nil, fmt.Errorf("cannot parse date: %w", err)
		}

		return t, nil
	}
}

// ToISO8601 converts a time to ISO8601 string format.
func ToISO8601() builders.Transformer {
	return func(value interface{}) (interface{}, error) {
		switch v := value.(type) {
		case time.Time:
			return v.Format(time.RFC3339), nil
		case string:
			// Try to parse common formats and convert to ISO8601
			formats := []string{
				"2006-01-02",
				"01/02/2006",
				"02/01/2006",
				"2006-01-02 15:04:05",
				"Jan 2, 2006",
			}

			for _, format := range formats {
				if t, err := time.Parse(format, v); err == nil {
					return t.Format(time.RFC3339), nil
				}
			}

			return nil, fmt.Errorf("cannot parse date string: %s", v)
		default:
			return nil, fmt.Errorf("expected time.Time or string, got %T", value)
		}
	}
}

// DateFormat converts between date formats.
func DateFormat(fromFormat, toFormat string) builders.Transformer {
	return func(value interface{}) (interface{}, error) {
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", value)
		}

		t, err := time.Parse(fromFormat, str)
		if err != nil {
			return nil, fmt.Errorf("cannot parse date with format %s: %w", fromFormat, err)
		}

		return t.Format(toFormat), nil
	}
}

// ToTimezone converts time to specified timezone.
func ToTimezone(location string) builders.Transformer {
	return func(value interface{}) (interface{}, error) {
		var t time.Time

		switch v := value.(type) {
		case time.Time:
			t = v
		case string:
			parsed, err := time.Parse(time.RFC3339, v)
			if err != nil {
				return nil, fmt.Errorf("cannot parse time string: %w", err)
			}
			t = parsed
		default:
			return nil, fmt.Errorf("expected time.Time or string, got %T", value)
		}

		loc, err := time.LoadLocation(location)
		if err != nil {
			return nil, fmt.Errorf("invalid timezone: %w", err)
		}

		return t.In(loc), nil
	}
}

// StartOfDay sets time to 00:00:00.
func StartOfDay() builders.Transformer {
	return func(value interface{}) (interface{}, error) {
		var t time.Time

		switch v := value.(type) {
		case time.Time:
			t = v
		case string:
			parsed, err := time.Parse("2006-01-02", v[:10])
			if err != nil {
				return nil, fmt.Errorf("cannot parse date: %w", err)
			}
			t = parsed
		default:
			return nil, fmt.Errorf("expected time.Time or string, got %T", value)
		}

		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()), nil
	}
}

// EndOfDay sets time to 23:59:59.
func EndOfDay() builders.Transformer {
	return func(value interface{}) (interface{}, error) {
		var t time.Time

		switch v := value.(type) {
		case time.Time:
			t = v
		case string:
			parsed, err := time.Parse("2006-01-02", v[:10])
			if err != nil {
				return nil, fmt.Errorf("cannot parse date: %w", err)
			}
			t = parsed
		default:
			return nil, fmt.Errorf("expected time.Time or string, got %T", value)
		}

		return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location()), nil
	}
}
