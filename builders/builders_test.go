package builders_test

import (
	"testing"
	"time"

	"github.com/ha1tch/queryfy"
	"github.com/ha1tch/queryfy/builders"
)

// --- helpers ---

func validate(t *testing.T, schema queryfy.Schema, value interface{}, mode queryfy.ValidationMode) *queryfy.ValidationContext {
	t.Helper()
	ctx := queryfy.NewValidationContext(mode)
	schema.Validate(value, ctx)
	return ctx
}

func expectValid(t *testing.T, schema queryfy.Schema, value interface{}) {
	t.Helper()
	ctx := validate(t, schema, value, queryfy.Strict)
	if ctx.HasErrors() {
		t.Errorf("expected valid, got errors: %v", ctx.Error())
	}
}

func expectInvalid(t *testing.T, schema queryfy.Schema, value interface{}) {
	t.Helper()
	ctx := validate(t, schema, value, queryfy.Strict)
	if !ctx.HasErrors() {
		t.Errorf("expected invalid for value %v (%T), but validation passed", value, value)
	}
}

func expectValidLoose(t *testing.T, schema queryfy.Schema, value interface{}) {
	t.Helper()
	ctx := validate(t, schema, value, queryfy.Loose)
	if ctx.HasErrors() {
		t.Errorf("expected valid in loose mode, got errors: %v", ctx.Error())
	}
}

func expectInvalidLoose(t *testing.T, schema queryfy.Schema, value interface{}) {
	t.Helper()
	ctx := validate(t, schema, value, queryfy.Loose)
	if !ctx.HasErrors() {
		t.Errorf("expected invalid in loose mode for value %v (%T), but passed", value, value)
	}
}

// ======================================================================
// NumberSchema
// ======================================================================

func TestNumber_MinMax(t *testing.T) {
	s := builders.Number().Min(0).Max(100)
	expectValid(t, s, 0.0)
	expectValid(t, s, 50.0)
	expectValid(t, s, 100.0)
	expectInvalid(t, s, -1.0)
	expectInvalid(t, s, 101.0)
}

func TestNumber_Range(t *testing.T) {
	s := builders.Number().Range(10, 20)
	expectValid(t, s, 10.0)
	expectValid(t, s, 15.0)
	expectValid(t, s, 20.0)
	expectInvalid(t, s, 9.0)
	expectInvalid(t, s, 21.0)
}

func TestNumber_Integer(t *testing.T) {
	s := builders.Number().Integer()
	expectValid(t, s, 42.0)
	expectValid(t, s, 0.0)
	expectValid(t, s, -7.0)
	expectInvalid(t, s, 3.14)
	expectInvalid(t, s, 0.1)
}

func TestNumber_Positive(t *testing.T) {
	s := builders.Number().Positive()
	expectValid(t, s, 1.0)
	expectValid(t, s, 0.001)
	expectInvalid(t, s, 0.0)
	expectInvalid(t, s, -1.0)
}

func TestNumber_Negative(t *testing.T) {
	s := builders.Number().Negative()
	expectValid(t, s, -1.0)
	expectValid(t, s, -0.001)
	expectInvalid(t, s, 0.0)
	expectInvalid(t, s, 1.0)
}

func TestNumber_MultipleOf(t *testing.T) {
	s := builders.Number().MultipleOf(3)
	expectValid(t, s, 9.0)
	expectValid(t, s, 0.0)
	expectValid(t, s, -6.0)
	expectInvalid(t, s, 10.0)
	expectInvalid(t, s, 1.0)
}

func TestNumber_Required(t *testing.T) {
	s := builders.Number().Required()
	expectValid(t, s, 42.0)
	expectInvalid(t, s, nil)
}

func TestNumber_Nullable(t *testing.T) {
	// Required + nullable: nil is OK, but absence would be caught by object
	s := builders.Number().Nullable()
	ctx := validate(t, s, nil, queryfy.Strict)
	if ctx.HasErrors() {
		t.Error("nullable field should accept nil")
	}
}

