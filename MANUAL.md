# Queryfy Manual

Complete usage guide for queryfy v0.3.0. For an overview and quick start,
see [README.md](README.md).

## Table of Contents

- [Schema Definition](#schema-definition)
- [Validation Modes](#validation-modes)
- [Nullable and Optional](#nullable-and-optional)
- [Querying Data](#querying-data)
- [Wildcard Queries](#wildcard-queries)
- [Iteration Methods](#iteration-methods)
- [Low-Level Query API](#low-level-query-api)
- [Composite Schemas](#composite-schemas)
- [Custom Validators](#custom-validators)
- [Data Transformation](#data-transformation)
- [Transform Convenience Methods](#transform-convenience-methods)
- [Built-In Transformers](#built-in-transformers)
- [ValidateAndTransform](#validateandtransform)
- [Reusable Validator](#reusable-validator)
- [DateTime Validation](#datetime-validation)
- [Dependent Field Validation](#dependent-field-validation)
- [Error Handling](#error-handling)
- [Schema Composition](#schema-composition)
- [Schema Compilation](#schema-compilation)
- [Schema Introspection](#schema-introspection)
- [Custom Format Registry](#custom-format-registry)
- [Type Metadata](#type-metadata)
- [Schema Equality and Diff](#schema-equality-and-diff)
- [Field Walker](#field-walker)
- [JSON Schema Interoperability](#json-schema-interoperability)
- [Async Validation](#async-validation)

---

## Schema Definition

Define schemas using the fluent builder pattern. All builders are in the
`builders` package.

```go
import "github.com/ha1tch/queryfy/builders"

userSchema := builders.Object().
    Field("id", builders.String().Pattern(`^[A-Z]{3}-[0-9]{6}$`)).
    Field("email", builders.String().Email().Required()).
    Field("age", builders.Number().Min(0).Max(150)).
    Field("active", builders.Bool().Required()).
    Field("roles", builders.Array().Of(
        builders.String().Enum("admin", "user", "guest"),
    )).
    Field("address", builders.Object().
        Field("street", builders.String().Required()).
        Field("city", builders.String().Required()).
        Field("zipCode", builders.String().Pattern(`^[0-9]{5}$`)),
    )
```

Available schema types:

| Builder | Type | Key methods |
|---------|------|-------------|
| `builders.String()` | string | `MinLength`, `MaxLength`, `Length`, `Pattern`, `Email`, `URL`, `UUID`, `Enum` |
| `builders.Number()` | number | `Min`, `Max`, `Range`, `MultipleOf`, `Integer`, `Positive`, `Negative` |
| `builders.Bool()` | boolean | |
| `builders.Object()` | object | `Field`, `Fields`, `RequiredFields`, `AllowAdditional` |
| `builders.Array()` | array | `Of`, `MinItems`, `MaxItems`, `Length`, `UniqueItems` |
| `builders.DateTime()` | datetime | `ISO8601`, `DateOnly`, `YMD`, `DMY`, `MDY`, `Format`, `Past`, `Future`, `Age`, `Between`, `BusinessDay`, `StrictFormat` |

All schema types support `.Required()`, `.Optional()`, `.Nullable()`, and
`.Custom(fn)` for adding custom validator functions.

## Validation Modes

**Strict mode** (default): all fields must match the schema exactly. Extra
fields cause validation errors.

```go
err := qf.Validate(data, schema) // strict mode
```

**Loose mode**: allows extra fields and validates type compatibility. The
string `"42"` is considered valid for a number field, but the data is not
modified — it remains a string in the map.

```go
err := qf.ValidateWithMode(data, schema, qf.Loose)
```

**Controllable additional properties**: you can override the mode-based
default on a per-object basis:

```go
// Reject extra fields even in loose mode
strict := builders.Object().
    Field("name", builders.String()).
    AllowAdditional(false)

// Allow extra fields even in strict mode
flexible := builders.Object().
    Field("name", builders.String()).
    AllowAdditional(true)
```

## Nullable and Optional

These compose independently:

| Combination | Absent field | `null` value | Wrong type |
|-------------|-------------|-------------|------------|
| (default) | ✓ pass | ✗ fail | ✗ fail |
| `.Required()` | ✗ fail | ✗ fail | ✗ fail |
| `.Nullable()` | ✓ pass | ✓ pass | ✗ fail |
| `.Required().Nullable()` | ✗ fail | ✓ pass | ✗ fail |

```go
// Field must be present but may be null
builders.String().Required().Nullable()

// Field may be absent; if present, must not be null
builders.String().Optional()
```

## Querying Data

Query using path expressions with dot notation and array indexing:

```go
name, _ := qf.Query(data, "customer.firstName")
firstItem, _ := qf.Query(data, "items[0]")
street, _ := qf.Query(data, "customer.address.street")
price, _ := qf.Query(data, "items[0].product.price")
```

## Wildcard Queries

Wildcard `[*]` expands across all elements in an array:

```go
// Get all prices
prices, _ := qf.Query(data, "items[*].price")
// Returns: []interface{}{75.25, 49.99, 120.00}

// Nested wildcards
totals, _ := qf.Query(data, "customers[*].orders[*].total")
```

## Iteration Methods

```go
// Execute a function for each matched element
qf.Each(data, "items[*]", func(index int, value interface{}) error {
    item := value.(map[string]interface{})
    fmt.Printf("Item %d: %s\n", index, item["name"])
    return nil
})

// Collect transformed results
names, _ := qf.Collect(data, "items[*].name", func(value interface{}) (interface{}, error) {
    return strings.ToUpper(value.(string)), nil
})

// Validate each matched element against a schema
err := qf.ValidateEach(data, "items[*]", itemSchema, qf.Strict)
```

## Low-Level Query API

For direct access to the query engine:

```go
import "github.com/ha1tch/queryfy/query"

// Execute without caching
result, err := query.Execute(data, "items[0].price")

// Execute with path caching (faster for repeated queries)
result, err := query.ExecuteCached(data, "items[0].price")

// Clear the path cache
query.ClearCache()
```

## Composite Schemas

Combine schemas with boolean logic:

```go
// Email OR phone required
contactSchema := builders.Or(
    builders.String().Email(),
    builders.String().Pattern(`^\+?[1-9]\d{9,14}$`),
)

// Multiple conditions
ageSchema := builders.And(
    builders.Number().Min(0),
    builders.Number().Max(150),
    builders.Number().Integer(),
)

// NOT condition
nonEmptyString := builders.And(
    builders.String(),
    builders.Not(builders.String().Length(0)),
)
```

## Custom Validators

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

## Data Transformation

Transform data during validation using the `Transform` wrapper and `.Add()`:

```go
import "github.com/ha1tch/queryfy/builders/transformers"

emailSchema := builders.Transform(
    builders.String().Email().Required(),
).Add(transformers.Trim()).
  Add(transformers.Lowercase())

priceSchema := builders.Transform(
    builders.Number().Min(0).Required(),
).Add(transformers.RemoveCurrencySymbols()).
  Add(transformers.ToFloat64()).
  Add(transformers.Round(2))

// Custom transformer function
normalizePhone := func(value interface{}) (interface{}, error) {
    phone := value.(string)
    return regexp.MustCompile(`\D`).ReplaceAllString(phone, ""), nil
}

phoneSchema := builders.Transform(
    builders.String().Pattern(`^\d{10}$`).Required(),
).Add(normalizePhone)
```

### Transform Convenience Methods

All leaf schema types have a `.Transform()` convenience that wraps the schema
in a `TransformSchema` and adds the first transformer in one call:

```go
// These are equivalent:
builders.Transform(builders.String().Email()).Add(transformers.Trim())
builders.String().Email().Transform(transformers.Trim())

// Chain more transformers with .Add()
builders.String().Email().
    Transform(transformers.Trim()).
    Add(transformers.Lowercase())
```

Available on: `StringSchema`, `NumberSchema`, `BoolSchema`, `ArraySchema`,
`ObjectSchema`, `DateTimeSchema`.

### Built-In Transformers

All transformers are in `builders/transformers`.

**String transformers:**

| Transformer | Description |
|-------------|-------------|
| `Trim()` | Remove leading/trailing whitespace |
| `Lowercase()` | Convert to lowercase |
| `Uppercase()` | Convert to uppercase |
| `TitleCase()` | Capitalise first letter of each word |
| `RemoveSpaces()` | Remove all spaces |
| `NormalizeWhitespace()` | Collapse runs of whitespace to single spaces |
| `Replace(old, new)` | Replace all occurrences |
| `RemoveNonAlphanumeric()` | Strip non-alphanumeric characters |
| `Truncate(maxLength)` | Truncate to maximum length |
| `PadLeft(minLength, padChar)` | Pad on the left to minimum length |

**Number transformers:**

| Transformer | Description |
|-------------|-------------|
| `ToFloat64()` | Convert numeric strings to float64 |
| `ToInt()` | Convert to integer |
| `Round(decimals)` | Round to N decimal places |
| `Clamp(min, max)` | Clamp value to range |
| `Percentage()` | Multiply by 100 |
| `FromPercentage()` | Divide by 100 |

**Date transformers:**

| Transformer | Description |
|-------------|-------------|
| `ParseDate(format)` | Parse string to time using Go format |
| `ToISO8601()` | Format as ISO 8601 |
| `DateFormat(from, to)` | Convert between date formats |
| `ToTimezone(location)` | Convert to timezone |
| `StartOfDay()` | Set time to 00:00:00 |
| `EndOfDay()` | Set time to 23:59:59 |

**Phone transformers:**

| Transformer | Description |
|-------------|-------------|
| `NormalizePhone(defaultCountry)` | Normalize with country detection |
| `NormalizePhoneWithCountry(country)` | Normalize for a specific country |
| `FormatPhone(country)` | Format in national style |

**General transformers:**

| Transformer | Description |
|-------------|-------------|
| `ToString()` | Convert any value to string |
| `ToBoolean()` | Convert truthy/falsy values to bool |
| `Default(value)` | Provide a default for nil values |
| `RemoveCurrencySymbols()` | Strip $, €, £, ¥, etc. |
| `Chain(transformers...)` | Compose multiple transformers |
| `Conditional(predicate, transformer)` | Apply transformer only when predicate is true |

### ValidateAndTransform

To get the transformed result after validation:

```go
// On a TransformSchema directly
ctx := qf.NewValidationContext(qf.Strict)
transformed, err := emailSchema.ValidateAndTransform(emailInput, ctx)

// Package-level convenience (works with any schema)
transformed, err := qf.ValidateAndTransform(data, schema, qf.Strict)

// Async variant with context
transformed, err := qf.ValidateAndTransformAsync(goCtx, data, schema, qf.Strict)
```

### Reusable Validator

For repeated validations with the same schema:

```go
v := qf.NewValidator(orderSchema)

err := v.Validate(order1)
err = v.Validate(order2)

// With mode
err = v.ValidateWithMode(order3, qf.Loose)
```

## DateTime Validation

```go
// Date only (YYYY-MM-DD)
birthDate := builders.DateTime().
    DateOnly().
    Past().
    Age(18, 100).
    Required()

// Full ISO 8601 timestamp
createdAt := builders.DateTime().
    ISO8601().
    Required()

// Custom format with constraints
appointment := builders.DateTime().
    Format("2006-01-02 15:04").
    Future().
    BusinessDay().
    Between(businessStart, businessEnd)

// Strict format enforcement (rejects partial matches)
strict := builders.DateTime().
    ISO8601().
    StrictFormat()
```

## Dependent Field Validation

Validate fields conditionally based on other field values:

```go
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

Available conditions: `WhenEquals`, `WhenNotEquals`, `WhenExists`,
`WhenNotExists`, `WhenIn`, `WhenGreaterThan`, `WhenLessThan`, `WhenTrue`,
`WhenFalse`. Combine with `WhenAll` (AND) and `WhenAny` (OR).

Shortcuts: `RequiredWhen(condition, schema)` and
`RequiredUnless(condition, schema)`.

## Error Handling

Validation errors are returned as `*ValidationError` containing a slice of
`FieldError` values:

```go
err := qf.Validate(data, schema)
if err != nil {
    validationErr := err.(*qf.ValidationError)
    for _, fe := range validationErr.Errors {
        fmt.Printf("  %s: %s (got: %v)\n", fe.Path, fe.Message, fe.Value)
    }
}
```

Create errors programmatically in custom validators:

```go
qf.NewFieldError("user.email", "domain not allowed", value)
qf.NewValidationError(fieldErr1, fieldErr2)
qf.WrapError(err, "user.preferences")
```

`MustValidate` panics on failure (useful in tests and initialization):

```go
qf.MustValidate(config, configSchema)
```

## Schema Composition

Build reusable schema components:

```go
addressSchema := builders.Object().
    Field("street", builders.String().Required()).
    Field("city", builders.String().Required()).
    Field("zipCode", builders.String().Pattern(`^[0-9]{5}$`))

customerSchema := builders.Object().
    Field("name", builders.String().Required()).
    Field("billingAddress", addressSchema.Required()).
    Field("shippingAddress", addressSchema)
```

## Schema Compilation

`Compile()` pre-processes a schema into an optimised form. See
[README.md](README.md#performance) for benchmark results.

```go
// Compile once at init time
var schema = qf.Compile(builders.Object().
    Field("id", builders.String().Required()).
    Field("amount", builders.Number().Min(0)))

// Use normally — same Schema interface
err := qf.Validate(data, schema)

// Access the original schema if needed
compiled := schema.(*qf.CompiledSchema)
original := compiled.Inner()
```

`ValidationContext` can be reused across validations with `Reset()` to avoid
allocation overhead:

```go
ctx := qf.NewValidationContext(qf.Strict)
for _, item := range items {
    ctx.Reset()
    schema.Validate(item, ctx)
    if ctx.HasErrors() {
        // handle
    }
}
```

## Schema Introspection

Read schema constraints programmatically:

```go
obj := builders.Object().
    Field("name", builders.String().MinLength(1).MaxLength(100).Required()).
    Field("age", builders.Number().Min(0).Max(150))

// Field access
nameSchema, ok := obj.GetField("name")
fields := obj.FieldNames()           // []string{"name", "age"}
required := obj.RequiredFieldNames()  // []string{"name"}

// String constraints
str := nameSchema.(*builders.StringSchema)
min, max := str.LengthConstraints()  // *int(1), *int(100)
pattern := str.PatternString()       // "" (no pattern set)
format := str.FormatType()           // "" (no format set)
enums := str.EnumValues()            // nil (no enum set)

// Number constraints
num, _ := obj.GetField("age")
n := num.(*builders.NumberSchema)
lo, hi := n.RangeConstraints()       // *float64(0), *float64(150)
mult := n.MultipleOfValue()          // nil
isInt := n.IsInteger()               // false

// Array constraints
arr := builders.Array().Of(builders.String()).MinItems(1).MaxItems(10).UniqueItems()
elem := arr.ElementSchema()          // *builders.StringSchema
min, max := arr.ItemCountConstraints() // *int(1), *int(10)
unique := arr.IsUniqueItems()        // true

// Object constraints
allow, explicit := obj.AllowsAdditional() // true, false (mode default)
```

## Custom Format Registry

Register custom string formats that integrate with JSON Schema import:

```go
// Register a format validator
builders.RegisterFormat("credit-card", func(value interface{}) error {
    str := value.(string)
    if !luhnCheck(str) {
        return fmt.Errorf("invalid credit card number")
    }
    return nil
})

// Use in schemas via Custom() with a lookup
cardValidator := builders.LookupFormat("credit-card")
cardSchema := builders.String().Custom(cardValidator)

// List all registered formats
allFormats := builders.RegisteredFormats()
```

Registered formats are automatically recognised during JSON Schema import
when a `"format"` keyword is encountered. For example, importing a JSON Schema
with `"format": "credit-card"` will apply the registered validator.

## Type Metadata

Attach arbitrary metadata to schemas. Metadata round-trips through JSON Schema
export/import via the `x-queryfy-meta` extension:

```go
schema := builders.String().
    Meta("displayName", "Email Address").
    Meta("helpText", "Enter your work email")

// Read metadata
name, ok := schema.GetMeta("displayName") // "Email Address", true

// Enumerate all metadata
allMeta := schema.AllMeta() // map[string]interface{}{...}
```

Available on all schema types: `StringSchema`, `NumberSchema`, `BoolSchema`,
`ObjectSchema`, `ArraySchema`, `DateTimeSchema`.

## Schema Equality and Diff

Compare schemas structurally:

```go
// Structural equality (ignores metadata)
equal := builders.Equal(schemaA, schemaB)

// Canonical hash for indexing
hash := builders.Hash(schema)

// Detailed diff between schema versions
diff, err := builders.Diff(oldSchema, newSchema)
if diff != nil && diff.HasChanges() {
    for _, path := range diff.Added {
        fmt.Printf("added: %s\n", path)
    }
    for _, path := range diff.Removed {
        fmt.Printf("removed: %s\n", path)
    }
    for _, change := range diff.Changed {
        fmt.Printf("changed: %s (%s -> %s) %s\n",
            change.Path, change.OldType, change.NewType, change.Details)
    }
}
```

## Field Walker

Recursively traverse all fields in a schema tree:

```go
builders.Walk(schema, func(path string, s qf.Schema) error {
    fmt.Printf("%-30s %s\n", path, s.Type())
    return nil
})
```

Output for a nested object:

```
name                           string
age                            number
address                        object
address.street                 string
address.city                   string
```

## JSON Schema Interoperability

Import JSON Schema documents as queryfy builders and export queryfy schemas
as JSON Schema. Supports a subset of Draft 2020-12 / Draft 7.

For full details, see the
[JSON Schema Interoperability Guide](https://github.com/ha1tch/queryfy/blob/main/doc/jsonschema-interop-EN.md).

```go
import "github.com/ha1tch/queryfy/builders/jsonschema"

// Import
schema, errors := jsonschema.FromJSON(jsonBytes, nil)
if len(errors) > 0 {
    // Non-fatal conversion warnings (unsupported keywords, etc.)
}

// Export
jsonBytes, err := jsonschema.ToJSON(schema, nil)

// Export to map (for further manipulation)
schemaMap := jsonschema.ToMap(schema, nil)
```

Round-trip verification: import a JSON Schema, export it, re-import the output,
and verify structural equality with `builders.Equal()`.

## Async Validation

For validators that need I/O (database lookups, API calls):

```go
schema := builders.Object().
    Field("email", builders.String().Email().Required()).
    AsyncCustom(func(ctx context.Context, value interface{}) error {
        obj := value.(map[string]interface{})
        email := obj["email"].(string)
        exists, err := checkEmailInDB(ctx, email)
        if err != nil {
            return err
        }
        if exists {
            return fmt.Errorf("email already registered")
        }
        return nil
    })

// Validate with context (supports cancellation and timeouts)
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

transformed, err := qf.ValidateAndTransformAsync(ctx, data, schema, qf.Strict)
```

Async validators run after synchronous validation passes. Context cancellation
propagates through all async validators at both field and object level.
