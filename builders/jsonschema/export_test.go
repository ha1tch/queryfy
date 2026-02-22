package jsonschema_test

import (
	"encoding/json"
	"testing"

	"github.com/ha1tch/queryfy"
	"github.com/ha1tch/queryfy/builders"
	"github.com/ha1tch/queryfy/builders/jsonschema"
)

// ======================================================================
// Basic type export
// ======================================================================

func TestExport_String(t *testing.T) {
	schema := builders.String().MinLength(1).MaxLength(100).Pattern("^[a-z]+$")
	out := jsonschema.ToMap(schema, nil)

	assertMapValue(t, out, "type", "string")
	assertMapValue(t, out, "minLength", 1)
	assertMapValue(t, out, "maxLength", 100)
	assertMapValue(t, out, "pattern", "^[a-z]+$")
}

func TestExport_StringEmail(t *testing.T) {
	schema := builders.String().Email()
	out := jsonschema.ToMap(schema, nil)

	assertMapValue(t, out, "type", "string")
	assertMapValue(t, out, "format", "email")
}

func TestExport_StringEnum(t *testing.T) {
	schema := builders.String().Enum("red", "green", "blue")
	out := jsonschema.ToMap(schema, nil)

	enum, ok := out["enum"].([]interface{})
	if !ok {
		t.Fatalf("expected enum array, got %T", out["enum"])
	}
	if len(enum) != 3 || enum[0] != "red" {
		t.Errorf("unexpected enum: %v", enum)
	}
}

func TestExport_Number(t *testing.T) {
	schema := builders.Number().Min(0).Max(100).MultipleOf(5)
	out := jsonschema.ToMap(schema, nil)

	assertMapValue(t, out, "type", "number")
	assertMapValue(t, out, "minimum", 0.0)
	assertMapValue(t, out, "maximum", 100.0)
	assertMapValue(t, out, "multipleOf", 5.0)
}

func TestExport_Integer(t *testing.T) {
	schema := builders.Number().Integer().Min(1)
	out := jsonschema.ToMap(schema, nil)

	assertMapValue(t, out, "type", "integer")
	assertMapValue(t, out, "minimum", 1.0)
}

func TestExport_Boolean(t *testing.T) {
	schema := builders.Bool()
	out := jsonschema.ToMap(schema, nil)
	assertMapValue(t, out, "type", "boolean")
}

// ======================================================================
// Object export
// ======================================================================

func TestExport_Object(t *testing.T) {
	schema := builders.Object().
		Field("name", builders.String().Required().MinLength(1)).
		Field("age", builders.Number().Integer().Min(0)).
		AllowAdditional(false)

	out := jsonschema.ToMap(schema, nil)

	assertMapValue(t, out, "type", "object")
	assertMapValue(t, out, "additionalProperties", false)

	props, ok := out["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected properties map, got %T", out["properties"])
	}
	if _, exists := props["name"]; !exists {
		t.Error("expected 'name' in properties")
	}
	if _, exists := props["age"]; !exists {
		t.Error("expected 'age' in properties")
	}

	required, ok := out["required"].([]interface{})
	if !ok {
		t.Fatalf("expected required array, got %T", out["required"])
	}
	if len(required) != 1 || required[0] != "name" {
		t.Errorf("expected required=['name'], got %v", required)
	}
}

func TestExport_NestedObject(t *testing.T) {
	schema := builders.Object().
		Field("address", builders.Object().
			Field("city", builders.String().Required()).
			Field("zip", builders.String().Pattern("^[0-9]{5}$")))

	out := jsonschema.ToMap(schema, nil)
	props := out["properties"].(map[string]interface{})
	addr := props["address"].(map[string]interface{})

	assertMapValue(t, addr, "type", "object")
	addrProps := addr["properties"].(map[string]interface{})
	if _, exists := addrProps["city"]; !exists {
		t.Error("expected 'city' in address properties")
	}
}

// ======================================================================
// Array export
// ======================================================================

func TestExport_Array(t *testing.T) {
	schema := builders.Array().
		Of(builders.String().MinLength(1)).
		MinItems(1).MaxItems(10).UniqueItems()

	out := jsonschema.ToMap(schema, nil)

	assertMapValue(t, out, "type", "array")
	assertMapValue(t, out, "minItems", 1)
	assertMapValue(t, out, "maxItems", 10)
	assertMapValue(t, out, "uniqueItems", true)

	items, ok := out["items"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected items map, got %T", out["items"])
	}
	assertMapValue(t, items, "type", "string")
}

// ======================================================================
// Nullable export
// ======================================================================

func TestExport_Nullable(t *testing.T) {
	schema := builders.String().Nullable()
	out := jsonschema.ToMap(schema, nil)

	typeVal, ok := out["type"].([]interface{})
	if !ok {
		t.Fatalf("expected type array for nullable, got %T", out["type"])
	}
	if len(typeVal) != 2 || typeVal[0] != "string" || typeVal[1] != "null" {
		t.Errorf("expected [string, null], got %v", typeVal)
	}
}

