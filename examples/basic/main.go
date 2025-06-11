package main

import (
	"encoding/json"
	"fmt"
	"log"

	qf "github.com/ha1tch/queryfy"
	"github.com/ha1tch/queryfy/builders"
)

func main() {
	fmt.Println("=== Queryfy Basic Example ===")

	// Example 1: Basic E-commerce Order Validation
	basicExample()

	// Example 2: Composite Schema with AND/OR/NOT
	compositeExample()

	// Example 3: Loose Mode with Type Coercion
	looseModeExample()
}

func basicExample() {
	fmt.Println("1. Basic E-commerce Order Validation")
	fmt.Println("------------------------------------")

	// Define a schema for an e-commerce order
	orderSchema := builders.Object().
		Field("orderId", builders.String().Required()).
		Field("customer", builders.Object().
			Field("email", builders.String().Email().Required()).
			Field("name", builders.String().Required())).
		Field("items", builders.Array().
			MinItems(1).
			Of(builders.Object().
				Field("productId", builders.String().Required()).
				Field("quantity", builders.Number().Min(1).Integer().Required()).
				Field("price", builders.Number().Min(0).Required()))).
		Field("total", builders.Number().Min(0).Required())

	// Valid order data
	validOrder := map[string]interface{}{
		"orderId": "ORD-12345",
		"customer": map[string]interface{}{
			"email": "john@example.com",
			"name":  "John Doe",
		},
		"items": []interface{}{
			map[string]interface{}{
				"productId": "PROD-001",
				"quantity":  2,
				"price":     29.99,
			},
			map[string]interface{}{
				"productId": "PROD-002",
				"quantity":  1,
				"price":     49.99,
			},
		},
		"total": 109.97,
	}

	// Validate the order
	fmt.Println("Validating order...")
	if err := qf.Validate(validOrder, orderSchema); err != nil {
		fmt.Printf("[X] Validation failed: %v\n", err)
	} else {
		fmt.Println("[✓] Order is valid!")
	}

	// Query the data
	fmt.Println("\nQuerying order data:")

	// Get customer email
	email, err := qf.Query(validOrder, "customer.email")
	if err != nil {
		fmt.Printf("[X] Query failed: %v\n", err)
	} else {
		fmt.Printf("Customer email: %v\n", email)
	}

	// Get first item price
	firstPrice, err := qf.Query(validOrder, "items[0].price")
	if err != nil {
		fmt.Printf("[X] Query failed: %v\n", err)
	} else {
		fmt.Printf("First item price: $%.2f\n", firstPrice)
	}

	// Invalid order - missing required field
	fmt.Println("\nTesting invalid order:")
	invalidOrder := map[string]interface{}{
		"orderId": "ORD-12346",
		"customer": map[string]interface{}{
			"name": "Jane Doe",
			// Missing required email
		},
		"items": []interface{}{
			map[string]interface{}{
				"productId": "PROD-003",
				"quantity":  0,   // Invalid: less than minimum
				"price":     -10, // Invalid: negative price
			},
		},
		"total": 0,
	}

	if err := qf.Validate(invalidOrder, orderSchema); err != nil {
		fmt.Printf("[X] Validation errors:\n%v\n", err)
	}
	fmt.Println()
}

