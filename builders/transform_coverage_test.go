package builders_test

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

// ======================================================================
// TransformSchema — core validation and transformation
// ======================================================================

func TestTransform_BasicTrim(t *testing.T) {
	s := builders.Transform(builders.String().Required()).
		Add(transformers.Trim())

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	result, err := s.ValidateAndTransform("  hello  ", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

func TestTransform_ChainedTransformers(t *testing.T) {
	s := builders.Transform(builders.String().Required()).
		Add(transformers.Trim()).
		Add(transformers.Lowercase())

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	result, err := s.ValidateAndTransform("  HELLO WORLD  ", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello world" {
		t.Errorf("expected 'hello world', got %q", result)
	}
}

func TestTransform_ValidationFailsBeforeTransform(t *testing.T) {
	s := builders.Transform(builders.String().Email().Required()).
		Add(transformers.Lowercase())

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	_, err := s.ValidateAndTransform("not-an-email", ctx)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestTransform_RequiredInner(t *testing.T) {
	s := builders.Transform(builders.String().Required())
	if !s.IsRequired() {
		t.Error("Transform wrapping Required schema should report IsRequired=true")
	}
}

func TestTransform_RequiredOuter(t *testing.T) {
	s := builders.Transform(builders.String()).Required()
	if !s.IsRequired() {
		t.Error("Transform with .Required() should report IsRequired=true")
	}
}

func TestTransform_Validate(t *testing.T) {
	// Validate (not ValidateAndTransform) should still validate the inner schema
	s := builders.Transform(builders.String().MinLength(3).Required()).
		Add(transformers.Trim())

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	s.Validate("ab", ctx)
	if !ctx.HasErrors() {
		t.Error("expected validation error for short string")
	}
}

func TestTransform_NilValue(t *testing.T) {
	s := builders.Transform(builders.String().Required())
	ctx := queryfy.NewValidationContext(queryfy.Strict)
	_, err := s.ValidateAndTransform(nil, ctx)
	if err == nil {
		t.Error("expected error for nil on required field")
	}
}

func TestTransform_Type(t *testing.T) {
	s := builders.Transform(builders.String())
	if s.Type() != queryfy.TypeString {
		t.Errorf("expected TypeString, got %v", s.Type())
	}

	s2 := builders.Transform(builders.Number())
	if s2.Type() != queryfy.TypeNumber {
		t.Errorf("expected TypeNumber, got %v", s2.Type())
	}
}

// ======================================================================
// TransformSchema — convenience methods on leaf schemas
// ======================================================================

func TestString_Transform(t *testing.T) {
	s := builders.String().Required().Transform(transformers.Trim())
	ctx := queryfy.NewValidationContext(queryfy.Strict)
	result, err := s.ValidateAndTransform("  padded  ", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "padded" {
		t.Errorf("expected 'padded', got %q", result)
	}
}

func TestNumber_Transform(t *testing.T) {
	s := builders.Number().Required().Transform(transformers.Round(2))
	ctx := queryfy.NewValidationContext(queryfy.Strict)
	result, err := s.ValidateAndTransform(3.14159, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 3.14 {
		t.Errorf("expected 3.14, got %v", result)
	}
}

func TestBool_Transform(t *testing.T) {
	// Transform-then-validate order means transformer output must still
	// pass the inner schema. Use a bool->bool transform (e.g., negate).
	negate := func(value interface{}) (interface{}, error) {
		b, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("expected bool")
		}
		return !b, nil
	}
	s := builders.Bool().Required().Transform(negate)
	ctx := queryfy.NewValidationContext(queryfy.Strict)
	result, err := s.ValidateAndTransform(true, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != false {
		t.Errorf("expected false (negated), got %v", result)
	}
}

// ======================================================================
// ObjectSchema.ValidateAndTransform
// ======================================================================

func TestObject_ValidateAndTransform(t *testing.T) {
	schema := builders.Object().
		Field("name", builders.Transform(
			builders.String().Required(),
		).Add(transformers.Trim())).
		Field("email", builders.Transform(
			builders.String().Email().Required(),
		).Add(transformers.Trim()).Add(transformers.Lowercase()))

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	result, err := schema.ValidateAndTransform(map[string]interface{}{
		"name":  "  Alice  ",
		"email": "  ALICE@EXAMPLE.COM  ",
	}, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := result.(map[string]interface{})
	if m["name"] != "Alice" {
		t.Errorf("expected 'Alice', got %q", m["name"])
	}
	if m["email"] != "alice@example.com" {
		t.Errorf("expected 'alice@example.com', got %q", m["email"])
	}
}

func TestObject_ValidateAndTransform_StrictExtra(t *testing.T) {
	schema := builders.Object().
		Field("name", builders.String())

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	_, err := schema.ValidateAndTransform(map[string]interface{}{
		"name":  "Alice",
		"extra": "should fail",
	}, ctx)
	if err == nil {
		t.Fatal("expected error for extra field in strict mode")
	}
}

func TestObject_ValidateAndTransform_LooseExtra(t *testing.T) {
	schema := builders.Object().
		Field("name", builders.String())

	ctx := queryfy.NewValidationContext(queryfy.Loose)
	result, err := schema.ValidateAndTransform(map[string]interface{}{
		"name":  "Alice",
		"extra": "preserved",
	}, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]interface{})
	if m["extra"] != "preserved" {
		t.Error("loose mode should preserve extra fields")
	}
}

func TestObject_ValidateAndTransform_Nil(t *testing.T) {
	schema := builders.Object().Required()
	ctx := queryfy.NewValidationContext(queryfy.Strict)
	_, err := schema.ValidateAndTransform(nil, ctx)
	if err == nil {
		t.Error("expected error for nil on required object")
	}
}

func TestObject_ValidateAndTransform_CustomValidator(t *testing.T) {
	schema := builders.Object().
		Field("password", builders.Transform(
			builders.String().Required(),
		).Add(transformers.Trim())).
		Field("confirm", builders.Transform(
			builders.String().Required(),
		).Add(transformers.Trim())).
		Custom(func(value interface{}) error {
			m := value.(map[string]interface{})
			if m["password"] != m["confirm"] {
				return fmt.Errorf("passwords do not match")
			}
			return nil
		})

	// Matching
	ctx := queryfy.NewValidationContext(queryfy.Strict)
	_, err := schema.ValidateAndTransform(map[string]interface{}{
		"password": "  secret  ",
		"confirm":  "  secret  ",
	}, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Mismatching
	ctx2 := queryfy.NewValidationContext(queryfy.Strict)
	_, err2 := schema.ValidateAndTransform(map[string]interface{}{
		"password": "secret",
		"confirm":  "different",
	}, ctx2)
	if err2 == nil {
		t.Fatal("expected password mismatch error")
	}
}

// ======================================================================
// ArraySchema — MinItems, MaxItems, Items
// ======================================================================

func TestArray_MinItems(t *testing.T) {
	s := builders.Array().MinItems(2).Of(builders.Number())
	expectValid(t, s, []interface{}{1.0, 2.0})
	expectValid(t, s, []interface{}{1.0, 2.0, 3.0})
	expectInvalid(t, s, []interface{}{1.0})
}

func TestArray_MaxItems(t *testing.T) {
	s := builders.Array().MaxItems(3).Of(builders.Number())
	expectValid(t, s, []interface{}{1.0, 2.0, 3.0})
	expectValid(t, s, []interface{}{1.0})
	expectInvalid(t, s, []interface{}{1.0, 2.0, 3.0, 4.0})
}

func TestArray_Items_Alias(t *testing.T) {
	// Items() is an alias for Of()
	s := builders.Array().Items(builders.String().MinLength(1))
	expectValid(t, s, []interface{}{"a", "bc"})
	expectInvalid(t, s, []interface{}{"a", ""})
}

func TestArray_Required(t *testing.T) {
	s := builders.Array().Required()
	expectInvalid(t, s, nil)
}

// ======================================================================
// ArraySchema.ValidateAndTransform
// ======================================================================

func TestArray_ValidateAndTransform(t *testing.T) {
	schema := builders.Array().Of(
		builders.Transform(
			builders.String().Required(),
		).Add(transformers.Trim()).Add(transformers.Lowercase()),
	)

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	result, err := schema.ValidateAndTransform([]interface{}{
		"  HELLO  ",
		"  WORLD  ",
	}, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := result.([]interface{})
	if items[0] != "hello" || items[1] != "world" {
		t.Errorf("expected ['hello', 'world'], got %v", items)
	}
}

func TestArray_ValidateAndTransform_ObjectElements(t *testing.T) {
	schema := builders.Array().Of(
		builders.Object().
			Field("name", builders.Transform(
				builders.String().Required(),
			).Add(transformers.Trim())),
	)

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	result, err := schema.ValidateAndTransform([]interface{}{
		map[string]interface{}{"name": "  Alice  "},
		map[string]interface{}{"name": "  Bob  "},
	}, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := result.([]interface{})
	first := items[0].(map[string]interface{})
	if first["name"] != "Alice" {
		t.Errorf("expected 'Alice', got %q", first["name"])
	}
}

// ======================================================================
// Async validation — within builders package
// ======================================================================

func TestTransform_AsyncCustom(t *testing.T) {
	s := builders.Transform(builders.String().Required()).
		Add(transformers.Lowercase()).
		AsyncCustom(func(ctx context.Context, value interface{}) error {
			if value.(string) == "forbidden" {
				return fmt.Errorf("value is forbidden")
			}
			return nil
		})

	if !s.HasAsyncValidators() {
		t.Error("expected HasAsyncValidators=true")
	}

	// OK
	ctx := queryfy.NewValidationContext(queryfy.Strict)
	result, err := s.ValidateAndTransformAsync(context.Background(), "ALLOWED", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "allowed" {
		t.Errorf("expected 'allowed', got %v", result)
	}

	// Forbidden
	ctx2 := queryfy.NewValidationContext(queryfy.Strict)
	_, err2 := s.ValidateAndTransformAsync(context.Background(), "FORBIDDEN", ctx2)
	if err2 == nil {
		t.Fatal("expected async error for forbidden value")
	}
}

func TestObject_AsyncValidateAndTransform(t *testing.T) {
	schema := builders.Object().
		Field("email", builders.Transform(
			builders.String().Email().Required(),
		).Add(transformers.Lowercase()).
			AsyncCustom(func(ctx context.Context, value interface{}) error {
				if value.(string) == "taken@example.com" {
					return fmt.Errorf("email taken")
				}
				return nil
			})).
		AsyncCustom(func(ctx context.Context, value interface{}) error {
			// Object-level async
			return nil
		})

	if !schema.HasAsyncValidators() {
		t.Error("expected HasAsyncValidators=true")
	}

	// OK
	ctx := queryfy.NewValidationContext(queryfy.Strict)
	result, err := schema.ValidateAndTransformAsync(context.Background(), map[string]interface{}{
		"email": "NEW@EXAMPLE.COM",
	}, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]interface{})
	if m["email"] != "new@example.com" {
		t.Errorf("expected 'new@example.com', got %v", m["email"])
	}

	// Taken
	ctx2 := queryfy.NewValidationContext(queryfy.Strict)
	_, err2 := schema.ValidateAndTransformAsync(context.Background(), map[string]interface{}{
		"email": "TAKEN@EXAMPLE.COM",
	}, ctx2)
	if err2 == nil {
		t.Fatal("expected async error for taken email")
	}
}

func TestArray_AsyncValidateAndTransform(t *testing.T) {
	schema := builders.Array().Items(
		builders.Transform(builders.String().Required()).
			Add(transformers.Lowercase()).
			AsyncCustom(func(ctx context.Context, value interface{}) error {
				if value.(string) == "blocked" {
					return fmt.Errorf("blocked value")
				}
				return nil
			}),
	).AsyncCustom(func(ctx context.Context, value interface{}) error {
		return nil // array-level check
	})

	if !schema.HasAsyncValidators() {
		t.Error("expected HasAsyncValidators=true")
	}

	// OK
	ctx := queryfy.NewValidationContext(queryfy.Strict)
	result, err := schema.ValidateAndTransformAsync(context.Background(), []interface{}{"OK", "FINE"}, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	items := result.([]interface{})
	if items[0] != "ok" {
		t.Errorf("expected 'ok', got %v", items[0])
	}

	// Blocked
	ctx2 := queryfy.NewValidationContext(queryfy.Strict)
	_, err2 := schema.ValidateAndTransformAsync(context.Background(), []interface{}{"BLOCKED"}, ctx2)
	if err2 == nil {
		t.Fatal("expected async error for blocked value")
	}
}

func TestAsync_ContextCancellation(t *testing.T) {
	schema := builders.Transform(builders.String().Required()).
		AsyncCustom(func(ctx context.Context, value interface{}) error {
			<-ctx.Done()
			return ctx.Err()
		})

	goCtx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	_, err := schema.ValidateAndTransformAsync(goCtx, "test", ctx)
	if err == nil {
		t.Fatal("expected cancellation error")
	}
}

func TestAsync_SyncFailSkipsAsync(t *testing.T) {
	asyncCalled := false
	schema := builders.Transform(builders.String().Email().Required()).
		AsyncCustom(func(ctx context.Context, value interface{}) error {
			asyncCalled = true
			return nil
		})

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	schema.ValidateAndTransformAsync(context.Background(), "invalid", ctx)
	if asyncCalled {
		t.Error("async should not run when sync validation fails")
	}
}

func TestHasAsyncValidators_Negative(t *testing.T) {
	s := builders.Transform(builders.String())
	if s.HasAsyncValidators() {
		t.Error("plain transform should have no async validators")
	}

	o := builders.Object().Field("x", builders.String())
	if o.HasAsyncValidators() {
		t.Error("plain object should have no async validators")
	}

	a := builders.Array().Of(builders.String())
	if a.HasAsyncValidators() {
		t.Error("plain array should have no async validators")
	}
}

// ======================================================================
// Dependent conditions — remaining untested ones
// ======================================================================

func TestDependent_WhenNotEquals(t *testing.T) {
	schema := builders.Object().WithDependencies().
		Field("status", builders.String().Required()).
		DependentField("reason",
			builders.Dependent("reason").
				On("status").
				When(builders.WhenNotEquals("status", "approved")).
				Then(builders.String().Required()))

	// Approved — no reason needed
	expectValid(t, schema, map[string]interface{}{"status": "approved"})

	// Rejected — reason required
	expectValid(t, schema, map[string]interface{}{"status": "rejected", "reason": "too late"})
	expectInvalid(t, schema, map[string]interface{}{"status": "rejected"})
}

func TestDependent_WhenNotExists(t *testing.T) {
	schema := builders.Object().WithDependencies().
		Field("email", builders.String().Optional()).
		DependentField("phone",
			builders.Dependent("phone").
				On("email").
				When(builders.WhenNotExists("email")).
				Then(builders.String().Required()))

	// Email present — phone not required
	expectValid(t, schema, map[string]interface{}{"email": "a@b.com"})

	// No email — phone required
	expectValid(t, schema, map[string]interface{}{"phone": "555-1234"})
	expectInvalid(t, schema, map[string]interface{}{})
}

func TestDependent_WhenIn(t *testing.T) {
	schema := builders.Object().WithDependencies().
		Field("country", builders.String().Required()).
		DependentField("state",
			builders.Dependent("state").
				On("country").
				When(builders.WhenIn("country", "US", "CA")).
				Then(builders.String().Required()))

	expectValid(t, schema, map[string]interface{}{"country": "US", "state": "TX"})
	expectValid(t, schema, map[string]interface{}{"country": "CA", "state": "ON"})
	expectValid(t, schema, map[string]interface{}{"country": "UK"}) // no state needed
	expectInvalid(t, schema, map[string]interface{}{"country": "US"}) // state needed
}

func TestDependent_WhenLessThan(t *testing.T) {
	schema := builders.Object().WithDependencies().
		Field("score", builders.Number().Required()).
		DependentField("remediation",
			builders.Dependent("remediation").
				On("score").
				When(builders.WhenLessThan("score", 50)).
				Then(builders.String().Required()))

	expectValid(t, schema, map[string]interface{}{"score": 80.0})
	expectValid(t, schema, map[string]interface{}{"score": 30.0, "remediation": "retake"})
	expectInvalid(t, schema, map[string]interface{}{"score": 30.0})
}

func TestDependent_WhenFalse(t *testing.T) {
	schema := builders.Object().WithDependencies().
		Field("automated", builders.Bool().Required()).
		DependentField("operator",
			builders.Dependent("operator").
				On("automated").
				When(builders.WhenFalse("automated")).
				Then(builders.String().Required()))

	expectValid(t, schema, map[string]interface{}{"automated": true})
	expectValid(t, schema, map[string]interface{}{"automated": false, "operator": "human"})
	expectInvalid(t, schema, map[string]interface{}{"automated": false})
}

func TestDependent_WhenAll(t *testing.T) {
	schema := builders.Object().WithDependencies().
		Field("premium", builders.Bool().Required()).
		Field("country", builders.String().Required()).
		DependentField("taxId",
			builders.Dependent("taxId").
				On("premium", "country").
				When(builders.WhenAll(
					builders.WhenTrue("premium"),
					builders.WhenEquals("country", "US"),
				)).
				Then(builders.String().Required()))

	// Both conditions met
	expectValid(t, schema, map[string]interface{}{
		"premium": true, "country": "US", "taxId": "123",
	})
	expectInvalid(t, schema, map[string]interface{}{
		"premium": true, "country": "US",
	})
	// Only one condition met
	expectValid(t, schema, map[string]interface{}{
		"premium": false, "country": "US",
	})
	expectValid(t, schema, map[string]interface{}{
		"premium": true, "country": "UK",
	})
}

func TestDependent_WhenAny(t *testing.T) {
	schema := builders.Object().WithDependencies().
		Field("admin", builders.Bool().Required()).
		Field("superuser", builders.Bool().Required()).
		DependentField("mfaCode",
			builders.Dependent("mfaCode").
				On("admin", "superuser").
				When(builders.WhenAny(
					builders.WhenTrue("admin"),
					builders.WhenTrue("superuser"),
				)).
				Then(builders.String().Required()))

	// Either condition triggers
	expectValid(t, schema, map[string]interface{}{
		"admin": true, "superuser": false, "mfaCode": "123456",
	})
	expectInvalid(t, schema, map[string]interface{}{
		"admin": true, "superuser": false,
	})
	expectInvalid(t, schema, map[string]interface{}{
		"admin": false, "superuser": true,
	})
	// Neither triggers
	expectValid(t, schema, map[string]interface{}{
		"admin": false, "superuser": false,
	})
}

func TestDependent_RequiredWhen(t *testing.T) {
	dep := builders.RequiredWhen(
		builders.WhenTrue("active"),
		builders.String(),
	)
	// Just verify it builds without panic and produces a DependentSchema
	if dep == nil {
		t.Error("RequiredWhen returned nil")
	}
}

func TestDependent_RequiredUnless(t *testing.T) {
	dep := builders.RequiredUnless(
		builders.WhenTrue("exempt"),
		builders.String(),
	)
	if dep == nil {
		t.Error("RequiredUnless returned nil")
	}
}

// ======================================================================
// Composite — compositeErrorMessage path
// ======================================================================

func TestOr_ErrorMessage(t *testing.T) {
	// Test that Or produces a meaningful error when no schema matches
	s := builders.Or(
		builders.Number().Min(100),
		builders.String().Email(),
	)
	ctx := validate(t, s, "nope", queryfy.Strict)
	if !ctx.HasErrors() {
		t.Error("expected error from Or with no match")
	}
	errStr := ctx.Error().Error()
	if !strings.Contains(errStr, "none of the schemas") && !strings.Contains(errStr, "validation") {
		t.Errorf("error message should indicate no schema matched, got: %s", errStr)
	}
}
