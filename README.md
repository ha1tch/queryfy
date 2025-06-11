# Queryfy

Validate, Query, and Transform dynamic data in Go

Queryfy is a Go package for working with map-based data structures. It provides schema validation, querying capabilities, and type safety for scenarios involving dynamic data like JSON APIs and configuration files.

## Features

- Schema validation with strict or loose modes
- Query nested data structures using simple expressions
- Type checking before marshal/unmarshal operations
- Clear error messages with path information
- Flexible validation modes for different use cases

Existing Go solutions address only parts of the dynamic data problem. Libraries like `go-playground/validator` excel at struct validation but don't handle map[string]interface{} well. `gojsonschema` provides JSON Schema validation for maps but lacks querying capabilities and requires verbose schema definitions. `tidwall/gjson` offers excellent querying but no validation. `mapstructure` handles type conversion but doesn't validate schemas. Queryfy combines all three needs—validation, querying, and schema definition—in a single, cohesive package designed specifically for Go's map-based dynamic data, with a fluent API that feels natural to Go developers.

## Installation

```bash
go get github.com/ha1tch/queryfy
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/ha1tch/queryfy"
    qf "github.com/ha1tch/queryfy"
)

func main() {
    // Define a schema
    schema := qf.Object().
        Field("customerId", qf.String().Required()).
        Field("amount", qf.Number().Min(0).Required()).
        Field("items", qf.Array().Of(
            qf.Object().
                Field("productId", qf.String().Required()).
                Field("quantity", qf.Number().Min(1)).
                Field("price", qf.Number().Min(0))
        ).MinItems(1))

    // Your data
    orderData := map[string]interface{}{
        "customerId": "CUST-123",
        "amount": 150.50,
        "items": []interface{}{
            map[string]interface{}{
                "productId": "PROD-456",
                "quantity":  2,
                "price":     75.25,
            },
        },
    }

    // Validate
    if err := qf.Validate(orderData, schema); err != nil {
        fmt.Printf("Validation failed: %v\n", err)
        return
    }

    // Query
    total, _ := qf.Query(orderData, "sum(items[*].price)")
    fmt.Printf("Total price: %v\n", total)
}
```

## Core Concepts

### Schema Definition

Define schemas using the builder pattern:

```go
userSchema := qf.Object().
    Field("id", qf.String().Pattern("^[A-Z]{3}-[0-9]{6}$")).
    Field("email", qf.String().Email().Required()).
    Field("age", qf.Number().Min(0).Max(150)).
    Field("roles", qf.Array().Of(qf.String().Enum("admin", "user", "guest"))).
    Field("address", qf.Object().
        Field("street", qf.String().Required()).
        Field("city", qf.String().Required()).
        Field("zipCode", qf.String().Pattern("^[0-9]{5}$"))
    )
```

### Validation Modes

**Strict Mode** (Default): All fields must match the schema exactly.

```go
validator := qf.NewValidator(schema).Strict()
```

**Loose Mode**: Allows extra fields and provides safe type coercion.

```go
validator := qf.NewValidator(schema).Loose()
```

### Querying Data

Query using JSONPath-like syntax:

```go
// Simple path access
name, _ := qf.Query(data, "customer.firstName")

// Array access
firstItem, _ := qf.Query(data, "items[0]")

// Wildcard selection
allPrices, _ := qf.Query(data, "items[*].price")

// Filtering
expensiveItems, _ := qf.Query(data, "items[?price > 100]")

// Aggregations
sum, _ := qf.Query(data, "sum(items[*].price)")
count, _ := qf.Query(data, "count(items[*])")
avg, _ := qf.Query(data, "avg(items[*].quantity)")
```

## Advanced Usage

### Custom Validators

```go
phoneValidator := qf.Custom(func(value interface{}) error {
    str, ok := value.(string)
    if !ok {
        return fmt.Errorf("expected string, got %T", value)
    }
    if !isValidPhone(str) {
        return fmt.Errorf("invalid phone number: %s", str)
    }
    return nil
})

schema := qf.Object().
    Field("phone", phoneValidator.Required())
```

### Schema Composition

```go
addressSchema := qf.Object().
    Field("street", qf.String().Required()).
    Field("city", qf.String().Required())

customerSchema := qf.Object().
    Field("name", qf.String().Required()).
    Field("billingAddress", addressSchema).
    Field("shippingAddress", addressSchema)
```

### Pre-Marshal Validation

```go
func processOrder(data map[string]interface{}) error {
    if err := qf.Validate(data, orderSchema); err != nil {
        return fmt.Errorf("invalid order data: %w", err)
    }
    
    bytes, _ := json.Marshal(data)
    return sendToAPI(bytes)
}
```

## Real-World Example

```go
// E-commerce order validation
orderSchema := qf.Object().
    Field("orderId", qf.String().Required()).
    Field("customer", qf.Object().
        Field("email", qf.String().Email().Required()).
        Field("phone", qf.String().Required())
    ).
    Field("payment", qf.Object().
        Field("method", qf.String().Enum("CARD", "CASH", "DIGITAL_WALLET")).
        Field("amount", qf.Number().Min(0).Required()).
        Field("currency", qf.String().Length(3).Required())
    ).
    Field("items", qf.Array().MinItems(1).Of(
        qf.Object().
            Field("productId", qf.String().Required()).
            Field("quantity", qf.Number().Min(1).Required()).
            Field("price", qf.Number().Min(0).Required())
    ))

// Validate incoming order
if err := qf.Validate(orderData, orderSchema.Strict()); err != nil {
    validationErr := err.(*qf.ValidationError)
    for _, fieldErr := range validationErr.Errors {
        log.Printf("Field %s: %s", fieldErr.Path, fieldErr.Message)
    }
}

// Query order data
customerEmail, _ := qf.Query(orderData, "customer.email")
totalAmount, _ := qf.Query(orderData, "sum(items[*].price * items[*].quantity)")
highValueItems, _ := qf.Query(orderData, "items[?price > 100].productId")
```

## Performance Considerations

Queryfy is designed for production use:

- Schemas are compiled once and can be reused
- Minimal reflection for better performance
- Concurrent-safe validation
- Query results are cached when possible

```go
// Compile schema once
compiledSchema := qf.Compile(schema)

// Reuse for multiple validations
for _, order := range orders {
    if err := compiledSchema.Validate(order); err != nil {
        // Handle error
    }
}
```

## License

Copyright 2025 h@duck.com

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
