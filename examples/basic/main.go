package main

import (
	"fmt"
	"strings"

	qf "github.com/ha1tch/queryfy"
	"github.com/ha1tch/queryfy/builders"
	"github.com/ha1tch/queryfy/builders/transformers"
)

func main() {
	fmt.Println("=== Queryfy Transformation Examples ===")

	// Example 1: Basic string transformations
	stringTransformExample()

	// Example 2: Number transformations
	numberTransformExample()

	// Example 3: Data sanitization
	dataSanitizationExample()

	// Example 4: Complex transformations with validation
	complexTransformExample()
}

func stringTransformExample() {
	fmt.Println("\n1. String Transformations")
	fmt.Println("-------------------------")

	// Email normalization schema - build validation first, then wrap with transform
	emailSchema := builders.Transform(
		builders.String().
			Email().
			Required(),
	).Add(transformers.Trim()).
	Add(transformers.Lowercase())

	testEmails := []string{
		"  John.Doe@EXAMPLE.COM  ",
		"ADMIN@COMPANY.COM",
		"  support@test.com",
		"Invalid Email",
	}

	for _, email := range testEmails {
		fmt.Printf("\nInput: %q\n", email)
		
		// Using high-level API which properly checks errors
		if err := qf.Validate(email, emailSchema); err != nil {
			fmt.Printf("  [X] Validation failed: %v\n", err)
		} else {
			fmt.Println("  [✓] Valid email")
			
			// To see transformations, we need to use the context
			ctx := qf.NewValidationContext(qf.Strict)
			emailSchema.Validate(email, ctx)
			for _, transform := range ctx.Transformations() {
				fmt.Printf("  → Transformed from %q to %q\n", 
					transform.Original, transform.Result)
			}
		}
	}

	// Username normalization
	fmt.Println("\n\nUsername Normalization:")
	
	usernameSchema := builders.Transform(
		builders.String().
			MinLength(3).
			MaxLength(20).
			Pattern(`^[a-z0-9]+$`).
			Required(),
	).Add(transformers.Trim()).
	Add(transformers.Lowercase()).
	Add(transformers.RemoveSpaces())

	testUsernames := []string{
		"  John Doe  ",
		"USER_NAME_123",
		"test user",
		"ValidUser123",
	}

	for _, username := range testUsernames {
		fmt.Printf("\nInput: %q\n", username)
		
		ctx := qf.NewValidationContext(qf.Strict)
		usernameSchema.Validate(username, ctx)
		
		// Show transformations first
		if len(ctx.Transformations()) > 0 {
			last := ctx.Transformations()[len(ctx.Transformations())-1]
			fmt.Printf("  → Final form: %q\n", last.Result)
		}
		
		// Then check validation
		if ctx.HasErrors() {
			fmt.Println("  [X] Validation failed:")
			for _, err := range ctx.Errors() {
				fmt.Printf("      %s\n", err.Message)
			}
		} else {
			fmt.Println("  [✓] Valid username")
		}
	}
}

