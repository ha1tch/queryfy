// datetime.go - Date/Time validation builder for Queryfy
package builders

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ha1tch/queryfy"
)

// DateTimeSchema validates date/time values.
type DateTimeSchema struct {
	queryfy.BaseSchema
	format     string // The expected format (e.g., time.RFC3339, "2006-01-02")
	minTime    *time.Time
	maxTime    *time.Time
	validators []queryfy.ValidatorFunc
}

// DateTime creates a new date/time schema builder.
func DateTime() *DateTimeSchema {
	return &DateTimeSchema{
		BaseSchema: queryfy.BaseSchema{
			SchemaType: queryfy.TypeDateTime,
		},
		format: time.RFC3339, // Default to RFC3339
	}
}

// Required marks the field as required.
func (s *DateTimeSchema) Required() *DateTimeSchema {
	s.SetRequired(true)
	return s
}

// Optional marks the field as optional (default).
func (s *DateTimeSchema) Optional() *DateTimeSchema {
	s.SetRequired(false)
	return s
}

// Nullable allows the field to be null.
func (s *DateTimeSchema) Nullable() *DateTimeSchema {
	s.SetNullable(true)
	return s
}

// Format sets the expected date/time format.
// Common formats:
//   - time.RFC3339: "2006-01-02T15:04:05Z07:00"
//   - time.DateOnly: "2006-01-02"
//   - "2006-01-02 15:04:05"
//   - "01/02/2006"
//   - "02-Jan-2006"
func (s *DateTimeSchema) Format(format string) *DateTimeSchema {
	s.format = format
	return s
}

// ISO8601 sets the format to ISO8601 (RFC3339).
func (s *DateTimeSchema) ISO8601() *DateTimeSchema {
	s.format = time.RFC3339
	return s
}

// DateOnly sets the format to date only (YYYY-MM-DD).
func (s *DateTimeSchema) DateOnly() *DateTimeSchema {
	s.format = "2006-01-02"
	return s
}

// YMD sets the format to YYYY-MM-DD (ISO format).
func (s *DateTimeSchema) YMD() *DateTimeSchema {
	s.format = "2006-01-02"
	return s
}

// DMY sets the format to DD/MM/YYYY (used in UK, Europe, Latin America, and most of the world).
func (s *DateTimeSchema) DMY() *DateTimeSchema {
	s.format = "02/01/2006"
	return s
}

// MDY sets the format to MM/DD/YYYY (used primarily in the US).
func (s *DateTimeSchema) MDY() *DateTimeSchema {
	s.format = "01/02/2006"
	return s
}

// Min sets the minimum allowed date/time.
func (s *DateTimeSchema) Min(min time.Time) *DateTimeSchema {
	s.minTime = &min
	return s
}

// MinString sets the minimum allowed date/time from a string.
func (s *DateTimeSchema) MinString(min string) *DateTimeSchema {
	if t, err := time.Parse(s.format, min); err == nil {
		s.minTime = &t
	} else {
		// Store error to be reported during validation
		s.validators = append(s.validators, func(value interface{}) error {
			return fmt.Errorf("invalid min time format: %s", err.Error())
		})
	}
	return s
}

// Max sets the maximum allowed date/time.
func (s *DateTimeSchema) Max(max time.Time) *DateTimeSchema {
	s.maxTime = &max
	return s
}

// MaxString sets the maximum allowed date/time from a string.
func (s *DateTimeSchema) MaxString(max string) *DateTimeSchema {
	if t, err := time.Parse(s.format, max); err == nil {
		s.maxTime = &t
	} else {
		// Store error to be reported during validation
		s.validators = append(s.validators, func(value interface{}) error {
			return fmt.Errorf("invalid max time format: %s", err.Error())
		})
	}
	return s
}

// Between sets both minimum and maximum date/time.
func (s *DateTimeSchema) Between(min, max time.Time) *DateTimeSchema {
	s.minTime = &min
	s.maxTime = &max
	return s
}

// BetweenStrings sets both minimum and maximum date/time from strings.
func (s *DateTimeSchema) BetweenStrings(min, max string) *DateTimeSchema {
	s.MinString(min)
	s.MaxString(max)
	return s
}

// Future requires the date/time to be in the future.
func (s *DateTimeSchema) Future() *DateTimeSchema {
	s.validators = append(s.validators, func(value interface{}) error {
		t, ok := value.(time.Time)
		if !ok {
			// This will be caught by type validation
			return nil
		}
		if !t.After(time.Now()) {
			return fmt.Errorf("must be in the future")
		}
		return nil
	})
	return s
}

