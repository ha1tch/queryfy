# JSON Schema Interoperability

Queryfy can import JSON Schema documents into its native builder representation
and export queryfy schemas back to JSON Schema. This enables integration with
systems that already define their contracts in JSON Schema — API gateways,
schema registries, OpenAPI specifications, form generators — without requiring
those systems to adopt queryfy's builder API.

## Installation

The JSON Schema package is included with queryfy. No additional dependencies
are required.

```go
import "github.com/ha1tch/queryfy/builders/jsonschema"
```

## Importing JSON Schema

Use `FromJSON` to convert a JSON Schema document into a queryfy schema:

```go
schemaJSON := []byte(`{
    "type": "object",
    "properties": {
        "email": {"type": "string", "format": "email"},
        "age": {"type": "integer", "minimum": 13}
    },
    "required": ["email"]
}`)

schema, errs := jsonschema.FromJSON(schemaJSON, nil)
if len(errs) > 0 {
    // Handle conversion errors (see Error Handling below)
}

// Use it like any queryfy schema
err := queryfy.Validate(data, schema)
```

The returned schema is a standard queryfy schema composed of the same builders
you would use when writing schemas by hand. An imported `"type": "string"` with
`"minLength": 5` is identical to `builders.String().MinLength(5)`.

### Import Options

```go
schema, errs := jsonschema.FromJSON(data, &jsonschema.Options{
    StrictMode:   true,  // Error on unsupported features (default: false)
    StoreUnknown: true,  // Preserve unrecognised keywords as metadata (default: false)
})
```

**StrictMode** controls how unsupported JSON Schema features are handled. When
false (the default), unsupported features produce warnings in the error slice
but import continues. When true, unsupported features produce errors. In both
cases, the returned error slice contains every issue encountered, so callers
can inspect and decide.

**StoreUnknown** captures unrecognised keywords (such as `x-` extensions) as
schema metadata via queryfy's `Meta` API. This is useful for round-tripping:
extensions survive import, are accessible via `GetMeta`, and are emitted on
export when `IncludeMeta` is enabled.

## Exporting to JSON Schema

Use `ToJSON` to convert a queryfy schema to a JSON Schema document:

```go
schema := builders.Object().
    Field("name", builders.String().Required().MinLength(1)).
    Field("score", builders.Number().Integer().Min(0).Max(100))

data, err := jsonschema.ToJSON(schema, nil)
// data is valid JSON Schema
```

For programmatic manipulation before serialisation, use `ToMap`:

```go
m := jsonschema.ToMap(schema, nil)
m["description"] = "Player profile"
data, _ := json.MarshalIndent(m, "", "  ")
```

### Export Options

```go
data, err := jsonschema.ToJSON(schema, &jsonschema.ExportOptions{
    SchemaURI:   "https://json-schema.org/draft/2020-12/schema",
    ID:          "https://example.com/my-schema.json",
    IncludeMeta: true,
})
```

**SchemaURI** sets the `$schema` keyword in the output. Omit or leave empty
to exclude it.

**ID** sets the `$id` keyword. Omit or leave empty to exclude it.

**IncludeMeta** includes metadata stored on queryfy schemas (via `Meta`) as
extension keywords in the output. This is the export counterpart of
`StoreUnknown` on import.

## Round-tripping

Import and export are designed to be complementary. A JSON Schema document
can be imported, exported, and re-imported with structural fidelity:

```go
// Import
schema1, _ := jsonschema.FromJSON(originalJSON, nil)

// Export
exported, _ := jsonschema.ToJSON(schema1, nil)

// Re-import
schema2, _ := jsonschema.FromJSON(exported, nil)

// Both schemas validate identically
```

This is verified by round-trip tests in the test suite. The guarantee applies
to the supported JSON Schema subset; unsupported features (see below) are
either skipped or rejected at import time and therefore do not participate in
round-tripping.

Custom extensions (`x-` keywords) survive round-tripping when `StoreUnknown`
is enabled on import and `IncludeMeta` is enabled on export.

## Compiling Imported Schemas

Imported schemas can be compiled like any other queryfy schema:

```go
schema, _ := jsonschema.FromJSON(data, nil)
compiled := queryfy.Compile(schema)

// Compiled schema validates with a flat function-call chain
err := queryfy.Validate(input, compiled)
```

This is particularly useful when an imported schema is validated against many
inputs — the compilation step flattens all constraint checks into a
pre-resolved slice of function calls, eliminating per-validation branching on
which constraints are configured.

## Supported JSON Schema Features

### Types

| JSON Schema type | queryfy builder |
|---|---|
| `"string"` | `builders.String()` |
| `"number"` | `builders.Number()` |
| `"integer"` | `builders.Number().Integer()` |
| `"boolean"` | `builders.Bool()` |
| `"object"` | `builders.Object()` |
| `"array"` | `builders.Array()` |

### String constraints

| Keyword | Builder method |
|---|---|
| `minLength` | `.MinLength()` |
| `maxLength` | `.MaxLength()` |
| `pattern` | `.Pattern()` |
| `enum` | `.Enum()` |
| `format: "email"` | `.Email()` |
| `format: "uri"` | `.URL()` |
| `format: "uuid"` | `.UUID()` |
| `format: "date-time"` | Stored as metadata |
| `format: "date"` | Stored as metadata |
| Custom formats | `.FormatString()` if registered, otherwise metadata |

### Number constraints

