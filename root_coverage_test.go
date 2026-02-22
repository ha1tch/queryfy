package queryfy_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/ha1tch/queryfy"
	"github.com/ha1tch/queryfy/builders"
	"github.com/ha1tch/queryfy/builders/transformers"
)

// ======================================================================
// Validator wrapper (NewValidator / Strict / Loose)
// ======================================================================

func TestValidator_Strict(t *testing.T) {
	schema := builders.String().Required()
	v := queryfy.NewValidator(schema).Strict()
	err := v.Validate("hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidator_Loose(t *testing.T) {
	schema := builders.String()
	v := queryfy.NewValidator(schema).Loose()
	// Loose mode: number accepted as string
	err := v.Validate(42)
	if err != nil {
		t.Fatalf("unexpected error in loose mode: %v", err)
	}
}

func TestValidator_Validate_Error(t *testing.T) {
	schema := builders.String().Required()
	v := queryfy.NewValidator(schema)
	err := v.Validate(nil)
	if err == nil {
		t.Fatal("expected error for nil on required field")
	}
}

// ======================================================================
// Compile and MustValidate
// ======================================================================

func TestCompile(t *testing.T) {
	schema := builders.Object().
		Field("name", builders.String().Required())
	compiled := queryfy.Compile(schema)
	if compiled == nil {
		t.Fatal("Compile returned nil")
	}
}

func TestMustValidate_OK(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("MustValidate panicked unexpectedly: %v", r)
		}
	}()
	schema := builders.String().Required()
	queryfy.MustValidate("hello", schema) // should not panic
}

func TestMustValidate_Panics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected MustValidate to panic on invalid data")
		}
	}()
	schema := builders.String().Required()
	queryfy.MustValidate(nil, schema) // should panic
}

// ======================================================================
// ValidateValue — all type branches
// ======================================================================

func TestValidateValue_String(t *testing.T) {
	ctx := queryfy.NewValidationContext(queryfy.Strict)
	if !queryfy.ValidateValue("hello", queryfy.TypeString, ctx) {
		t.Error("expected string to validate as TypeString")
	}
	if ctx.HasErrors() {
		t.Error("unexpected errors")
	}

	ctx2 := queryfy.NewValidationContext(queryfy.Strict)
	queryfy.ValidateValue(42, queryfy.TypeString, ctx2)
	if !ctx2.HasErrors() {
		t.Error("expected error for int as TypeString")
	}
}

func TestValidateValue_Number(t *testing.T) {
	ctx := queryfy.NewValidationContext(queryfy.Strict)
	if !queryfy.ValidateValue(42.0, queryfy.TypeNumber, ctx) {
		t.Error("expected float64 to validate as TypeNumber")
	}

	ctx2 := queryfy.NewValidationContext(queryfy.Strict)
	if !queryfy.ValidateValue(42, queryfy.TypeNumber, ctx2) {
		t.Error("expected int to validate as TypeNumber")
	}

	ctx3 := queryfy.NewValidationContext(queryfy.Strict)
	queryfy.ValidateValue("not", queryfy.TypeNumber, ctx3)
	if !ctx3.HasErrors() {
		t.Error("expected error for string as TypeNumber")
	}
}

func TestValidateValue_LooseStringToNumber(t *testing.T) {
	ctx := queryfy.NewValidationContext(queryfy.Loose)
	if !queryfy.ValidateValue("42", queryfy.TypeNumber, ctx) {
		t.Error("expected string '42' to validate as TypeNumber in loose mode")
	}
}

func TestValidateValue_Object(t *testing.T) {
	ctx := queryfy.NewValidationContext(queryfy.Strict)
	if !queryfy.ValidateValue(map[string]interface{}{"a": 1}, queryfy.TypeObject, ctx) {
		t.Error("expected map to validate as TypeObject")
	}

	ctx2 := queryfy.NewValidationContext(queryfy.Strict)
	queryfy.ValidateValue("not-obj", queryfy.TypeObject, ctx2)
	if !ctx2.HasErrors() {
		t.Error("expected error for string as TypeObject")
	}
}