// Past requires the date/time to be in the past.
func (s *DateTimeSchema) Past() *DateTimeSchema {
	s.validators = append(s.validators, func(value interface{}) error {
		t, ok := value.(time.Time)
		if !ok {
			// This will be caught by type validation
			return nil
		}
		if !t.Before(time.Now()) {
			return fmt.Errorf("must be in the past")
		}
		return nil
	})
	return s
}

// Age validates that the date represents an age within the specified range.
// Useful for birth dates.
func (s *DateTimeSchema) Age(minAge, maxAge int) *DateTimeSchema {
	s.validators = append(s.validators, func(value interface{}) error {
		t, ok := value.(time.Time)
		if !ok {
			return nil
		}

		age := calculateAge(t)
		if age < minAge {
			return fmt.Errorf("age must be at least %d years (current: %d)", minAge, age)
		}
		if age > maxAge {
			return fmt.Errorf("age must be at most %d years (current: %d)", maxAge, age)
		}
		return nil
	})
	return s
}

// Weekday validates that the date falls on specific weekdays.
func (s *DateTimeSchema) Weekday(days ...time.Weekday) *DateTimeSchema {
	s.validators = append(s.validators, func(value interface{}) error {
		t, ok := value.(time.Time)
		if !ok {
			return nil
		}

		currentDay := t.Weekday()
		for _, day := range days {
			if currentDay == day {
				return nil
			}
		}

		return fmt.Errorf("must be on %s", formatWeekdays(days))
	})
	return s
}

// BusinessDay validates that the date is a business day (Mon-Fri).
func (s *DateTimeSchema) BusinessDay() *DateTimeSchema {
	return s.Weekday(time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday)
}

// Custom adds a custom validator function.
func (s *DateTimeSchema) Custom(fn queryfy.ValidatorFunc) *DateTimeSchema {
	s.validators = append(s.validators, fn)
	return s
}

// Validate implements the Schema interface.
func (s *DateTimeSchema) Validate(value interface{}, ctx *queryfy.ValidationContext) error {
	if !s.CheckRequired(value, ctx) {
		return nil
	}

	// Try to get time.Time directly first
	t, isTime := value.(time.Time)

	if !isTime {
		// Try to parse string in loose mode or if it's a string
		str, isString := value.(string)
		if isString || ctx.Mode() == queryfy.Loose {
			if !isString {
				// In loose mode, try to convert to string
				str, isString = queryfy.ConvertToString(value)
				if !isString {
					ctx.AddError(fmt.Sprintf("cannot convert %T to date/time", value), value)
					return nil
				}
			}

			// Parse the string
			parsed, err := time.Parse(s.format, str)
			if err != nil {
				// Try some common formats if the specified format fails
				if parsedAlt, err2 := tryCommonFormats(str); err2 == nil {
					t = parsedAlt
				} else {
					ctx.AddError(fmt.Sprintf("invalid date/time format (expected %s): %s", s.format, err.Error()), str)
					return nil
				}
			} else {
				t = parsed
			}
		} else {
			ctx.AddError(fmt.Sprintf("expected time.Time or string, got %T", value), value)
			return nil
		}
	}

	// Now validate the parsed time

	// Range validation
	if s.minTime != nil && t.Before(*s.minTime) {
		ctx.AddError(fmt.Sprintf("must be after %s", s.minTime.Format(s.format)), t.Format(s.format))
	}

	if s.maxTime != nil && t.After(*s.maxTime) {
		ctx.AddError(fmt.Sprintf("must be before %s", s.maxTime.Format(s.format)), t.Format(s.format))
	}

	// Custom validators - pass the parsed time
	for _, validator := range s.validators {
		if err := validator(t); err != nil {
			ctx.AddError(err.Error(), t.Format(s.format))
		}
	}

	return nil
}

// Type implements the Schema interface.
func (s *DateTimeSchema) Type() queryfy.SchemaType {
	return queryfy.TypeDateTime
}

// Helper functions

// calculateAge calculates age in years from a birth date.
func calculateAge(birthDate time.Time) int {
	now := time.Now()
	years := now.Year() - birthDate.Year()

	// Adjust if birthday hasn't occurred this year
	if now.YearDay() < birthDate.YearDay() {
		years--
	}

	return years
}

