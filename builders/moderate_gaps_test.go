package builders_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ha1tch/queryfy"
	"github.com/ha1tch/queryfy/builders"
	"github.com/ha1tch/queryfy/builders/transformers"
)

// ======================================================================
// Object ValidateAndTransformAsync — uncovered paths
// ======================================================================

func TestObjectVaTAsync_FieldTransformAndAsyncValidator(t *testing.T) {
	// Covers: async field-level traversal, object-level async validators,
	// sync+async combined flow
	schema := builders.Object().
		Field("email", builders.Transform(builders.String().Required().Email()).
			Add(transformers.Lowercase())).
		Field("name", builders.String().Required()).
		AsyncCustom(func(ctx context.Context, value interface{}) error {
			m := value.(map[string]interface{})
			if m["name"] == "blocked" {
				return fmt.Errorf("user is blocked")
			}
			return nil
		})

	// Happy path
	ctx1 := queryfy.NewValidationContext(queryfy.Strict)
	result, err := schema.ValidateAndTransformAsync(context.Background(),
		map[string]interface{}{"email": "USER@EXAMPLE.COM", "name": "Alice"}, ctx1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]interface{})
	if m["email"] != "user@example.com" {
		t.Errorf("expected lowercase email, got %q", m["email"])
	}

	// Async validator rejects
	ctx2 := queryfy.NewValidationContext(queryfy.Strict)
	_, err2 := schema.ValidateAndTransformAsync(context.Background(),
		map[string]interface{}{"email": "x@y.com", "name": "blocked"}, ctx2)
	if err2 == nil {
		t.Error("expected async validator error")
	}
}

func TestObjectVaTAsync_SyncFailSkipsAsync(t *testing.T) {
	// Covers: sync validation fails → async not called
	asyncCalled := false
	schema := builders.Object().
		Field("id", builders.String().Required()).
		AsyncCustom(func(ctx context.Context, value interface{}) error {
			asyncCalled = true
			return nil
		})

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	// Missing required field "id"
	schema.ValidateAndTransformAsync(context.Background(),
		map[string]interface{}{}, ctx)
	if asyncCalled {
		t.Error("async should not be called when sync validation fails")
	}
}

func TestObjectVaTAsync_ContextCancellation_BeforeAsync(t *testing.T) {
	// Covers: context cancelled before async phase
	schema := builders.Object().
		Field("name", builders.String()).
		AsyncCustom(func(ctx context.Context, value interface{}) error {
			return nil
		})

	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	_, err := schema.ValidateAndTransformAsync(cancelCtx,
		map[string]interface{}{"name": "Alice"}, ctx)
	if err == nil {
		t.Error("expected cancellation error")
	}
}

func TestObjectVaTAsync_ContextCancellation_DuringFieldAsync(t *testing.T) {
	// Covers: context cancelled between field async validators
	// Note: map iteration order is non-deterministic, so we can't assert
	// which field runs first. We just verify cancellation is handled.
	cancelCtx, cancel := context.WithCancel(context.Background())

	schema := builders.Object().
		Field("a", builders.Transform(builders.String()).
			AsyncCustom(func(ctx context.Context, value interface{}) error {
				cancel()
				return nil
			})).
		Field("b", builders.Transform(builders.String()).
			AsyncCustom(func(ctx context.Context, value interface{}) error {
				// May or may not run depending on iteration order
				return nil
			}))

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	schema.ValidateAndTransformAsync(cancelCtx,
		map[string]interface{}{"a": "x", "b": "y"}, ctx)
	// Key assertion: no panic, cancellation handled gracefully
}

func TestObjectVaTAsync_ContextCancellation_DuringObjectAsync(t *testing.T) {
	// Covers: context cancelled between object-level async validators
	cancelCtx, cancel := context.WithCancel(context.Background())

	schema := builders.Object().
		Field("x", builders.String()).
		AsyncCustom(func(ctx context.Context, value interface{}) error {
			cancel()
			return nil
		}).
		AsyncCustom(func(ctx context.Context, value interface{}) error {
			t.Error("second async should not run after cancellation")
			return nil
		})

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	schema.ValidateAndTransformAsync(cancelCtx,
		map[string]interface{}{"x": "val"}, ctx)
}