func TestValidateValue_Array(t *testing.T) {
	ctx := queryfy.NewValidationContext(queryfy.Strict)
	if !queryfy.ValidateValue([]interface{}{1}, queryfy.TypeArray, ctx) {
		t.Error("expected slice to validate as TypeArray")
	}

	ctx2 := queryfy.NewValidationContext(queryfy.Strict)
	queryfy.ValidateValue("not-arr", queryfy.TypeArray, ctx2)
	if !ctx2.HasErrors() {
		t.Error("expected error for string as TypeArray")
	}
}

// ======================================================================
// Error types
// ======================================================================

func TestFieldError(t *testing.T) {
	fe := queryfy.NewFieldError("user.email", "invalid email", "bad@")
	if fe.Error() == "" {
		t.Error("FieldError.Error() should not be empty")
	}
	if fe.Path != "user.email" {
		t.Errorf("expected path 'user.email', got %q", fe.Path)
	}
}

func TestValidationError(t *testing.T) {
	ve := queryfy.NewValidationError()
	if ve == nil {
		t.Fatal("NewValidationError returned nil")
	}
	if ve.HasErrors() {
		t.Error("new ValidationError should have no errors")
	}

	ve.Add("path", "message", "value")
	ve.AddError(queryfy.NewFieldError("other", "msg2", nil))
	if !ve.HasErrors() {
		t.Error("should have errors after adding")
	}
	if ve.Error() == "" {
		t.Error("Error() should not be empty")
	}
}

func TestWrapError(t *testing.T) {
	// Wrap a ValidationError — should prepend path
	ve := queryfy.NewValidationError(
		queryfy.NewFieldError("email", "invalid", "bad"),
	)
	wrapped := queryfy.WrapError(ve, "user")
	if wrapped == nil {
		t.Fatal("WrapError returned nil for validation error")
	}

	// Wrap nil should return nil
	if queryfy.WrapError(nil, "path") != nil {
		t.Error("WrapError(nil) should return nil")
	}

	// Wrap a non-ValidationError
	var plainErr error = queryfy.NewFieldError("", "oops", nil)
	wrapped2 := queryfy.WrapError(plainErr, "field")
	if wrapped2 == nil {
		t.Error("WrapError should wrap non-nil errors")
	}
}

// ======================================================================
// Context — Transformations tracking
// ======================================================================

func TestContext_Transformations(t *testing.T) {
	ctx := queryfy.NewValidationContext(queryfy.Strict)

	if ctx.HasTransformations() {
		t.Error("fresh context should have no transformations")
	}

	ctx.RecordTransformation("original", "transformed", "test")

	if !ctx.HasTransformations() {
		t.Error("context should have transformations after recording")
	}

	xforms := ctx.Transformations()
	if len(xforms) != 1 {
		t.Fatalf("expected 1 transformation, got %d", len(xforms))
	}
	if xforms[0].Original != "original" || xforms[0].Result != "transformed" {
		t.Errorf("unexpected transformation: %+v", xforms[0])
	}
}

func TestContext_AddFieldError(t *testing.T) {
	ctx := queryfy.NewValidationContext(queryfy.Strict)
	fe := queryfy.NewFieldError("field", "message", nil)
	ctx.AddFieldError(fe)
	if !ctx.HasErrors() {
		t.Error("context should have errors after AddFieldError")
	}
}

// ======================================================================
// ConvertToString edge cases
// ======================================================================

func TestConvertToString(t *testing.T) {
	// bool
	s, ok := queryfy.ConvertToString(true)
	if !ok || s != "true" {
		t.Errorf("expected 'true', got %q (ok=%v)", s, ok)
	}

	// int
	s, ok = queryfy.ConvertToString(42)
	if !ok || s != "42" {
		t.Errorf("expected '42', got %q (ok=%v)", s, ok)
	}

	// float
	s, ok = queryfy.ConvertToString(3.14)
	if !ok || s != "3.14" {
		t.Errorf("expected '3.14', got %q (ok=%v)", s, ok)
	}

	// string passthrough
	s, ok = queryfy.ConvertToString("hello")
	if !ok || s != "hello" {
		t.Errorf("expected 'hello', got %q (ok=%v)", s, ok)
	}
}

