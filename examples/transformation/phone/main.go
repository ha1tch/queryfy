package main

import (
	"fmt"

	qf "github.com/ha1tch/queryfy"
	"github.com/ha1tch/queryfy/builders"
	"github.com/ha1tch/queryfy/builders/transformers"
)

func main() {
	fmt.Println("=== Queryfy Phone Normalization Examples ===")

	// Example 1: US phone normalization
	usPhoneExample()

	// Example 2: South American countries
	southAmericanPhoneExample()

	// Example 3: Multi-country support with detection
	multiCountryExample()

	// Example 4: Phone validation in forms
	formValidationExample()
}

func usPhoneExample() {
	fmt.Println("\n1. US Phone Normalization")
	fmt.Println("-------------------------")

	phoneSchema := builders.Transform(
		builders.String().
			Pattern(`^\+1\d{10}$`).
			Required(),
	).Add(transformers.NormalizePhone("US"))

	testPhones := []string{
		"(555) 123-4567",
		"555-123-4567",
		"555.123.4567",
		"5551234567",
		"+1 555 123 4567",
		"1-555-123-4567",
		"555 123 4567",
		"(555)123-4567",
		"123-4567", // Too short
		"555-CALL-NOW", // Letters
	}

	for _, phone := range testPhones {
		fmt.Printf("\nInput: %q\n", phone)
		
		ctx := qf.NewValidationContext(qf.Strict)
		phoneSchema.Validate(phone, ctx)
		
		// Show transformation result
		if len(ctx.Transformations()) > 0 {
			fmt.Printf("  → Normalized to: %v\n", ctx.Transformations()[0].Result)
		}
		
		// Check validation
		if ctx.HasErrors() {
			fmt.Printf("  [X] Invalid: %s\n", ctx.Errors()[0].Message)
		} else {
			fmt.Println("  [✓] Valid US phone")
		}
	}
}

func southAmericanPhoneExample() {
	fmt.Println("\n\n2. South American Phone Normalization")
	fmt.Println("-------------------------------------")

	// Test different South American countries
	countries := []struct {
		code  string
		name  string
		phones []string
	}{
		{
			code: "AR",
			name: "Argentina",
			phones: []string{
				"011 15 1234 5678",    // Buenos Aires mobile
				"11 15 1234-5678",     // Alternative format
				"+54 9 11 1234 5678",  // International
				"0351 15 123 4567",    // Córdoba mobile
				"15 1234 5678",        // Local mobile
			},
		},
		{
			code: "BR", 
			name: "Brazil",
			phones: []string{
				"(11) 91234-5678",     // São Paulo mobile
				"11 91234-5678",       // Without parentheses
				"+55 11 91234-5678",   // International
				"(21) 98765-4321",     // Rio mobile
				"85 987654321",        // Ceará mobile
				"011 91234 5678",      // With carrier code
			},
		},
		{
			code: "CL",
			name: "Chile",
			phones: []string{
				"9 1234 5678",         // Mobile
				"+56 9 1234 5678",     // International
				"09 1234 5678",        // With leading 0
				"912345678",           // No spaces
			},
		},
		{
			code: "CO",
			name: "Colombia", 
			phones: []string{
				"301 234 5678",        // Mobile
				"3012345678",          // No spaces
				"+57 301 234 5678",    // International
				"320 987 6543",        // Another mobile
			},
		},
		{
			code: "UY",
			name: "Uruguay",
			phones: []string{
				"099 123 456",         // Mobile
				"94 123 456",          // Mobile without 0
				"+598 99 123 456",     // International
				"091234567",           // No spaces
			},
		},
	}

	for _, country := range countries {
		fmt.Printf("\n%s (%s):\n", country.name, country.code)
		
		phoneSchema := builders.Transform(
			builders.String().Required(),
		).Add(transformers.NormalizePhone(country.code))
		
		for _, phone := range country.phones {
			ctx := qf.NewValidationContext(qf.Strict)
			phoneSchema.Validate(phone, ctx)
			
			fmt.Printf("  %q", phone)
			
			// Check if transformation succeeded
			if len(ctx.Transformations()) > 0 && !ctx.HasErrors() {
				fmt.Printf(" → %v\n", ctx.Transformations()[0].Result)
			} else if ctx.HasErrors() {
				fmt.Printf(" → [X] Failed: %s\n", ctx.Errors()[0].Message)
			}
		}
	}
}

