package builders_test

import (
	"testing"

	"github.com/ha1tch/queryfy/builders"
)

// ======================================================================
// 3.3 Schema Diff
// ======================================================================

func TestDiff_Identical(t *testing.T) {
	old := builders.Object().
		Field("name", builders.String().Required()).
		Field("age", builders.Number())

	new := builders.Object().
		Field("name", builders.String().Required()).
		Field("age", builders.Number())

	diff, err := builders.Diff(old, new)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff.HasChanges() {
		t.Error("identical schemas should produce no changes")
	}
}

func TestDiff_AddedField(t *testing.T) {
	old := builders.Object().
		Field("name", builders.String())

	new := builders.Object().
		Field("name", builders.String()).
		Field("email", builders.String().Email())

	diff, err := builders.Diff(old, new)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(diff.Added) != 1 || diff.Added[0] != "email" {
		t.Errorf("expected [email] added, got %v", diff.Added)
	}
	if len(diff.Removed) != 0 {
		t.Errorf("expected no removals, got %v", diff.Removed)
	}
}

func TestDiff_RemovedField(t *testing.T) {
	old := builders.Object().
		Field("name", builders.String()).
		Field("legacy", builders.String())

	new := builders.Object().
		Field("name", builders.String())

	diff, err := builders.Diff(old, new)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(diff.Removed) != 1 || diff.Removed[0] != "legacy" {
		t.Errorf("expected [legacy] removed, got %v", diff.Removed)
	}
}

func TestDiff_ChangedType(t *testing.T) {
	old := builders.Object().
		Field("value", builders.String())

	new := builders.Object().
		Field("value", builders.Number())

	diff, err := builders.Diff(old, new)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(diff.Changed) != 1 {
		t.Fatalf("expected 1 change, got %d", len(diff.Changed))
	}
	if diff.Changed[0].Path != "value" {
		t.Errorf("expected path 'value', got %q", diff.Changed[0].Path)
	}
	if diff.Changed[0].Details == "" {
		t.Error("expected non-empty details")
	}
}

func TestDiff_ChangedConstraints(t *testing.T) {
	old := builders.Object().
		Field("name", builders.String().MinLength(1).MaxLength(50))

	new := builders.Object().
		Field("name", builders.String().MinLength(1).MaxLength(255))

	diff, err := builders.Diff(old, new)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(diff.Changed) != 1 {
		t.Fatalf("expected 1 change, got %d", len(diff.Changed))
	}
	if diff.Changed[0].Path != "name" {
		t.Errorf("expected path 'name', got %q", diff.Changed[0].Path)
	}
}

func TestDiff_BecameRequired(t *testing.T) {
	old := builders.Object().
		Field("email", builders.String())

	new := builders.Object().
		Field("email", builders.String().Required())

	diff, err := builders.Diff(old, new)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(diff.Changed) != 1 {
		t.Fatalf("expected 1 change, got %d", len(diff.Changed))
	}
}

func TestDiff_NestedChanges(t *testing.T) {
	old := builders.Object().
		Field("user", builders.Object().
			Field("name", builders.String()).
			Field("email", builders.String()))

	new := builders.Object().
		Field("user", builders.Object().
			Field("name", builders.String()).
			Field("email", builders.String().Email()).
			Field("phone", builders.String()))

	diff, err := builders.Diff(old, new)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// user.phone added, user.email changed (format added)
	if len(diff.Added) != 1 {
		t.Errorf("expected 1 addition, got %v", diff.Added)
	}
	if len(diff.Changed) != 1 {
		t.Errorf("expected 1 change, got %v", diff.Changed)
	}
}

func TestDiff_ArrayElementChange(t *testing.T) {
	old := builders.Object().
		Field("tags", builders.Array().Of(builders.String()))

	new := builders.Object().
		Field("tags", builders.Array().Of(builders.String().MinLength(1)))

	diff, err := builders.Diff(old, new)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// tags[*] should show as changed
	if len(diff.Changed) == 0 {
		t.Error("expected change in array element schema")
	}
}

func TestDiff_NonObject(t *testing.T) {
	old := builders.String().MinLength(1)
	new := builders.String().MinLength(5)

	diff, err := builders.Diff(old, new)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Non-objects get a single root change
	if !diff.HasChanges() {
		t.Error("expected changes for different non-object schemas")
	}
}

func TestDiff_Combined(t *testing.T) {
	old := builders.Object().
		Field("id", builders.String().Required()).
		Field("name", builders.String().MinLength(1)).
		Field("deprecated", builders.String())

	new := builders.Object().
		Field("id", builders.String().Required()).
		Field("name", builders.String().MinLength(1).MaxLength(255)).
		Field("email", builders.String().Email())

	diff, err := builders.Diff(old, new)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(diff.Added) != 1 || diff.Added[0] != "email" {
		t.Errorf("expected email added, got %v", diff.Added)
	}
	if len(diff.Removed) != 1 || diff.Removed[0] != "deprecated" {
		t.Errorf("expected deprecated removed, got %v", diff.Removed)
	}
	if len(diff.Changed) != 1 || diff.Changed[0].Path != "name" {
		t.Errorf("expected name changed, got %v", diff.Changed)
	}
}

func TestDiff_HasChanges(t *testing.T) {
	same := builders.String()
	diff, _ := builders.Diff(same, same)
	if diff.HasChanges() {
		t.Error("same schema should have no changes")
	}
}
