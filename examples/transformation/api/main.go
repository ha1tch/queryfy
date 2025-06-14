package main

import (
	"encoding/json"
	"fmt"
	"strings"

	qf "github.com/ha1tch/queryfy"
	"github.com/ha1tch/queryfy/builders"
	"github.com/ha1tch/queryfy/builders/transformers"
)

func main() {
	fmt.Println("=== Queryfy API Transformation Example ===")

	// Example 1: User registration with transformations
	userRegistrationExample()

	// Example 2: Product data normalization
	productDataExample()

	// Example 3: Configuration processing
	configProcessingExample()
}

func userRegistrationExample() {
	fmt.Println("\n1. User Registration API")
	fmt.Println("------------------------")

	// Create schemas with validation first, then wrap with transforms
	userSchema := builders.Object().
		Field("username",
			builders.Transform(
				builders.String().
					MinLength(3).
					MaxLength(20).
					Pattern(`^[a-z0-9_]+$`).
					Required(),
			).Add(transformers.Trim()).
			Add(transformers.Lowercase()).
			Add(transformers.RemoveSpaces())).
		Field("email",
			builders.Transform(
				builders.String().
					Email().
					Required(),
			).Add(transformers.Trim()).
			Add(transformers.Lowercase())).
		Field("password",
			builders.String().
				MinLength(8).
				Pattern(`[A-Za-z]`). // Has letter
				Pattern(`[0-9]`).    // Has number
				Required()).
		Field("phone",
			builders.Transform(
				builders.String().
					Pattern(`^\+\d{11,14}$`).
					Optional(),
			).Add(transformers.Trim()).
			Add(transformers.NormalizePhone("US"))).
		Field("birthDate",
			builders.Transform(
				builders.DateTime().
					Past().
					Age(18, 120).
					Required(),
			).Add(transformers.ParseDate("2006-01-02")))

	// Test data - as JSON to simulate API requests
	testRequests := []string{
		`{
			"username": "  JohnDoe123  ",
			"email": "JOHN@EXAMPLE.COM  ",
			"password": "SecurePass123",
			"phone": "(555) 123-4567",
			"birthDate": "1990-05-15"
		}`,
		`{
			"username": "invalid user!",
			"email": "not-an-email",
			"password": "weak",
			"phone": "12345",
			"birthDate": "2010-01-01"
		}`,
	}

	for i, jsonRequest := range testRequests {
		fmt.Printf("\nRequest %d:\n", i+1)
		
		// Parse JSON request
		var userData map[string]interface{}
		if err := json.Unmarshal([]byte(jsonRequest), &userData); err != nil {
			fmt.Printf("Invalid JSON: %v\n", err)
			continue
		}
		
		fmt.Printf("Input: %s\n", jsonRequest)

		// Validate with context to see all errors and transformations
		ctx := qf.NewValidationContext(qf.Strict)
		userSchema.Validate(userData, ctx)
		
		// Show transformations
		if len(ctx.Transformations()) > 0 {
			fmt.Println("Transformations applied:")
			for _, t := range ctx.Transformations() {
				if fmt.Sprintf("%v", t.Original) != fmt.Sprintf("%v", t.Result) {
					fmt.Printf("  - %s: %v → %v\n", t.Path, t.Original, t.Result)
				}
			}
		}
		
		// Show validation result
		if ctx.HasErrors() {
			fmt.Println("Response: 400 Bad Request")
			fmt.Println("Errors:")
			for _, e := range ctx.Errors() {
				fmt.Printf("  - %s: %s\n", e.Path, e.Message)
			}
		} else {
			fmt.Println("Response: 201 Created")
			fmt.Println("User successfully registered!")
		}
	}
}