func multiCountryExample() {
	fmt.Println("\n\n3. Multi-Country Phone Detection")
	fmt.Println("--------------------------------")

	// Schema that accepts phones from multiple countries
	// and normalizes based on detected country
	multiCountrySchema := builders.Transform(
		builders.String().
			Pattern(`^\+\d{1,3}\d{7,12}$`). // Generic international format
			Required(),
	).Add(func(value interface{}) (interface{}, error) {
		phone, ok := value.(string)
		if !ok {
			return value, fmt.Errorf("expected string, got %T", value)
		}
		
		// Try to detect country and normalize
		// This is a simplified example - real implementation in transformers.NormalizePhone
		countries := []string{"US", "AR", "BR", "CL", "CO", "UY", "MX", "PE", "UK"}
		
		for _, country := range countries {
			normalizer := transformers.NormalizePhone(country)
			result, err := normalizer(phone)
			if err == nil {
				return result, nil
			}
		}
		
		return nil, fmt.Errorf("could not determine phone format")
	})

	testPhones := []struct {
		phone    string
		expected string
	}{
		{"+1 555 123 4567", "US"},
		{"+54 9 11 1234 5678", "AR"},
		{"+55 11 91234 5678", "BR"},
		{"+56 9 1234 5678", "CL"},
		{"+57 301 234 5678", "CO"},
		{"+598 99 123 456", "UY"},
		{"+44 7700 900123", "UK"},
		{"555-123-4567", "US (needs country code)"},
		{"9 1234 5678", "Ambiguous"},
	}

	fmt.Println("Testing international phone detection:")
	for _, test := range testPhones {
		ctx := qf.NewValidationContext(qf.Strict)
		multiCountrySchema.Validate(test.phone, ctx)
		
		fmt.Printf("\n%q (%s)\n", test.phone, test.expected)
		
		if len(ctx.Transformations()) > 0 && !ctx.HasErrors() {
			fmt.Printf("  [✓] Normalized to: %v\n", ctx.Transformations()[0].Result)
		} else if ctx.HasErrors() {
			fmt.Printf("  [X] Failed: %s\n", ctx.Errors()[0].Message)
		}
	}
}

func formValidationExample() {
	fmt.Println("\n\n4. Phone Validation in Forms")
	fmt.Println("----------------------------")

	// Customer form with country-specific phone validation
	customerSchema := builders.Object().
		Field("name", builders.String().Required()).
		Field("country", builders.String().
			Enum("US", "AR", "BR", "CL", "CO", "UY", "MX", "PE").
			Required()).
		Field("phone", builders.String().Required()).
		Field("alternatePhone", builders.String().Optional()).
		Custom(func(value interface{}) error {
			// Custom validation that uses country field to validate phone
			data, ok := value.(map[string]interface{})
			if !ok {
				return fmt.Errorf("expected object")
			}
			
			country, _ := data["country"].(string)
			phone, _ := data["phone"].(string)
			
			if country == "" || phone == "" {
				return nil // Let required validation handle this
			}
			
			// Normalize phone based on country
			normalizer := transformers.NormalizePhone(country)
			normalized, err := normalizer(phone)
			if err != nil {
				return fmt.Errorf("invalid phone number for %s: %w", country, err)
			}
			
			// Update the phone with normalized version
			data["phone"] = normalized
			
			// Also normalize alternate phone if provided
			if altPhone, ok := data["alternatePhone"].(string); ok && altPhone != "" {
				altNormalized, err := normalizer(altPhone)
				if err != nil {
					return fmt.Errorf("invalid alternate phone number for %s: %w", country, err)
				}
				data["alternatePhone"] = altNormalized
			}
			
			return nil
		})

	testCustomers := []map[string]interface{}{
		{
			"name":           "John Doe",
			"country":        "US",
			"phone":          "(555) 123-4567",
			"alternatePhone": "555-987-6543",
		},
		{
			"name":    "María García",
			"country": "AR",
			"phone":   "11 15 1234 5678",
		},
		{
			"name":    "João Silva",
			"country": "BR",
			"phone":   "(11) 91234-5678",
		},
		{
			"name":    "Invalid Customer",
			"country": "US",
			"phone":   "123", // Too short
		},
	}

	for _, customer := range testCustomers {
		fmt.Printf("\nCustomer: %s (%s)\n", customer["name"], customer["country"])
		fmt.Printf("Input phone: %s\n", customer["phone"])
		
		ctx := qf.NewValidationContext(qf.Strict)
		customerSchema.Validate(customer, ctx)
		
		if ctx.HasErrors() {
			fmt.Printf("[X] Validation failed:\n")
			for _, err := range ctx.Errors() {
				fmt.Printf("  - %s\n", err.Message)
			}
		} else {
			fmt.Printf("[✓] Valid customer data\n")
			fmt.Printf("Normalized phone: %s\n", customer["phone"])
			if altPhone, ok := customer["alternatePhone"]; ok {
				fmt.Printf("Normalized alt phone: %s\n", altPhone)
			}
		}
	}
}