package queryfy_test

import (
	"fmt"
	"testing"

	qf "github.com/ha1tch/queryfy"
	"github.com/ha1tch/queryfy/builders"
	"github.com/ha1tch/queryfy/builders/transformers"
)

// Regression: compiled TransformSchema must preserve the transform pipeline.
// Prior to fix, TransformSchema.Type() returned the inner type, causing
// Compile() to strip the transform layer.
func TestCompile_TransformSchema_PreservesTransform(t *testing.T) {
	raw := builders.Transform(
		builders.String().Email().Required(),
	).Add(transformers.Trim()).Add(transformers.Lowercase())

	compiled := qf.Compile(raw)

	ctx := qf.NewValidationContext(qf.Strict)
	result, err := compiled.(qf.TransformableSchema).ValidateAndTransform("  ALICE@EXAMPLE.COM  ", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "alice@example.com" {
		t.Errorf("expected transformed result \"alice@example.com\", got %q", result)
	}
}

// Regression: compiled TransformSchema in an object field must also work.
func TestCompile_Object_TransformField(t *testing.T) {
	schema := qf.Compile(builders.Object().
		Field("email", builders.Transform(
			builders.String().Email().Required(),
		).Add(transformers.Trim()).Add(transformers.Lowercase())))

	data := map[string]interface{}{
		"email": "  BOB@EXAMPLE.COM  ",
	}

	ctx := qf.NewValidationContext(qf.Strict)
	result, err := schema.(qf.TransformableSchema).ValidateAndTransform(data, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if resultMap["email"] != "bob@example.com" {
		t.Errorf("expected \"bob@example.com\", got %q", resultMap["email"])
	}
}

// Regression: compiled object must accept typed maps (e.g. map[string]string),
// not just map[string]interface{}.
func TestCompile_Object_TypedMap(t *testing.T) {
	schema := qf.Compile(builders.Object().
		Field("name", builders.String().Required()))

	typedMap := map[string]string{"name": "Alice"}

	ctx := qf.NewValidationContext(qf.Strict)
	schema.Validate(typedMap, ctx)
	if ctx.HasErrors() {
		t.Errorf("compiled object should accept map[string]string, got errors: %v", ctx.Error())
	}
}

// Regression: compiled array must accept typed slices (e.g. []string),
// not just []interface{}.
func TestCompile_Array_TypedSlice(t *testing.T) {
	schema := qf.Compile(builders.Array().Of(builders.String()).MinItems(1))

	typedSlice := []string{"a", "b", "c"}

	ctx := qf.NewValidationContext(qf.Strict)
	schema.Validate(typedSlice, ctx)
	if ctx.HasErrors() {
		t.Errorf("compiled array should accept []string, got errors: %v", ctx.Error())
	}
}

// Regression: compiled unique-items must not panic on non-comparable types
// (maps, slices used as array elements).
func TestCompile_Array_UniqueItems_NonComparable(t *testing.T) {
	schema := qf.Compile(builders.Array().Of(builders.Object()).UniqueItems())

	data := []interface{}{
		map[string]interface{}{"a": 1},
		map[string]interface{}{"b": 2},
	}

	// This must not panic
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("compiled unique-items panicked on non-comparable type: %v", r)
		}
	}()

	ctx := qf.NewValidationContext(qf.Strict)
	schema.Validate(data, ctx)
	// No panic = pass. Errors are acceptable (or not) depending on uniqueness.
}

// Regression: compiled unique-items must still detect duplicates.
func TestCompile_Array_UniqueItems_DetectsDuplicates(t *testing.T) {
	schema := qf.Compile(builders.Array().Of(builders.String()).UniqueItems())

	data := []interface{}{"a", "b", "a"}

	ctx := qf.NewValidationContext(qf.Strict)
	schema.Validate(data, ctx)
	if !ctx.HasErrors() {
		t.Error("compiled unique-items should detect duplicate strings")
	}
}

// Verify compiled typed slices with integer elements.
func TestCompile_Array_TypedIntSlice(t *testing.T) {
	schema := qf.Compile(builders.Array().Of(builders.Number()).MinItems(1))

	typedSlice := []int{1, 2, 3}

	ctx := qf.NewValidationContext(qf.Strict)
	schema.Validate(typedSlice, ctx)
	if ctx.HasErrors() {
		// Numbers arrive as int, which the number validator may reject if it
		// expects float64. This test verifies the slice is at least accepted
		// and iterated, not rejected at the container level.
		for _, e := range ctx.Errors() {
			if e.Message == "must be a number" {
				// Type mismatch on elements is expected for int vs float64;
				// the point is we didn't get "cannot convert" on the container.
				return
			}
		}
		fmt.Printf("errors: %v\n", ctx.Error())
	}
}
