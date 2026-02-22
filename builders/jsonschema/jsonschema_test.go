package jsonschema_test

import (
	"testing"

	"github.com/ha1tch/queryfy"
	"github.com/ha1tch/queryfy/builders/jsonschema"
)

// ======================================================================
// Basic type mapping
// ======================================================================

func TestFromJSON_String(t *testing.T) {
	schema, errs := jsonschema.FromJSON([]byte(`{
		"type": "string",
		"minLength": 1,
		"maxLength": 100
	}`), nil)
	assertNoErrors(t, errs)
	assertValid(t, schema, "hello")
	assertInvalid(t, schema, "")
	assertInvalid(t, schema, 42.0)
}

func TestFromJSON_StringEmail(t *testing.T) {
	schema, errs := jsonschema.FromJSON([]byte(`{
		"type": "string",
		"format": "email"
	}`), nil)
	assertNoErrors(t, errs)
	assertValid(t, schema, "user@example.com")
	assertInvalid(t, schema, "not-an-email")
}

func TestFromJSON_StringURL(t *testing.T) {
	schema, errs := jsonschema.FromJSON([]byte(`{
		"type": "string",
		"format": "uri"
	}`), nil)
	assertNoErrors(t, errs)
	assertValid(t, schema, "https://example.com")
}

func TestFromJSON_StringUUID(t *testing.T) {
	schema, errs := jsonschema.FromJSON([]byte(`{
		"type": "string",
		"format": "uuid"
	}`), nil)
	assertNoErrors(t, errs)
	assertValid(t, schema, "550e8400-e29b-41d4-a716-446655440000")
}

func TestFromJSON_StringPattern(t *testing.T) {
	schema, errs := jsonschema.FromJSON([]byte(`{
		"type": "string",
		"pattern": "^[A-Z]{2}[0-9]{4}$"
	}`), nil)
	assertNoErrors(t, errs)
	assertValid(t, schema, "AB1234")
	assertInvalid(t, schema, "ab1234")
	assertInvalid(t, schema, "ABC123")
}

func TestFromJSON_StringEnum(t *testing.T) {
	schema, errs := jsonschema.FromJSON([]byte(`{
		"type": "string",
		"enum": ["red", "green", "blue"]
	}`), nil)
	assertNoErrors(t, errs)
	assertValid(t, schema, "red")
	assertValid(t, schema, "blue")
	assertInvalid(t, schema, "yellow")
}

func TestFromJSON_Number(t *testing.T) {
	schema, errs := jsonschema.FromJSON([]byte(`{
		"type": "number",
		"minimum": 0,
		"maximum": 100
	}`), nil)
	assertNoErrors(t, errs)
	assertValid(t, schema, 50.0)
	assertValid(t, schema, 0.0)
	assertValid(t, schema, 100.0)
	assertInvalid(t, schema, -1.0)
	assertInvalid(t, schema, 101.0)
}

func TestFromJSON_NumberMultipleOf(t *testing.T) {
	schema, errs := jsonschema.FromJSON([]byte(`{
		"type": "number",
		"multipleOf": 5
	}`), nil)
	assertNoErrors(t, errs)
	assertValid(t, schema, 10.0)
	assertValid(t, schema, 25.0)
	assertInvalid(t, schema, 7.0)
}

func TestFromJSON_Integer(t *testing.T) {
	schema, errs := jsonschema.FromJSON([]byte(`{
		"type": "integer",
		"minimum": 1,
		"maximum": 10
	}`), nil)
	assertNoErrors(t, errs)
	assertValid(t, schema, 5.0) // JSON numbers are float64
	assertInvalid(t, schema, 5.5)
}

func TestFromJSON_Boolean(t *testing.T) {
	schema, errs := jsonschema.FromJSON([]byte(`{
		"type": "boolean"
	}`), nil)
	assertNoErrors(t, errs)
	assertValid(t, schema, true)
	assertValid(t, schema, false)
	assertInvalid(t, schema, "true")
}

// ======================================================================
// Object schemas
// ======================================================================

