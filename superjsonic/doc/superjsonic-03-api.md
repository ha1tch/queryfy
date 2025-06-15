# Queryfy with Superjsonic - Complete API Reference

## Table of Contents

1. [Package Overview](#package-overview)
2. [Core Types](#core-types)
3. [Main API](#main-api)
4. [Schema Builders](#schema-builders)
5. [Validation API](#validation-api)
6. [Query API](#query-api)
7. [Codec Interface](#codec-interface)
8. [Error Types](#error-types)
9. [Constants and Enums](#constants-and-enums)
10. [Examples](#examples)

---

## Package Overview

### Package queryfy

```go
import "github.com/yourusername/queryfy"
```

Queryfy provides high-performance JSON validation, querying, and transformation for Go. It uses Superjsonic internally for 5-8x faster validation than encoding/json while maintaining zero dependencies.

### Key Features
- Zero-allocation JSON validation
- Type-safe schema builders
- Pluggable JSON codec support
- Comprehensive error reporting
- Query support for validated data

### Basic Usage

```go
// Create a schema
schema := builders.Object().
    Field("name", builders.String().Required()).
    Field("age", builders.Number().Min(0).Max(150))

// Create validator
qf := queryfy.New()

// Validate JSON
data := []byte(`{"name": "John", "age": 30}`)
if err := qf.Validate(data, schema); err != nil {
    log.Fatal(err)
}

// Validate and unmarshal
var person Person
if err := qf.ValidateInto(data, schema, &person); err != nil {
    log.Fatal(err)
}
```

---

## Core Types

### Queryfy

The main type for validation operations.

```go
type Queryfy struct {
    // Private fields
}
```

#### Constructor

```go
func New() *Queryfy
```

Creates a new Queryfy instance with default configuration (using encoding/json codec).

**Example:**
```go
qf := queryfy.New()
```

### Schema

Interface representing a validation schema.

```go
type Schema interface {
    Validate(value interface{}) error
    GetType() SchemaType
    // Private methods
}
```

Schemas are created using builder functions rather than directly implementing this interface.

### JSONCodec

Interface for pluggable JSON marshaling/unmarshaling.

```go
type JSONCodec interface {
    Marshal(v interface{}) ([]byte, error)
    Unmarshal(data []byte, v interface{}) error
}
```

---

## Main API

### Queryfy Methods

#### Validate

```go
func (q *Queryfy) Validate(data []byte, schema Schema) error
```

Validates JSON data against a schema without unmarshaling. Uses Superjsonic for high-performance validation.

**Parameters:**
- `data`: Raw JSON bytes to validate
- `schema`: Schema to validate against

**Returns:**
- `error`: Validation error with detailed information, or nil if valid

**Performance:**
- Zero allocations
- 5-8x faster than standard JSON parsing
- Fail-fast on structural errors

**Example:**
```go
schema := builders.Object().
    Field("email", builders.String().Email())

err := qf.Validate([]byte(`{"email": "invalid-email"}`), schema)
// err: "email: must be a valid email address"
```

#### ValidateInto

```go
func (q *Queryfy) ValidateInto(data []byte, schema Schema, v interface{}) error
```

Validates JSON data and unmarshals it into the provided interface using the configured codec.

**Parameters:**
- `data`: Raw JSON bytes to validate
- `schema`: Schema to validate against
- `v`: Pointer to unmarshal into

**Returns:**
- `error`: Validation or unmarshal error, or nil if successful

**Example:**
```go
type User struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

schema := builders.Object().
    Field("name", builders.String().Required()).
    Field("email", builders.String().Email())

var user User
err := qf.ValidateInto(data, schema, &user)
```

#### WithCodec

```go
func (q *Queryfy) WithCodec(codec JSONCodec) *Queryfy
```

Returns a new Queryfy instance with the specified JSON codec.

**Parameters:**
- `codec`: JSON codec implementation

**Returns:**
- `*Queryfy`: New instance with specified codec

**Example:**
```go
import jsoniter "github.com/json-iterator/go"

qf := queryfy.New().WithCodec(jsoniter.ConfigFastest)
```

#### Query

```go
func (q *Queryfy) Query(data []byte, path string) (interface{}, error)
```

Extracts a value from JSON data using a path expression.

**Parameters:**
- `data`: Raw JSON bytes
- `path`: Query path (e.g., "user.address.city")

**Returns:**
- `interface{}`: Extracted value
- `error`: Query error or nil

**Example:**
```go
data := []byte(`{"user": {"name": "John", "age": 30}}`)
age, err := qf.Query(data, "user.age")
// age: float64(30)
```

#### Transform

```go
func (q *Queryfy) Transform(data []byte, schema Schema) ([]byte, error)
```

Validates and transforms JSON data according to schema rules.

**Parameters:**
- `data`: Raw JSON bytes
- `schema`: Schema with transformation rules

**Returns:**
- `[]byte`: Transformed JSON
- `error`: Validation/transformation error

**Example:**
```go
schema := builders.Object().
    Field("email", builders.String().
        Transform(transformers.Lowercase()).
        Transform(transformers.Trim()))

result, _ := qf.Transform([]byte(`{"email": " JOHN@EXAMPLE.COM "}`), schema)
// result: {"email": "john@example.com"}
```

---

## Schema Builders

### Package builders

```go
import "github.com/yourusername/queryfy/builders"
```

Type-safe schema builders for creating validation schemas.

### Object Schema

```go
func Object() *ObjectBuilder
```

Creates an object schema builder.

#### ObjectBuilder Methods

```go
// Add a required field
func (b *ObjectBuilder) Field(name string, schema Schema) *ObjectBuilder

// Add an optional field
func (b *ObjectBuilder) OptionalField(name string, schema Schema) *ObjectBuilder

// Add a dependent field
func (b *ObjectBuilder) DependentField(name string, dep DependentSchema) *ObjectBuilder

// Allow additional properties
func (b *ObjectBuilder) AdditionalProperties(allow bool) *ObjectBuilder

// Set required fields
func (b *ObjectBuilder) Required(fields ...string) *ObjectBuilder

// Add custom validation
func (b *ObjectBuilder) Custom(fn func(value interface{}) error) *ObjectBuilder
```

**Example:**
```go
userSchema := builders.Object().
    Field("id", builders.String().UUID()).
    Field("email", builders.String().Email()).
    OptionalField("age", builders.Number().Min(0).Max(150)).
    Required("id", "email")
```

### String Schema

```go
func String() *StringBuilder
```

Creates a string schema builder.

#### StringBuilder Methods

```go
// Length constraints
func (b *StringBuilder) MinLength(min int) *StringBuilder
func (b *StringBuilder) MaxLength(max int) *StringBuilder
func (b *StringBuilder) Length(exactly int) *StringBuilder

// Pattern matching
func (b *StringBuilder) Pattern(pattern string) *StringBuilder
func (b *StringBuilder) Email() *StringBuilder
func (b *StringBuilder) URL() *StringBuilder
func (b *StringBuilder) UUID() *StringBuilder
func (b *StringBuilder) DateTime() *StringBuilder

// Enumeration
func (b *StringBuilder) Enum(values ...string) *StringBuilder

// Transformation
func (b *StringBuilder) Transform(t Transformer) *StringBuilder

// Requirements
func (b *StringBuilder) Required() *StringBuilder
func (b *StringBuilder) Optional() *StringBuilder
```

**Example:**
```go
emailSchema := builders.String().
    Email().
    Required().
    Transform(transformers.Lowercase())
```

### Number Schema

```go
func Number() *NumberBuilder
```

Creates a number schema builder.

#### NumberBuilder Methods

```go
// Range constraints
func (b *NumberBuilder) Min(min float64) *NumberBuilder
func (b *NumberBuilder) Max(max float64) *NumberBuilder
func (b *NumberBuilder) ExclusiveMin(min float64) *NumberBuilder
func (b *NumberBuilder) ExclusiveMax(max float64) *NumberBuilder

// Type constraints
func (b *NumberBuilder) Integer() *NumberBuilder
func (b *NumberBuilder) MultipleOf(value float64) *NumberBuilder

// Requirements
func (b *NumberBuilder) Required() *NumberBuilder
func (b *NumberBuilder) Optional() *NumberBuilder
```

**Example:**
```go
ageSchema := builders.Number().
    Integer().
    Min(0).
    Max(150).
    Required()
```

### Boolean Schema

```go
func Boolean() *BooleanBuilder
```

Creates a boolean schema builder.

#### BooleanBuilder Methods

```go
func (b *BooleanBuilder) Required() *BooleanBuilder
func (b *BooleanBuilder) Optional() *BooleanBuilder
func (b *BooleanBuilder) Default(value bool) *BooleanBuilder
```

### Array Schema

```go
func Array() *ArrayBuilder
```

Creates an array schema builder.

#### ArrayBuilder Methods

```go
// Item constraints
func (b *ArrayBuilder) Items(schema Schema) *ArrayBuilder
func (b *ArrayBuilder) UniqueItems(unique bool) *ArrayBuilder

// Length constraints
func (b *ArrayBuilder) MinItems(min int) *ArrayBuilder
func (b *ArrayBuilder) MaxItems(max int) *ArrayBuilder

// Requirements
func (b *ArrayBuilder) Required() *ArrayBuilder
func (b *ArrayBuilder) Optional() *ArrayBuilder
```

**Example:**
```go
tagsSchema := builders.Array().
    Items(builders.String().MinLength(1)).
    UniqueItems(true).
    MinItems(1).
    MaxItems(10)
```

### DateTime Schema

```go
func DateTime() *DateTimeBuilder
```

Creates a datetime schema builder.

#### DateTimeBuilder Methods

```go
// Format constraints
func (b *DateTimeBuilder) Format(format string) *DateTimeBuilder
func (b *DateTimeBuilder) ISO8601() *DateTimeBuilder
func (b *DateTimeBuilder) RFC3339() *DateTimeBuilder

// Range constraints
func (b *DateTimeBuilder) After(time time.Time) *DateTimeBuilder
func (b *DateTimeBuilder) Before(time time.Time) *DateTimeBuilder
func (b *DateTimeBuilder) Between(start, end time.Time) *DateTimeBuilder

// Age constraints
func (b *DateTimeBuilder) Age(min, max int) *DateTimeBuilder
func (b *DateTimeBuilder) Future() *DateTimeBuilder
func (b *DateTimeBuilder) Past() *DateTimeBuilder

// Business rules
func (b *DateTimeBuilder) BusinessDay() *DateTimeBuilder
func (b *DateTimeBuilder) Weekday() *DateTimeBuilder
```

**Example:**
```go
birthDateSchema := builders.DateTime().
    ISO8601().
    Age(18, 100).
    Past()
```

### Dependent Schema

```go
func Dependent(fieldName string) *DependentBuilder
```

Creates a dependent field schema builder.

#### DependentBuilder Methods

```go
// Condition
func (b *DependentBuilder) When(condition Condition) *DependentBuilder

// Schema to apply
func (b *DependentBuilder) Then(schema Schema) *DependentBuilder
```

**Example:**
```go
builders.Dependent("cardNumber").
    When(builders.WhenEquals("paymentMethod", "credit_card")).
    Then(builders.String().Pattern(`^\d{16}$`).Required())
```

---

## Validation API

### ValidationError

```go
type ValidationError struct {
    Code    string              // Error code (e.g., "REQUIRED_FIELD")
    Message string              // Human-readable message
    Path    string              // JSON path to error (e.g., "users[0].email")
    Details map[string]string   // Additional error details
}

func (e *ValidationError) Error() string
```

### ValidationErrors

Container for multiple validation errors.

```go
type ValidationErrors struct {
    Errors []ValidationError
}

func (e *ValidationErrors) Error() string
func (e *ValidationErrors) Count() int
func (e *ValidationErrors) First() *ValidationError
```

### Common Validation Error Codes

- `REQUIRED_FIELD`: Required field is missing
- `TYPE_MISMATCH`: Value has wrong type
- `INVALID_FORMAT`: Format validation failed
- `OUT_OF_RANGE`: Numeric value outside allowed range
- `TOO_SHORT`: String/array shorter than minimum
- `TOO_LONG`: String/array longer than maximum
- `PATTERN_MISMATCH`: Regex pattern not matched
- `INVALID_EMAIL`: Email validation failed
- `INVALID_URL`: URL validation failed
- `INVALID_UUID`: UUID format invalid
- `CUSTOM_VALIDATION_FAILED`: Custom validator returned error

---

## Query API

### Query Syntax

Queryfy supports a simple path-based query syntax:

- **Field access**: `user.name`
- **Array index**: `users[0]`
- **Nested access**: `users[0].address.city`

### Query Methods

#### Query

```go
func (q *Queryfy) Query(data []byte, path string) (interface{}, error)
```

Extract single value.

#### QueryString

```go
func (q *Queryfy) QueryString(data []byte, path string) (string, error)
```

Extract value as string.

#### QueryNumber

```go
func (q *Queryfy) QueryNumber(data []byte, path string) (float64, error)
```

Extract value as number.

#### QueryBool

```go
func (q *Queryfy) QueryBool(data []byte, path string) (bool, error)
```

Extract value as boolean.

#### QueryArray

```go
func (q *Queryfy) QueryArray(data []byte, path string) ([]interface{}, error)
```

Extract value as array.

#### QueryObject

```go
func (q *Queryfy) QueryObject(data []byte, path string) (map[string]interface{}, error)
```

Extract value as object.

---

## Codec Interface

### JSONCodec

```go
type JSONCodec interface {
    Marshal(v interface{}) ([]byte, error)
    Unmarshal(data []byte, v interface{}) error
}
```

### Built-in Codecs

#### DefaultCodec

```go
type DefaultCodec struct{}
```

Uses encoding/json (standard library).

### Third-Party Codec Examples

#### JsoniterCodec

```go
import jsoniter "github.com/json-iterator/go"

type JsoniterCodec struct {
    api jsoniter.API
}

func NewJsoniterCodec() JsoniterCodec {
    return JsoniterCodec{
        api: jsoniter.ConfigFastest,
    }
}

func (c JsoniterCodec) Marshal(v interface{}) ([]byte, error) {
    return c.api.Marshal(v)
}

func (c JsoniterCodec) Unmarshal(data []byte, v interface{}) error {
    return c.api.Unmarshal(data, v)
}
```

#### SonicCodec

```go
import "github.com/bytedance/sonic"

type SonicCodec struct{}

func (SonicCodec) Marshal(v interface{}) ([]byte, error) {
    return sonic.Marshal(v)
}

func (SonicCodec) Unmarshal(data []byte, v interface{}) error {
    return sonic.Unmarshal(data, v)
}
```

---

## Error Types

### ParseError

```go
type ParseError struct {
    Message string
    Offset  int
    Line    int
    Column  int
}

func (e *ParseError) Error() string
```

Indicates JSON parsing failed.

### QueryError

```go
type QueryError struct {
    Path    string
    Message string
}

func (e *QueryError) Error() string
```

Indicates query execution failed.

### SchemaError

```go
type SchemaError struct {
    Message string
}

func (e *SchemaError) Error() string
```

Indicates schema definition error.

---

## Constants and Enums

### SchemaType

```go
type SchemaType int

const (
    TypeObject SchemaType = iota
    TypeArray
    TypeString
    TypeNumber
    TypeBoolean
    TypeNull
    TypeDateTime
)
```

### JSONQuality (Future)

```go
type JSONQuality int

const (
    JSONQualityFresh  JSONQuality = iota  // Definitely valid
    JSONQualityMild                       // Likely valid
    JSONQualityFishy                      // Possibly corrupted
    JSONQualityRotten                     // Definitely corrupted
)
```

---

## Examples

### Basic Validation

```go
package main

import (
    "log"
    "github.com/yourusername/queryfy"
    "github.com/yourusername/queryfy/builders"
)

func main() {
    // Define schema
    schema := builders.Object().
        Field("name", builders.String().MinLength(1).Required()).
        Field("age", builders.Number().Min(0).Max(150).Required()).
        Field("email", builders.String().Email().Required())
    
    // Create validator
    qf := queryfy.New()
    
    // Validate data
    data := []byte(`{
        "name": "John Doe",
        "age": 30,
        "email": "john@example.com"
    }`)
    
    if err := qf.Validate(data, schema); err != nil {
        log.Fatal(err)
    }
    
    log.Println("Valid!")
}
```

### Using Custom Codec

```go
import jsoniter "github.com/json-iterator/go"

// Create codec wrapper
type FastCodec struct{}

func (FastCodec) Marshal(v interface{}) ([]byte, error) {
    return jsoniter.ConfigFastest.Marshal(v)
}

func (FastCodec) Unmarshal(data []byte, v interface{}) error {
    return jsoniter.ConfigFastest.Unmarshal(data, v)
}

// Use it
qf := queryfy.New().WithCodec(FastCodec{})
```

### Complex Schema with Dependencies

```go
schema := builders.Object().
    Field("paymentMethod", builders.String().Enum("card", "paypal", "bank")).
    DependentField("cardNumber",
        builders.Dependent("cardNumber").
            When(builders.WhenEquals("paymentMethod", "card")).
            Then(builders.String().Pattern(`^\d{16}$`).Required())).
    DependentField("paypalEmail",
        builders.Dependent("paypalEmail").
            When(builders.WhenEquals("paymentMethod", "paypal")).
            Then(builders.String().Email().Required()))
```

### Querying Validated Data

```go
// First validate
if err := qf.Validate(data, schema); err != nil {
    return err
}

// Then query specific fields
userName, _ := qf.QueryString(data, "user.name")
userAge, _ := qf.QueryNumber(data, "user.age")
firstTag, _ := qf.QueryString(data, "tags[0]")
```

### Error Handling

```go
err := qf.Validate(data, schema)
if err != nil {
    switch e := err.(type) {
    case *queryfy.ValidationErrors:
        // Multiple validation errors
        for _, ve := range e.Errors {
            log.Printf("Error at %s: %s", ve.Path, ve.Message)
        }
    case *queryfy.ParseError:
        // JSON parsing failed
        log.Printf("Parse error at line %d: %s", e.Line, e.Message)
    default:
        // Other error
        log.Printf("Error: %v", err)
    }
}
```

### Transformation Pipeline

```go
import "github.com/yourusername/queryfy/transformers"

schema := builders.Object().
    Field("email", builders.String().
        Email().
        Transform(transformers.Lowercase()).
        Transform(transformers.Trim())).
    Field("name", builders.String().
        Transform(transformers.TrimSpace()).
        Transform(transformers.TitleCase())).
    Field("tags", builders.Array().
        Items(builders.String().
            Transform(transformers.Lowercase())))

transformed, err := qf.Transform(data, schema)
```

---

## Performance Notes

### Validation Performance

- **Validate()**: Uses Superjsonic, 5-8x faster than encoding/json
- **ValidateInto()**: Validation speed + codec unmarshal time
- **Query()**: Direct access without full unmarshal

### Memory Usage

- **Zero allocations** during validation
- Parser pooling for concurrent usage
- Optimized for large arrays

### Best Practices

1. **Reuse Queryfy instances** - They're safe for concurrent use
2. **Use Validate() when possible** - Faster than ValidateInto()
3. **Choose appropriate codec** - Based on your performance needs
4. **Pre-compile schemas** - Build schemas once, reuse many times

---

## Version Information

- **Current Version**: 1.0.0
- **Minimum Go Version**: 1.18
- **Dependencies**: None (zero-dependency library)

---

*Last Updated: [Current Date]*