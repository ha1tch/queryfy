# Queryfy vs Other Go Libraries

## Feature Comparison Table

| Feature | **Queryfy** | **go-playground/validator** | **gojsonschema** | **tidwall/gjson** |
|---------|------------|----------------------------|------------------|-------------------|
| **Primary Purpose** | Validation + Querying | Struct validation | JSON Schema validation | JSON querying |
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
| **Dynamic Data** | ✅ Excellent | ⚠️ Different API¹ | ✅ Good | ✅ Excellent |
| **Performance** | ✅ Good (cached queries) | ✅ Excellent² | ⚠️ Moderate | ✅ Excellent |
| **Dependencies** | ✅ None | ⚠️ Several | ⚠️ Several | ✅ None |
| **Learning Curve** | ✅ Easy | ✅ Easy (structs) / ⚠️ Moderate (maps) | ❌ Complex | ✅ Easy |
| **Schema Format** | Go code | Struct tags / Rule maps | JSON/YAML | N/A |
| **Array Operations** | ✅ Index access | ⚠️ Limited | ✅ Full | ✅ Advanced |
| **Nested Objects** | ✅ Full support | ✅ Full support | ✅ Full support | ✅ Full support |
| **Schema Reuse** | ✅ Via Go variables | ⚠️ Limited | ✅ Via $ref | N/A |
| **API Style** | Fluent builder for maps | Struct tags / Rule maps | Configuration | Function calls |

¹ go-playground/validator supports map validation through ValidateMap, but requires defining rules in a separate map structure  
² Performance comparison shown for struct validation; map validation performance may differ

## Detailed Comparison

### **Use Case Fit**

| Use Case | Best Choice | Why |
|----------|-------------|-----|
| **JSON API validation** | Queryfy | Handles dynamic data + can query validated data |
| **Go struct validation** | go-playground/validator | Purpose-built for structs |
| **OpenAPI/Swagger schemas** | gojsonschema | Industry standard format |
| **Fast JSON extraction** | gjson | Optimized for read-only queries |
| **Config file validation** | Queryfy | Dynamic data + can query after validation |
| **Form validation** | Queryfy or validator | Depends if data is structured or dynamic |
| **Map-based validation** | Queryfy | Fluent API designed for maps |

### **Code Style Examples**

**Queryfy** - Fluent builder optimized for maps
```go
schema := builders.Object().
    Field("email", builders.String().Email().Required()).
    Field("age", builders.Number().Min(18))

err := qf.Validate(data, schema)
email, _ := qf.Query(data, "email")
```

**go-playground/validator** - Struct tags (primary use case)
```go
type User struct {
    Email string `validate:"required,email"`
    Age   int    `validate:"min=18"`
}
err := validator.Struct(user)
```

**go-playground/validator** - Map validation (secondary use case)
```go
rules := map[string]interface{}{
    "email": "required,email",
    "age": "min=18",
}
errs := validator.ValidateMap(data, rules)
```

**gojsonschema** - JSON Schema standard
```json
{
    "type": "object",
    "properties": {
        "email": {"type": "string", "format": "email"},
        "age": {"type": "number", "minimum": 18}
    },
    "required": ["email"]
}
```

**gjson** - Query-only
```go
email := gjson.Get(jsonStr, "email").String()
age := gjson.Get(jsonStr, "age").Int()
// No validation
```

### **Key Differentiators**

- **Queryfy**: Only solution that combines validation + querying for dynamic map-based data with a fluent API
- **validator**: Best for compile-time safety with Go structs; can handle maps but with a different approach
- **gojsonschema**: Best for JSON Schema standard compliance and interoperability
- **gjson**: Fastest for pure JSON querying without validation

### **When to Choose Queryfy**

Choose Queryfy when you need:
- ✅ To validate AND query `map[string]interface{}` data
- ✅ A fluent, intuitive API for defining schemas in Go code
- ✅ Type coercion in loose mode for flexible input handling
- ✅ Clear error messages with exact field paths
- ✅ Zero external dependencies
- ✅ Composable validation logic (AND/OR/NOT)

### **When to Choose Alternatives**

- **go-playground/validator**: When working primarily with Go structs and you want compile-time type safety
- **gojsonschema**: When you need JSON Schema compatibility or are working with OpenAPI/Swagger
- **gjson**: When you only need fast querying without validation

## Type Safety Clarification

**Queryfy** provides compile-time type safety at the schema definition level:
```go
// ✅ Compile-time safety: wrong method won't compile
schema := builders.String().Min(5)  // Compile error: Min not available on String

// ✅ Compile-time safety: wrong parameter type won't compile  
schema := builders.Number().Min("five")  // Compile error: wrong type

// ✅ Type-safe schema composition
schema := builders.Object().
    Field("age", builders.Number().Min(0)).  // Correct types enforced
    Field("email", builders.String().Email())
```

The runtime aspect only applies to the actual data validation (as with any map[string]interface{} data).

## Summary

Queryfy fills a specific gap in the Go ecosystem: elegant validation and querying of dynamic, map-based data. While other libraries can handle some aspects of this use case, Queryfy provides the best developer experience for this specific need through its purpose-built API design and compile-time safe schema definitions.