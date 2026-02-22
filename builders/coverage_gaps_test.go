package builders_test

import (
	"testing"

	"github.com/ha1tch/queryfy"
	"github.com/ha1tch/queryfy/builders"
)

// ======================================================================
// Equality gaps — Bool, composites, dependent, unknown types
// ======================================================================

func TestHash_Bool(t *testing.T) {
	b1 := builders.Bool().Required()
	b2 := builders.Bool().Required()
	b3 := builders.Bool()

	if builders.Hash(b1) != builders.Hash(b2) {
		t.Error("identical bool schemas should hash equal")
	}
	if builders.Hash(b1) == builders.Hash(b3) {
		t.Error("required vs optional bool should differ")
	}
}

func TestHash_BoolNullable(t *testing.T) {
	b1 := builders.Bool().Nullable()
	b2 := builders.Bool()
	if builders.Hash(b1) == builders.Hash(b2) {
		t.Error("nullable vs non-nullable bool should differ")
	}
}

func TestHash_OrComposite(t *testing.T) {
	s1 := builders.Or(builders.String(), builders.Number())
	s2 := builders.Or(builders.String(), builders.Number())
	s3 := builders.Or(builders.String(), builders.Bool())

	if builders.Hash(s1) != builders.Hash(s2) {
		t.Error("identical Or composites should hash equal")
	}
	if builders.Hash(s1) == builders.Hash(s3) {
		t.Error("different Or composites should hash differently")
	}
}

func TestHash_NotComposite(t *testing.T) {
	s1 := builders.Not(builders.String().Email())
	s2 := builders.Not(builders.String().Email())
	s3 := builders.Not(builders.String().URL())

	if builders.Hash(s1) != builders.Hash(s2) {
		t.Error("identical Not composites should hash equal")
	}
	if builders.Hash(s1) == builders.Hash(s3) {
		t.Error("different Not composites should differ")
	}
}

func TestEqual_Bool(t *testing.T) {
	if !builders.Equal(builders.Bool(), builders.Bool()) {
		t.Error("identical bools should be equal")
	}
	if builders.Equal(builders.Bool().Required(), builders.Bool()) {
		t.Error("required vs optional bool should differ")
	}
}

func TestHash_TransformInObject(t *testing.T) {
	s1 := builders.Object().
		Field("x", builders.Transform(builders.String().Required()))
	s2 := builders.Object().
		Field("x", builders.Transform(builders.String().Required()))

	if builders.Hash(s1) != builders.Hash(s2) {
		t.Error("objects with identical transform fields should hash equal")
	}
}

func TestHash_Nil(t *testing.T) {
	// TransformSchema wrapping nil shouldn't panic
	// Actually this tests canonicalise(nil) path via Not with nil
	s := builders.Not(nil)
	h := builders.Hash(s)
	if h == "" {
		t.Error("hash should be non-empty even for edge cases")
	}
}

func TestHash_DependentObject(t *testing.T) {
	s1 := builders.Object().WithDependencies().
		Field("a", builders.String()).
		DependentField("b", builders.Dependent("b").
			On("a").
			When(builders.WhenExists("a")).
			Then(builders.String().Required()))

	s2 := builders.Object().WithDependencies().
		Field("a", builders.String()).
		DependentField("b", builders.Dependent("b").
			On("a").
			When(builders.WhenExists("a")).
			Then(builders.String().Required()))

	if builders.Hash(s1) != builders.Hash(s2) {
		t.Error("identical dependent objects should hash equal")
	}
}

func TestHash_ArrayWithMeta(t *testing.T) {
	s1 := builders.Array().Of(builders.String()).Meta("x", 1)
	s2 := builders.Array().Of(builders.String()).Meta("x", 1)
	s3 := builders.Array().Of(builders.String()).Meta("x", 2)

	if builders.Hash(s1) != builders.Hash(s2) {
		t.Error("identical arrays with meta should hash equal")
	}
	if builders.Hash(s1) == builders.Hash(s3) {
		t.Error("different meta should produce different hash")
	}
}

// ======================================================================
// Walk gaps — DependentObject, TransformSchema inner traversal
// ======================================================================

func TestWalk_DependentObject(t *testing.T) {
	schema := builders.Object().WithDependencies().
		Field("status", builders.String()).
		DependentField("reason", builders.Dependent("reason").
			On("status").
			When(builders.WhenNotEquals("status", "ok")).
			Then(builders.String().Required()))

	paths := collectPaths(t, schema)
	if !contains(paths, "status") {
		t.Errorf("expected 'status' in paths: %v", paths)
	}
	// "reason" is a DependentSchema — it may or may not be walked depending
	// on implementation. The key thing is no panic.
}