func productDataExample() {
	fmt.Println("\n\n2. Product API")
	fmt.Println("--------------")

	// SKU transformer - custom business logic
	skuTransformer := func(value interface{}) (interface{}, error) {
		sku := strings.ToUpper(strings.TrimSpace(value.(string)))
		sku = strings.ReplaceAll(sku, " ", "-")
		sku = strings.ReplaceAll(sku, "_", "-")
		// Remove multiple dashes
		for strings.Contains(sku, "--") {
			sku = strings.ReplaceAll(sku, "--", "-")
		}
		return sku, nil
	}

	// Product schema with comprehensive transformations
	productSchema := builders.Object().
		Field("sku",
			builders.Transform(
				builders.String().
					Pattern(`^[A-Z0-9-]+$`).
					MinLength(5).
					Required(),
			).Add(skuTransformer)).
		Field("name",
			builders.Transform(
				builders.String().
					MinLength(3).
					MaxLength(100).
					Required(),
			).Add(transformers.Trim()).
			Add(transformers.NormalizeWhitespace())).
		Field("price",
			builders.Transform(
				builders.Number().Min(0.01).Required(),
			).Add(transformers.RemoveCurrencySymbols()).
			Add(transformers.ToFloat64()).
			Add(transformers.Round(2))).
		Field("quantity",
			builders.Transform(
				builders.Number().Min(0).Integer().Required(),
			).Add(transformers.ToInt())).
		Field("category",
			builders.String().
				Enum("electronics", "clothing", "home", "toys").
				Required()).
		Field("tags",
			builders.Array().Of(
				builders.Transform(
					builders.String(),
				).Add(transformers.Trim()).
				Add(transformers.Lowercase()),
			).MinItems(1).MaxItems(5).Optional())

	// Test product submissions
	testProducts := []string{
		`{
			"sku": "prod 123 abc",
			"name": "  Amazing   Widget   ",
			"price": "$29.997",
			"quantity": 10,
			"category": "electronics",
			"tags": [" Electronics ", " GADGET ", "new"]
		}`,
		`{
			"sku": "bad",
			"name": "x",
			"price": "$0.00",
			"quantity": -5,
			"category": "invalid",
			"tags": ["tag1", "tag2", "tag3", "tag4", "tag5", "tag6"]
		}`,
	}

	for i, jsonProduct := range testProducts {
		fmt.Printf("\nProduct Submission %d:\n", i+1)
		
		var productData map[string]interface{}
		if err := json.Unmarshal([]byte(jsonProduct), &productData); err != nil {
			fmt.Printf("Invalid JSON: %v\n", err)
			continue
		}

		ctx := qf.NewValidationContext(qf.Strict)
		productSchema.Validate(productData, ctx)
		
		// Show transformations
		if len(ctx.Transformations()) > 0 {
			fmt.Println("Transformations applied:")
			for _, t := range ctx.Transformations() {
				fmt.Printf("  %s: %v → %v\n", t.Path, t.Original, t.Result)
			}
		}
		
		// API Response
		if ctx.HasErrors() {
			fmt.Println("\nResponse: 400 Bad Request")
			fmt.Println("Validation errors:")
			for _, e := range ctx.Errors() {
				fmt.Printf("  - %s: %s\n", e.Path, e.Message)
			}
		} else {
			fmt.Println("\nResponse: 200 OK")
			fmt.Println("Product saved successfully!")
		}
	}
}

