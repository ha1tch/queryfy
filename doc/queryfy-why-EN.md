# Queryfy? Why?

## The Hidden Problem in Every Go Service

Go developers face a fundamental tension. While Go's type system excels at compile-time safety, real-world applications must handle dynamic data from APIs, webhooks, databases, and configuration files. This creates a philosophical split that runs through most Go codebases.

### The Two-Personality Codebase

Every Go web service contains two distinct approaches to data handling:

**The struct world** - What we show in code reviews:
```go
type User struct {
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"min=18"`
}
```

**The dynamic world** - What actually handles production data:
```go
func HandleWebhook(payload map[string]interface{}) error {
    // Pages of type assertions and manual validation
    if user, ok := payload["user"].(map[string]interface{}); ok {
        if email, ok := user["email"].(string); ok {
            // More nested checking...
        }
    }
}
```

This isn't just about code style. It represents a false dichotomy the Go community has internalized: structs are "proper Go" while `map[string]interface{}` is a "necessary evil."

## Nobody Writes Struct-Only Services

Here's the reality: no production Go service is purely struct-based. Every service handles:
- Webhooks with unknown structure
- Flexible API query parameters
- GraphQL variables
- Multi-tenant configurations
- External API responses

Yet our tooling assumes the struct-only world. We end up with:
```go
// go-playground/validator for structs
// + manual validation for dynamic parts
// + gjson for querying nested data
// + mapstructure for type conversion
// + 200 lines of glue code
// + inconsistent error handling
// = Your current "solution"
```

## Queryfy's Fundamental Insight

Queryfy recognizes that **safety isn't in the data structure, it's in how we describe what we expect**. A schema is just as type-safe as a struct definition:

```go
// This is as safe as a struct definition
userSchema := builders.Object().
    Field("email", builders.String().Email().Required()).
    Field("age", builders.Number().Min(18).Max(120))

// And works uniformly across all data types
err := qf.Validate(structData, userSchema)    // ✓
err := qf.Validate(mapData, userSchema)       // ✓
err := qf.Validate(jsonBytes, userSchema)     // ✓
```

## The Unexplored: Compile-Time Safety for Dynamic Data

### Build-Time Type Safety

Queryfy brings Go's compile-time guarantees to dynamic validation through its builder API:

```go
// These are compile-time errors
schema := builders.Number().Email()        // ❌ Won't compile
schema := builders.String().Min(5).Email() // ❌ Email() not available after Min()

// The IDE guides you to valid compositions
schema := builders.String().Email().Required() // ✓ Full autocomplete support
```

This isn't just convenience—it's a fundamental shift. You get the same safety building validation rules that you get defining structs.

### The IDE Experience

Unlike string-based validation tags, every Queryfy method is:
- **Discoverable**: Type `builders.` and see all options
- **Contextual**: After `.String()`, only string methods appear
- **Documented**: Inline documentation for every method
- **Refactorable**: Rename fields across your entire codebase safely

## Dissolving the False Dichotomy

Queryfy shows that the struct vs. dynamic split was never necessary. You don't need two mental models, two validation approaches, or two sets of error handling.

### One Mental Model for Everything

Instead of switching between "struct mode" and "dynamic mode":

```go
// Before: Different approaches for different data
func validateStruct(u User) error { 
    return validator.Validate(u) 
}

func validateMap(data map[string]interface{}) error {
    // 50 lines of manual validation
}

// After: One approach for all data
func validate(data interface{}) error {
    return qf.Validate(data, userSchema)
}
```

### Progressive Data Refinement

Queryfy provides a clear pipeline from unknown to known:

```go
// 1. Receive unknown data
rawData := receiveWebhook()

// 2. Validate structure
if err := qf.Validate(rawData, schema); err != nil {
    return err
}

// 3. Transform and clean
cleaned, _ := qf.ValidateAndTransform(rawData, schema)

// 4. Query specific values (no type assertions!)
email, _ := qf.Query(cleaned, "user.email")

// 5. Convert to struct when needed
user, _ := qf.ToStructT[User](cleaned)
```

Each step is explicit, type-safe, and testable.

## The Transformation Innovation

Queryfy doesn't just validate—it helps fix common data issues:

```go
// Define validation AND transformation together
emailSchema := builders.Transform(
    builders.String().Email()
).Add(transformers.Trim()).
  Add(transformers.Lowercase())