func TestObjectVaTAsync_NonTransformableField(t *testing.T) {
	// Covers: the default branch where fieldSchema doesn't implement VaT
	schema := builders.Object().
		Field("flag", builders.Bool()).
		AsyncCustom(func(ctx context.Context, value interface{}) error {
			return nil
		})

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	result, _ := schema.ValidateAndTransformAsync(context.Background(),
		map[string]interface{}{"flag": true}, ctx)
	m := result.(map[string]interface{})
	if m["flag"] != true {
		t.Error("non-transformable field should pass through")
	}
}

func TestObjectVaTAsync_ExtraFieldsRejected(t *testing.T) {
	// Covers: rejectsExtra path in async
	schema := builders.Object().
		Field("name", builders.String()).
		AllowAdditional(false).
		AsyncCustom(func(ctx context.Context, value interface{}) error {
			return nil
		})

	ctx := queryfy.NewValidationContext(queryfy.Loose) // loose mode, but AllowAdditional(false)
	_, err := schema.ValidateAndTransformAsync(context.Background(),
		map[string]interface{}{"name": "ok", "extra": "bad"}, ctx)
	if err == nil {
		t.Error("expected extra field rejection")
	}
}

// ======================================================================
// Array ValidateAndTransform — uncovered paths
// ======================================================================