// formatWeekdays formats a list of weekdays for error messages.
func formatWeekdays(days []time.Weekday) string {
	if len(days) == 0 {
		return ""
	}

	names := make([]string, len(days))
	for i, day := range days {
		names[i] = day.String()
	}

	if len(names) == 1 {
		return names[0]
	}

	return names[0] + " through " + names[len(names)-1]
}

// tryCommonFormats attempts to parse a date string using common formats.
func tryCommonFormats(str string) (time.Time, error) {
	// First, try the most common worldwide formats (DD/MM/YYYY used by most of the world)
	worldwideFormats := []string{
		time.RFC3339,
		"2006-01-02", // ISO format (unambiguous)
		"2006-01-02 15:04:05",
		"02/01/2006", // DD/MM/YYYY - British/European/Latin American format
		"2/1/2006",   // D/M/YYYY
		"02-01-2006", // DD-MM-YYYY
		"2-1-2006",   // D-M-YYYY
		"02.01.2006", // DD.MM.YYYY (common in Europe)
		"2.1.2006",   // D.M.YYYY
		"2006/01/02", // YYYY/MM/DD
		"02-Jan-2006",
		"2-Jan-2006",
		"Jan 2, 2006",
		"January 2, 2006",
		"2 January 2006",      // Common in UK
		"2nd January 2006",    // With ordinal
		"2006-01-02T15:04:05", // RFC3339 without timezone
		"15:04:05",
		"3:04 PM",
		"2006-01-02 15:04", // Without seconds
		"02/01/06",         // DD/MM/YY
		"2/1/06",           // D/M/YY
	}

	// Try worldwide formats first
	for _, format := range worldwideFormats {
		if t, err := time.Parse(format, str); err == nil {
			return t, nil
		}
	}

	// Special handling for slash-separated dates to detect US format
	if strings.Contains(str, "/") {
		parts := strings.Split(str, "/")
		if len(parts) == 3 {
			first, err1 := strconv.Atoi(parts[0])
			second, err2 := strconv.Atoi(parts[1])

			if err1 == nil && err2 == nil {
				// If day > 12, it must be US format MM/DD/YYYY
				if first <= 12 && second > 12 && second <= 31 {
					// Try US formats
					if t, err := time.Parse("01/02/2006", str); err == nil {
						return t, nil
					}
					if t, err := time.Parse("1/2/2006", str); err == nil {
						return t, nil
					}
				}
				// If month > 12, it must be DD/MM/YYYY (already tried above)
				// If ambiguous (both <= 12), US format is secondary choice
				if first <= 12 && second <= 12 {
					// Already tried DD/MM above, now try MM/DD
					if t, err := time.Parse("01/02/2006", str); err == nil {
						return t, nil
					}
					if t, err := time.Parse("1/2/2006", str); err == nil {
						return t, nil
					}
				}
			}
		}
	}

	// Try remaining formats
	additionalFormats := []string{
		"01-02-2006",             // MM-DD-YYYY (US with dashes)
		"1-2-2006",               // M-D-YYYY
		"Monday, 2 January 2006", // Full format
		"Mon, 2 Jan 2006",        // Abbreviated
		"01/02/06",               // MM/DD/YY
		"1/2/06",                 // M/D/YY
	}

	for _, format := range additionalFormats {
		if t, err := time.Parse(format, str); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date/time from: %s", str)
}

// --- Examples of usage ---
/*
// Basic date validation
dateSchema := builders.DateTime().
    DateOnly().
    Required()

// Birth date validation (age 18-100)
birthDateSchema := builders.DateTime().
    DateOnly().
    Past().
    Age(18, 100).
    Required()

// Appointment scheduling (future business days only)
appointmentSchema := builders.DateTime().
    Format("2006-01-02 15:04").
    Future().
    BusinessDay().
    Between(
        time.Now().Add(24*time.Hour),
        time.Now().Add(30*24*time.Hour),
    ).
    Required()

// Event date validation
eventSchema := builders.Object().
    Field("startDate", builders.DateTime().
        ISO8601().
        Future().
        Required()).
    Field("endDate", builders.DateTime().
        ISO8601().
        Custom(func(value interface{}) error {
            // Custom validation: end must be after start
            // This would need access to the parent object
            return nil
        }).
        Required())

// Flexible date parsing in loose mode
flexibleDateSchema := builders.DateTime().
    Required() // Will try multiple formats automatically
*/
