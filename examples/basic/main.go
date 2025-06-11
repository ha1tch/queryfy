package main

import (
	"encoding/json"
	"fmt"
	"log"

	qf "github.com/ha1tch/queryfy"
	"github.com/ha1tch/queryfy/builders"
)

func main() {
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
				Field("quantity", builders.Number().Min(1).Required()).
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
		log.Printf("Validation failed: %v\n", err)
	} else {
		fmt.Println("Order is valid!")
	}

	// Query the data
	fmt.Println("\nQuerying order data:")

	// Get customer email
	email, err := qf.Query(validOrder, "customer.email")
	if err != nil {
		log.Printf("Query failed: %v\n", err)
	} else {
		fmt.Printf("Customer email: %v\n", email)
	}

	// Get first item price
	firstPrice, err := qf.Query(validOrder, "items[0].price")
	if err != nil {
		log.Printf("Query failed: %v\n", err)
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
		"items": []interface{}{},
		"total": 0,
	}

	if err := qf.Validate(invalidOrder, orderSchema); err != nil {
		fmt.Printf("Validation errors:\n%v\n", err)
	}

	// Demonstrate JSON validation
	fmt.Println("\nValidating JSON:")
	jsonData := `{
		"orderId": "ORD-JSON",
		"customer": {
			"email": "test@example.com",
			"name": "Test User"
		},
		"items": [
			{
				"productId": "PROD-JSON",
				"quantity": 1,
				"price": 99.99
			}
		],
		"total": 99.99
	}`

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}

	if err := qf.Validate(data, orderSchema); err != nil {
		fmt.Printf("JSON validation failed: %v\n", err)
	} else {
		fmt.Println("JSON is valid!")
	}
}