// "  John.Doe@EXAMPLE.COM  " → "john.doe@example.com"
```

This solves real problems:
- APIs that send numbers as strings
- Phone numbers in various formats
- Inconsistent date formats
- Currency values with symbols

The transformation pipeline is:
- **Auditable**: Every transformation is recorded
- **Composable**: Chain simple operations
- **Testable**: Pure functions with no side effects

## Practical Features for Real-World Messiness

### Strict vs. Loose Modes

Queryfy acknowledges that data isn't always perfect:

```go
// Strict mode: For internal services
err := qf.Validate(data, schema)

// Loose mode: For external APIs
err := qf.ValidateWithMode(data, schema, qf.Loose)
// "42" validates as number 42
// Extra fields are ignored
```

### Dependent Field Validation

Real forms have complex relationships:

```go
paymentSchema := builders.Object().WithDependencies().
    Field("method", builders.String().Enum("card", "paypal")).
    DependentField("cardNumber",
        builders.Dependent("cardNumber").
            When(builders.WhenEquals("method", "card")).
            Then(builders.String().Required().Pattern(`^\d{16}$`)))
```

### Custom Business Logic

Embed complex validation directly:

```go
.Custom(func(value interface{}) error {
    order := value.(map[string]interface{})
    items := order["items"].([]interface{})
    total := order["total"].(float64)
    
    calculatedTotal := calculateItemsTotal(items)
    if math.Abs(total - calculatedTotal) > 0.01 {
        return fmt.Errorf("total doesn't match items sum")
    }
    return nil
})
```

## Where Queryfy Shines

### API Gateways
Different validation per route, handling multiple upstream formats:
```go
routeSchemas := map[string]queryfy.Schema{
    "/v1/users":  userSchemaV1,
    "/v2/users":  userSchemaV2,
    "/webhooks":  webhookSchema,
}
```

### Multi-Tenant B2B
Different validation rules per customer:
```go
customerSchemas := map[string]queryfy.Schema{
    "enterprise-customer": strictSchema,
    "startup-customer":    lenientSchema,
}
```

### GraphQL Servers
Variables are inherently dynamic:
```go
func validateVariables(query string, variables map[string]interface{}) error {
    schema := getSchemaForQuery(query)
    return qf.Validate(variables, schema)
}
```

## The Philosophy: Compose at Build-Time, Validate at Run-Time

This principle has profound implications:

1. **All validation behaviors are known at compile time**
2. **Runtime only selects between pre-composed behaviors**
3. **No dynamic schema generation from user input**
4. **Every possible validation can be tested**

This enables:
- Blue-green deployments with validation changes
- A/B testing of validation rules
- Progressive rollout of stricter validation
- Complete audit trails

## Performance and Implementation Details

### Zero-Allocation Optimizations
- Pre-allocated path slices for typical nesting
- String builders instead of concatenation
- Type switches to avoid reflection where possible
- Query path compilation and caching

### Error Messages as Documentation
Every error is actionable:
- `"must be a valid email address"` not `"validation failed"`
- `"length must be at least 8, got 5"` not `"invalid length"`
- `"must be one of: admin, user, guest"` shows valid options

## The Bottom Line

Queryfy isn't just another validation library. It's a reconciliation between two parts of Go development that have been unnecessarily at odds. It shows that:

1. **Dynamic data handling can be as safe as struct handling**
2. **You don't need different mental models for different data types**
3. **Validation can help fix data, not just reject it**
4. **Complex business rules can be expressed declaratively**

By bringing compile-time safety to runtime validation, Queryfy makes working with `map[string]interface{}` as pleasant and safe as working with structs. It's not about choosing between flexibility and safety—it's about having both.

## Integration Path

Queryfy complements existing code:

1. Keep using struct tags for pure struct validation
2. Use Queryfy for dynamic data handling
3. Share schemas between services as Go packages
4. Gradually migrate validation logic as needed

The result is cleaner code, fewer bugs, consistent error handling, and confidence when dealing with external data. Most importantly, it eliminates the guilt and complexity around dynamic data handling, making it a first-class citizen in your Go applications.
