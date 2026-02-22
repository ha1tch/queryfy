package queryfy_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ha1tch/queryfy"
	"github.com/ha1tch/queryfy/builders"
	"github.com/ha1tch/queryfy/builders/transformers"
)

// simulateDBLookup simulates a database uniqueness check.
func simulateDBLookup(existing map[string]bool) queryfy.AsyncValidatorFunc {
	return func(ctx context.Context, value interface{}) error {
		// Simulate I/O delay
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Millisecond):
		}

		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string for uniqueness check")
		}
		if existing[str] {
			return fmt.Errorf("value %q already exists", str)
		}
		return nil
	}
}

// TestTransformSchemaAsync_Basic tests async validation on a TransformSchema.
func TestTransformSchemaAsync_Basic(t *testing.T) {
	existingEmails := map[string]bool{
		"taken@example.com": true,
	}

	schema := builders.Transform(
		builders.String().Email().Required(),
	).Add(transformers.Trim()).
		Add(transformers.Lowercase()).
		AsyncCustom(simulateDBLookup(existingEmails))

	// Valid unique email
	ctx := queryfy.NewValidationContext(queryfy.Strict)
	result, err := schema.ValidateAndTransformAsync(context.Background(), "  NEW@EXAMPLE.COM  ", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "new@example.com" {
		t.Errorf("expected 'new@example.com', got %q", result)
	}

	// Taken email — async validator should fail
	ctx2 := queryfy.NewValidationContext(queryfy.Strict)
	_, err2 := schema.ValidateAndTransformAsync(context.Background(), "  TAKEN@EXAMPLE.COM  ", ctx2)
	if err2 == nil {
		t.Fatal("expected async validation error for taken email")
	}
	if !strings.Contains(err2.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err2)
	}
}

// TestTransformSchemaAsync_SyncFailSkipsAsync tests that async validators
// are not invoked when sync validation fails.
func TestTransformSchemaAsync_SyncFailSkipsAsync(t *testing.T) {
	asyncCalled := false

	schema := builders.Transform(
		builders.String().Email().Required(),
	).Add(transformers.Trim()).
		AsyncCustom(func(ctx context.Context, value interface{}) error {
			asyncCalled = true
			return nil
		})

	// Invalid email — sync should fail, async should never run
	ctx := queryfy.NewValidationContext(queryfy.Strict)
	_, err := schema.ValidateAndTransformAsync(context.Background(), "not-an-email", ctx)
	if err == nil {
		t.Fatal("expected sync validation error")
	}
	if asyncCalled {
		t.Error("async validator was called despite sync validation failure")
	}
}

// TestTransformSchemaAsync_ContextCancellation tests that async validators
// respect context cancellation.
func TestTransformSchemaAsync_ContextCancellation(t *testing.T) {
	schema := builders.Transform(
		builders.String().Required(),
	).AsyncCustom(func(ctx context.Context, value interface{}) error {
		// This would block forever without cancellation
		<-ctx.Done()
		return ctx.Err()
	})

	goCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	_, err := schema.ValidateAndTransformAsync(goCtx, "test", ctx)
	if err == nil {
		t.Fatal("expected cancellation error")
	}
}

// TestObjectAsync_FieldLevel tests async validation on individual fields
// within an object.
func TestObjectAsync_FieldLevel(t *testing.T) {
	existingUsernames := map[string]bool{
		"admin": true,
		"root":  true,
	}

	schema := builders.Object().
		Field("username", builders.Transform(
			builders.String().MinLength(3).Required(),
		).Add(transformers.Trim()).
			Add(transformers.Lowercase()).
			AsyncCustom(simulateDBLookup(existingUsernames))).
		Field("email", builders.Transform(
			builders.String().Email().Required(),
		).Add(transformers.Trim()).Add(transformers.Lowercase()))

	// Valid — unique username
	data := map[string]interface{}{
		"username": "  NewUser  ",
		"email":    "  NEW@EXAMPLE.COM  ",
	}

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	result, err := schema.ValidateAndTransformAsync(context.Background(), data, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultMap := result.(map[string]interface{})
	if resultMap["username"] != "newuser" {
		t.Errorf("expected 'newuser', got %q", resultMap["username"])
	}

	// Taken username — async should fail
	data2 := map[string]interface{}{
		"username": "  Admin  ",
		"email":    "  ADMIN@EXAMPLE.COM  ",
	}

	ctx2 := queryfy.NewValidationContext(queryfy.Strict)
	_, err2 := schema.ValidateAndTransformAsync(context.Background(), data2, ctx2)
	if err2 == nil {
		t.Fatal("expected async validation error for taken username")
	}
	if !strings.Contains(err2.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err2)
	}
}