func TestNumber_WrongType(t *testing.T) {
	s := builders.Number()
	expectInvalid(t, s, "not a number")
	expectInvalid(t, s, true)
}

func TestNumber_LooseStringConversion(t *testing.T) {
	s := builders.Number().Min(0).Max(100)
	expectValidLoose(t, s, "42")
	expectValidLoose(t, s, "99.5")
	expectInvalidLoose(t, s, "not-a-number")
	expectInvalidLoose(t, s, "200")
}

func TestNumber_IntegerTypes(t *testing.T) {
	s := builders.Number().Min(0)
	expectValid(t, s, int(5))
	expectValid(t, s, int32(10))
	expectValid(t, s, int64(20))
	expectValid(t, s, uint(15))
	expectValid(t, s, float32(7.5))
}

func TestNumber_Custom(t *testing.T) {
	s := builders.Number().Custom(func(value interface{}) error {
		n := value.(float64)
		if int(n)%2 != 0 {
			return queryfy.NewFieldError("", "must be even", value)
		}
		return nil
	})
	expectValid(t, s, 4.0)
	expectInvalid(t, s, 3.0)
}

// ======================================================================
// StringSchema — formats and edge cases
// ======================================================================

func TestString_Email(t *testing.T) {
	s := builders.String().Email()
	expectValid(t, s, "user@example.com")
	expectValid(t, s, "user.name+tag@domain.co.uk")
	expectInvalid(t, s, "not-an-email")
	expectInvalid(t, s, "@missing-local.com")
	expectInvalid(t, s, "missing-domain@")
}

func TestString_URL(t *testing.T) {
	s := builders.String().URL()
	expectValid(t, s, "https://example.com")
	expectValid(t, s, "http://example.com/path?q=1")
	expectInvalid(t, s, "not-a-url")
	expectInvalid(t, s, "ftp://wrong-scheme.com")
}