| Keyword | Builder method |
|---|---|
| `minimum` | `.Min()` |
| `maximum` | `.Max()` |
| `exclusiveMinimum` | `.Min()` + metadata for exact value |
| `exclusiveMaximum` | `.Max()` + metadata for exact value |
| `multipleOf` | `.MultipleOf()` |

### Object keywords

| Keyword | Builder method |
|---|---|
| `properties` | `.Field()` for each property |
| `required` | `.Required()` on individual field schemas |
| `additionalProperties: false` | `.AllowAdditional(false)` |
| `additionalProperties: true` | `.AllowAdditional(true)` |
| `additionalProperties: {schema}` | `.AllowAdditional(true)` + warning |

### Array keywords

| Keyword | Builder method |
|---|---|
| `items` | `.Of()` |
| `minItems` | `.MinItems()` |
| `maxItems` | `.MaxItems()` |
| `uniqueItems` | `.UniqueItems()` |

### Other

| Keyword | Handling |
|---|---|
| `nullable: true` | `.Nullable()` (OpenAPI 3.0 style) |
| `type: ["string", "null"]` | `.Nullable()` (JSON Schema style) |
| `$schema`, `$id`, `$comment` | Recognised and ignored (no warning) |
| `title`, `description` | Recognised and ignored |
| `default`, `examples`, `const` | Recognised and ignored |

### Type inference

When `type` is omitted, the importer infers the type from context:

- If `properties` is present, the type is inferred as `"object"`.
- If `items` is present, the type is inferred as `"array"`.
- Otherwise, an error is produced.

## Unsupported JSON Schema Features

The following keywords are explicitly unsupported. In default mode, they
produce warnings and are skipped. In strict mode, they produce errors.

| Keyword | Reason |
|---|---|
| `$ref`, `$defs`, `definitions` | Schema references would require a resolution engine and cycle detection. Use queryfy builders to compose schemas directly. |
| `oneOf`, `anyOf`, `allOf`, `not` | Composite schemas map to queryfy's `Or`, `And`, `Not` builders, but the semantics differ in edge cases. Use queryfy's composite builders directly for precise control. |
| `if`, `then`, `else` | Conditional schemas have no direct queryfy equivalent. Use `builders.Dependent()` for conditional field requirements. |
| `dependentRequired`, `dependentSchemas` | Use `builders.Dependent()` directly. |
| `patternProperties` | No queryfy equivalent. Define fields explicitly. |
| `unevaluatedProperties`, `unevaluatedItems` | These require tracking which properties were "evaluated" across composed schemas — not applicable without `allOf`/`oneOf`. |
| `prefixItems` | Tuple validation is not supported. Use `items` for homogeneous arrays. |
| `contains` | Use queryfy's `Each` or `ValidateEach` for element-level checks. |
| `additionalItems` | Use `items` instead. |

This subset covers the features that appear in the vast majority of real-world
JSON Schema documents. The unsupported features are primarily composition
mechanisms (`$ref`, `allOf`, `oneOf`) and advanced validation constructs
(`if`/`then`/`else`, `patternProperties`) that represent a small fraction of
usage but a large fraction of implementation complexity.

## Error Handling

`FromJSON` returns a flat slice of `ConversionError` values alongside the
schema. Each error describes a single issue with its location in the document:

```go
schema, errs := jsonschema.FromJSON(data, nil)
for _, e := range errs {
    fmt.Println(e.Path)      // "properties.address.properties.city"
    fmt.Println(e.Keyword)   // "$ref"
    fmt.Println(e.Message)   // "schema references are not supported (skipped)"
    fmt.Println(e.IsWarning) // true (in default mode)
}
```

The error path uses dot-separated notation matching the JSON Schema document
structure.

A schema is always returned, even when errors are present. In default mode,
the schema represents the best-effort conversion of the supported parts. In
strict mode, the schema is still usable but may be incomplete where unsupported
features were encountered. Callers should check the error slice and decide
whether to proceed based on their requirements.

## Example: API Gateway Integration

A common use case is validating incoming requests against JSON Schema
definitions that are already maintained as part of an API specification:

```go
// Load schema from file, database, or schema registry
schemaBytes, err := os.ReadFile("schemas/create-user.json")
if err != nil {
    log.Fatal(err)
}

// Import once at startup
schema, errs := jsonschema.FromJSON(schemaBytes, &jsonschema.Options{
    StrictMode: true,
})
if len(errs) > 0 {
    log.Fatalf("schema import failed: %v", errs)
}

// Compile for repeated use
compiled := queryfy.Compile(schema)

// Validate each request
func handleCreateUser(w http.ResponseWriter, r *http.Request) {
    var body map[string]interface{}
    json.NewDecoder(r.Body).Decode(&body)

    if err := queryfy.Validate(body, compiled); err != nil {
        http.Error(w, err.Error(), 400)
        return
    }
    // proceed with validated data
}
```

## Example: Schema Evolution

Export can be used to snapshot a queryfy schema as JSON Schema for versioning
or documentation:

```go
// Your queryfy schema (source of truth)
schema := builders.Object().
    Field("id", builders.String().Required().Pattern("^[a-z0-9-]+$")).
    Field("name", builders.String().Required().MinLength(1).MaxLength(200)).
    Field("email", builders.String().Email()).
    Field("created_at", builders.String())

// Export for documentation or schema registry
data, _ := jsonschema.ToJSON(schema, &jsonschema.ExportOptions{
    SchemaURI: "https://json-schema.org/draft/2020-12/schema",
    ID:        "https://api.example.com/schemas/user/v3",
})

os.WriteFile("schemas/user-v3.json", data, 0644)
```