func compositeExample() {
	fmt.Println("2. Composite Schema with AND/OR/NOT")
	fmt.Println("-----------------------------------")

	// Example 2.1: OR validation - Email OR Phone required
	fmt.Println("\n2.1 OR Validation - Contact Info")
	fmt.Println("~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~")

	contactSchema := builders.Or(
		builders.String().Email(),
		builders.String().Pattern(`^\+?[1-9]\d{9,14}$`), // International phone
	).Required()

	// Test various contact values
	testContacts := []struct {
		name  string
		value interface{}
	}{
		{"Valid email", "user@example.com"},
		{"Valid phone", "+1234567890"},
		{"Invalid format", "not-email-or-phone"},
		{"Empty string", ""},
		{"Number", 12345},
	}

	for _, test := range testContacts {
		fmt.Printf("Testing %s (%v): ", test.name, test.value)
		if err := qf.Validate(test.value, contactSchema); err != nil {
			fmt.Printf("[X] Failed\n")
		} else {
			fmt.Printf("[✓] Valid\n")
		}
	}

	// Example 2.2: AND validation - Complex number requirements
	fmt.Println("\n2.2 AND Validation - Age Requirements")
	fmt.Println("~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~")

	ageSchema := builders.And(
		builders.Number().Min(18),   // Must be >= 18
		builders.Number().Max(65),   // Must be <= 65
		builders.Number().Integer(), // Must be integer
	).Required()

	testAges := []struct {
		name  string
		value interface{}
	}{
		{"Valid age 25", 25},
		{"Valid age 18", 18},
		{"Valid age 65", 65},
		{"Too young", 17},
		{"Too old", 66},
		{"Decimal", 25.5},
		{"String", "25"},
	}

	for _, test := range testAges {
		fmt.Printf("Testing %s (%v): ", test.name, test.value)
		if err := qf.Validate(test.value, ageSchema); err != nil {
			fmt.Printf("[X] Failed\n")
		} else {
			fmt.Printf("[✓] Valid\n")
		}
	}

	// Example 2.3: NOT validation - Exclude certain values
	fmt.Println("\n2.3 NOT Validation - Prohibited Values")
	fmt.Println("~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~")

	// Not an empty string
	nonEmptyString := builders.And(
		builders.String(),
		builders.Not(builders.String().Length(0)),
	)

	// Not a test/example email
	realEmailSchema := builders.And(
		builders.String().Email(),
		builders.Not(builders.String().Pattern(`.*@(test|example)\.com$`)),
	)

	testStrings := []struct {
		name   string
		schema qf.Schema
		value  interface{}
	}{
		{"Non-empty: 'hello'", nonEmptyString, "hello"},
		{"Non-empty: ''", nonEmptyString, ""},
		{"Non-empty: '   '", nonEmptyString, "   "},
		{"Real email: user@gmail.com", realEmailSchema, "user@gmail.com"},
		{"Real email: user@example.com", realEmailSchema, "user@example.com"},
		{"Real email: user@test.com", realEmailSchema, "user@test.com"},
	}

	for _, test := range testStrings {
		fmt.Printf("Testing %s: ", test.name)
		if err := qf.Validate(test.value, test.schema); err != nil {
			fmt.Printf("[X] Failed\n")
		} else {
			fmt.Printf("[✓] Valid\n")
		}
	}

	// Example 2.4: Complex user schema combining AND/OR/NOT
	fmt.Println("\n2.4 Complex User Schema")
	fmt.Println("~~~~~~~~~~~~~~~~~~~~~~~")

	userSchema := builders.Object().
		Field("username", builders.And(
			builders.String().MinLength(3).MaxLength(20),
			builders.Not(builders.String().Pattern(`\s`)), // No whitespace
		).Required()).
		Field("contact", builders.Or(
			builders.String().Email(),
			builders.String().Pattern(`^\+?[1-9]\d{9,14}$`),
		).Required()).
		Field("age", builders.And(
			builders.Number().Min(13),
			builders.Number().Max(120),
			builders.Number().Integer(),
		)).
		Field("bio", builders.And(
			builders.String(),
			builders.Not(builders.String().Length(0)),
		)).
		Field("role", builders.And(
			builders.String().Enum("user", "moderator", "admin"),
			builders.Not(builders.String().Length(0)),
		).Required())

	// Test users
	testUsers := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "Valid user with email",
			data: map[string]interface{}{
				"username": "john_doe",
				"contact":  "john@example.com",
				"age":      25,
				"bio":      "Software developer",
				"role":     "user",
			},
		},
		{
			name: "Valid user with phone",
			data: map[string]interface{}{
				"username": "jane123",
				"contact":  "+1234567890",
				"age":      30,
				"bio":      "Product manager",
				"role":     "moderator",
			},
		},
		{
			name: "Invalid - username with space",
			data: map[string]interface{}{
				"username": "john doe",
				"contact":  "john@example.com",
				"age":      25,
				"bio":      "Developer",
				"role":     "user",
			},
		},
		{
			name: "Invalid - no contact info",
			data: map[string]interface{}{
				"username": "nocontact",
				"contact":  "not-email-or-phone",
				"age":      25,
				"bio":      "Mystery person",
				"role":     "user",
			},
		},
		{
			name: "Invalid - empty bio",
			data: map[string]interface{}{
				"username": "emptybio",
				"contact":  "empty@example.com",
				"age":      25,
				"bio":      "",
				"role":     "user",
			},
		},
		{
			name: "Invalid - wrong role",
			data: map[string]interface{}{
				"username": "wrongrole",
				"contact":  "wrong@example.com",
				"age":      25,
				"bio":      "Has wrong role",
				"role":     "superuser",
			},
		},
	}

	for _, test := range testUsers {
		fmt.Printf("\nTesting %s:\n", test.name)
		if err := qf.Validate(test.data, userSchema); err != nil {
			fmt.Printf("[X] Validation failed:\n%v\n", err)
		} else {
			fmt.Printf("[✓] User is valid!\n")
		}
	}

	// Example 2.5: Fluent chained composite schemas
	fmt.Println("\n2.5 Fluent Chained Composite Schemas")
	fmt.Println("~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~")

	// Chained password validation: 8-20 chars AND (has letter AND has number) AND NOT common password
	passwordSchema := builders.And(
		builders.String().MinLength(8).MaxLength(20),
		builders.And(
			builders.String().Pattern(`[a-zA-Z]`), // Has letter
			builders.String().Pattern(`[0-9]`),    // Has number
		),
		builders.Not(
			builders.String().Enum("password", "12345678", "qwerty123", "admin123"),
		),
	).Required()

	// Product code: (starts with PROD- OR starts with ITEM-) AND 8 chars AND NOT test code
	productCodeSchema := builders.And(
		builders.Or(
			builders.String().Pattern(`^PROD-`),
			builders.String().Pattern(`^ITEM-`),
		),
		builders.String().Length(8),
		builders.Not(
			builders.String().Pattern(`TEST`),
		),
	).Required()

	// Complex price validation: (number AND > 0) AND (< 1000 OR special high value product)
	priceSchema := builders.Or(
		builders.And(
			builders.Number().Min(0.01),
			builders.Number().Max(999.99),
		),
		builders.And(
			builders.Number().Min(1000),
			builders.Custom(func(value interface{}) error {
				// Custom validation for high-value items
				if v, ok := value.(float64); ok && v >= 1000 {
					return nil // Approved high-value
				}
				return fmt.Errorf("high-value items need approval")
			}),
		),
	).Required()

	// Status field: string AND (one of valid statuses OR (custom AND NOT empty))
	statusSchema := builders.Or(
		builders.String().Enum("active", "pending", "completed", "cancelled"),
		builders.And(
			builders.String().Pattern(`^CUSTOM-`),
			builders.Not(builders.String().Length(7)), // Not just "CUSTOM-"
		),
	).Required()

	// Test passwords
	fmt.Println("\nPassword validation:")
	testPasswords := []struct {
		name  string
		value string
	}{
		{"Valid password", "Secret123"},
		{"Too short", "abc123"},
		{"No numbers", "SecretPassword"},
		{"No letters", "12345678"},
		{"Common password", "password"},
		{"Another common", "qwerty123"},
	}

	for _, test := range testPasswords {
		fmt.Printf("  %-20s: ", test.name)
		if err := qf.Validate(test.value, passwordSchema); err != nil {
			fmt.Printf("[X] Invalid\n")
		} else {
			fmt.Printf("[✓] Valid\n")
		}
	}

	// Test product codes
	fmt.Println("\nProduct code validation:")
	testCodes := []struct {
		name  string
		value string
	}{
		{"Valid PROD", "PROD-123"},
		{"Valid ITEM", "ITEM-456"},
		{"Test code", "PROD-TEST"},
		{"Wrong prefix", "CODE-123"},
		{"Too short", "PROD-12"},
		{"Too long", "PROD-1234"},
	}

	for _, test := range testCodes {
		fmt.Printf("  %-20s: ", test.name)
		if err := qf.Validate(test.value, productCodeSchema); err != nil {
			fmt.Printf("[X] Invalid\n")
		} else {
			fmt.Printf("[✓] Valid\n")
		}
	}

	// Test prices
	fmt.Println("\nPrice validation:")
	testPrices := []struct {
		name  string
		value interface{}
	}{
		{"Normal price", 99.99},
		{"Minimum price", 0.01},
		{"Maximum normal", 999.99},
		{"High value OK", 5000.00},
		{"Zero price", 0.0},
		{"Negative price", -10.0},
	}

	for _, test := range testPrices {
		fmt.Printf("  %-20s: ", test.name)
		if err := qf.Validate(test.value, priceSchema); err != nil {
			fmt.Printf("[X] Invalid\n")
		} else {
			fmt.Printf("[✓] Valid\n")
		}
	}

	// Test status values
	fmt.Println("\nStatus validation:")
	testStatuses := []struct {
		name  string
		value string
	}{
		{"Standard active", "active"},
		{"Standard pending", "pending"},
		{"Custom valid", "CUSTOM-SPECIAL"},
		{"Custom empty", "CUSTOM-"},
		{"Invalid status", "unknown"},
		{"Wrong custom", "custom-123"},
	}

	for _, test := range testStatuses {
		fmt.Printf("  %-20s: ", test.name)
		if err := qf.Validate(test.value, statusSchema); err != nil {
			fmt.Printf("[X] Invalid\n")
		} else {
			fmt.Printf("[✓] Valid\n")
		}
	}

	// Example 2.6: Deeply nested composite schemas
	fmt.Println("\n2.6 Deeply Nested Composite Schemas")
	fmt.Println("~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~")

	// Account validation with complex rules
	accountSchema := builders.Object().
		Field("accountId", builders.And(
			builders.String().Pattern(`^ACC-\d{6}$`),
			builders.Not(builders.String().Pattern(`0{6}`)), // Not all zeros
		).Required()).
		Field("accountType", builders.Or(
			builders.String().Enum("personal", "business"),
			builders.And(
				builders.String().Pattern(`^CUSTOM_`),
				builders.String().MinLength(10),
			),
		).Required()).
		Field("balance", builders.And(
			builders.Number(),
			builders.Or(
				builders.Number().Min(0), // Normal: non-negative
				builders.And( // Or overdraft with conditions
					builders.Number().Min(-1000),
					builders.Custom(func(v interface{}) error {
						// Overdraft requires special validation
						return nil // Simplified for example
					}),
				),
			),
		).Required()).
		Field("features", builders.Array().Of(
			builders.And(
				builders.String(),
				builders.Not(builders.String().Length(0)),
				builders.Or(
					builders.String().Enum("overdraft", "interest", "cashback"),
					builders.String().Pattern(`^BETA_`),
				),
			),
		))

	// Test accounts
	testAccounts := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "Valid personal account",
			data: map[string]interface{}{
				"accountId":   "ACC-123456",
				"accountType": "personal",
				"balance":     1500.50,
				"features":    []interface{}{"interest", "cashback"},
			},
		},
		{
			name: "Valid business with overdraft",
			data: map[string]interface{}{
				"accountId":   "ACC-789012",
				"accountType": "business",
				"balance":     -500.00,
				"features":    []interface{}{"overdraft", "BETA_INSTANT_TRANSFER"},
			},
		},
		{
			name: "Invalid - all zeros account ID",
			data: map[string]interface{}{
				"accountId":   "ACC-000000",
				"accountType": "personal",
				"balance":     100.00,
			},
		},
		{
			name: "Invalid - bad custom type",
			data: map[string]interface{}{
				"accountId":   "ACC-111111",
				"accountType": "CUSTOM_X", // Too short
				"balance":     100.00,
			},
		},
		{
			name: "Invalid - excessive overdraft",
			data: map[string]interface{}{
				"accountId":   "ACC-222222",
				"accountType": "business",
				"balance":     -2000.00,
			},
		},
	}

	for _, test := range testAccounts {
		fmt.Printf("\nTesting %s:\n", test.name)
		if err := qf.Validate(test.data, accountSchema); err != nil {
			fmt.Printf("[X] Validation failed:\n%v\n", err)
		} else {
			fmt.Printf("[✓] Account is valid!\n")
		}
	}

	fmt.Println()
}