func configProcessingExample() {
	fmt.Println("\n\n3. Configuration API")
	fmt.Println("--------------------")

	// Environment variable expander - simulates env var resolution
	envExpander := func(value interface{}) (interface{}, error) {
		str, ok := value.(string)
		if !ok {
			// Not a string, return as-is
			return value, nil
		}
		if strings.HasPrefix(str, "${") && strings.HasSuffix(str, "}") {
			// In real app, would use os.Getenv()
			envVar := str[2 : len(str)-1]
			switch envVar {
			case "API_KEY":
				return "secret-key-123", nil
			case "DB_HOST":
				return "localhost", nil
			case "PORT":
				return "8080", nil
			default:
				return "", fmt.Errorf("environment variable %s not found", envVar)
			}
		}
		return str, nil
	}

	// Configuration schema with environment variable expansion
	configSchema := builders.Object().
		Field("apiKey",
			builders.Transform(
				builders.String().MinLength(10).Required(),
			).Add(envExpander)).
		Field("database", builders.Object().
			Field("host",
				builders.Transform(
					builders.String().Required(),
				).Add(envExpander)).
			Field("port",
				builders.Transform(
					builders.Number().Min(1).Max(65535).Required(),
				).Add(envExpander).
				Add(transformers.ToInt())).
			Field("poolSize",
				builders.Number().Min(1).Max(100).Required())).
		Field("cache", builders.Object().
			Field("enabled",
				builders.Transform(
					builders.Bool().Required(),
				).Add(transformers.ToBoolean())).
			Field("ttl",
				builders.Number().Min(60).Max(86400).Optional())).
		Field("features", builders.Object().
			Field("rateLimit",
				builders.Transform(
					builders.Bool().Required(),
				).Add(transformers.ToBoolean())).
			Field("cacheEnabled",
				builders.Transform(
					builders.Bool().Required(),
				).Add(transformers.ToBoolean())))

	// Test configurations
	testConfigs := []struct {
		name string
		json string
	}{
		{
			name: "Valid configuration",
			json: `{
				"apiKey": "${API_KEY}",
				"database": {
					"host": "${DB_HOST}",
					"port": "${PORT}",
					"poolSize": 10
				},
				"cache": {
					"enabled": "true",
					"ttl": 3600
				},
				"features": {
					"rateLimit": "yes",
					"cacheEnabled": "1"
				}
			}`,
		},
		{
			name: "Invalid configuration",
			json: `{
				"apiKey": "short",
				"database": {
					"host": "${UNKNOWN_HOST}",
					"port": 99999,
					"poolSize": 0
				},
				"cache": {
					"enabled": "maybe",
					"ttl": 30
				},
				"features": {
					"rateLimit": "false",
					"cacheEnabled": "no"
				}
			}`,
		},
	}

	for _, test := range testConfigs {
		fmt.Printf("\n%s:\n", test.name)
		
		var configData map[string]interface{}
		if err := json.Unmarshal([]byte(test.json), &configData); err != nil {
			fmt.Printf("Invalid JSON: %v\n", err)
			continue
		}

		// Pretty print input
		fmt.Println("Input:")
		fmt.Println(test.json)

		ctx := qf.NewValidationContext(qf.Strict)
		configSchema.Validate(configData, ctx)
		
		// Show transformations
		if len(ctx.Transformations()) > 0 {
			fmt.Println("\nTransformations:")
			// Group by type for clarity
			envVars := []qf.TransformationRecord{}
			typeConversions := []qf.TransformationRecord{}
			
			for _, t := range ctx.Transformations() {
				if strings.Contains(fmt.Sprintf("%v", t.Original), "${") {
					envVars = append(envVars, t)
				} else {
					typeConversions = append(typeConversions, t)
				}
			}
			
			if len(envVars) > 0 {
				fmt.Println("  Environment variables resolved:")
				for _, t := range envVars {
					fmt.Printf("    %s: %v → %v\n", t.Path, t.Original, t.Result)
				}
			}
			
			if len(typeConversions) > 0 {
				fmt.Println("  Type conversions:")
				for _, t := range typeConversions {
					fmt.Printf("    %s: %v (%T) → %v (%T)\n", 
						t.Path, t.Original, t.Original, t.Result, t.Result)
				}
			}
		}
		
		// Show result
		if ctx.HasErrors() {
			fmt.Println("\nResponse: 400 Bad Request")
			fmt.Println("Configuration errors:")
			for _, e := range ctx.Errors() {
				fmt.Printf("  - %s: %s\n", e.Path, e.Message)
			}
		} else {
			fmt.Println("\nResponse: 200 OK")
			fmt.Println("Configuration accepted!")
			
			// Show final transformed state
			// Note: In a real application, you would use ValidateAndTransform 
			// to get the transformed data structure
			fmt.Println("\nFinal configuration state:")
			fmt.Println("(Note: Original data shown - use ValidateAndTransform to get transformed values)")
		}
	}
}