// TestObjectAsync_ObjectLevel tests async validators registered on the
// object schema itself (cross-field async checks).
func TestObjectAsync_ObjectLevel(t *testing.T) {
	// Async cross-field validator: check that username + email combo is unique
	existingCombos := map[string]bool{
		"alice:alice@example.com": true,
	}

	schema := builders.Object().
		Field("username", builders.Transform(
			builders.String().Required(),
		).Add(transformers.Lowercase())).
		Field("email", builders.Transform(
			builders.String().Email().Required(),
		).Add(transformers.Lowercase())).
		AsyncCustom(func(ctx context.Context, value interface{}) error {
			m := value.(map[string]interface{})
			combo := fmt.Sprintf("%s:%s", m["username"], m["email"])
			if existingCombos[combo] {
				return fmt.Errorf("username+email combination already registered")
			}
			return nil
		})

	// Unique combo
	ctx := queryfy.NewValidationContext(queryfy.Strict)
	_, err := schema.ValidateAndTransformAsync(context.Background(), map[string]interface{}{
		"username": "Bob",
		"email":    "bob@example.com",
	}, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Taken combo
	ctx2 := queryfy.NewValidationContext(queryfy.Strict)
	_, err2 := schema.ValidateAndTransformAsync(context.Background(), map[string]interface{}{
		"username": "Alice",
		"email":    "alice@example.com",
	}, ctx2)
	if err2 == nil {
		t.Fatal("expected async error for duplicate combo")
	}
	if !strings.Contains(err2.Error(), "already registered") {
		t.Errorf("expected 'already registered' error, got: %v", err2)
	}
}

// TestObjectAsync_SyncFailSkipsAsync tests that object-level async validators
// are skipped when sync validation fails.
func TestObjectAsync_SyncFailSkipsAsync(t *testing.T) {
	asyncCalled := false

	schema := builders.Object().
		Field("name", builders.String().Required()).
		AsyncCustom(func(ctx context.Context, value interface{}) error {
			asyncCalled = true
			return nil
		})

	// Missing required field — sync fails
	ctx := queryfy.NewValidationContext(queryfy.Strict)
	_, err := schema.ValidateAndTransformAsync(context.Background(), map[string]interface{}{}, ctx)
	if err == nil {
		t.Fatal("expected sync validation error")
	}
	if asyncCalled {
		t.Error("object-level async validator was called despite sync failure")
	}
}

// TestObjectAsync_MultipleAsyncValidators tests that multiple async validators
// run sequentially and all errors are collected.
func TestObjectAsync_MultipleAsyncValidators(t *testing.T) {
	schema := builders.Object().
		Field("code", builders.String().Required()).
		AsyncCustom(func(ctx context.Context, value interface{}) error {
			return fmt.Errorf("check-1 failed")
		}).
		AsyncCustom(func(ctx context.Context, value interface{}) error {
			return fmt.Errorf("check-2 failed")
		})

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	_, err := schema.ValidateAndTransformAsync(context.Background(), map[string]interface{}{
		"code": "ABC",
	}, ctx)
	if err == nil {
		t.Fatal("expected async errors")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "check-1 failed") {
		t.Error("missing check-1 error")
	}
	if !strings.Contains(errStr, "check-2 failed") {
		t.Error("missing check-2 error")
	}
}

// TestArrayAsync_ElementLevel tests async validation on array elements.
func TestArrayAsync_ElementLevel(t *testing.T) {
	existingTags := map[string]bool{
		"reserved": true,
	}

	schema := builders.Array().Items(
		builders.Transform(
			builders.String().Required(),
		).Add(transformers.Lowercase()).
			AsyncCustom(simulateDBLookup(existingTags)),
	).Required()

	// Valid tags
	ctx := queryfy.NewValidationContext(queryfy.Strict)
	result, err := schema.ValidateAndTransformAsync(context.Background(), []interface{}{"Go", "Rust"}, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	items := result.([]interface{})
	if items[0] != "go" || items[1] != "rust" {
		t.Errorf("expected ['go', 'rust'], got %v", items)
	}

	// One reserved tag
	ctx2 := queryfy.NewValidationContext(queryfy.Strict)
	_, err2 := schema.ValidateAndTransformAsync(context.Background(), []interface{}{"Go", "RESERVED"}, ctx2)
	if err2 == nil {
		t.Fatal("expected async error for reserved tag")
	}
	if !strings.Contains(err2.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err2)
	}
}

// TestHasAsyncValidators tests the HasAsyncValidators introspection method.
func TestHasAsyncValidators(t *testing.T) {
	// No async
	syncSchema := builders.Object().
		Field("name", builders.String().Required())
	if syncSchema.HasAsyncValidators() {
		t.Error("expected no async validators on sync-only schema")
	}

	// Async on field
	fieldAsync := builders.Object().
		Field("email", builders.Transform(
			builders.String().Required(),
		).AsyncCustom(func(ctx context.Context, value interface{}) error {
			return nil
		}))
	if !fieldAsync.HasAsyncValidators() {
		t.Error("expected async validators detected on field schema")
	}

	// Async on object level
	objAsync := builders.Object().
		Field("name", builders.String()).
		AsyncCustom(func(ctx context.Context, value interface{}) error {
			return nil
		})
	if !objAsync.HasAsyncValidators() {
		t.Error("expected async validators detected on object schema")
	}

	// TransformSchema
	transformNoAsync := builders.Transform(builders.String())
	if transformNoAsync.HasAsyncValidators() {
		t.Error("expected no async validators on plain transform schema")
	}

	transformWithAsync := builders.Transform(builders.String()).
		AsyncCustom(func(ctx context.Context, value interface{}) error {
			return nil
		})
	if !transformWithAsync.HasAsyncValidators() {
		t.Error("expected async validators on transform schema with AsyncCustom")
	}
}

// TestObjectAsync_ContextCancellationMidway tests that context cancellation
// stops async validation between field validators.
func TestObjectAsync_ContextCancellationMidway(t *testing.T) {
	secondCalled := false

	schema := builders.Object().
		AsyncCustom(func(ctx context.Context, value interface{}) error {
			// First validator succeeds but takes a moment
			time.Sleep(5 * time.Millisecond)
			return nil
		}).
		AsyncCustom(func(ctx context.Context, value interface{}) error {
			secondCalled = true
			return nil
		})

	goCtx, cancel := context.WithTimeout(context.Background(), 3*time.Millisecond)
	defer cancel()

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	_, err := schema.ValidateAndTransformAsync(goCtx, map[string]interface{}{
		"x": 1,
	}, ctx)

	// Should have a cancellation error
	if err == nil {
		t.Fatal("expected cancellation error")
	}
	if secondCalled {
		t.Error("second async validator should not have been called after cancellation")
	}
}

// TestObjectAsync_NestedObjectWithAsync tests async validation through
// nested object structures.
func TestObjectAsync_NestedObjectWithAsync(t *testing.T) {
	existingEmails := map[string]bool{
		"taken@example.com": true,
	}

	schema := builders.Object().
		Field("user", builders.Object().
			Field("email", builders.Transform(
				builders.String().Email().Required(),
			).Add(transformers.Lowercase()).
				AsyncCustom(simulateDBLookup(existingEmails))))

	// Unique email in nested object
	ctx := queryfy.NewValidationContext(queryfy.Strict)
	result, err := schema.ValidateAndTransformAsync(context.Background(), map[string]interface{}{
		"user": map[string]interface{}{
			"email": "NEW@EXAMPLE.COM",
		},
	}, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	user := result.(map[string]interface{})["user"].(map[string]interface{})
	if user["email"] != "new@example.com" {
		t.Errorf("expected 'new@example.com', got %q", user["email"])
	}

	// Taken email in nested object
	ctx2 := queryfy.NewValidationContext(queryfy.Strict)
	_, err2 := schema.ValidateAndTransformAsync(context.Background(), map[string]interface{}{
		"user": map[string]interface{}{
			"email": "TAKEN@EXAMPLE.COM",
		},
	}, ctx2)
	if err2 == nil {
		t.Fatal("expected async error for taken email in nested object")
	}
}

// TestSyncIgnoresAsync verifies that sync methods completely ignore
// async validators — the core contract.
func TestSyncIgnoresAsync(t *testing.T) {
	asyncCalled := false

	schema := builders.Object().
		Field("email", builders.Transform(
			builders.String().Email().Required(),
		).Add(transformers.Lowercase()).
			AsyncCustom(func(ctx context.Context, value interface{}) error {
				asyncCalled = true
				return fmt.Errorf("should never see this")
			})).
		AsyncCustom(func(ctx context.Context, value interface{}) error {
			asyncCalled = true
			return fmt.Errorf("should never see this either")
		})

	data := map[string]interface{}{
		"email": "TEST@EXAMPLE.COM",
	}

	// Sync Validate
	ctx := queryfy.NewValidationContext(queryfy.Strict)
	schema.Validate(data, ctx)
	if asyncCalled {
		t.Error("async validator was invoked during sync Validate")
	}

	// Sync ValidateAndTransform
	asyncCalled = false
	ctx2 := queryfy.NewValidationContext(queryfy.Strict)
	result, err := schema.ValidateAndTransform(data, ctx2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if asyncCalled {
		t.Error("async validator was invoked during sync ValidateAndTransform")
	}

	resultMap := result.(map[string]interface{})
	if resultMap["email"] != "test@example.com" {
		t.Errorf("expected 'test@example.com', got %q", resultMap["email"])
	}
}
