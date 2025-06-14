package main

import (
	"fmt"
	"time"

	qf "github.com/ha1tch/queryfy"
	"github.com/ha1tch/queryfy/builders"
)

func main() {
	fmt.Println("=== Queryfy DateTime Validation Examples ===")

	// Example 1: Basic date validation
	basicDateExample()

	// Example 2: Birth date validation
	birthDateExample()

	// Example 3: Event scheduling validation
	eventSchedulingExample()

	// Example 4: Business hours validation
	businessHoursExample()
}

func basicDateExample() {
	fmt.Println("1. Basic Date Validation")
	fmt.Println("------------------------")

	// Simple date schema
	dateSchema := builders.DateTime().
		DateOnly().
		Required()

	testDates := []struct {
		name  string
		value interface{}
	}{
		{"Valid date string", "2024-06-15"},
		{"Invalid format", "15/06/2024"},
		{"US format", "06/15/2024"},
		{"Not a date", "not-a-date"},
		{"Number", 12345},
		{"Time object", time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)},
	}

	for _, test := range testDates {
		fmt.Printf("Testing %s (%v): ", test.name, test.value)
		if err := qf.Validate(test.value, dateSchema); err != nil {
			fmt.Printf("[X] Failed\n")
		} else {
			fmt.Printf("[✓] Valid\n")
		}
	}

	// Test with loose mode
	fmt.Println("\nLoose mode (tries multiple formats):")
	for _, test := range testDates {
		fmt.Printf("Testing %s (%v): ", test.name, test.value)
		if err := qf.ValidateWithMode(test.value, dateSchema, qf.Loose); err != nil {
			fmt.Printf("[X] Failed\n")
		} else {
			fmt.Printf("[✓] Valid\n")
		}
	}
}

func birthDateExample() {
	fmt.Println("\n\n2. Birth Date Validation")
	fmt.Println("------------------------")

	// Birth date schema with age validation
	birthDateSchema := builders.DateTime().
		DateOnly().
		Past().
		Age(18, 120).
		Required()

	// User registration schema
	userSchema := builders.Object().
		Field("username", builders.String().Required()).
		Field("email", builders.String().Email().Required()).
		Field("birthDate", birthDateSchema)

	testUsers := []map[string]interface{}{
		{
			"username":  "john_doe",
			"email":     "john@example.com",
			"birthDate": "1990-06-15", // Valid age
		},
		{
			"username":  "too_young",
			"email":     "young@example.com",
			"birthDate": "2010-01-01", // Too young
		},
		{
			"username":  "future_birth",
			"email":     "future@example.com",
			"birthDate": "2025-01-01", // In the future
		},
		{
			"username":  "ancient",
			"email":     "old@example.com",
			"birthDate": "1850-01-01", // Too old
		},
	}

	for _, user := range testUsers {
		fmt.Printf("\nValidating user %s:\n", user["username"])
		if err := qf.Validate(user, userSchema); err != nil {
			fmt.Printf("[X] Validation failed: %v\n", err)
		} else {
			fmt.Printf("[✓] User is valid!\n")
		}
	}
}

func eventSchedulingExample() {
	fmt.Println("\n\n3. Event Scheduling Validation")
	fmt.Println("------------------------------")

	// Event schema with custom validation
	eventSchema := builders.Object().
		Field("title", builders.String().MinLength(5).Required()).
		Field("startDate", builders.DateTime().
			ISO8601().
			Future().
			Required()).
		Field("endDate", builders.DateTime().
			ISO8601().
			Future().
			Required()).
		Custom(func(value interface{}) error {
			// Cross-field validation: end must be after start
			event, ok := value.(map[string]interface{})
			if !ok {
				return nil
			}

			startStr, _ := event["startDate"].(string)
			endStr, _ := event["endDate"].(string)

			if startStr != "" && endStr != "" {
				start, err1 := time.Parse(time.RFC3339, startStr)
				end, err2 := time.Parse(time.RFC3339, endStr)

				if err1 == nil && err2 == nil {
					if !end.After(start) {
						return fmt.Errorf("end date must be after start date")
					}
					if end.Sub(start) > 24*30*time.Hour {
						return fmt.Errorf("event cannot span more than 30 days")
					}
				}
			}
			return nil
		})

	// Test events
	now := time.Now()
	tomorrow := now.Add(24 * time.Hour)
	nextWeek := now.Add(7 * 24 * time.Hour)
	nextMonth := now.Add(31 * 24 * time.Hour)

	testEvents := []map[string]interface{}{
		{
			"title":     "Valid Conference",
			"startDate": tomorrow.Format(time.RFC3339),
			"endDate":   tomorrow.Add(8 * time.Hour).Format(time.RFC3339),
		},
		{
			"title":     "Past Event",
			"startDate": now.Add(-24 * time.Hour).Format(time.RFC3339),
			"endDate":   now.Add(-20 * time.Hour).Format(time.RFC3339),
		},
		{
			"title":     "End Before Start",
			"startDate": nextWeek.Format(time.RFC3339),
			"endDate":   tomorrow.Format(time.RFC3339),
		},
		{
			"title":     "Too Long Event",
			"startDate": tomorrow.Format(time.RFC3339),
			"endDate":   nextMonth.Add(7 * 24 * time.Hour).Format(time.RFC3339),
		},
	}

	for _, event := range testEvents {
		fmt.Printf("\nValidating event '%s':\n", event["title"])
		if err := qf.Validate(event, eventSchema); err != nil {
			fmt.Printf("[X] Validation failed: %v\n", err)
		} else {
			fmt.Printf("[✓] Event is valid!\n")
		}
	}
}