func TestFromJSON_Object(t *testing.T) {
	schema, errs := jsonschema.FromJSON([]byte(`{
		"type": "object",
		"properties": {
			"name": {"type": "string", "minLength": 1},
			"age": {"type": "integer", "minimum": 0}
		},
		"required": ["name"]
	}`), nil)
	assertNoErrors(t, errs)

	// Valid
	assertValid(t, schema, map[string]interface{}{
		"name": "Alice",
		"age":  25.0,
	})

	// Missing required
	ctx := queryfy.NewValidationContext(queryfy.Strict)
	schema.Validate(map[string]interface{}{"age": 25.0}, ctx)
	if !ctx.HasErrors() {
		t.Error("expected error for missing required field 'name'")
	}
}

func TestFromJSON_ObjectAdditionalPropertiesFalse(t *testing.T) {
	schema, errs := jsonschema.FromJSON([]byte(`{
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		},
		"additionalProperties": false
	}`), nil)
	assertNoErrors(t, errs)

	// Extra fields should fail even in loose mode
	ctx := queryfy.NewValidationContext(queryfy.Loose)
	schema.Validate(map[string]interface{}{
		"name":  "Alice",
		"extra": "bad",
	}, ctx)
	if !ctx.HasErrors() {
		t.Error("expected error for extra field with additionalProperties: false")
	}
}

func TestFromJSON_ObjectAdditionalPropertiesTrue(t *testing.T) {
	schema, errs := jsonschema.FromJSON([]byte(`{
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		},
		"additionalProperties": true
	}`), nil)
	assertNoErrors(t, errs)

	// Extra fields should pass even in strict mode
	ctx := queryfy.NewValidationContext(queryfy.Strict)
	schema.Validate(map[string]interface{}{
		"name":  "Alice",
		"extra": "fine",
	}, ctx)
	if ctx.HasErrors() {
		t.Errorf("extra fields should be allowed: %v", ctx.Error())
	}
}

func TestFromJSON_NestedObject(t *testing.T) {
	schema, errs := jsonschema.FromJSON([]byte(`{
		"type": "object",
		"properties": {
			"address": {
				"type": "object",
				"properties": {
					"city": {"type": "string", "minLength": 1},
					"zip": {"type": "string", "pattern": "^[0-9]{5}$"}
				},
				"required": ["city"]
			}
		},
		"required": ["address"]
	}`), nil)
	assertNoErrors(t, errs)

	assertValid(t, schema, map[string]interface{}{
		"address": map[string]interface{}{
			"city": "Houston",
			"zip":  "77001",
		},
	})

	// Missing nested required
	ctx := queryfy.NewValidationContext(queryfy.Strict)
	schema.Validate(map[string]interface{}{
		"address": map[string]interface{}{
			"zip": "77001",
		},
	}, ctx)
	if !ctx.HasErrors() {
		t.Error("expected error for missing nested required field 'city'")
	}
}

// ======================================================================
// Array schemas
// ======================================================================

func TestFromJSON_Array(t *testing.T) {
	schema, errs := jsonschema.FromJSON([]byte(`{
		"type": "array",
		"items": {"type": "string"},
		"minItems": 1,
		"maxItems": 5
	}`), nil)
	assertNoErrors(t, errs)

	assertValid(t, schema, []interface{}{"a", "b"})
	assertInvalid(t, schema, []interface{}{}) // minItems
}

func TestFromJSON_ArrayOfObjects(t *testing.T) {
	schema, errs := jsonschema.FromJSON([]byte(`{
		"type": "array",
		"items": {
			"type": "object",
			"properties": {
				"id": {"type": "integer"},
				"name": {"type": "string"}
			},
			"required": ["id"]
		}
	}`), nil)
	assertNoErrors(t, errs)

	assertValid(t, schema, []interface{}{
		map[string]interface{}{"id": 1.0, "name": "Alice"},
		map[string]interface{}{"id": 2.0},
	})
}

func TestFromJSON_ArrayUniqueItems(t *testing.T) {
	schema, errs := jsonschema.FromJSON([]byte(`{
		"type": "array",
		"items": {"type": "string"},
		"uniqueItems": true
	}`), nil)
	assertNoErrors(t, errs)

	assertValid(t, schema, []interface{}{"a", "b", "c"})
	assertInvalid(t, schema, []interface{}{"a", "b", "a"})
}