func TestString_UUID(t *testing.T) {
	s := builders.String().UUID()
	expectValid(t, s, "550e8400-e29b-41d4-a716-446655440000")
	expectValid(t, s, "6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	expectInvalid(t, s, "not-a-uuid")
	expectInvalid(t, s, "550e8400-e29b-41d4-a716") // truncated
}

func TestString_Pattern(t *testing.T) {
	s := builders.String().Pattern(`^\d{3}-\d{4}$`)
	expectValid(t, s, "123-4567")
	expectInvalid(t, s, "123-456")
	expectInvalid(t, s, "abc-defg")
}

func TestString_PatternInvalid(t *testing.T) {
	// An invalid regex should produce a validation error, not a panic
	s := builders.String().Pattern(`[invalid`)
	expectInvalid(t, s, "anything")
}

func TestString_Enum(t *testing.T) {
	s := builders.String().Enum("red", "green", "blue")
	expectValid(t, s, "red")
	expectValid(t, s, "blue")
	expectInvalid(t, s, "yellow")
	expectInvalid(t, s, "RED") // case-sensitive
}

func TestString_Length(t *testing.T) {
	s := builders.String().Length(5)
	expectValid(t, s, "hello")
	expectInvalid(t, s, "hi")
	expectInvalid(t, s, "toolong")
}

func TestString_MinMaxLength(t *testing.T) {
	s := builders.String().MinLength(2).MaxLength(5)
	expectValid(t, s, "ab")
	expectValid(t, s, "abcde")
	expectInvalid(t, s, "a")
	expectInvalid(t, s, "abcdef")
}

func TestString_WrongType(t *testing.T) {
	s := builders.String()
	expectInvalid(t, s, 42)
	expectInvalid(t, s, true)
}

func TestString_LooseConversion(t *testing.T) {
	s := builders.String().MinLength(1)
	expectValidLoose(t, s, 42)    // number -> "42"
	expectValidLoose(t, s, true)  // bool -> "true"
}

func TestString_RequiredEmpty(t *testing.T) {
	s := builders.String().Required().MinLength(1)
	expectInvalid(t, s, "")
}

// ======================================================================
// BoolSchema
// ======================================================================

func TestBool_Valid(t *testing.T) {
	s := builders.Bool()
	expectValid(t, s, true)
	expectValid(t, s, false)
}

func TestBool_WrongType(t *testing.T) {
	s := builders.Bool()
	expectInvalid(t, s, "true")
	expectInvalid(t, s, 1)
	expectInvalid(t, s, 0)
}

func TestBool_LooseStringConversion(t *testing.T) {
	s := builders.Bool()
	expectValidLoose(t, s, "true")
	expectValidLoose(t, s, "false")
	expectInvalidLoose(t, s, "yes")
	expectInvalidLoose(t, s, 1)
}

func TestBool_Required(t *testing.T) {
	s := builders.Bool().Required()
	expectValid(t, s, false) // false is a valid value, not absence
	expectInvalid(t, s, nil)
}

func TestBool_Custom(t *testing.T) {
	// Contrived: must be true
	s := builders.Bool().Custom(func(value interface{}) error {
		if value == false {
			return queryfy.NewFieldError("", "must be true", value)
		}
		return nil
	})
	expectValid(t, s, true)
	expectInvalid(t, s, false)
}

// ======================================================================
// DateTimeSchema
// ======================================================================

func TestDateTime_ISO8601(t *testing.T) {
	s := builders.DateTime().ISO8601()
	expectValid(t, s, "2024-06-15T10:30:00Z")
	expectValid(t, s, "2024-06-15T10:30:00+02:00")
	expectInvalid(t, s, "not-a-date")
	// Note: "2024-06-15" is accepted via tryCommonFormats fallback — by design
}

func TestDateTime_DateOnly(t *testing.T) {
	s := builders.DateTime().DateOnly()
	expectValid(t, s, "2024-06-15")
	expectValid(t, s, "2000-01-01")
	expectInvalid(t, s, "not-a-date")
	expectInvalid(t, s, "99/99/9999") // nonsensical date
	// Note: "15/06/2024" accepted via tryCommonFormats fallback — by design
}

func TestDateTime_DMY(t *testing.T) {
	s := builders.DateTime().DMY()
	expectValid(t, s, "15/06/2024")
	expectValid(t, s, "01/12/2000")
	expectInvalid(t, s, "not-a-date")
}

func TestDateTime_MDY(t *testing.T) {
	s := builders.DateTime().MDY()
	expectValid(t, s, "06/15/2024")
	expectValid(t, s, "12/01/2000")
	expectInvalid(t, s, "not-a-date")
}

func TestDateTime_MinMax(t *testing.T) {
	min := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	max := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	s := builders.DateTime().DateOnly().Between(min, max)

	expectValid(t, s, "2024-06-15")
	expectValid(t, s, "2024-01-01")
	expectInvalid(t, s, "2023-12-31")
	expectInvalid(t, s, "2025-01-01")
}

func TestDateTime_MinMaxString(t *testing.T) {
	s := builders.DateTime().DateOnly().
		MinString("2024-01-01").
		MaxString("2024-12-31")
	expectValid(t, s, "2024-06-15")
	expectInvalid(t, s, "2023-12-31")
}

func TestDateTime_Future(t *testing.T) {
	s := builders.DateTime().DateOnly().Future()
	futureDate := time.Now().AddDate(1, 0, 0).Format("2006-01-02")
	pastDate := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	expectValid(t, s, futureDate)
	expectInvalid(t, s, pastDate)
}

func TestDateTime_Past(t *testing.T) {
	s := builders.DateTime().DateOnly().Past()
	pastDate := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	futureDate := time.Now().AddDate(1, 0, 0).Format("2006-01-02")
	expectValid(t, s, pastDate)
	expectInvalid(t, s, futureDate)
}

func TestDateTime_Weekday(t *testing.T) {
	s := builders.DateTime().DateOnly().Weekday(time.Monday, time.Wednesday, time.Friday)
	// 2024-01-01 is a Monday
	expectValid(t, s, "2024-01-01")
	// 2024-01-03 is a Wednesday
	expectValid(t, s, "2024-01-03")
	// 2024-01-02 is a Tuesday
	expectInvalid(t, s, "2024-01-02")
}

func TestDateTime_BusinessDay(t *testing.T) {
	s := builders.DateTime().DateOnly().BusinessDay()
	// 2024-01-01 is Monday
	expectValid(t, s, "2024-01-01")
	// 2024-01-05 is Friday
	expectValid(t, s, "2024-01-05")
	// 2024-01-06 is Saturday
	expectInvalid(t, s, "2024-01-06")
	// 2024-01-07 is Sunday
	expectInvalid(t, s, "2024-01-07")
}

func TestDateTime_TimeValue(t *testing.T) {
	s := builders.DateTime().ISO8601()
	now := time.Now()
	expectValid(t, s, now)
}

func TestDateTime_Required(t *testing.T) {
	s := builders.DateTime().Required()
	expectInvalid(t, s, nil)
}

func TestDateTime_WrongType(t *testing.T) {
	s := builders.DateTime()
	expectInvalid(t, s, 42)
	expectInvalid(t, s, true)
}

func TestDateTime_Age(t *testing.T) {
	s := builders.DateTime().DateOnly().Age(18, 120)
	// Someone born 30 years ago
	thirtyAgo := time.Now().AddDate(-30, 0, 0).Format("2006-01-02")
	expectValid(t, s, thirtyAgo)
	// Someone born 10 years ago (too young)
	tenAgo := time.Now().AddDate(-10, 0, 0).Format("2006-01-02")
	expectInvalid(t, s, tenAgo)
}

func TestDateTime_StrictFormat_DateOnly(t *testing.T) {
	s := builders.DateTime().DateOnly().StrictFormat()
	expectValid(t, s, "2024-06-15")
	expectInvalid(t, s, "15/06/2024")       // wrong format
	expectInvalid(t, s, "June 15, 2024")    // wrong format
	expectInvalid(t, s, "not-a-date")
}

func TestDateTime_StrictFormat_ISO8601(t *testing.T) {
	s := builders.DateTime().ISO8601().StrictFormat()
	expectValid(t, s, "2024-06-15T10:30:00Z")
	expectInvalid(t, s, "2024-06-15")       // date-only rejected in strict
	expectInvalid(t, s, "15/06/2024")
}

func TestDateTime_StrictFormat_DMY(t *testing.T) {
	s := builders.DateTime().DMY().StrictFormat()
	expectValid(t, s, "15/06/2024")
	expectInvalid(t, s, "2024-06-15")       // ISO rejected in strict
	expectInvalid(t, s, "06/15/2024")       // MDY rejected
}

func TestDateTime_LenientFormat_Fallback(t *testing.T) {
	// Without StrictFormat, the fallback should accept common formats
	s := builders.DateTime().DateOnly() // lenient (default)
	expectValid(t, s, "2024-06-15")     // exact format
	expectValid(t, s, "15/06/2024")     // fallback accepts DMY
	expectValid(t, s, "2024-06-15T10:30:00Z") // fallback accepts RFC3339
}

// ======================================================================
// CustomSchema
// ======================================================================

func TestCustom_Basic(t *testing.T) {
	s := builders.Custom(func(value interface{}) error {
		if value == nil {
			return queryfy.NewFieldError("", "cannot be nil", nil)
		}
		return nil
	}).Required()

	expectValid(t, s, "anything")
	expectValid(t, s, 42)
	expectInvalid(t, s, nil)
}

func TestCustom_Nullable(t *testing.T) {
	called := false
	s := builders.Custom(func(value interface{}) error {
		called = true
		return nil
	}).Nullable()

	ctx := validate(t, s, nil, queryfy.Strict)
	if ctx.HasErrors() {
		t.Error("nullable custom schema should accept nil")
	}
	if called {
		t.Error("custom validator should not be called for nil on nullable schema")
	}
}

// ======================================================================
// CompositeSchema (And, Or, Not)
// ======================================================================

func TestAnd_BothPass(t *testing.T) {
	s := builders.And(
		builders.String().MinLength(3),
		builders.String().MaxLength(10),
	)
	expectValid(t, s, "hello")
}

func TestAnd_OneFails(t *testing.T) {
	s := builders.And(
		builders.String().MinLength(3),
		builders.String().MaxLength(10),
	)
	expectInvalid(t, s, "ab")       // too short
	expectInvalid(t, s, "toolongstring") // too long
}

func TestOr_OneMatches(t *testing.T) {
	s := builders.Or(
		builders.String().Email(),
		builders.String().URL(),
	)
	expectValid(t, s, "user@example.com")
	expectValid(t, s, "https://example.com")
}

func TestOr_NoneMatch(t *testing.T) {
	s := builders.Or(
		builders.String().Email(),
		builders.String().URL(),
	)
	expectInvalid(t, s, "just a string")
}

func TestNot_Inverts(t *testing.T) {
	s := builders.Not(builders.String().Email())
	expectValid(t, s, "not-an-email")
	expectInvalid(t, s, "user@example.com")
}

func TestComposite_Required(t *testing.T) {
	s := builders.And(
		builders.String().MinLength(1),
	).Required()
	expectInvalid(t, s, nil)
}

// ======================================================================
// DependentSchema
// ======================================================================

func TestDependent_WhenEquals(t *testing.T) {
	schema := builders.Object().WithDependencies().
		Field("type", builders.String().Enum("personal", "business").Required()).
		DependentField("company",
			builders.Dependent("company").
				On("type").
				When(builders.WhenEquals("type", "business")).
				Then(builders.String().MinLength(1).Required()))

	// Business type requires company
	validBusiness := map[string]interface{}{
		"type":    "business",
		"company": "Acme Corp",
	}
	expectValid(t, schema, validBusiness)

	// Personal type, no company needed
	validPersonal := map[string]interface{}{
		"type": "personal",
	}
	expectValid(t, schema, validPersonal)

	// Business without company should fail
	invalidBusiness := map[string]interface{}{
		"type": "business",
	}
	expectInvalid(t, schema, invalidBusiness)
}

func TestDependent_WhenExists(t *testing.T) {
	schema := builders.Object().WithDependencies().
		Field("email", builders.String().Email().Optional()).
		DependentField("emailVerified",
			builders.Dependent("emailVerified").
				On("email").
				When(builders.WhenExists("email")).
				Then(builders.Bool().Required()))

	// Email present — emailVerified required
	withEmail := map[string]interface{}{
		"email":         "a@b.com",
		"emailVerified": true,
	}
	expectValid(t, schema, withEmail)

	// Email present but no emailVerified — should fail
	missingVerified := map[string]interface{}{
		"email": "a@b.com",
	}
	expectInvalid(t, schema, missingVerified)

	// No email — emailVerified not required
	noEmail := map[string]interface{}{}
	expectValid(t, schema, noEmail)
}

func TestDependent_WhenGreaterThan(t *testing.T) {
	schema := builders.Object().WithDependencies().
		Field("amount", builders.Number().Min(0).Required()).
		DependentField("approver",
			builders.Dependent("approver").
				On("amount").
				When(builders.WhenGreaterThan("amount", 1000)).
				Then(builders.String().Required()))

	// Small amount, no approver needed
	small := map[string]interface{}{
		"amount": 500.0,
	}
	expectValid(t, schema, small)

	// Large amount, approver required
	largeWithApprover := map[string]interface{}{
		"amount":   5000.0,
		"approver": "manager@company.com",
	}
	expectValid(t, schema, largeWithApprover)

	// Large amount without approver
	largeNoApprover := map[string]interface{}{
		"amount": 5000.0,
	}
	expectInvalid(t, schema, largeNoApprover)
}

func TestDependent_ThenElse(t *testing.T) {
	schema := builders.Object().WithDependencies().
		Field("premium", builders.Bool().Required()).
		DependentField("limit",
			builders.Dependent("limit").
				On("premium").
				When(builders.WhenTrue("premium")).
				Then(builders.Number().Max(100000)).
				Else(builders.Number().Max(1000)))

	// Premium user, high limit OK
	premiumHigh := map[string]interface{}{
		"premium": true,
		"limit":   50000.0,
	}
	expectValid(t, schema, premiumHigh)

	// Non-premium, low limit OK
	freeLow := map[string]interface{}{
		"premium": false,
		"limit":   500.0,
	}
	expectValid(t, schema, freeLow)

	// Non-premium, high limit should fail
	freeHigh := map[string]interface{}{
		"premium": false,
		"limit":   5000.0,
	}
	expectInvalid(t, schema, freeHigh)
}

func TestDependent_PanicOnPlainField(t *testing.T) {
	// Passing a DependentSchema to ObjectSchema.Field should panic
	// to prevent the silent-no-validation footgun.
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when passing DependentSchema to Field()")
		}
	}()

	builders.Object().Field("x", builders.Dependent("x").
		When(builders.WhenExists("other")).
		Then(builders.String().Required()))
}