func numberTransformExample() {
	fmt.Println("\n\n2. Number Transformations")
	fmt.Println("-------------------------")

	// Price normalization using Transform wrapper
	priceSchema := builders.Transform(
		builders.Number().Min(0).Required(),
	).Add(transformers.RemoveCurrencySymbols()).
	Add(transformers.ToFloat64()).
	Add(transformers.Round(2))

	testPrices := []string{
		"$19.99",
		"€ 42.997",
		"£100.00",
		"$1,234.56",
		"-$10.00",
	}

	fmt.Println("Price validation and transformation:")
	for _, price := range testPrices {
		fmt.Printf("\nInput: %q\n", price)
		
		ctx := qf.NewValidationContext(qf.Strict)
		result, err := priceSchema.ValidateAndTransform(price, ctx)
		
		if err != nil {
			fmt.Printf("  [X] Failed: %v\n", err)
		} else {
			fmt.Printf("  [✓] Transformed to: %.2f\n", result)
		}
	}

	// Percentage conversion
	fmt.Println("\n\nPercentage Conversion:")
	
	percentSchema := builders.Transform(
		builders.Number().
			Min(0).
			Max(1).
			Required(),
	).Add(transformers.FromPercentage())

	testPercents := []interface{}{
		25,      // 25% -> 0.25
		50.5,    // 50.5% -> 0.505
		100,     // 100% -> 1.0
		150,     // 150% -> 1.5 (invalid, max is 1)
	}

	for _, percent := range testPercents {
		fmt.Printf("\nInput: %v%%\n", percent)
		
		ctx := qf.NewValidationContext(qf.Strict)
		percentSchema.Validate(percent, ctx)
		
		// Show transformation
		if len(ctx.Transformations()) > 0 {
			fmt.Printf("  → Converted to: %v\n", ctx.Transformations()[0].Result)
		}
		
		// Check validation
		if ctx.HasErrors() {
			fmt.Printf("  [X] Validation failed: %s\n", ctx.Errors()[0].Message)
		} else {
			fmt.Printf("  [✓] Valid percentage")
		}
	}
}

func dataSanitizationExample() {
	fmt.Println("\n\n3. Data Sanitization")
	fmt.Println("--------------------")

	// User input sanitization
	userInputSchema := builders.Object().
		Field("name",
			builders.Transform(
				builders.String().
					MinLength(2).
					Required(),
			).Add(transformers.Trim()).
			Add(transformers.NormalizeWhitespace())).
		Field("bio",
			builders.Transform(
				builders.String().Optional(),
			).Add(transformers.Trim()).
			Add(transformers.Truncate(200))).
		Field("website",
			builders.Transform(
				builders.String().
					Pattern(`^https?://.*`).
					Optional(),
			).Add(transformers.Trim()).
			Add(transformers.Lowercase()).
			Add(func(value interface{}) (interface{}, error) {
				// Ensure URL has protocol
				url := value.(string)
				if url != "" && !strings.HasPrefix(url, "http://") && 
				   !strings.HasPrefix(url, "https://") {
					return "https://" + url, nil
				}
				return url, nil
			}))

	testUsers := []map[string]interface{}{
		{
			"name":    "  John    Doe  ",
			"bio":     "  This is a   very   long bio that goes on and on and on and on and on and on and on and on and on and on and on and on and on and on and on and on and on and on and on and on and on and on and on and on and on and on and should be truncated  ",
			"website": "EXAMPLE.COM",
		},
		{
			"name":    "Jane Smith",
			"bio":     "Software developer",
			"website": "https://janesmith.dev",
		},
		{
			"name":    " ", // Too short after trim
			"bio":     "",
			"website": "not a url",
		},
	}

	for i, userData := range testUsers {
		fmt.Printf("\nUser %d:\n", i+1)
		fmt.Printf("Input: %+v\n", userData)
		
		ctx := qf.NewValidationContext(qf.Strict)
		userInputSchema.Validate(userData, ctx)
		
		// Show transformations
		if len(ctx.Transformations()) > 0 {
			fmt.Println("Transformations applied:")
			for _, transform := range ctx.Transformations() {
				if fmt.Sprintf("%v", transform.Original) != fmt.Sprintf("%v", transform.Result) {
					fmt.Printf("  %s: %q → %q\n", 
						transform.Path, transform.Original, transform.Result)
				}
			}
		}
		
		// Check validation
		if ctx.HasErrors() {
			fmt.Println("[X] Validation failed:")
			for _, err := range ctx.Errors() {
				fmt.Printf("  - %s: %s\n", err.Path, err.Message)
			}
		} else {
			fmt.Println("[✓] Valid user data")
		}
	}
}