// ======================================================================
// Nullable
// ======================================================================

func TestFromJSON_NullableOpenAPI(t *testing.T) {
	// OpenAPI 3.0 style: nullable: true
	schema, errs := jsonschema.FromJSON([]byte(`{
		"type": "string",
		"nullable": true
	}`), nil)
	assertNoErrors(t, errs)
	assertValid(t, schema, nil)
	assertValid(t, schema, "hello")
}

func TestFromJSON_NullableTypeArray(t *testing.T) {
	// JSON Schema style: type: ["string", "null"]
	schema, errs := jsonschema.FromJSON([]byte(`{
		"type": ["string", "null"]
	}`), nil)
	assertNoErrors(t, errs)
	assertValid(t, schema, nil)
	assertValid(t, schema, "hello")
}

// ======================================================================
// Type inference
// ======================================================================

func TestFromJSON_InferObject(t *testing.T) {
	// No explicit type, but has properties → infer object
	schema, errs := jsonschema.FromJSON([]byte(`{
		"properties": {
			"name": {"type": "string"}
		}
	}`), nil)
	assertNoErrors(t, errs)
	assertValid(t, schema, map[string]interface{}{"name": "Alice"})
}

func TestFromJSON_InferArray(t *testing.T) {
	// No explicit type, but has items → infer array
	schema, errs := jsonschema.FromJSON([]byte(`{
		"items": {"type": "number"}
	}`), nil)
	assertNoErrors(t, errs)
	assertValid(t, schema, []interface{}{1.0, 2.0})
}

// ======================================================================
// Unsupported features
// ======================================================================

func TestFromJSON_RefStrict(t *testing.T) {
	_, errs := jsonschema.FromJSON([]byte(`{
		"$ref": "#/definitions/Foo"
	}`), &jsonschema.Options{StrictMode: true})
	assertHasError(t, errs, "$ref")
}

func TestFromJSON_RefNonStrict(t *testing.T) {
	_, errs := jsonschema.FromJSON([]byte(`{
		"type": "string",
		"$ref": "#/definitions/Foo"
	}`), nil)
	// Should produce warning, not error
	assertHasWarning(t, errs, "$ref")
}

func TestFromJSON_OneOf(t *testing.T) {
	_, errs := jsonschema.FromJSON([]byte(`{
		"oneOf": [
			{"type": "string"},
			{"type": "number"}
		]
	}`), &jsonschema.Options{StrictMode: true})
	assertHasError(t, errs, "oneOf")
}

func TestFromJSON_AllOf(t *testing.T) {
	_, errs := jsonschema.FromJSON([]byte(`{
		"allOf": [
			{"type": "object", "properties": {"a": {"type": "string"}}},
			{"type": "object", "properties": {"b": {"type": "number"}}}
		]
	}`), &jsonschema.Options{StrictMode: true})
	assertHasError(t, errs, "allOf")
}

func TestFromJSON_IfThenElse(t *testing.T) {
	_, errs := jsonschema.FromJSON([]byte(`{
		"type": "object",
		"if": {"properties": {"type": {"const": "business"}}},
		"then": {"required": ["taxId"]}
	}`), &jsonschema.Options{StrictMode: true})
	assertHasError(t, errs, "if")
}

func TestFromJSON_DependentRequired(t *testing.T) {
	_, errs := jsonschema.FromJSON([]byte(`{
		"type": "object",
		"dependentRequired": {"email": ["name"]}
	}`), &jsonschema.Options{StrictMode: true})
	assertHasError(t, errs, "dependentRequired")
}

// ======================================================================
// Error handling
// ======================================================================

func TestFromJSON_InvalidJSON(t *testing.T) {
	_, errs := jsonschema.FromJSON([]byte(`{broken`), nil)
	if len(errs) == 0 {
		t.Fatal("expected error for invalid JSON")
	}
	if errs[0].Keyword != "json" {
		t.Errorf("expected json error, got %q", errs[0].Keyword)
	}
}