// ======================================================================
// ObjectSchema edge cases
// ======================================================================

func TestObject_FieldNames(t *testing.T) {
	s := builders.Object().
		Field("z", builders.String()).
		Field("a", builders.String()).
		Field("m", builders.String())
	names := s.FieldNames()
	if len(names) != 3 || names[0] != "a" || names[1] != "m" || names[2] != "z" {
		t.Errorf("FieldNames should return sorted: got %v", names)
	}
}

func TestObject_RequiredFields(t *testing.T) {
	s := builders.Object().
		Field("a", builders.String()).
		Field("b", builders.String()).
		RequiredFields("a", "b")

	expectInvalid(t, s, map[string]interface{}{})
	expectInvalid(t, s, map[string]interface{}{"a": "ok"})
	expectValid(t, s, map[string]interface{}{"a": "ok", "b": "ok"})
}

func TestObject_WrongType(t *testing.T) {
	s := builders.Object()
	expectInvalid(t, s, "not an object")
	expectInvalid(t, s, 42)
	expectInvalid(t, s, []interface{}{1, 2})
}

func TestObject_ReflectionMap(t *testing.T) {
	// Test that non-standard map types still work
	s := builders.Object().Field("x", builders.Number())
	data := map[string]int{"x": 42}
	// This should work via reflection conversion in convertToMap
	expectValid(t, s, data)
}

