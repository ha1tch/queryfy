package queryfy_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ha1tch/queryfy"
	"github.com/ha1tch/queryfy/builders"
	"github.com/ha1tch/queryfy/builders/transformers"
)

// TestArrayItems verifies that Items() works as an alias for Of().
func TestArrayItems(t *testing.T) {
	schema := builders.Array().Items(
		builders.String().MinLength(2),
	).Required()

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	err := schema.Validate([]interface{}{"hello", "world"}, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.HasErrors() {
		t.Fatalf("unexpected validation errors: %v", ctx.Error())
	}

	// Should fail with short strings
	ctx2 := queryfy.NewValidationContext(queryfy.Strict)
	schema.Validate([]interface{}{"a"}, ctx2)
	if !ctx2.HasErrors() {
		t.Fatal("expected validation error for short string")
	}
}

// TestObjectValidateAndTransform_SimpleFields tests basic field transformation.
func TestObjectValidateAndTransform_SimpleFields(t *testing.T) {
	schema := builders.Object().
		Field("email", builders.Transform(
			builders.String().Email().Required(),
		).Add(transformers.Trim()).Add(transformers.Lowercase())).
		Field("name", builders.String().Required())

	data := map[string]interface{}{
		"email": "  ALICE@EXAMPLE.COM  ",
		"name":  "Alice",
	}

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	result, err := schema.ValidateAndTransform(data, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}

	// Transformed field
	if resultMap["email"] != "alice@example.com" {
		t.Errorf("expected 'alice@example.com', got %q", resultMap["email"])
	}

	// Non-transformed field preserved
	if resultMap["name"] != "Alice" {
		t.Errorf("expected 'Alice', got %q", resultMap["name"])
	}

	// Original data unmodified
	if data["email"] != "  ALICE@EXAMPLE.COM  " {
		t.Error("original data was mutated")
	}
}

// TestObjectValidateAndTransform_Nested tests nested object transformation.
func TestObjectValidateAndTransform_Nested(t *testing.T) {
	schema := builders.Object().
		Field("orderId", builders.Transform(
			builders.String().Required(),
		).Add(transformers.Trim()).Add(transformers.Uppercase())).
		Field("customer", builders.Object().
			Field("name", builders.Transform(
				builders.String().Required(),
			).Add(transformers.Trim())).
			Field("email", builders.Transform(
				builders.String().Email().Required(),
			).Add(transformers.Trim()).Add(transformers.Lowercase())))

	data := map[string]interface{}{
		"orderId": "  ord-001  ",
		"customer": map[string]interface{}{
			"name":  "  John Doe  ",
			"email": "  JOHN@EXAMPLE.COM  ",
		},
	}

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	result, err := schema.ValidateAndTransform(data, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultMap := result.(map[string]interface{})
	if resultMap["orderId"] != "ORD-001" {
		t.Errorf("expected 'ORD-001', got %q", resultMap["orderId"])
	}

	customer := resultMap["customer"].(map[string]interface{})
	if customer["name"] != "John Doe" {
		t.Errorf("expected 'John Doe', got %q", customer["name"])
	}
	if customer["email"] != "john@example.com" {
		t.Errorf("expected 'john@example.com', got %q", customer["email"])
	}
}

// TestObjectValidateAndTransform_WithArray tests array-of-objects transformation.
func TestObjectValidateAndTransform_WithArray(t *testing.T) {
	schema := builders.Object().
		Field("items", builders.Array().Items(
			builders.Object().
				Field("sku", builders.Transform(
					builders.String().Required(),
				).Add(transformers.Trim()).Add(transformers.Uppercase())).
				Field("label", builders.String().Required())))

	data := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{
				"sku":   "  abc-001  ",
				"label": "Widget",
			},
			map[string]interface{}{
				"sku":   "  xyz-999  ",
				"label": "Gadget",
			},
		},
	}

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	result, err := schema.ValidateAndTransform(data, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultMap := result.(map[string]interface{})
	items := resultMap["items"].([]interface{})

	item0 := items[0].(map[string]interface{})
	if item0["sku"] != "ABC-001" {
		t.Errorf("expected 'ABC-001', got %q", item0["sku"])
	}
	if item0["label"] != "Widget" {
		t.Errorf("expected 'Widget', got %q", item0["label"])
	}

	item1 := items[1].(map[string]interface{})
	if item1["sku"] != "XYZ-999" {
		t.Errorf("expected 'XYZ-999', got %q", item1["sku"])
	}
}

// TestObjectValidateAndTransform_ValidationErrors tests that validation errors
// are correctly reported even when transformations are involved.
func TestObjectValidateAndTransform_ValidationErrors(t *testing.T) {
	schema := builders.Object().
		Field("email", builders.Transform(
			builders.String().Email().Required(),
		).Add(transformers.Trim()).Add(transformers.Lowercase())).
		Field("age", builders.Number().Min(18).Required())

	data := map[string]interface{}{
		"email": "  not-an-email  ",
		"age":   15,
	}

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	_, err := schema.ValidateAndTransform(data, ctx)
	if err == nil {
		t.Fatal("expected validation error")
	}

	valErr, ok := err.(*queryfy.ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}

	// Should have errors for both fields
	if len(valErr.Errors) < 2 {
		t.Errorf("expected at least 2 errors, got %d: %v", len(valErr.Errors), valErr.Errors)
	}

	// Check that specific error paths are reported
	paths := make(map[string]bool)
	for _, fe := range valErr.Errors {
		paths[fe.Path] = true
	}
	if !paths["email"] {
		t.Error("expected error at path 'email'")
	}
	if !paths["age"] {
		t.Error("expected error at path 'age'")
	}
}

// TestObjectValidateAndTransform_RequiredMissing tests that missing required
// fields are correctly detected.
func TestObjectValidateAndTransform_RequiredMissing(t *testing.T) {
	schema := builders.Object().
		Field("name", builders.String().Required()).
		Field("email", builders.Transform(
			builders.String().Email().Required(),
		).Add(transformers.Lowercase()))

	data := map[string]interface{}{
		// Both required fields missing
	}

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	_, err := schema.ValidateAndTransform(data, ctx)
	if err == nil {
		t.Fatal("expected validation error for missing required fields")
	}

	valErr := err.(*queryfy.ValidationError)
	if len(valErr.Errors) < 2 {
		t.Errorf("expected at least 2 errors, got %d", len(valErr.Errors))
	}
}

// TestObjectValidateAndTransform_LooseMode tests that extra fields are
// preserved in loose mode.
func TestObjectValidateAndTransform_LooseMode(t *testing.T) {
	schema := builders.Object().
		Field("email", builders.Transform(
			builders.String().Email().Required(),
		).Add(transformers.Trim()).Add(transformers.Lowercase()))

	data := map[string]interface{}{
		"email":  "  BOB@EXAMPLE.COM  ",
		"extra1": "preserved",
		"extra2": 42,
	}

	ctx := queryfy.NewValidationContext(queryfy.Loose)
	result, err := schema.ValidateAndTransform(data, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultMap := result.(map[string]interface{})
	if resultMap["email"] != "bob@example.com" {
		t.Errorf("expected 'bob@example.com', got %q", resultMap["email"])
	}
	if resultMap["extra1"] != "preserved" {
		t.Error("extra field 'extra1' was not preserved in loose mode")
	}
	if resultMap["extra2"] != 42 {
		t.Error("extra field 'extra2' was not preserved in loose mode")
	}
}

// TestObjectValidateAndTransform_StrictModeExtraFields tests that extra fields
// cause errors in strict mode.
func TestObjectValidateAndTransform_StrictModeExtraFields(t *testing.T) {
	schema := builders.Object().
		Field("name", builders.String().Required())

	data := map[string]interface{}{
		"name":      "Alice",
		"unwanted":  true,
	}

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	_, err := schema.ValidateAndTransform(data, ctx)
	if err == nil {
		t.Fatal("expected error for extra field in strict mode")
	}
	if !strings.Contains(err.Error(), "unexpected field") {
		t.Errorf("expected 'unexpected field' error, got: %v", err)
	}
}

// TestObjectValidateAndTransform_NilValue tests nil input handling.
func TestObjectValidateAndTransform_NilValue(t *testing.T) {
	schema := builders.Object().Required()
	ctx := queryfy.NewValidationContext(queryfy.Strict)
	_, err := schema.ValidateAndTransform(nil, ctx)
	if err == nil {
		t.Fatal("expected error for nil required object")
	}
}

// TestObjectValidateAndTransform_CustomValidator tests that custom validators
// receive the transformed result map.
func TestObjectValidateAndTransform_CustomValidator(t *testing.T) {
	var receivedValue interface{}
	schema := builders.Object().
		Field("code", builders.Transform(
			builders.String().Required(),
		).Add(transformers.Uppercase())).
		Custom(func(value interface{}) error {
			receivedValue = value
			m := value.(map[string]interface{})
			if m["code"] != "ABC" {
				return fmt.Errorf("custom: expected 'ABC' in custom validator")
			}
			return nil
		})

	data := map[string]interface{}{
		"code": "abc",
	}

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	_, err := schema.ValidateAndTransform(data, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the custom validator received the transformed value
	m := receivedValue.(map[string]interface{})
	if m["code"] != "ABC" {
		t.Errorf("custom validator received untransformed value: %v", m["code"])
	}
}

// TestArrayValidateAndTransform_Elements tests array element transformation.
func TestArrayValidateAndTransform_Elements(t *testing.T) {
	schema := builders.Array().Items(
		builders.Transform(
			builders.String().Required(),
		).Add(transformers.Trim()).Add(transformers.Uppercase()),
	).Required()

	data := []interface{}{"  hello  ", "  world  "}

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	result, err := schema.ValidateAndTransform(data, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := result.([]interface{})
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0] != "HELLO" {
		t.Errorf("expected 'HELLO', got %q", items[0])
	}
	if items[1] != "WORLD" {
		t.Errorf("expected 'WORLD', got %q", items[1])
	}
}

// TestArrayValidateAndTransform_ObjectElements tests array of transformed objects.
func TestArrayValidateAndTransform_ObjectElements(t *testing.T) {
	schema := builders.Array().Items(
		builders.Object().
			Field("tag", builders.Transform(
				builders.String().Required(),
			).Add(transformers.Trim()).Add(transformers.Lowercase())),
	).MinItems(1)

	data := []interface{}{
		map[string]interface{}{"tag": "  GO  "},
		map[string]interface{}{"tag": "  RUST  "},
	}

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	result, err := schema.ValidateAndTransform(data, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := result.([]interface{})
	tag0 := items[0].(map[string]interface{})["tag"]
	tag1 := items[1].(map[string]interface{})["tag"]
	if tag0 != "go" {
		t.Errorf("expected 'go', got %q", tag0)
	}
	if tag1 != "rust" {
		t.Errorf("expected 'rust', got %q", tag1)
	}
}

// TestArrayValidateAndTransform_NoElementSchema tests arrays without element schemas.
func TestArrayValidateAndTransform_NoElementSchema(t *testing.T) {
	schema := builders.Array().MinItems(1).Required()

	data := []interface{}{1, "two", true}

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	result, err := schema.ValidateAndTransform(data, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := result.([]interface{})
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if items[0] != 1 || items[1] != "two" || items[2] != true {
		t.Error("original values not preserved")
	}
}

// TestArrayValidateAndTransform_ValidationErrors tests error reporting in arrays.
func TestArrayValidateAndTransform_ValidationErrors(t *testing.T) {
	schema := builders.Array().Items(
		builders.String().MinLength(3).Required(),
	).Required()

	data := []interface{}{"hello", "ab", "ok"}

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	_, err := schema.ValidateAndTransform(data, ctx)
	if err == nil {
		t.Fatal("expected validation error")
	}

	valErr := err.(*queryfy.ValidationError)
	// "ab" and "ok" are both < 3 chars
	if len(valErr.Errors) < 2 {
		t.Errorf("expected at least 2 errors, got %d", len(valErr.Errors))
	}
}

// TestDeepNesting tests a three-level-deep transformation scenario:
// Object -> Array -> Object with transforms at every level.
func TestDeepNesting(t *testing.T) {
	schema := builders.Object().
		Field("company", builders.Transform(
			builders.String().Required(),
		).Add(transformers.Trim())).
		Field("departments", builders.Array().Items(
			builders.Object().
				Field("name", builders.Transform(
					builders.String().Required(),
				).Add(transformers.Trim()).Add(transformers.Uppercase())).
				Field("tags", builders.Array().Items(
					builders.Transform(
						builders.String(),
					).Add(transformers.Trim()).Add(transformers.Lowercase())))))

	data := map[string]interface{}{
		"company": "  Acme Corp  ",
		"departments": []interface{}{
			map[string]interface{}{
				"name": "  engineering  ",
				"tags": []interface{}{"  GO  ", "  RUST  "},
			},
			map[string]interface{}{
				"name": "  marketing  ",
				"tags": []interface{}{"  SEO  ", "  PPC  "},
			},
		},
	}

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	result, err := schema.ValidateAndTransform(data, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := result.(map[string]interface{})
	if r["company"] != "Acme Corp" {
		t.Errorf("expected 'Acme Corp', got %q", r["company"])
	}

	depts := r["departments"].([]interface{})
	eng := depts[0].(map[string]interface{})
	if eng["name"] != "ENGINEERING" {
		t.Errorf("expected 'ENGINEERING', got %q", eng["name"])
	}

	engTags := eng["tags"].([]interface{})
	if engTags[0] != "go" || engTags[1] != "rust" {
		t.Errorf("expected ['go', 'rust'], got %v", engTags)
	}

	mkt := depts[1].(map[string]interface{})
	if mkt["name"] != "MARKETING" {
		t.Errorf("expected 'MARKETING', got %q", mkt["name"])
	}

	mktTags := mkt["tags"].([]interface{})
	if mktTags[0] != "seo" || mktTags[1] != "ppc" {
		t.Errorf("expected ['seo', 'ppc'], got %v", mktTags)
	}
}