func TestFromJSON_InvalidPattern(t *testing.T) {
	_, errs := jsonschema.FromJSON([]byte(`{
		"type": "string",
		"pattern": "[invalid"
	}`), nil)
	assertHasError(t, errs, "pattern")
}

func TestFromJSON_MissingType(t *testing.T) {
	_, errs := jsonschema.FromJSON([]byte(`{
		"minLength": 5
	}`), nil)
	// No type and no inference possible → error
	assertHasError(t, errs, "type")
}

func TestFromJSON_UnknownType(t *testing.T) {
	_, errs := jsonschema.FromJSON([]byte(`{
		"type": "banana"
	}`), nil)
	assertHasError(t, errs, "type")
}

// ======================================================================
// Options: StoreUnknown
// ======================================================================

func TestFromJSON_StoreUnknown(t *testing.T) {
	schema, errs := jsonschema.FromJSON([]byte(`{
		"type": "string",
		"x-custom": "hello",
		"x-priority": 5
	}`), &jsonschema.Options{StoreUnknown: true})
	assertNoErrors(t, errs)

	meta := getMeta(t, schema, "x-custom")
	if meta != "hello" {
		t.Errorf("expected x-custom='hello', got %v", meta)
	}
	meta2 := getMeta(t, schema, "x-priority")
	if meta2 != 5.0 { // JSON numbers are float64
		t.Errorf("expected x-priority=5, got %v", meta2)
	}
}

func TestFromJSON_StoreUnknown_Disabled(t *testing.T) {
	schema, _ := jsonschema.FromJSON([]byte(`{
		"type": "string",
		"x-custom": "hello"
	}`), nil)

	meta := getMeta(t, schema, "x-custom")
	if meta != nil {
		t.Error("expected no metadata when StoreUnknown is false")
	}
}

// ======================================================================
// ConversionError formatting
// ======================================================================

func TestConversionError_String(t *testing.T) {
	e := jsonschema.ConversionError{
		Path:    "properties.name",
		Keyword: "type",
		Message: "unsupported type",
	}
	s := e.Error()
	if s != "error at properties.name: type: unsupported type" {
		t.Errorf("unexpected error string: %s", s)
	}

	w := jsonschema.ConversionError{
		Keyword:   "$ref",
		Message:   "skipped",
		IsWarning: true,
	}
	ws := w.Error()
	if ws != "warning: $ref: skipped" {
		t.Errorf("unexpected warning string: %s", ws)
	}
}

// ======================================================================
// ExclusiveMinimum / ExclusiveMaximum
// ======================================================================

func TestFromJSON_ExclusiveMinMax(t *testing.T) {
	schema, errs := jsonschema.FromJSON([]byte(`{
		"type": "number",
		"exclusiveMinimum": 0,
		"exclusiveMaximum": 100
	}`), nil)
	assertNoErrors(t, errs)

	// These are approximated as min/max; exact exclusive stored as metadata
	assertValid(t, schema, 50.0)
}

// ======================================================================
// UnknownFormat
// ======================================================================

func TestFromJSON_UnknownFormat(t *testing.T) {
	_, errs := jsonschema.FromJSON([]byte(`{
		"type": "string",
		"format": "isbn"
	}`), nil)
	// Should warn, not error
	assertHasWarning(t, errs, "format")
}

// ======================================================================
// Date-time format stored as metadata
// ======================================================================

func TestFromJSON_DateTimeFormat(t *testing.T) {
	schema, errs := jsonschema.FromJSON([]byte(`{
		"type": "string",
		"format": "date-time"
	}`), nil)
	assertNoErrors(t, errs)

	meta := getMeta(t, schema, "format")
	if meta != "date-time" {
		t.Errorf("expected format metadata 'date-time', got %v", meta)
	}
}

// ======================================================================
// AdditionalProperties as schema (warning)
// ======================================================================

func TestFromJSON_AdditionalPropertiesSchema(t *testing.T) {
	_, errs := jsonschema.FromJSON([]byte(`{
		"type": "object",
		"properties": {"name": {"type": "string"}},
		"additionalProperties": {"type": "string"}
	}`), nil)
	assertHasWarning(t, errs, "additionalProperties")
}