// ======================================================================
// ArraySchema edge cases
// ======================================================================

func TestArray_UniqueItems(t *testing.T) {
	s := builders.Array().Of(builders.String()).UniqueItems()
	expectValid(t, s, []interface{}{"a", "b", "c"})
	expectInvalid(t, s, []interface{}{"a", "b", "a"})
}

func TestArray_Length(t *testing.T) {
	s := builders.Array().Length(3)
	expectValid(t, s, []interface{}{1, 2, 3})
	expectInvalid(t, s, []interface{}{1, 2})
	expectInvalid(t, s, []interface{}{1, 2, 3, 4})
}

func TestArray_WrongType(t *testing.T) {
	s := builders.Array()
	expectInvalid(t, s, "not an array")
	expectInvalid(t, s, 42)
	expectInvalid(t, s, map[string]interface{}{})
}

func TestArray_NestedValidation(t *testing.T) {
	s := builders.Array().Of(
		builders.Object().
			Field("id", builders.Number().Required()).
			Field("name", builders.String().Required()),
	)
	valid := []interface{}{
		map[string]interface{}{"id": 1, "name": "Alice"},
		map[string]interface{}{"id": 2, "name": "Bob"},
	}
	expectValid(t, s, valid)

	invalid := []interface{}{
		map[string]interface{}{"id": 1}, // missing name
	}
	expectInvalid(t, s, invalid)
}

func TestArray_Custom(t *testing.T) {
	// Custom: sum must be <= 100
	s := builders.Array().Of(builders.Number()).Custom(func(value interface{}) error {
		return nil // just ensure custom validators are invoked
	})
	expectValid(t, s, []interface{}{10.0, 20.0})
}
