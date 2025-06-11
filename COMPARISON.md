Here's a comprehensive table showing Queryfy side by side with comparable libraries:

| Feature | **Queryfy** | **go-playground/validator** | **gojsonschema** | **tidwall/gjson** |
|---------|------------|----------------------------|------------------|-------------------|
| **Primary Purpose** | Validation + Querying | Struct validation | JSON Schema validation | JSON querying |
| **Target Data Type** | `map[string]interface{}` | Go structs | JSON/maps | JSON strings |
| **Validation** | ✅ Full support | ✅ Full support | ✅ Full support | ❌ None |
| **Querying** | ✅ Dot notation + arrays | ❌ None | ❌ None | ✅ Advanced paths |
| **Schema Definition** | Fluent builder API | Struct tags | JSON Schema | N/A |
| **Type Safety** | ✅ Compile + runtime | ✅ Compile-time | ⚠️ Runtime only | ⚠️ Runtime only |
| **Error Messages** | ✅ Path-based, clear | ✅ Field-based | ✅ JSON pointer | N/A |
| **Composable Schemas** | ✅ AND/OR/NOT | ⚠️ Limited | ✅ Full JSON Schema | N/A |
| **Custom Validators** | ✅ Yes | ✅ Yes | ⚠️ Via regex/format | N/A |
| **Validation Modes** | ✅ Strict/Loose | ❌ Strict only | ⚠️ Depends on schema | N/A |
| **Type Coercion** | ✅ In loose mode | ❌ None | ⚠️ Limited | ✅ Automatic |
| **Dynamic Data** | ✅ Excellent | ❌ Poor | ✅ Good | ✅ Excellent |
| **Performance** | ✅ Good (cached queries) | ✅ Excellent | ⚠️ Moderate | ✅ Excellent |
| **Dependencies** | ✅ None | ⚠️ Several | ⚠️ Several | ✅ None |
| **Learning Curve** | ✅ Easy | ✅ Easy | ❌ Complex | ✅ Easy |
| **Schema Format** | Go code | Struct tags | JSON/YAML | N/A |
| **Array Operations** | ✅ Index access | ⚠️ Limited | ✅ Full | ✅ Advanced |
| **Nested Objects** | ✅ Full support | ✅ Full support | ✅ Full support | ✅ Full support |
| **Schema Reuse** | ✅ Via Go variables | ⚠️ Limited | ✅ Via $ref | N/A |
| **API Style** | Fluent/Builder | Declarative tags | Configuration | Function calls |

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

### **Code Style Examples**

**Queryfy**
```go
schema := builders.Object().
    Field("email", builders.String().Email().Required()).
    Field("age", builders.Number().Min(18))

err := qf.Validate(data, schema)
email, _ := qf.Query(data, "email")
```

**go-playground/validator**
```go
type User struct {
    Email string `validate:"required,email"`
    Age   int    `validate:"min=18"`
}
err := validator.Struct(user)
```

**gojsonschema**
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

**gjson**
```go
email := gjson.Get(jsonStr, "email").String()
age := gjson.Get(jsonStr, "age").Int()
// No validation
```

### **Key Differentiators**

- **Queryfy**: Only solution that combines validation + querying for dynamic data
- **validator**: Best for compile-time safety with Go structs
- **gojsonschema**: Best for JSON Schema standard compliance
- **gjson**: Fastest for pure JSON querying without validation