func TestArrayVaT_NonTransformableElement(t *testing.T) {
	// Covers: element schema that doesn't implement VaT (the else branch)
	schema := builders.Array().Of(builders.Bool()).MinItems(1)

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	result, err := schema.ValidateAndTransform([]interface{}{true, false}, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr := result.([]interface{})
	if len(arr) != 2 || arr[0] != true {
		t.Errorf("expected [true, false], got %v", arr)
	}
}

func TestArrayVaT_NoElementSchema(t *testing.T) {
	// Covers: nil elementSchema path
	schema := builders.Array().MinItems(1)

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	result, err := schema.ValidateAndTransform([]interface{}{"a", 1, true}, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr := result.([]interface{})
	if len(arr) != 3 {
		t.Error("expected 3 elements passthrough")
	}
}

func TestArrayVaT_UniqueItems(t *testing.T) {
	// Covers: uniqueItems validation in VaT path
	schema := builders.Array().Of(builders.String()).UniqueItems()

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	_, err := schema.ValidateAndTransform([]interface{}{"a", "b", "a"}, ctx)
	if err == nil {
		t.Error("expected unique items error")
	}
}

func TestArrayVaT_MaxItems(t *testing.T) {
	// Covers: maxItems validation in VaT
	schema := builders.Array().MaxItems(2)

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	_, err := schema.ValidateAndTransform([]interface{}{1, 2, 3}, ctx)
	if err == nil {
		t.Error("expected maxItems error")
	}
}

func TestArrayVaT_CustomValidator(t *testing.T) {
	// Covers: custom validator in VaT
	schema := builders.Array().Custom(func(value interface{}) error {
		return fmt.Errorf("custom array error")
	})

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	_, err := schema.ValidateAndTransform([]interface{}{1}, ctx)
	if err == nil {
		t.Error("expected custom validator error")
	}
}

// ======================================================================
// Array ValidateAndTransformAsync — uncovered paths
// ======================================================================

func TestArrayVaTAsync_ElementAsync(t *testing.T) {
	// Covers: element schema with async validators
	schema := builders.Array().Of(
		builders.Transform(builders.String().Required()).
			AsyncCustom(func(ctx context.Context, value interface{}) error {
				if value.(string) == "bad" {
					return fmt.Errorf("rejected")
				}
				return nil
			}))

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	_, err := schema.ValidateAndTransformAsync(context.Background(),
		[]interface{}{"good", "bad"}, ctx)
	if err == nil {
		t.Error("expected element async validator error")
	}
}

func TestArrayVaTAsync_ArrayLevelAsync(t *testing.T) {
	// Covers: array-level async validators
	schema := builders.Array().Of(builders.String()).
		AsyncCustom(func(ctx context.Context, value interface{}) error {
			return fmt.Errorf("array-level async error")
		})

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	_, err := schema.ValidateAndTransformAsync(context.Background(),
		[]interface{}{"a"}, ctx)
	if err == nil {
		t.Error("expected array-level async error")
	}
}

func TestArrayVaTAsync_ContextCancellation_BeforeElements(t *testing.T) {
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()

	schema := builders.Array().Of(
		builders.Transform(builders.String()).
			AsyncCustom(func(ctx context.Context, value interface{}) error {
				return nil
			}))

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	_, err := schema.ValidateAndTransformAsync(cancelCtx,
		[]interface{}{"a"}, ctx)
	if err == nil {
		t.Error("expected cancellation error")
	}
}

func TestArrayVaTAsync_ContextCancellation_DuringElements(t *testing.T) {
	cancelCtx, cancel := context.WithCancel(context.Background())

	schema := builders.Array().Of(
		builders.Transform(builders.String()).
			AsyncCustom(func(ctx context.Context, value interface{}) error {
				cancel()
				return nil
			}))

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	// Two elements — first cancels context, second should not run
	schema.ValidateAndTransformAsync(cancelCtx,
		[]interface{}{"a", "b"}, ctx)
}

func TestArrayVaTAsync_ContextCancellation_DuringArrayValidators(t *testing.T) {
	cancelCtx, cancel := context.WithCancel(context.Background())

	schema := builders.Array().Of(builders.String()).
		AsyncCustom(func(ctx context.Context, value interface{}) error {
			cancel()
			return nil
		}).
		AsyncCustom(func(ctx context.Context, value interface{}) error {
			t.Error("second async should not run")
			return nil
		})

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	schema.ValidateAndTransformAsync(cancelCtx, []interface{}{"a"}, ctx)
}

func TestArrayVaTAsync_SyncFailSkipsAsync(t *testing.T) {
	asyncCalled := false
	schema := builders.Array().Of(builders.Number().Required()).
		AsyncCustom(func(ctx context.Context, value interface{}) error {
			asyncCalled = true
			return nil
		})

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	schema.ValidateAndTransformAsync(context.Background(),
		[]interface{}{"not a number"}, ctx)
	if asyncCalled {
		t.Error("async should not run when sync fails")
	}
}

// ======================================================================
// toFloat64 / toFloat64WithMode — type branches
// ======================================================================

func TestNumberValidate_IntegerTypes(t *testing.T) {
	// Covers: toFloat64 branches for int8, int16, int32, int64,
	// uint, uint8, uint16, uint32, uint64, float32
	schema := builders.Number().Min(0).Max(1000)

	types := []struct {
		name  string
		value interface{}
	}{
		{"int", int(42)},
		{"int8", int8(42)},
		{"int16", int16(42)},
		{"int32", int32(42)},
		{"int64", int64(42)},
		{"uint", uint(42)},
		{"uint8", uint8(42)},
		{"uint16", uint16(42)},
		{"uint32", uint32(42)},
		{"uint64", uint64(42)},
		{"float32", float32(42.5)},
		{"float64", float64(42.5)},
	}

	for _, tt := range types {
		t.Run(tt.name, func(t *testing.T) {
			ctx := queryfy.NewValidationContext(queryfy.Strict)
			schema.Validate(tt.value, ctx)
			if ctx.HasErrors() {
				t.Errorf("expected %s(%v) to validate, got errors: %v", tt.name, tt.value, ctx.Error())
			}
		})
	}
}

func TestNumberValidate_IntegerTypes_OutOfRange(t *testing.T) {
	// Covers: toFloat64 type branches with failing range check
	schema := builders.Number().Max(10)

	types := []struct {
		name  string
		value interface{}
	}{
		{"int32", int32(999)},
		{"uint64", uint64(999)},
		{"float32", float32(999.0)},
	}

	for _, tt := range types {
		t.Run(tt.name, func(t *testing.T) {
			ctx := queryfy.NewValidationContext(queryfy.Strict)
			schema.Validate(tt.value, ctx)
			if !ctx.HasErrors() {
				t.Errorf("expected %s(%v) to fail max=10", tt.name, tt.value)
			}
		})
	}
}

func TestNumberValidate_LooseModeStringConversion(t *testing.T) {
	// Covers: toFloat64WithMode loose branch with string
	schema := builders.Number().Min(0).Max(100)

	ctx := queryfy.NewValidationContext(queryfy.Loose)
	schema.Validate("42.5", ctx)
	if ctx.HasErrors() {
		t.Errorf("loose mode should convert string '42.5' to number: %v", ctx.Error())
	}
}

func TestNumberValidate_LooseModeInvalidString(t *testing.T) {
	// Covers: toFloat64WithMode loose branch with non-numeric string
	schema := builders.Number()

	ctx := queryfy.NewValidationContext(queryfy.Loose)
	schema.Validate("not-a-number", ctx)
	if !ctx.HasErrors() {
		t.Error("loose mode should fail on non-numeric string")
	}
}

func TestNumberValidate_NonNumericType(t *testing.T) {
	// Covers: toFloat64WithMode default branch (returns 0, false)
	schema := builders.Number()

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	schema.Validate(true, ctx) // bool is not numeric
	if !ctx.HasErrors() {
		t.Error("expected error for bool in number field")
	}
}

// ======================================================================
// dependentToFloat64 — type branches
// ======================================================================

func TestDependentWhenGreaterThan_IntTypes(t *testing.T) {
	// Covers: dependentToFloat64 branches via WhenGreaterThan
	schema := builders.Object().WithDependencies().
		Field("score", builders.Number()).
		DependentField("bonus", builders.Dependent("bonus").
			On("score").
			When(builders.WhenGreaterThan("score", 50)).
			Then(builders.String().Required()))

	tests := []struct {
		name  string
		score interface{}
		need  bool
	}{
		{"float64 above", float64(60), true},
		{"int above", int(60), true},
		{"int64 above", int64(60), true},
		{"int32 above", int32(60), true},
		{"float32 above", float32(60), true},
		{"uint above", uint(60), true},
		{"uint64 above", uint64(60), true},
		{"uint32 above", uint32(60), true},
		{"below", float64(30), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := queryfy.NewValidationContext(queryfy.Strict)
			data := map[string]interface{}{"score": tt.score}
			if tt.need {
				// Should require bonus but it's missing → error
				schema.Validate(data, ctx)
				if !ctx.HasErrors() {
					t.Error("expected required bonus error")
				}
			} else {
				// Should not require bonus
				schema.Validate(data, ctx)
				if ctx.HasErrors() {
					t.Errorf("unexpected error: %v", ctx.Error())
				}
			}
		})
	}
}

// ======================================================================
// tryCommonFormats — uncovered format paths
// ======================================================================

func TestDateTimeValidate_CommonFormats(t *testing.T) {
	// Covers: tryCommonFormats branches for various date formats
	// Using lenient mode (not StrictFormat) so fallback kicks in
	schema := builders.DateTime() // default format, lenient

	dates := []struct {
		name  string
		value string
	}{
		{"RFC3339", "2024-06-15T14:30:00Z"},
		{"ISO date", "2024-06-15"},
		{"ISO datetime", "2024-06-15 14:30:00"},
		{"DD/MM/YYYY", "15/06/2024"},
		{"D/M/YYYY", "5/6/2024"},
		{"DD-MM-YYYY", "15-06-2024"},
		{"DD.MM.YYYY", "15.06.2024"},
		{"YYYY/MM/DD", "2024/06/15"},
		{"DD-Mon-YYYY", "15-Jun-2024"},
		{"Mon D, YYYY", "Jun 15, 2024"},
		{"Month D, YYYY", "June 15, 2024"},
		{"D Month YYYY", "15 June 2024"},
		{"RFC3339 no tz", "2024-06-15T14:30:00"},
		{"time only", "14:30:00"},
		{"time 12h", "2:30 PM"},
		{"datetime no sec", "2024-06-15 14:30"},
		{"DD/MM/YY", "15/06/24"},
	}

	for _, tt := range dates {
		t.Run(tt.name, func(t *testing.T) {
			ctx := queryfy.NewValidationContext(queryfy.Strict)
			schema.Validate(tt.value, ctx)
			if ctx.HasErrors() {
				t.Errorf("expected %q (%s) to be valid: %v", tt.value, tt.name, ctx.Error())
			}
		})
	}
}

func TestDateTimeValidate_InvalidFormat(t *testing.T) {
	schema := builders.DateTime()

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	schema.Validate("not-a-date-at-all", ctx)
	if !ctx.HasErrors() {
		t.Error("expected error for invalid date string")
	}
}

// ======================================================================
// DateTime Min/Max/YMD/Format/BetweenStrings — uncovered builder paths
// ======================================================================

func TestDateTimeMin(t *testing.T) {
	min := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	schema := builders.DateTime().DateOnly().Min(min)

	expectValid(t, schema, "2024-06-15")
	expectInvalid(t, schema, "2023-12-31")
}

func TestDateTimeMax(t *testing.T) {
	max := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	schema := builders.DateTime().DateOnly().Max(max)

	expectValid(t, schema, "2024-06-15")
	expectInvalid(t, schema, "2025-01-01")
}

func TestDateTimeYMD(t *testing.T) {
	schema := builders.DateTime().YMD()
	if schema.FormatString() != "2006-01-02" {
		t.Errorf("expected YMD format, got %q", schema.FormatString())
	}
}

func TestDateTimeFormat(t *testing.T) {
	schema := builders.DateTime().Format("02 Jan 2006")
	if schema.FormatString() != "02 Jan 2006" {
		t.Errorf("expected custom format, got %q", schema.FormatString())
	}
	expectValid(t, schema, "15 Jun 2024")
}

func TestDateTimeBetweenStrings(t *testing.T) {
	schema := builders.DateTime().DateOnly().BetweenStrings("2024-01-01", "2024-12-31")

	expectValid(t, schema, "2024-06-15")
	expectInvalid(t, schema, "2023-06-15")
	expectInvalid(t, schema, "2025-06-15")
}

func TestDateTimeMinString(t *testing.T) {
	schema := builders.DateTime().DateOnly().MinString("2024-01-01")
	expectValid(t, schema, "2024-06-15")
	expectInvalid(t, schema, "2023-12-31")
}

func TestDateTimeMaxString(t *testing.T) {
	schema := builders.DateTime().DateOnly().MaxString("2024-12-31")
	expectValid(t, schema, "2024-06-15")
	expectInvalid(t, schema, "2025-01-01")
}

func TestDateTimeCustom(t *testing.T) {
	schema := builders.DateTime().Custom(func(value interface{}) error {
		return fmt.Errorf("custom datetime error")
	})
	expectInvalid(t, schema, "2024-06-15")
}

// ======================================================================
// compositeErrorMessage
// ======================================================================

func TestCompositeOrValidation_ErrorMessage(t *testing.T) {
	// Covers: compositeErrorMessage via Or validation failure
	schema := builders.Or(
		builders.String().MinLength(1),
		builders.Number().Min(0),
		builders.Bool(),
	)

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	schema.Validate(nil, ctx)
	// The error should be generated but we just verify no panic
	// and that validation correctly fails
	if !ctx.HasErrors() {
		t.Error("expected validation error for nil in Or")
	}
}

// ======================================================================
// ObjectSchema.Fields (batch field add — 0%)
// ======================================================================

func TestObjectFields_BatchAdd(t *testing.T) {
	schema := builders.Object().Fields(map[string]queryfy.Schema{
		"a": builders.String(),
		"b": builders.Number(),
		"c": builders.Bool(),
	})

	// Verify all fields present
	for _, name := range []string{"a", "b", "c"} {
		if _, ok := schema.GetField(name); !ok {
			t.Errorf("expected field %q", name)
		}
	}
}

// ======================================================================
// DependentSchema — uncovered paths
// ======================================================================

func TestDependentSchemaWithDeps_Fields(t *testing.T) {
	// Covers: ObjectSchemaWithDependencies.Fields()
	schema := builders.Object().WithDependencies().
		Fields(map[string]queryfy.Schema{
			"a": builders.String(),
			"b": builders.Number(),
		}).
		DependentField("c", builders.Dependent("c").
			On("a").
			When(builders.WhenExists("a")).
			Then(builders.String()))

	if _, ok := schema.GetField("a"); !ok {
		t.Error("expected field a")
	}
}

func TestDependentSchemaWithDeps_RequiredFields(t *testing.T) {
	// Covers: ObjectSchemaWithDependencies.RequiredFields()
	schema := builders.Object().WithDependencies().
		Field("a", builders.String()).
		Field("b", builders.String()).
		RequiredFields("a", "b")

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	schema.Validate(map[string]interface{}{}, ctx)
	if !ctx.HasErrors() {
		t.Error("expected required field errors")
	}
}

// (helpers expectValid/expectInvalid defined in builders_test.go)

// ======================================================================
// toFloat64 (bare) — via Integer/Positive/Negative validators
// ======================================================================

func TestNumberInteger_VariousTypes(t *testing.T) {
	schema := builders.Number().Integer()

	types := []struct {
		name  string
		value interface{}
		valid bool
	}{
		{"float64 int", float64(42), true},
		{"float64 frac", float64(42.5), false},
		{"float32 int", float32(42), true},
		{"float32 frac", float32(42.5), false},
		{"int", int(42), true},
		{"int8", int8(42), true},
		{"int16", int16(42), true},
		{"int32", int32(42), true},
		{"int64", int64(42), true},
		{"uint", uint(42), true},
		{"uint8", uint8(42), true},
		{"uint16", uint16(42), true},
		{"uint32", uint32(42), true},
		{"uint64", uint64(42), true},
	}

	for _, tt := range types {
		t.Run(tt.name, func(t *testing.T) {
			ctx := queryfy.NewValidationContext(queryfy.Strict)
			schema.Validate(tt.value, ctx)
			if tt.valid && ctx.HasErrors() {
				t.Errorf("expected valid, got: %v", ctx.Error())
			}
			if !tt.valid && !ctx.HasErrors() {
				t.Error("expected invalid for fractional number")
			}
		})
	}
}

func TestNumberPositive_VariousTypes(t *testing.T) {
	schema := builders.Number().Positive()

	valid := []interface{}{float64(1), int(1), int32(1), uint(1), float32(0.5)}
	invalid := []interface{}{float64(-1), int(-1), int32(0), float64(0)}

	for _, v := range valid {
		ctx := queryfy.NewValidationContext(queryfy.Strict)
		schema.Validate(v, ctx)
		if ctx.HasErrors() {
			t.Errorf("expected %v (%T) positive, got: %v", v, v, ctx.Error())
		}
	}
	for _, v := range invalid {
		ctx := queryfy.NewValidationContext(queryfy.Strict)
		schema.Validate(v, ctx)
		if !ctx.HasErrors() {
			t.Errorf("expected %v (%T) to fail positive check", v, v)
		}
	}
}

func TestNumberNegative_VariousTypes(t *testing.T) {
	schema := builders.Number().Negative()

	valid := []interface{}{float64(-1), int(-1), int32(-5)}
	invalid := []interface{}{float64(1), int(0), uint(1)}

	for _, v := range valid {
		ctx := queryfy.NewValidationContext(queryfy.Strict)
		schema.Validate(v, ctx)
		if ctx.HasErrors() {
			t.Errorf("expected %v (%T) negative, got: %v", v, v, ctx.Error())
		}
	}
	for _, v := range invalid {
		ctx := queryfy.NewValidationContext(queryfy.Strict)
		schema.Validate(v, ctx)
		if !ctx.HasErrors() {
			t.Errorf("expected %v (%T) to fail negative check", v, v)
		}
	}
}