func complexTransformExample() {
	fmt.Println("\n\n4. Complex Transformations")
	fmt.Println("--------------------------")

	// Product SKU normalization
	skuTransformer := func(value interface{}) (interface{}, error) {
		sku := value.(string)
		// Convert to uppercase and replace spaces with dashes
		sku = strings.ToUpper(sku)
		sku = strings.ReplaceAll(sku, " ", "-")
		sku = strings.ReplaceAll(sku, "_", "-")
		// Remove multiple dashes
		for strings.Contains(sku, "--") {
			sku = strings.ReplaceAll(sku, "--", "-")
		}
		return sku, nil
	}

	// Order processing with transformations
	orderSchema := builders.Object().
		Field("orderId",
			builders.Transform(
				builders.String().
					Pattern(`^ORD-\d{6}$`).
					Required(),
			).Add(transformers.Uppercase())).
		Field("customer", builders.Object().
			Field("email",
				builders.Transform(
					builders.String().
						Email().
						Required(),
				).Add(transformers.Trim()).
				Add(transformers.Lowercase())).
			Field("phone",
				builders.Transform(
					builders.String().Optional(),
				).Add(transformers.NormalizePhone("US")))).
		Field("items", builders.Array().Of(
			builders.Object().
				Field("sku",
					builders.Transform(
						builders.String().
							Pattern(`^[A-Z0-9-]+$`).
							Required(),
					).Add(transformers.Trim()).
					Add(skuTransformer)).
				Field("quantity",
					builders.Transform(
						builders.Number().
							Min(1).
							Required(),
					).Add(transformers.ToInt())).
				Field("price",
					builders.Transform(
						builders.Number().Required(),
					).Add(transformers.RemoveCurrencySymbols()).
					Add(transformers.ToFloat64()).
					Add(transformers.Round(2))))).
		Field("notes",
			builders.Transform(
				builders.String().Optional(),
			).Add(transformers.Trim()).
			Add(transformers.Default("No notes provided")))

	testOrder := map[string]interface{}{
		"orderId": "ord-123456",
		"customer": map[string]interface{}{
			"email": "  CUSTOMER@EXAMPLE.COM  ",
			"phone": "(555) 123-4567",
		},
		"items": []interface{}{
			map[string]interface{}{
				"sku":      "prod 123 abc",
				"quantity": "2",
				"price":    "$29.997",
			},
			map[string]interface{}{
				"sku":      "item__xyz__789",
				"quantity": 1.0,
				"price":    "€ 15.50",
			},
		},
		"notes": "   ",
	}

	fmt.Println("Input order:")
	fmt.Printf("%+v\n", testOrder)
	
	ctx := qf.NewValidationContext(qf.Strict)
	orderSchema.Validate(testOrder, ctx)
	
	// Always show transformations first
	if len(ctx.Transformations()) > 0 {
		fmt.Println("\nTransformations applied:")
		for _, transform := range ctx.Transformations() {
			fmt.Printf("  %s: %v → %v\n", 
				transform.Path, transform.Original, transform.Result)
		}
	}
	
	// Then show validation result
	if ctx.HasErrors() {
		fmt.Printf("\n[X] Order validation failed:\n")
		for _, err := range ctx.Errors() {
			fmt.Printf("  - %s: %s\n", err.Path, err.Message)
		}
	} else {
		fmt.Println("\n[✓] Order is valid!")
	}

	// Example of getting transformed data
	fmt.Println("\n\nUsing ValidateAndTransform for data retrieval:")
	
	emailInput := "  TEST@EXAMPLE.COM  "
	emailTransformSchema := builders.Transform(
		builders.String().Email(),
	).Add(transformers.Trim()).Add(transformers.Lowercase())
	
	ctx2 := qf.NewValidationContext(qf.Strict)
	transformed, err := emailTransformSchema.ValidateAndTransform(emailInput, ctx2)
	if err != nil {
		fmt.Printf("Failed to transform: %v\n", err)
	} else {
		fmt.Printf("Original: %q → Transformed: %q\n", emailInput, transformed)
	}
}