# Queryfy
#### Validate, Query, and Transform Dynamic Data in Go

Queryfy is a Go package for working with map-based data structures. It provides schema validation and querying capabilities for scenarios involving dynamic data like JSON APIs and configuration files.

## Features

- **Schema validation** with strict or loose modes
- **Query nested data** using dot notation and array indexing
- **Type safety** with validation-time type checking
- **Clear error messages** with exact field paths
- **Composable schemas** with AND/OR/NOT logic
- **Fluent builder API** for intuitive schema definitions
- **Data transformation** with built-in and custom transformers
- **DateTime validation** with comprehensive format support
- **Dependent field validation** for conditional requirements
**Work in progress**
- **Superjsonic** our subproject aiming to produce a fast parser for JSON validation
https://github.com/ha1tch/queryfy/tree/main/superjsonic#readme

## Why Queryfy?

Existing Go solutions address only parts of the dynamic data problem. Libraries like `go-playground/validator` excel at struct validation but don't handle `map[string]interface{}` well. `gojsonschema` provides JSON Schema validation but lacks querying capabilities and requires verbose schema definitions. `tidwall/gjson` offers excellent querying but no validation. Queryfy combines validation and querying in a single, cohesive package designed specifically for Go's map-based dynamic data.

## Documentation
- **[Why Queryfy?](https://github.com/ha1tch/queryfy/blob/main/doc/queryfy-why-EN.md)** - Understanding the problem Queryfy solves and why it matters
- **[Design Philosophy](https://github.com/ha1tch/queryfy/blob/main/doc/design-and-innovations_EN_v0.1.0_2025-06-13.md)** - Core design principles and innovations
- **[Comparison Guide](https://github.com/ha1tch/queryfy/blob/main/doc/COMPARISON.md)** - How Queryfy compares to go-playground/validator, gjson, and other libraries
- **[Philosophy: Compose at Build-Time](https://github.com/ha1tch/queryfy/blob/main/doc/philosophy-en-01.md)** - The fundamental principle behind Queryfy
- **[Philosophy: Practical Ramifications](https://github.com/ha1tch/queryfy/blob/main/doc/philosophy-en-02.md)** - Real-world implications and patterns

### En Español

- **[¿Por qué Queryfy?](https://github.com/ha1tch/queryfy/blob/main/doc/queryfy-why-ES.md)** - Entendiendo el problema que Queryfy resuelve
- **[Filosofía de Diseño](https://github.com/ha1tch/queryfy/blob/main/doc/design-and-innovations_ES_v0.1.0_2025-06-13.md)** - Principios de diseño e innovaciones
- **[Filosofía: Componer en Tiempo de Compilación](https://github.com/ha1tch/queryfy/blob/main/doc/philosophy-es-01.md)** - El principio fundamental
- **[Filosofía: Ramificaciones Prácticas](https://github.com/ha1tch/queryfy/blob/main/doc/philosophy-es-02.md)** - Implicaciones y patrones del mundo real

### Research & Background

- **[What Developers Need](https://github.com/ha1tch/queryfy/blob/main/doc/research-01-market-needs.md)** - Analysis of Go ecosystem validation needs
- **[How Queryfy Delivers](https://github.com/ha1tch/queryfy/blob/main/doc/research-02-how-qf-delivers.md)** - Mapping features to developer needs

## Installation

```bash
go get github.com/ha1tch/queryfy
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    
    qf "github.com/ha1tch/queryfy"
    "github.com/ha1tch/queryfy/builders"
)

func main() {
    // Define a schema
    schema := builders.Object().
        Field("customerId", builders.String().Required()).
        Field("amount", builders.Number().Min(0).Required()).
        Field("items", builders.Array().Of(
            builders.Object().
                Field("productId", builders.String().Required()).
                Field("quantity", builders.Number().Min(1)).
                Field("price", builders.Number().Min(0))
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
        log.Printf("Validation failed: %v\n", err)
        return
    }

    // Query
    firstPrice, _ := qf.Query(orderData, "items[0].price")
    fmt.Printf("First item price: $%.2f\n", firstPrice)
}
```

## Core Concepts

### Schema Definition

Define schemas using the fluent builder pattern:

```go
userSchema := builders.Object().
    Field("id", builders.String().Pattern("^[A-Z]{3}-[0-9]{6}$")).
    Field("email", builders.String().Email().Required()).
    Field("age", builders.Number().Min(0).Max(150)).
    Field("roles", builders.Array().Of(builders.String().Enum("admin", "user", "guest"))).
    Field("address", builders.Object().
        Field("street", builders.String().Required()).
        Field("city", builders.String().Required()).
        Field("zipCode", builders.String().Pattern("^[0-9]{5}$"))
    )
```

### Validation Modes

**Strict Mode** (Default): All fields must match the schema exactly. Extra fields cause validation errors.

```go
err := qf.Validate(data, schema) // Uses strict mode by default
```

**Loose Mode**: Allows extra fields and validates type compatibility. For example, the string "42" is considered valid for a number field.

```go
err := qf.ValidateWithMode(data, schema, qf.Loose)
```

**Note**: In v0.1.0, loose mode only validates type compatibility. It does not transform the actual data. The string "42" will validate as a number but remain a string in your data structure.

### Querying Data

Query using simple path expressions:

```go
// Simple field access
name, _ := qf.Query(data, "customer.firstName")

// Array access by index
firstItem, _ := qf.Query(data, "items[0]")

// Nested access
street, _ := qf.Query(data, "customer.address.street")

// Complex paths
price, _ := qf.Query(data, "items[0].product.price")
```

### Composite Schemas (AND/OR/NOT)

Create complex validation logic by combining schemas:

```go
// Email OR phone required
contactSchema := builders.Or(
    builders.String().Email(),
    builders.String().Pattern(`^\+?[1-9]\d{9,14}$`) // International phone
)

// Multiple conditions with AND
ageSchema := builders.And(
    builders.Number().Min(0),
    builders.Number().Max(150),
    builders.Number().Integer()
)

// NOT condition
nonEmptyString := builders.And(
    builders.String(),
    builders.Not(builders.String().Length(0))
)

// Use in object schema
schema := builders.Object().
    Field("contact", contactSchema.Required()).
    Field("age", ageSchema).
    Field("description", nonEmptyString)
```

## Advanced Usage

### Custom Validators

Create custom validation logic:

```go
phoneValidator := builders.Custom(func(value interface{}) error {
    str, ok := value.(string)
    if !ok {
        return fmt.Errorf("expected string, got %T", value)
    }
    if !isValidPhone(str) {
        return fmt.Errorf("invalid phone number: %s", str)
    }
    return nil
})

schema := builders.Object().
    Field("phone", phoneValidator.Required())
```

### Data Transformation

Transform data during validation using built-in or custom transformers:

```go
import "github.com/ha1tch/queryfy/builders/transformers"

// Use built-in transformers
emailSchema := builders.Transform(
    builders.String().Email().Required(),
).Add(transformers.Trim()).
  Add(transformers.Lowercase())

// Number transformations
priceSchema := builders.Transform(
    builders.Number().Min(0).Required(),
).Add(transformers.RemoveCurrencySymbols()).
  Add(transformers.ToFloat64()).
  Add(transformers.Round(2))

// Custom transformer
normalizePhone := func(value interface{}) (interface{}, error) {
    phone := value.(string)
    // Remove all non-digits
    return regexp.MustCompile(`\D`).ReplaceAllString(phone, ""), nil
}

phoneSchema := builders.Transform(
    builders.String().Pattern(`^\d{10}$`).Required(),
).Add(normalizePhone)

// Using ValidateAndTransform to get transformed values
ctx := qf.NewValidationContext(qf.Strict)
transformed, err := emailSchema.ValidateAndTransform(emailInput, ctx)
```

### DateTime Validation

Comprehensive date and time validation with multiple format support:

```go
// Date only validation
birthDateSchema := builders.DateTime().
    DateOnly().              // YYYY-MM-DD format
    Past().                  // Must be in the past
    Age(18, 100).           // Age between 18 and 100
    Required()

// Timestamp validation
createdAtSchema := builders.DateTime().
    ISO8601().              // Full ISO8601 format
    Required()

// Custom format
appointmentSchema := builders.DateTime().
    Format("2006-01-02 15:04").
    Future().               // Must be in the future
    BusinessDay().          // Monday-Friday only
    Between(businessStart, businessEnd)
```

### Dependent Field Validation

Validate fields based on the values of other fields:

```go
// Payment form with conditional fields
paymentSchema := builders.Object().WithDependencies().
    Field("paymentMethod", builders.String().
        Enum("credit_card", "paypal", "bank_transfer")).
    DependentField("cardNumber",
        builders.Dependent("cardNumber").
            When(builders.WhenEquals("paymentMethod", "credit_card")).
            Then(builders.String().Pattern(`^\d{16}$`).Required())).
    DependentField("paypalEmail",
        builders.Dependent("paypalEmail").
            When(builders.WhenEquals("paymentMethod", "paypal")).
            Then(builders.String().Email().Required())).
    DependentField("accountNumber",
        builders.Dependent("accountNumber").
            When(builders.WhenEquals("paymentMethod", "bank_transfer")).
            Then(builders.String().Required()))
```

### Schema Composition

Build reusable schema components:

```go
// Reusable address schema
addressSchema := builders.Object().
    Field("street", builders.String().Required()).
    Field("city", builders.String().Required()).
    Field("zipCode", builders.String().Pattern("^[0-9]{5}$"))

// Use in multiple places
customerSchema := builders.Object().
    Field("name", builders.String().Required()).
    Field("billingAddress", addressSchema.Required()).
    Field("shippingAddress", addressSchema)
```

### Pre-Marshal Validation

Ensure data is valid before JSON marshaling:

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
orderSchema := builders.Object().
    Field("orderId", builders.String().Required()).
    Field("customer", builders.Object().
        Field("email", builders.Transform(
            builders.String().Email().Required(),
        ).Add(transformers.Trim()).Add(transformers.Lowercase())).
        Field("phone", builders.String().Optional())
    ).Required().
    Field("payment", builders.Object().
        Field("method", builders.String().Enum("CARD", "CASH", "DIGITAL_WALLET")).
        Field("amount", builders.Number().Min(0).Required()).
        Field("currency", builders.String().Length(3).Required())
    ).Required().
    Field("items", builders.Array().MinItems(1).Of(
        builders.Object().
            Field("productId", builders.String().Required()).
            Field("quantity", builders.Number().Min(1).Integer().Required()).
            Field("price", builders.Transform(
                builders.Number().Min(0).Required(),
            ).Add(transformers.Round(2)))
    ).Required())

// Validate incoming order
if err := qf.Validate(orderData, orderSchema); err != nil {
    validationErr := err.(*qf.ValidationError)
    for _, fieldErr := range validationErr.Errors {
        log.Printf("Field %s: %s", fieldErr.Path, fieldErr.Message)
    }
    return err
}

// Query order data
customerEmail, _ := qf.Query(orderData, "customer.email")
totalAmount, _ := qf.Query(orderData, "payment.amount")
firstProductId, _ := qf.Query(orderData, "items[0].productId")
```

## Error Messages

Queryfy provides clear, actionable error messages with exact field paths:

```
validation failed:
  customer.email: must be a valid email address, got "not-an-email"
  items[0].quantity: must be >= 1, got 0
  items[2].productId: field is required
  payment.method: must be one of: CARD, CASH, DIGITAL_WALLET, got "CHECK"
```

## Performance

Queryfy is designed for production use:

- Validation schemas can be defined once and reused
- Query paths are cached after first use
- Minimal reflection through type-switch optimization
- No external dependencies

```go
// Create schema once
var orderSchema = builders.Object().
    Field("id", builders.String().Required()).
    Field("amount", builders.Number().Min(0))

// Reuse for multiple validations
for _, order := range orders {
    if err := qf.Validate(order, orderSchema); err != nil {
        // Handle error
    }
}
```

## Roadmap

### v0.1.0 (Released)
- [✓] Schema validation with builder API
- [✓] Basic path queries (dot notation, array indexing)
- [✓] Composite schemas (AND/OR/NOT)
- [✓] Strict and loose validation modes
- [✓] Custom validators
- [✓] Clear error messages with paths

### v0.2.0 (Current Release)
- [✓] Data transformation pipeline with builder pattern
- [✓] DateTime validation with comprehensive format support
- [✓] Dependent field validation for conditional requirements
- [✓] Phone normalization for multiple countries
- [✓] Built-in transformers (string, number, date operations)
- [✓] Transform chaining with `.Add()` method

### v0.3.0 (Planned)
- [ ] Data transformation in loose mode (modify actual data)
- [ ] Wildcard queries (`items[*].price`)
- [ ] Schema compilation for better performance
- [ ] Iteration methods (`Each`, `Collect`, `ValidateEach`)
- [ ] Enhanced transform API (direct method chaining)

### v0.4.0 (Future)
- [ ] Filter expressions (`items[?price > 100]`)
- [ ] Aggregation functions (`sum()`, `avg()`, `count()`)
- [ ] JSON Schema compatibility
- [ ] Struct conversion (`ToStruct`, `ValidateToStruct`)

## License

Copyright 2025 h@ual.fi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

<http://www.apache.org/licenses/LICENSE-2.0>

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