// ======================================================================
// ValidateWithMode
// ======================================================================

func TestValidateWithMode(t *testing.T) {
	schema := builders.String().Required()

	err := queryfy.ValidateWithMode("hello", schema, queryfy.Strict)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err2 := queryfy.ValidateWithMode(nil, schema, queryfy.Strict)
	if err2 == nil {
		t.Fatal("expected error for nil on required field")
	}

	// Loose mode — number converts to string
	err3 := queryfy.ValidateWithMode(42, builders.String(), queryfy.Loose)
	if err3 != nil {
		t.Fatalf("expected loose mode to accept number as string: %v", err3)
	}
}

// ======================================================================
// SchemaType.String()
// ======================================================================

func TestSchemaType_String(t *testing.T) {
	if queryfy.TypeString.String() == "" {
		t.Error("TypeString.String() should not be empty")
	}
	if queryfy.TypeNumber.String() == "" {
		t.Error("TypeNumber.String() should not be empty")
	}
}

// ======================================================================
// Top-level ValidateAndTransform
// ======================================================================

func TestValidateAndTransform_WithTransformable(t *testing.T) {
	schema := builders.Object().
		Field("name", builders.Transform(
			builders.String().Required(),
		).Add(transformers.Trim()))

	result, err := queryfy.ValidateAndTransform(
		map[string]interface{}{"name": "  Alice  "},
		schema, queryfy.Strict,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]interface{})
	if m["name"] != "Alice" {
		t.Errorf("expected 'Alice', got %q", m["name"])
	}
}

func TestValidateAndTransform_NonTransformable(t *testing.T) {
	// Plain StringSchema doesn't implement TransformableSchema
	// Should fall back to plain validation
	schema := builders.String().Required()
	result, err := queryfy.ValidateAndTransform("hello", schema, queryfy.Strict)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello" {
		t.Errorf("expected 'hello', got %v", result)
	}
}

func TestValidateAndTransform_Error(t *testing.T) {
	schema := builders.String().Required()
	_, err := queryfy.ValidateAndTransform(nil, schema, queryfy.Strict)
	if err == nil {
		t.Fatal("expected error for nil on required field")
	}
}

func TestValidateAndTransformAsync_WithAsync(t *testing.T) {
	schema := builders.Transform(builders.String().Required()).
		Add(transformers.Lowercase()).
		AsyncCustom(func(ctx context.Context, value interface{}) error {
			if value.(string) == "blocked" {
				return fmt.Errorf("blocked")
			}
			return nil
		})

	result, err := queryfy.ValidateAndTransformAsync(
		context.Background(), "ALLOWED", schema, queryfy.Strict,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "allowed" {
		t.Errorf("expected 'allowed', got %v", result)
	}

	_, err2 := queryfy.ValidateAndTransformAsync(
		context.Background(), "BLOCKED", schema, queryfy.Strict,
	)
	if err2 == nil {
		t.Fatal("expected error for blocked value")
	}
}

func TestValidateAndTransformAsync_FallbackToSync(t *testing.T) {
	// Schema with no async validators should fall back to sync
	schema := builders.Transform(builders.String().Required()).
		Add(transformers.Trim())

	result, err := queryfy.ValidateAndTransformAsync(
		context.Background(), "  hello  ", schema, queryfy.Strict,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello" {
		t.Errorf("expected 'hello', got %v", result)
	}
}

func TestValidateAndTransformAsync_NonTransformable(t *testing.T) {
	schema := builders.String().Required()
	result, err := queryfy.ValidateAndTransformAsync(
		context.Background(), "hello", schema, queryfy.Strict,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello" {
		t.Errorf("expected 'hello', got %v", result)
	}
}