func businessHoursExample() {
	fmt.Println("\n\n4. Business Hours Validation")
	fmt.Println("----------------------------")

	// Appointment schema - must be during business days
	appointmentSchema := builders.Object().
		Field("patientName", builders.String().Required()).
		Field("appointmentDate", builders.DateTime().
			Format("2006-01-02 15:04").
			Future().
			BusinessDay().
			Custom(func(value interface{}) error {
				// Check business hours (9 AM - 5 PM)
				t, ok := value.(time.Time)
				if !ok {
					// Try parsing if string
					if str, ok := value.(string); ok {
						parsed, err := time.Parse("2006-01-02 15:04", str)
						if err != nil {
							return nil // Let the main validator handle this
						}
						t = parsed
					} else {
						return nil
					}
				}

				hour := t.Hour()
				if hour < 9 || hour >= 17 {
					return fmt.Errorf("appointments must be between 9:00 AM and 5:00 PM")
				}

				// No appointments during lunch (12-1 PM)
				if hour == 12 {
					return fmt.Errorf("no appointments during lunch hour (12-1 PM)")
				}

				return nil
			}).
			Required()).
		Field("duration", builders.Number().Min(15).Max(120).Required())

	// Test appointments - using dynamic future dates
	now := time.Now()
	// Find next Monday at midnight
	nextMonday := now
	for nextMonday.Weekday() != time.Monday {
		nextMonday = nextMonday.Add(24 * time.Hour)
	}
	// Reset to start of day
	nextMonday = time.Date(nextMonday.Year(), nextMonday.Month(), nextMonday.Day(), 0, 0, 0, 0, nextMonday.Location())

	// Ensure it's actually in the future
	if nextMonday.Before(now) || nextMonday.Equal(now) {
		nextMonday = nextMonday.Add(7 * 24 * time.Hour)
	}

	nextFriday := nextMonday.Add(4 * 24 * time.Hour)   // Friday is 4 days after Monday
	nextSaturday := nextMonday.Add(5 * 24 * time.Hour) // Saturday is 5 days after Monday

	testAppointments := []map[string]interface{}{
		{
			"patientName":     "John Smith",
			"appointmentDate": nextMonday.Add(10*time.Hour + 30*time.Minute).Format("2006-01-02 15:04"), // Valid: future Monday at 10:30 AM
			"duration":        30,
		},
		{
			"patientName":     "Jane Doe",
			"appointmentDate": nextSaturday.Add(14 * time.Hour).Format("2006-01-02 15:04"), // Invalid: Saturday
			"duration":        45,
		},
		{
			"patientName":     "Bob Johnson",
			"appointmentDate": nextFriday.Add(8 * time.Hour).Format("2006-01-02 15:04"), // Invalid: too early (8 AM)
			"duration":        30,
		},
		{
			"patientName":     "Alice Brown",
			"appointmentDate": nextFriday.Add(12*time.Hour + 30*time.Minute).Format("2006-01-02 15:04"), // Invalid: lunch hour
			"duration":        30,
		},
		{
			"patientName":     "Charlie Wilson",
			"appointmentDate": nextFriday.Add(16*time.Hour + 45*time.Minute).Format("2006-01-02 15:04"), // Valid: 4:45 PM
			"duration":        15,
		},
	}

	fmt.Printf("(Testing with Monday=%s, Friday=%s)\n",
		nextMonday.Format("2006-01-02"),
		nextFriday.Format("2006-01-02"))

	for _, apt := range testAppointments {
		fmt.Printf("\nValidating appointment for %s:\n", apt["patientName"])
		if err := qf.Validate(apt, appointmentSchema); err != nil {
			fmt.Printf("[X] Validation failed: %v\n", err)
		} else {
			fmt.Printf("[✓] Appointment is valid!\n")
		}
	}

	// Example with flexible date formats in loose mode
	fmt.Println("\n\n5. Flexible Date Parsing (Loose Mode)")
	fmt.Println("------------------------------------")

	flexSchema := builders.DateTime().Required()

	flexDates := []struct {
		format string
		value  string
	}{
		{"ISO8601", "2024-06-15T14:30:00Z"},
		{"Date only", "2024-06-15"},
		{"US format", "06/15/2024"},
		{"UK format", "15-Jun-2024"},
		{"Long format", "June 15, 2024"},
		{"With time", "2024-06-15 14:30:00"},
		{"Time only", "14:30:00"},
		{"DD/MM/YYYY", "31/12/2024"}, // Unambiguous DD/MM
		{"MM/DD/YYYY", "12/31/2024"}, // Unambiguous MM/DD
		{"Ambiguous", "01/02/2024"},  // Could be either
	}

	fmt.Println("Testing various date formats:")
	for _, test := range flexDates {
		fmt.Printf("  %-15s: %s => ", test.format, test.value)
		if err := qf.ValidateWithMode(test.value, flexSchema, qf.Loose); err != nil {
			fmt.Printf("[X] Failed\n")
		} else {
			fmt.Printf("[✓] Parsed\n")
		}
	}
}
