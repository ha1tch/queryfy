package main

import (
    "encoding/json"
    "fmt"
    "log"
    
    qf "github.com/ha1tch/queryfy"
    "github.com/ha1tch/queryfy/builders"
    "github.com/ha1tch/queryfy/builders/transformers"
)

func main() {
    fmt.Println("=== ValidateAndTransform: The Complete Solution ===")
    fmt.Println("No helper functions needed - Queryfy returns transformed data!")
    fmt.Println()

    // EXAMPLE 1: Simple string transformation
    fmt.Println("1. Simple String Transformation")
    fmt.Println("-------------------------------")
    
    emailSchema := builders.Transform(
        builders.String().Email().Required(),
    ).Add(transformers.Trim()).
      Add(transformers.Lowercase())

    messyEmail := "  JOHN.DOE@EXAMPLE.COM  "
    
    ctx := qf.NewValidationContext(qf.Strict)
    cleanEmail, err := emailSchema.ValidateAndTransform(messyEmail, ctx)
    
    fmt.Printf("Input:       %q\n", messyEmail)
    fmt.Printf("Output:      %q\n", cleanEmail) // "john.doe@example.com"
    fmt.Printf("Error:       %v\n", err)
    fmt.Println()

    // EXAMPLE 2: Object transformation - THE KEY EXAMPLE
    fmt.Println("2. Object Transformation - Complete Data Cleanup")
    fmt.Println("------------------------------------------------")
    
    userSchema := builders.Object().
        Field("email", builders.Transform(
            builders.String().Email().Required(),
        ).Add(transformers.Trim()).Add(transformers.Lowercase())).
        Field("username", builders.Transform(
            builders.String().MinLength(3).Required(),
        ).Add(transformers.Trim()).Add(transformers.Lowercase())).
        Field("phone", builders.Transform(
            builders.String().Optional(),
        ).Add(transformers.NormalizePhone("US"))).
        Field("age", builders.Transform(
            builders.Number().Min(0).Max(150).Required(),
        ).Add(transformers.ToInt()))

    messyUserData := map[string]interface{}{
        "email":    "  ALICE@COMPANY.COM  ",
        "username": "  AliceWonder  ",
        "phone":    "1 (555) 123-4567",
        "age":      "25", // String instead of number!
    }

    fmt.Println("BEFORE:")
    printJSON(messyUserData)

    ctx = qf.NewValidationContext(qf.Loose) // Note: Loose mode for type conversion
    cleanUserData, err := userSchema.ValidateAndTransform(messyUserData, ctx)
    
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("\nAFTER (this is the actual returned data):")
    printJSON(cleanUserData)
    
    // IMPORTANT: Show that cleanUserData is a NEW map with transformed values
    fmt.Println("\n✅ Key Point: cleanUserData is a NEW map with all transformations applied!")
    fmt.Printf("   Original email: %v\n", messyUserData["email"])
    fmt.Printf("   Cleaned email:  %v\n", cleanUserData.(map[string]interface{})["email"])
    
    // EXAMPLE 3: Nested transformation
    fmt.Println("\n3. Nested Object Transformation")
    fmt.Println("-------------------------------")
    
    orderSchema := builders.Object().
        Field("orderId", builders.Transform(
            builders.String().Required(),
        ).Add(transformers.Uppercase()).Add(transformers.Trim())).
        Field("customer", builders.Object().
            Field("name", builders.Transform(
                builders.String().Required(),
            ).Add(transformers.Trim()).Add(transformers.NormalizeWhitespace())).
            Field("email", builders.Transform(
                builders.String().Email().Required(),
            ).Add(transformers.Trim()).Add(transformers.Lowercase()))).
        Field("items", builders.Array().Items(
            builders.Object().
                Field("sku", builders.Transform(
                    builders.String().Required(),
                ).Add(transformers.Uppercase()).Add(transformers.Trim())).
                Field("quantity", builders.Transform(
                    builders.Number().Min(1).Required(),
                ).Add(transformers.ToInt()))))

    messyOrder := map[string]interface{}{
        "orderId": "  ord-2024-001  ",
        "customer": map[string]interface{}{
            "name":  "  John    Doe  ",
            "email": "  JOHN@EXAMPLE.COM  ",
        },
        "items": []interface{}{
            map[string]interface{}{
                "sku":      "  prod-abc  ",
                "quantity": "5", // String!
            },
            map[string]interface{}{
                "sku":      "  prod-xyz  ",
                "quantity": 3.0, // Float!
            },
        },
    }

    fmt.Println("COMPLEX NESTED DATA BEFORE:")
    printJSON(messyOrder)

    ctx = qf.NewValidationContext(qf.Loose)
    cleanOrder, err := orderSchema.ValidateAndTransform(messyOrder, ctx)
    
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("\nCOMPLEX NESTED DATA AFTER:")
    printJSON(cleanOrder)
    
    // ANTI-PATTERN: Show what NOT to do
    fmt.Println("\n❌ ANTI-PATTERN - Don't do this:")
    fmt.Println("```go")
    fmt.Println("// DON'T manually extract transformations from context")
    fmt.Println("err := schema.Validate(data, ctx)")
    fmt.Println("transformedData := extractTransformedData(data, ctx) // UNNECESSARY!")
    fmt.Println("```")
    
    fmt.Println("\n✅ CORRECT PATTERN - Do this instead:")
    fmt.Println("```go")
    fmt.Println("// DO use ValidateAndTransform directly")
    fmt.Println("transformedData, err := schema.ValidateAndTransform(data, ctx)")
    fmt.Println("```")

    // EXAMPLE 4: Show transformation tracking
    fmt.Println("\n4. Understanding Transformation Tracking")
    fmt.Println("---------------------------------------")
    
    fmt.Println("The context tracks all transformations for debugging:")
    
    debugSchema := builders.Transform(
        builders.String().MinLength(5).Required(),
    ).Add(transformers.Trim()).
      Add(transformers.Uppercase()).
      Add(transformers.NormalizeWhitespace())

    debugInput := "  hello   world  "
    ctx = qf.NewValidationContext(qf.Strict)
    debugOutput, _ := debugSchema.ValidateAndTransform(debugInput, ctx)
    
    fmt.Printf("\nInput:  %q\n", debugInput)
    fmt.Printf("Output: %q\n", debugOutput)
    
    fmt.Println("\nTransformation steps recorded in context:")
    for i, transform := range ctx.Transformations() {
        fmt.Printf("  Step %d (%s): %q → %q\n", 
            i+1, transform.Type, transform.Original, transform.Result)
    }

    // EXAMPLE 5: Error handling with transformations
    fmt.Println("\n5. Error Handling with Transformations")
    fmt.Println("--------------------------------------")
    
    strictEmailSchema := builders.Transform(
        builders.String().Email().Required(),
    ).Add(transformers.Trim()).Add(transformers.Lowercase())

    invalidEmails := []string{
        "not-an-email",
        "  VALID@EMAIL.COM  ",
        "",
        "missing@domain",
    }

    for _, email := range invalidEmails {
        ctx = qf.NewValidationContext(qf.Strict)
        result, err := strictEmailSchema.ValidateAndTransform(email, ctx)
        
        if err != nil {
            fmt.Printf("\nInput %q failed validation:\n", email)
            for _, validationErr := range ctx.Errors() {
                fmt.Printf("  - %s\n", validationErr.Message)
            }
        } else {
            fmt.Printf("\nInput %q → Output %q ✓\n", email, result)
        }
    }

    // EXAMPLE 6: Real-world scenario
    fmt.Println("\n6. Real-World API Request Processing")
    fmt.Println("------------------------------------")
    
    apiRequestSchema := builders.Object().
        Field("action", builders.Transform(
            builders.String().Enum("create", "update", "delete").Required(),
        ).Add(transformers.Lowercase()).Add(transformers.Trim())).
        Field("resource", builders.Transform(
            builders.String().Required(),
        ).Add(transformers.Lowercase()).Add(transformers.Trim())).
        Field("data", builders.Object().
            Field("title", builders.Transform(
                builders.String().MaxLength(100).Required(),
            ).Add(transformers.Trim()).Add(transformers.NormalizeWhitespace())).
            Field("tags", builders.Array().Items(
                builders.Transform(
                    builders.String(),
                ).Add(transformers.Trim()).Add(transformers.Lowercase()),
            ))).
        Field("timestamp", builders.Transform(
            builders.Number().Required(),
        ).Add(transformers.ToInt()))

    apiRequest := map[string]interface{}{
        "action":    "  CREATE  ",
        "resource":  "  Article  ",
        "data": map[string]interface{}{
            "title": "  My   Amazing    Article  ",
            "tags":  []interface{}{"  Technology  ", "  Programming  ", "  GO  "},
        },
        "timestamp": "1705320000", // Unix timestamp as string
    }

    fmt.Println("Raw API Request:")
    printJSON(apiRequest)

    ctx = qf.NewValidationContext(qf.Loose)
    processedRequest, err := apiRequestSchema.ValidateAndTransform(apiRequest, ctx)
    
    if err != nil {
        fmt.Printf("\nAPI request validation failed: %v\n", err)
        for _, e := range ctx.Errors() {
            fmt.Printf("  - %s: %s\n", e.Path, e.Message)
        }
    } else {
        fmt.Println("\nProcessed API Request (ready for handling):")
        printJSON(processedRequest)
        
        // Show how to use the cleaned data
        cleanData := processedRequest.(map[string]interface{})
        fmt.Printf("\nExtracted values:\n")
        fmt.Printf("  Action: %s\n", cleanData["action"])
        fmt.Printf("  Resource: %s\n", cleanData["resource"])
        
        data := cleanData["data"].(map[string]interface{})
        fmt.Printf("  Title: %s\n", data["title"])
        fmt.Printf("  Tags: %v\n", data["tags"])
        fmt.Printf("  Timestamp: %d\n", cleanData["timestamp"])
    }

    // FINAL SUMMARY
    fmt.Println("\n" + strings.Repeat("=", 60))
    fmt.Println("SUMMARY: ValidateAndTransform is ALL YOU NEED!")
    fmt.Println(strings.Repeat("=", 60))
    fmt.Println()
    fmt.Println("1. It validates your data according to the schema")
    fmt.Println("2. It applies ALL transformations automatically")
    fmt.Println("3. It returns a NEW data structure with clean values")
    fmt.Println("4. It works recursively for nested objects and arrays")
    fmt.Println("5. No helper functions or manual extraction needed!")
    fmt.Println()
    fmt.Println("Just call: cleanData, err := schema.ValidateAndTransform(messyData, ctx)")
}

func printJSON(data interface{}) {
    b, _ := json.MarshalIndent(data, "", "  ")
    fmt.Println(string(b))
}

// This function demonstrates what the reviewer unnecessarily created
// DO NOT USE THIS - IT'S SHOWN HERE ONLY AS AN ANTI-PATTERN
func unnecessaryHelperFunction(data map[string]interface{}, ctx *qf.ValidationContext) map[string]interface{} {
    // This entire function is UNNECESSARY because ValidateAndTransform
    // already returns the transformed data!
    result := make(map[string]interface{})
    for k, v := range data {
        result[k] = v
    }
    for _, transform := range ctx.Transformations() {
        if transform.Path != "" {
            // This manual extraction is NOT NEEDED
            result[transform.Path] = transform.Result
        }
    }
    return result
}

// Import strings package for the summary section
var strings = struct {
    Repeat func(s string, count int) string
}{
    Repeat: func(s string, count int) string {
        result := ""
        for i := 0; i < count; i++ {
            result += s
        }
        return result
    },
}