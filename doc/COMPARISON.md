# Queryfy vs Other Go Libraries

## Feature Comparison Table

| Feature | **Queryfy** | **go-playground/validator** | **gojsonschema** | **tidwall/gjson** |
|---------|------------|----------------------------|------------------|-------------------|
| **Primary Purpose** | Validation + Querying + Transformation | Struct validation | JSON Schema validation | JSON querying |
| **Target Data Type** | `map[string]interface{}` | Go structs | JSON/maps | JSON strings |
| **Validation** | ✅ Full support | ✅ Full support | ✅ Full support | ❌ None |
| **Querying** | ✅ Dot notation + arrays | ❌ None | ❌ None | ✅ Advanced paths |
| **Schema Definition** | Fluent builder API | Struct tags | JSON Schema | N/A |
| **Type Safety** | ✅ Compile-time (schema) / ⚠️ Runtime (data) | ✅ Compile-time (schema+data for structs) / ⚠️ Runtime (data for maps) | ⚠️ Runtime only | ⚠️ Runtime only |
| **Error Messages** | ✅ Path-based, clear | ✅ Field-based | ✅ JSON pointer | N/A |
| **Composable Schemas** | ✅ AND/OR/NOT | ⚠️ Limited | ✅ Full JSON Schema | N/A |
| **Custom Validators** | ✅ Yes | ✅ Yes | ⚠️ Via regex/format | N/A |
| **Validation Modes** | ✅ Strict/Loose | ❌ Strict only | ⚠️ Depends on schema | N/A |
| **Type Coercion** | ✅ In loose mode | ❌ None | ⚠️ Limited | ✅ Automatic |
| **Data Transformation** | ✅ Transform pipeline | ❌ None | ❌ None | ❌ None |
| **DateTime Validation** | ✅ Comprehensive | ⚠️ Basic via tags | ⚠️ Via format | ❌ None |
| **Dependent Fields** | ✅ Native support | ⚠️ Via custom¹ | ✅ Via dependencies | ❌ None |
| **Dynamic Data** | ✅ Excellent | ⚠️ Different API² | ✅ Good | ✅ Excellent |
| **Performance** | ✅ Good (cached queries) | ✅ Excellent³ | ⚠️ Moderate | ✅ Excellent |
| **Dependencies** | ✅ None | ⚠️ Several | ⚠️ Several | ✅ None |
| **Learning Curve** | ✅ Easy | ✅ Easy (structs) / ⚠️ Moderate (maps) | ❌ Complex | ✅ Easy |
| **Schema Format** | Go code | Struct tags / Rule maps | JSON/YAML | N/A |
| **Array Operations** | ✅ Index access | ⚠️ Limited | ✅ Full | ✅ Advanced |
| **Nested Objects** | ✅ Full support | ✅ Full support | ✅ Full support | ✅ Full support |
| **Schema Reuse** | ✅ Via Go variables | ⚠️ Limited | ✅ Via $ref | N/A |
| **API Style** | Fluent builder for maps | Struct tags / Rule maps | Configuration | Function calls |

¹ go-playground/validator supports dependent fields through custom validators but lacks native conditional validation  
² go-playground/validator supports map validation through ValidateMap, but requires defining rules in a separate map structure  
³ Performance comparison shown for struct validation; map validation performance may differ

## Detailed Comparison

### **Use Case Fit**

| Use Case | Best Choice | Why |
|----------|-------------|-----|
| **JSON API validation** | Queryfy | Handles dynamic data + can query + transform |
| **Go struct validation** | go-playground/validator | Purpose-built for structs |
| **OpenAPI/Swagger schemas** | gojsonschema | Industry standard format |
| **Fast JSON extraction** | gjson | Optimized for read-only queries |
| **Config file validation** | Queryfy | Dynamic data + can query after validation |
| **Form validation** | Queryfy or validator | Depends if data is structured or dynamic |
| **Map-based validation** | Queryfy | Fluent API designed for maps |
| **Data cleaning/normalization** | Queryfy | Built-in transformation pipeline |
| **Conditional validation** | Queryfy | Native dependent field support |

### **Code Style Examples**

**Queryfy** - Fluent builder optimized for maps with transformations
```go
schema := builders.Object().
    Field("email", builders.String().
        Transform(transformers.Lowercase()).
        Transform(transformers.Trim()).
        Email().Required()).
    Field("age", builders.Number().Min(0).Max(150)).
    Field("birthDate", builders.DateTime().
        DateOnly().
        Past().
        Age(18, 100)).
    DependentField("parentConsent",
        builders.Dependent("parentConsent").
            When(builders.WhenLessThan("age", 18)).
            Then(builders.Bool().Required()))
```