// ======================================================================
// $schema and $id are recognised (no warning)
// ======================================================================

func TestFromJSON_MetaKeywords(t *testing.T) {
	_, errs := jsonschema.FromJSON([]byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$id": "https://example.com/schema",
		"type": "string"
	}`), nil)
	assertNoErrors(t, errs)
}

// ======================================================================
// Complex real-world schema
// ======================================================================

func TestFromJSON_RealWorld_UserRegistration(t *testing.T) {
	schema, errs := jsonschema.FromJSON([]byte(`{
		"type": "object",
		"properties": {
			"email": {
				"type": "string",
				"format": "email",
				"maxLength": 254
			},
			"password": {
				"type": "string",
				"minLength": 8,
				"maxLength": 128
			},
			"age": {
				"type": "integer",
				"minimum": 13,
				"maximum": 150
			},
			"tags": {
				"type": "array",
				"items": {"type": "string", "maxLength": 50},
				"maxItems": 10,
				"uniqueItems": true
			},
			"address": {
				"type": "object",
				"properties": {
					"street": {"type": "string"},
					"city": {"type": "string", "minLength": 1},
					"zip": {"type": "string", "pattern": "^[0-9]{5}(-[0-9]{4})?$"}
				},
				"required": ["city"],
				"additionalProperties": false
			}
		},
		"required": ["email", "password"],
		"additionalProperties": false
	}`), nil)
	assertNoErrors(t, errs)

	// Valid input
	assertValid(t, schema, map[string]interface{}{
		"email":    "user@example.com",
		"password": "secret123",
		"age":      25.0,
		"tags":     []interface{}{"go", "dev"},
		"address": map[string]interface{}{
			"city": "Houston",
			"zip":  "77001",
		},
	})

	// Missing required
	ctx := queryfy.NewValidationContext(queryfy.Strict)
	schema.Validate(map[string]interface{}{
		"email": "user@example.com",
	}, ctx)
	if !ctx.HasErrors() {
		t.Error("expected error for missing password")
	}
}

// ======================================================================
// helpers
// ======================================================================

func assertNoErrors(t *testing.T, errs []jsonschema.ConversionError) {
	t.Helper()
	for _, e := range errs {
		if !e.IsWarning {
			t.Fatalf("unexpected error: %s", e.Error())
		}
	}
}

func assertValid(t *testing.T, schema queryfy.Schema, value interface{}) {
	t.Helper()
	ctx := queryfy.NewValidationContext(queryfy.Strict)
	schema.Validate(value, ctx)
	if ctx.HasErrors() {
		t.Errorf("expected %v to be valid, got: %v", value, ctx.Error())
	}
}

func assertInvalid(t *testing.T, schema queryfy.Schema, value interface{}) {
	t.Helper()
	ctx := queryfy.NewValidationContext(queryfy.Strict)
	schema.Validate(value, ctx)
	if !ctx.HasErrors() {
		t.Errorf("expected %v to be invalid", value)
	}
}

func assertHasError(t *testing.T, errs []jsonschema.ConversionError, keyword string) {
	t.Helper()
	for _, e := range errs {
		if e.Keyword == keyword && !e.IsWarning {
			return
		}
	}
	t.Errorf("expected error with keyword %q, got %v", keyword, errs)
}

func assertHasWarning(t *testing.T, errs []jsonschema.ConversionError, keyword string) {
	t.Helper()
	for _, e := range errs {
		if e.Keyword == keyword && e.IsWarning {
			return
		}
	}
	t.Errorf("expected warning with keyword %q, got %v", keyword, errs)
}

func getMeta(t *testing.T, schema queryfy.Schema, key string) interface{} {
	t.Helper()
	type metaGetter interface {
		GetMeta(string) (interface{}, bool)
	}
	if mg, ok := schema.(metaGetter); ok {
		val, _ := mg.GetMeta(key)
		return val
	}
	t.Fatalf("schema does not support GetMeta")
	return nil
}