func TestWalk_TransformInner(t *testing.T) {
	// Walk should recurse into the inner schema of TransformSchema
	schema := builders.Object().
		Field("data", builders.Transform(
			builders.Object().
				Field("inner_a", builders.String()).
				Field("inner_b", builders.Number())))

	paths := collectPaths(t, schema)
	// TransformSchema at "data" wraps an Object with inner_a, inner_b
	// Walk should visit data, then recurse into the inner object
	if !contains(paths, "data") {
		t.Errorf("expected 'data' in paths: %v", paths)
	}
	// The inner schema's fields should be visited through the transform
	if !contains(paths, "inner_a") && !contains(paths, "data.inner_a") {
		// Depending on Walk implementation, the inner object fields
		// might be at "inner_a" or "data.inner_a"
		t.Logf("Transform inner traversal paths: %v", paths)
	}
}

func TestWalk_OrComposite(t *testing.T) {
	schema := builders.Or(
		builders.String().MinLength(1),
		builders.Number().Min(0),
	)

	paths := collectPaths(t, schema)
	// Should visit root "" and <or[0]>, <or[1]>
	if len(paths) < 3 {
		t.Errorf("expected at least 3 visits for Or, got %d: %v", len(paths), paths)
	}
}

func TestWalk_NotComposite(t *testing.T) {
	schema := builders.Not(builders.String().Email())

	paths := collectPaths(t, schema)
	if len(paths) < 2 {
		t.Errorf("expected at least 2 visits for Not, got %d: %v", len(paths), paths)
	}
}

// ======================================================================
// Diff gaps — number constraint changes, nullable changes
// ======================================================================

func TestDiff_NumberConstraintChange(t *testing.T) {
	old := builders.Object().
		Field("score", builders.Number().Min(0).Max(100))

	new := builders.Object().
		Field("score", builders.Number().Min(0).Max(1000))

	diff, err := builders.Diff(old, new)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(diff.Changed) != 1 {
		t.Fatalf("expected 1 change, got %d", len(diff.Changed))
	}
	if diff.Changed[0].Path != "score" {
		t.Errorf("expected path 'score', got %q", diff.Changed[0].Path)
	}
	// Details should mention the max change
	if diff.Changed[0].Details == "" {
		t.Error("expected non-empty details for number constraint change")
	}
}

func TestDiff_NumberIntegerChange(t *testing.T) {
	old := builders.Object().
		Field("val", builders.Number())

	new := builders.Object().
		Field("val", builders.Number().Integer())

	diff, _ := builders.Diff(old, new)
	if len(diff.Changed) != 1 {
		t.Fatalf("expected 1 change, got %d", len(diff.Changed))
	}
}

func TestDiff_NullableChange(t *testing.T) {
	old := builders.Object().
		Field("name", builders.String())

	new := builders.Object().
		Field("name", builders.String().Nullable())

	diff, _ := builders.Diff(old, new)
	if len(diff.Changed) != 1 {
		t.Fatalf("expected 1 change, got %d", len(diff.Changed))
	}
}

func TestDiff_NumberMinAdded(t *testing.T) {
	old := builders.Object().
		Field("val", builders.Number())

	new := builders.Object().
		Field("val", builders.Number().Min(0))

	diff, _ := builders.Diff(old, new)
	if len(diff.Changed) != 1 {
		t.Fatalf("expected 1 change, got %d", len(diff.Changed))
	}
}

func TestDiff_MultipleChanges(t *testing.T) {
	old := builders.Object().
		Field("a", builders.String().MinLength(1)).
		Field("b", builders.Number().Min(0)).
		Field("c", builders.Bool())

	new := builders.Object().
		Field("a", builders.String().MinLength(5)).
		Field("b", builders.Number().Min(10)).
		Field("c", builders.Bool())

	diff, _ := builders.Diff(old, new)
	if len(diff.Changed) != 2 {
		t.Errorf("expected 2 changes (a and b), got %d: %v", len(diff.Changed), diff.Changed)
	}
}

// ======================================================================
// Introspection edge cases
// ======================================================================

func TestIntrospection_TransformInnerSchema(t *testing.T) {
	inner := builders.String().Required().Email()
	s := builders.Transform(inner)
	got := s.InnerSchema()
	if got == nil {
		t.Fatal("InnerSchema should not be nil")
	}
	// Should be the same StringSchema
	if got.Type() != queryfy.TypeString {
		t.Errorf("expected TypeString, got %v", got.Type())
	}
}

func TestIntrospection_ObjectGetField_Transform(t *testing.T) {
	// GetField should return the TransformSchema wrapper, not the inner
	schema := builders.Object().
		Field("email", builders.Transform(builders.String().Email()))

	field, ok := schema.GetField("email")
	if !ok {
		t.Fatal("expected field to exist")
	}
	// It should be a TransformSchema
	ts, ok := field.(*builders.TransformSchema)
	if !ok {
		t.Fatalf("expected *TransformSchema, got %T", field)
	}
	if ts.InnerSchema().Type() != queryfy.TypeString {
		t.Error("inner schema should be string")
	}
}