// ======================================================================
// Transform passthrough
// ======================================================================

func TestExport_Transform(t *testing.T) {
	schema := builders.Transform(builders.String().Email().MinLength(5))
	out := jsonschema.ToMap(schema, nil)

	assertMapValue(t, out, "type", "string")
	assertMapValue(t, out, "format", "email")
	assertMapValue(t, out, "minLength", 5)
}

// ======================================================================
// Options
// ======================================================================

func TestExport_WithSchemaURI(t *testing.T) {
	schema := builders.String()
	out := jsonschema.ToMap(schema, &jsonschema.ExportOptions{
		SchemaURI: "https://json-schema.org/draft/2020-12/schema",
		ID:        "https://example.com/my-schema",
	})

	assertMapValue(t, out, "$schema", "https://json-schema.org/draft/2020-12/schema")
	assertMapValue(t, out, "$id", "https://example.com/my-schema")
}

func TestExport_IncludeMeta(t *testing.T) {
	schema := builders.String()
	schema.Meta("x-priority", 5)
	schema.Meta("x-label", "Name")

	out := jsonschema.ToMap(schema, &jsonschema.ExportOptions{IncludeMeta: true})
	if out["x-priority"] != 5 {
		t.Errorf("expected x-priority=5, got %v", out["x-priority"])
	}
	if out["x-label"] != "Name" {
		t.Errorf("expected x-label='Name', got %v", out["x-label"])
	}
}

func TestExport_MetaDisabled(t *testing.T) {
	schema := builders.String()
	schema.Meta("x-custom", "val")

	out := jsonschema.ToMap(schema, nil)
	if _, exists := out["x-custom"]; exists {
		t.Error("expected no x-custom when IncludeMeta is false")
	}
}

// ======================================================================
// ToJSON produces valid JSON
// ======================================================================