**go-playground/validator** - Struct tags
```go
type User struct {
    Email    string    `validate:"required,email"`
    Age      int       `validate:"min=0,max=150"`
    BirthDate time.Time `validate:"required,ltefield=Now"`
}
```

**gojsonschema** - JSON Schema
```json
{
  "type": "object",
  "properties": {
    "email": {"type": "string", "format": "email"},
    "age": {"type": "integer", "minimum": 0, "maximum": 150},
    "birthDate": {"type": "string", "format": "date"}
  },
  "required": ["email"]
}
```

**gjson** - Query only
```go
email := gjson.Get(jsonStr, "user.email").String()
```

## Feature Details

### **Transformation Pipeline (Queryfy only)**

Queryfy uniquely offers data transformation during validation:

```go
// Transform messy input data
priceSchema := builders.String().
    Transform(transformers.RemoveCurrencySymbols()).
    Transform(transformers.ToFloat64()).
    Transform(transformers.Round(2)).
    Min(0)

// Phone normalization
phoneSchema := builders.String().
    Transform(transformers.NormalizePhone("US")).
    Pattern(`^\+1\d{10}$`)
```

### **DateTime Validation Comparison**

| Feature | Queryfy | validator | gojsonschema |
|---------|---------|-----------|--------------|
| Date formats | ✅ Multiple + custom | ⚠️ Via tags | ✅ RFC3339 |
| Age validation | ✅ Built-in | ❌ Custom needed | ❌ None |
| Business days | ✅ Built-in | ❌ Custom needed | ❌ None |
| Time ranges | ✅ Between() | ⚠️ Via tags | ⚠️ min/max |
| Relative dates | ✅ Past()/Future() | ⚠️ Limited | ❌ None |

### **Dependent Field Validation**

Queryfy provides native support for conditional validation:

```go
// Queryfy - Native dependent fields
paymentSchema := builders.Object().
    Field("method", builders.String().Enum("card", "paypal")).
    DependentField("cardNumber",
        builders.Dependent("cardNumber").
            When(builders.WhenEquals("method", "card")).
            Then(builders.String().Required()))
```

vs go-playground/validator (requires custom validator):

```go
// validator - Custom function needed
type Payment struct {
    Method     string `validate:"required,oneof=card paypal"`
    CardNumber string `validate:"required_if=Method card"`
}
```

## Migration Guide

### From go-playground/validator
```go
// Before (struct tags)
type User struct {
    Email string `validate:"required,email"`
}

// After (Queryfy)
schema := builders.Object().
    Field("email", builders.String().Email().Required())
```

### From gojsonschema
```go
// Before (JSON Schema)
schema := `{"properties": {"email": {"format": "email"}}}`

// After (Queryfy)
schema := builders.Object().
    Field("email", builders.String().Email())
```

## When to Choose Each Library

### Choose **Queryfy** when:
- Working with `map[string]interface{}` from JSON APIs
- Need to validate AND query the same data
- Want to transform/clean data during validation
- Need conditional/dependent field validation
- Prefer compile-time safe schema definitions
- Want a single library for validation + querying + transformation

### Choose **go-playground/validator** when:
- Primarily validating Go structs
- Already using struct tags extensively
- Need the absolute fastest performance
- Working with static data structures

### Choose **gojsonschema** when:
- Must use JSON Schema standard
- Sharing schemas across languages
- Integrating with OpenAPI/Swagger
- Need schema versioning/evolution

### Choose **gjson** when:
- Only need fast JSON queries
- No validation required
- Working with large JSON documents
- Read-only operations

## Performance Considerations

- **Queryfy**: Good performance with query caching. Transform operations add minimal overhead.
- **go-playground/validator**: Excellent for structs, uses caching and reflection optimization
- **gojsonschema**: Moderate, JSON Schema parsing overhead
- **gjson**: Excellent, optimized for fast path queries

## Summary

Queryfy fills a unique niche in the Go ecosystem by providing:
1. **Unified validation + querying + transformation** for dynamic data
2. **Compile-time safe** schema definitions with runtime flexibility
3. **Native support** for data transformation pipelines
4. **Built-in** datetime and dependent field validation
5. **Zero dependencies** while offering comprehensive features

While other libraries excel in their specific domains, Queryfy is the best choice when working with dynamic `map[string]interface{}` data that needs validation, querying, and transformation in a single, cohesive package.