func looseModeExample() {
	fmt.Println("3. Loose Mode with Type Coercion")
	fmt.Println("--------------------------------")

	// Schema expecting numbers
	dataSchema := builders.Object().
		Field("count", builders.Number().Min(0).Required()).
		Field("price", builders.Number().Required()).
		Field("active", builders.Bool().Required())

	// Data with string representations
	jsonData := `{
		"count": "42",
		"price": "19.99",
		"active": "true",
		"extra": "This field is not in schema"
	}`

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}

	// Validate in strict mode (will fail)
	fmt.Println("Validating in STRICT mode:")
	if err := qf.Validate(data, dataSchema); err != nil {
		fmt.Printf("[X] Validation failed (expected): %v\n", err)
	}

	// Validate in loose mode (will pass with coercion)
	fmt.Println("\nValidating in LOOSE mode:")
	if err := qf.ValidateWithMode(data, dataSchema, qf.Loose); err != nil {
		fmt.Printf("[X] Validation failed: %v\n", err)
	} else {
		fmt.Println("[✓] Data is valid with type coercion!")

		// Query the coerced values
		count, _ := qf.Query(data, "count")
		price, _ := qf.Query(data, "price")
		fmt.Printf("Count: %v (type: %T)\n", count, count)
		fmt.Printf("Price: %v (type: %T)\n", price, price)
	}
}
