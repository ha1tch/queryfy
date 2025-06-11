# README.custom-validators.md
> queryfy/examples/custom-validators/main.go

### Queryfy Custom Validators Example

This example explores using custom validators with Queryfy for complex validation scenarios that often arise in real applications.

## Why Custom Validators?

When working with dynamic data from APIs, configuration files, or user input, validation requirements can quickly become complex. This example demonstrates how custom validators can help address these challenges.

## Notable Features

### 1. **Business Logic in Validators**
Custom validators can encapsulate domain-specific rules:

```go
// Pricing rules example
priceConsistencyValidator := func(value interface{}) error {
    // Check margin requirements
    if regularPrice < costPrice * 1.20 {
        return fmt.Errorf("regular price must be at least 20%% above cost")
    }
    // Additional pricing logic...
}
```

### 2. **Combining Validators**
Multiple validation rules can be composed together:

```go
// Example: username validation
userSchema := builders.Object().
    Field("username", builders.And(
        builders.String().MinLength(3).MaxLength(20),
        builders.String().Pattern(`^[a-zA-Z0-9_\-\[\]{}]+$`),
        usernameValidator, // Additional custom rules
    ))
```

### 3. **Cross-Field Validation**
Validators can examine relationships between fields:

```go
// Example: validating order totals against shipping methods
orderValidator := func(value interface{}) error {
    order := value.(map[string]interface{})
    items := order["items"].([]interface{})
    shipping := order["shipping"].(map[string]interface{})
    
    // Apply rules based on multiple fields
}
```

### 4. **Conditional Rules**
Different validation rules can apply based on context:

```go
envValidator := func(value interface{}) error {
    config := value.(map[string]interface{})
    env := config["environment"].(string)
    
    switch env {
    case "development":
        // Development-specific checks
    case "production":
        // Production-specific checks
    }
}
```

### 5. **Descriptive Error Messages**
Custom validators can provide specific feedback:

```go
return fmt.Errorf("password is too weak (strength: %d/5)", strength)
return fmt.Errorf("minimum order amount is $10.00 (current: $%.2f)", total)
```

## Comparison with Other Approaches

### Struct Tags
```go
// Struct tags handle basic constraints
type User struct {
    Email string `validate:"required,email"`
    Age   int    `validate:"min=18,max=120"`
}
```

Custom validators extend this by allowing complex business logic, external validations, and detailed error messages.

### Manual Validation
```go
// Traditional approach: validation logic spread across functions
func ValidateOrder(order Order) error {
    if order.Total < 10 {
        return errors.New("order too small")
    }
    // More checks...
}
```

With Queryfy, validation rules can be organized in schemas alongside the data structure definition.

## Example Scenarios

This example includes four scenarios that demonstrate different validation patterns:

1. **User Registration**
   - Username restrictions and format validation
   - Password strength requirements
   - Email domain validation
   - Birth date verification

2. **Product Listings**
   - SKU format enforcement
   - Price relationship validation
   - Image requirements
   - Content quality checks

3. **Order Processing**
   - Payment method validation
   - Address format verification
   - Business rule enforcement
   - Order total calculations

4. **Configuration Management**
   - Environment-specific validation
   - Connection string formats
   - Security settings
   - Feature flag types

## Considerations

When using custom validators:

1. **Keep It Simple**: Each validator should focus on one concern
2. **Error Messages**: Provide enough detail to help users understand what went wrong
3. **Type Safety**: Check types before casting to avoid panics
4. **Testing**: Custom validators are functions that can be tested independently
5. **Performance**: Be mindful of expensive operations in validators that run frequently

## Running the Example

```bash
go run main.go
```

The example shows various validation scenarios with both valid and invalid data to demonstrate how the validators work in practice.

## Summary

Custom validators with Queryfy offer one approach to handling complex validation requirements. They're particularly useful when:
- Validation logic involves multiple fields
- Rules change based on context
- Business logic is complex
- Clear error messages are important

This pattern may help when dealing with dynamic data where traditional validation approaches become cumbersome. As with any pattern, it's worth considering whether the added flexibility is needed for your specific use case.