func TestExport_ToJSON(t *testing.T) {
	schema := builders.Object().
		Field("email", builders.String().Email().Required()).
		Field("score", builders.Number().Integer().Min(0).Max(100))

	data, err := jsonschema.ToJSON(schema, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if parsed["type"] != "object" {
		t.Errorf("expected type=object, got %v", parsed["type"])
	}
}

// ======================================================================
// Round-trip: import → export → import → compare
// ======================================================================

func TestRoundTrip_String(t *testing.T) {
	original := `{
		"type": "string",
		"minLength": 1,
		"maxLength": 50,
		"pattern": "^[a-zA-Z]+$"
	}`
	roundTrip(t, original)
}

func TestRoundTrip_Number(t *testing.T) {
	original := `{
		"type": "number",
		"minimum": 0,
		"maximum": 100,
		"multipleOf": 5
	}`
	roundTrip(t, original)
}

func TestRoundTrip_Integer(t *testing.T) {
	original := `{
		"type": "integer",
		"minimum": 1,
		"maximum": 999
	}`
	roundTrip(t, original)
}

func TestRoundTrip_Boolean(t *testing.T) {
	roundTrip(t, `{"type": "boolean"}`)
}

func TestRoundTrip_Object(t *testing.T) {
	original := `{
		"type": "object",
		"properties": {
			"name": {"type": "string", "minLength": 1},
			"age": {"type": "integer", "minimum": 0}
		},
		"required": ["name"],
		"additionalProperties": false
	}`
	roundTrip(t, original)
}

func TestRoundTrip_Array(t *testing.T) {
	original := `{
		"type": "array",
		"items": {"type": "string"},
		"minItems": 1,
		"maxItems": 10,
		"uniqueItems": true
	}`
	roundTrip(t, original)
}

func TestRoundTrip_Nullable(t *testing.T) {
	original := `{"type": ["string", "null"], "minLength": 1}`
	roundTrip(t, original)
}

func TestRoundTrip_Nested(t *testing.T) {
	original := `{
		"type": "object",
		"properties": {
			"address": {
				"type": "object",
				"properties": {
					"city": {"type": "string"},
					"zip": {"type": "string", "pattern": "^[0-9]{5}$"}
				},
				"required": ["city"]
			},
			"tags": {
				"type": "array",
				"items": {"type": "string", "maxLength": 50},
				"minItems": 0,
				"maxItems": 20
			}
		},
		"required": ["address"]
	}`
	roundTrip(t, original)
}

func TestRoundTrip_EmailFormat(t *testing.T) {
	original := `{"type": "string", "format": "email"}`
	roundTrip(t, original)
}

func TestRoundTrip_WithMeta(t *testing.T) {
	original := `{
		"type": "string",
		"x-custom": "hello",
		"x-priority": 5
	}`

	// Import with StoreUnknown
	schema, errs := jsonschema.FromJSON([]byte(original), &jsonschema.Options{StoreUnknown: true})
	assertNoErrors(t, errs)

	// Export with IncludeMeta
	exported, err := jsonschema.ToJSON(schema, &jsonschema.ExportOptions{IncludeMeta: true})
	if err != nil {
		t.Fatalf("export error: %v", err)
	}

	// Re-import
	schema2, errs2 := jsonschema.FromJSON(exported, &jsonschema.Options{StoreUnknown: true})
	assertNoErrors(t, errs2)

	// Both should validate the same data
	assertValid(t, schema, "test")
	assertValid(t, schema2, "test")

	// Meta should survive
	meta := getMeta(t, schema2, "x-custom")
	if meta != "hello" {
		t.Errorf("expected x-custom='hello' after round-trip, got %v", meta)
	}
}

// ======================================================================
// Round-trip helper
// ======================================================================

// roundTrip imports JSON Schema, exports it back, then re-imports and
// verifies both schemas validate the same test data.
func roundTrip(t *testing.T, originalJSON string) {
	t.Helper()

	// Step 1: Import
	schema1, errs := jsonschema.FromJSON([]byte(originalJSON), nil)
	assertNoErrors(t, errs)

	// Step 2: Export
	exported, err := jsonschema.ToJSON(schema1, nil)
	if err != nil {
		t.Fatalf("export error: %v", err)
	}

	// Step 3: Re-import
	schema2, errs2 := jsonschema.FromJSON(exported, nil)
	assertNoErrors(t, errs2)

	// Step 4: Structural comparison — parse both to maps and compare
	var original map[string]interface{}
	json.Unmarshal([]byte(originalJSON), &original)

	var reexported map[string]interface{}
	json.Unmarshal(exported, &reexported)

	compareJSONMaps(t, original, reexported, "")

	// Step 5: Both schemas should accept the same valid data
	testData := generateTestData(schema1)
	if testData != nil {
		ctx1 := queryfy.NewValidationContext(queryfy.Strict)
		schema1.Validate(testData, ctx1)

		ctx2 := queryfy.NewValidationContext(queryfy.Strict)
		schema2.Validate(testData, ctx2)

		if ctx1.HasErrors() != ctx2.HasErrors() {
			t.Errorf("validation mismatch: original=%v, round-tripped=%v",
				ctx1.HasErrors(), ctx2.HasErrors())
		}
	}
}

// compareJSONMaps compares two JSON maps structurally.
func compareJSONMaps(t *testing.T, expected, actual map[string]interface{}, path string) {
	t.Helper()

	for key, expVal := range expected {
		actVal, exists := actual[key]
		if !exists {
			t.Errorf("at %s: missing key %q in exported output", path, key)
			continue
		}

		fullPath := key
		if path != "" {
			fullPath = path + "." + key
		}

		switch ev := expVal.(type) {
		case map[string]interface{}:
			av, ok := actVal.(map[string]interface{})
			if !ok {
				t.Errorf("at %s: expected object, got %T", fullPath, actVal)
				continue
			}
			compareJSONMaps(t, ev, av, fullPath)

		case []interface{}:
			av, ok := actVal.([]interface{})
			if !ok {
				t.Errorf("at %s: expected array, got %T", fullPath, actVal)
				continue
			}
			if len(ev) != len(av) {
				t.Errorf("at %s: array length mismatch: %d vs %d", fullPath, len(ev), len(av))
				continue
			}
			// For simple arrays (required, enum, type), compare sorted
			// since order may differ
			expStrs := toStrSlice(ev)
			actStrs := toStrSlice(av)
			if expStrs != nil && actStrs != nil {
				for _, s := range expStrs {
					if !containsStr(actStrs, s) {
						t.Errorf("at %s: missing %q in exported array", fullPath, s)
					}
				}
			}

		default:
			if expVal != actVal {
				t.Errorf("at %s: expected %v (%T), got %v (%T)",
					fullPath, expVal, expVal, actVal, actVal)
			}
		}
	}
}

// generateTestData produces a minimal valid value for a schema type.
func generateTestData(schema queryfy.Schema) interface{} {
	switch schema.Type() {
	case queryfy.TypeString:
		return "test"
	case queryfy.TypeNumber:
		return 50.0
	case queryfy.TypeBool:
		return true
	case queryfy.TypeObject:
		return map[string]interface{}{}
	case queryfy.TypeArray:
		return []interface{}{}
	default:
		return nil
	}
}

func toStrSlice(arr []interface{}) []string {
	result := make([]string, 0, len(arr))
	for _, v := range arr {
		s, ok := v.(string)
		if !ok {
			return nil
		}
		result = append(result, s)
	}
	return result
}

func containsStr(arr []string, s string) bool {
	for _, v := range arr {
		if v == s {
			return true
		}
	}
	return false
}

// ======================================================================
// Helpers
// ======================================================================

func assertMapValue(t *testing.T, m map[string]interface{}, key string, expected interface{}) {
	t.Helper()
	val, ok := m[key]
	if !ok {
		t.Errorf("expected key %q in map", key)
		return
	}
	if val != expected {
		t.Errorf("expected %v (%T) for key %q, got %v (%T)", expected, expected, key, val, val)
	}
}
