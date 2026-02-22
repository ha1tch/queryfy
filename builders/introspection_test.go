package builders_test

import (
	"testing"
	"time"

	"github.com/ha1tch/queryfy"
	"github.com/ha1tch/queryfy/builders"
)

// ======================================================================
// 2.2 Metadata
// ======================================================================

func TestMeta_String(t *testing.T) {
	s := builders.String().
		Meta("x-index", true).
		Meta("db_column", "user_email")

	v, ok := s.GetMeta("x-index")
	if !ok || v != true {
		t.Error("expected x-index=true")
	}

	v2, ok := s.GetMeta("db_column")
	if !ok || v2 != "user_email" {
		t.Error("expected db_column=user_email")
	}

	_, ok = s.GetMeta("nonexistent")
	if ok {
		t.Error("expected nonexistent key to return false")
	}
}

func TestMeta_Number(t *testing.T) {
	s := builders.Number().
		Meta("decimal_precision", 18).
		Meta("decimal_scale", 4)

	all := s.AllMeta()
	if len(all) != 2 {
		t.Errorf("expected 2 metadata entries, got %d", len(all))
	}
	if all["decimal_precision"] != 18 {
		t.Error("wrong decimal_precision")
	}
	if all["decimal_scale"] != 4 {
		t.Error("wrong decimal_scale")
	}
}

func TestMeta_Object(t *testing.T) {
	s := builders.Object().
		Meta("format", "ref").
		Field("id", builders.String().Required())

	v, ok := s.GetMeta("format")
	if !ok || v != "ref" {
		t.Error("expected format=ref")
	}
}

func TestMeta_Array(t *testing.T) {
	s := builders.Array().
		Of(builders.String()).
		Meta("x-storage", "json_array")

	v, ok := s.GetMeta("x-storage")
	if !ok || v != "json_array" {
		t.Error("expected x-storage=json_array")
	}
}

func TestMeta_DateTime(t *testing.T) {
	s := builders.DateTime().DateOnly().Meta("db_type", "DATE")
	v, ok := s.GetMeta("db_type")
	if !ok || v != "DATE" {
		t.Error("expected db_type=DATE")
	}
}

func TestMeta_Bool(t *testing.T) {
	s := builders.Bool().Meta("default_value", false)
	v, ok := s.GetMeta("default_value")
	if !ok || v != false {
		t.Error("expected default_value=false")
	}
}

func TestMeta_Custom(t *testing.T) {
	s := builders.Custom(func(v interface{}) error { return nil }).
		Meta("description", "custom check")
	v, ok := s.GetMeta("description")
	if !ok || v != "custom check" {
		t.Error("expected description")
	}
}

func TestMeta_Transform(t *testing.T) {
	s := builders.Transform(builders.String()).
		Meta("pipeline", "sanitise")
	v, ok := s.GetMeta("pipeline")
	if !ok || v != "sanitise" {
		t.Error("expected pipeline=sanitise")
	}
}

func TestMeta_AllMeta_Empty(t *testing.T) {
	s := builders.String()
	all := s.AllMeta()
	if all != nil {
		t.Error("AllMeta on schema with no metadata should return nil")
	}
}

func TestMeta_DoesNotAffectValidation(t *testing.T) {
	// Metadata should be carried but never interpreted during validation
	s := builders.String().Required().Meta("nonsense", "value")
	ctx := queryfy.NewValidationContext(queryfy.Strict)
	s.Validate("hello", ctx)
	if ctx.HasErrors() {
		t.Error("metadata should not affect validation")
	}
}

// ======================================================================
// 1.2 String Introspection
// ======================================================================

func TestStringIntrospection_FormatType(t *testing.T) {
	if builders.String().Email().FormatType() != "email" {
		t.Error("expected email format")
	}
	if builders.String().URL().FormatType() != "url" {
		t.Error("expected url format")
	}
	if builders.String().UUID().FormatType() != "uuid" {
		t.Error("expected uuid format")
	}
	if builders.String().FormatType() != "" {
		t.Error("expected empty format")
	}
}

func TestStringIntrospection_EnumValues(t *testing.T) {
	s := builders.String().Enum("a", "b", "c")
	vals := s.EnumValues()
	if len(vals) != 3 || vals[0] != "a" || vals[1] != "b" || vals[2] != "c" {
		t.Errorf("expected [a, b, c], got %v", vals)
	}

	if builders.String().EnumValues() != nil {
		t.Error("expected nil enum on unconstrained string")
	}
}

func TestStringIntrospection_LengthConstraints(t *testing.T) {
	s := builders.String().MinLength(3).MaxLength(50)
	min, max := s.LengthConstraints()
	if min == nil || *min != 3 {
		t.Error("expected min=3")
	}
	if max == nil || *max != 50 {
		t.Error("expected max=50")
	}

	// Only min set
	s2 := builders.String().MinLength(1)
	min2, max2 := s2.LengthConstraints()
	if min2 == nil || *min2 != 1 {
		t.Error("expected min=1")
	}
	if max2 != nil {
		t.Error("expected max=nil")
	}

	// Neither set
	s3 := builders.String()
	min3, max3 := s3.LengthConstraints()
	if min3 != nil || max3 != nil {
		t.Error("expected nil/nil for unconstrained string")
	}
}

func TestStringIntrospection_Pattern(t *testing.T) {
	s := builders.String().Pattern(`^\d{3}-\d{4}$`)
	if s.PatternString() != `^\d{3}-\d{4}$` {
		t.Errorf("expected pattern, got %q", s.PatternString())
	}

	if builders.String().PatternString() != "" {
		t.Error("expected empty pattern on unconstrained string")
	}
}

// ======================================================================
// 1.2 Number Introspection
// ======================================================================

func TestNumberIntrospection_Range(t *testing.T) {
	s := builders.Number().Min(0).Max(100)
	min, max := s.RangeConstraints()
	if min == nil || *min != 0 {
		t.Error("expected min=0")
	}
	if max == nil || *max != 100 {
		t.Error("expected max=100")
	}

	// Unconstrained
	s2 := builders.Number()
	min2, max2 := s2.RangeConstraints()
	if min2 != nil || max2 != nil {
		t.Error("expected nil/nil for unconstrained number")
	}
}

func TestNumberIntrospection_IsInteger(t *testing.T) {
	if builders.Number().Integer().IsInteger() != true {
		t.Error("expected IsInteger=true after Integer()")
	}
	if builders.Number().IsInteger() != false {
		t.Error("expected IsInteger=false by default")
	}
}

func TestNumberIntrospection_MultipleOf(t *testing.T) {
	s := builders.Number().MultipleOf(5)
	m := s.MultipleOfValue()
	if m == nil || *m != 5 {
		t.Error("expected multipleOf=5")
	}

	if builders.Number().MultipleOfValue() != nil {
		t.Error("expected nil for unconstrained")
	}
}

// ======================================================================
// 1.2 Object Introspection
// ======================================================================

func TestObjectIntrospection_GetField(t *testing.T) {
	s := builders.Object().
		Field("name", builders.String().Required()).
		Field("age", builders.Number())

	nameSchema, ok := s.GetField("name")
	if !ok || nameSchema == nil {
		t.Fatal("expected name field to exist")
	}
	if nameSchema.Type() != queryfy.TypeString {
		t.Errorf("expected TypeString, got %v", nameSchema.Type())
	}

	_, ok = s.GetField("nonexistent")
	if ok {
		t.Error("expected nonexistent field to return false")
	}
}

func TestObjectIntrospection_RequiredFieldNames(t *testing.T) {
	s := builders.Object().
		Field("a", builders.String().Required()).
		Field("b", builders.Number()).
		Field("c", builders.Bool().Required()).
		Field("d", builders.String())

	names := s.RequiredFieldNames()
	if len(names) != 2 || names[0] != "a" || names[1] != "c" {
		t.Errorf("expected [a, c], got %v", names)
	}
}

func TestObjectIntrospection_RequiredFieldNames_ViaRequiredFields(t *testing.T) {
	// RequiredFields() sets requirements via the requiredFields map
	s := builders.Object().
		Field("x", builders.String()).
		Field("y", builders.String()).
		RequiredFields("x", "y")

	names := s.RequiredFieldNames()
	if len(names) != 2 || names[0] != "x" || names[1] != "y" {
		t.Errorf("expected [x, y], got %v", names)
	}
}

// ======================================================================
// 1.2 Array Introspection
// ======================================================================

func TestArrayIntrospection_ElementSchema(t *testing.T) {
	inner := builders.String().Email()
	s := builders.Array().Of(inner)

	elem := s.ElementSchema()
	if elem == nil {
		t.Fatal("expected non-nil element schema")
	}
	if elem.Type() != queryfy.TypeString {
		t.Errorf("expected TypeString, got %v", elem.Type())
	}

	if builders.Array().ElementSchema() != nil {
		t.Error("expected nil element schema when not set")
	}
}

func TestArrayIntrospection_ItemCountConstraints(t *testing.T) {
	s := builders.Array().MinItems(1).MaxItems(10)
	min, max := s.ItemCountConstraints()
	if min == nil || *min != 1 {
		t.Error("expected min=1")
	}
	if max == nil || *max != 10 {
		t.Error("expected max=10")
	}

	// Unconstrained
	s2 := builders.Array()
	min2, max2 := s2.ItemCountConstraints()
	if min2 != nil || max2 != nil {
		t.Error("expected nil/nil for unconstrained array")
	}
}

// ======================================================================
// 1.2 DateTime Introspection
// ======================================================================

func TestDateTimeIntrospection_FormatString(t *testing.T) {
	if builders.DateTime().DateOnly().FormatString() != "2006-01-02" {
		t.Error("expected date-only format")
	}
	if builders.DateTime().DMY().FormatString() != "02/01/2006" {
		t.Error("expected DMY format")
	}
	if builders.DateTime().ISO8601().FormatString() != "2006-01-02T15:04:05Z07:00" {
		t.Error("expected RFC3339 format")
	}
}

func TestDateTimeIntrospection_IsStrictFormat(t *testing.T) {
	if builders.DateTime().IsStrictFormat() {
		t.Error("expected not strict by default")
	}
	if !builders.DateTime().StrictFormat().IsStrictFormat() {
		t.Error("expected strict after StrictFormat()")
	}
}

func TestDateTimeIntrospection_TimeConstraints(t *testing.T) {
	min := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	max := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	s := builders.DateTime().Between(min, max)

	gotMin, gotMax := s.TimeConstraints()
	if gotMin == nil || !gotMin.Equal(min) {
		t.Error("expected min time")
	}
	if gotMax == nil || !gotMax.Equal(max) {
		t.Error("expected max time")
	}

	// Unconstrained
	s2 := builders.DateTime()
	m1, m2 := s2.TimeConstraints()
	if m1 != nil || m2 != nil {
		t.Error("expected nil/nil for unconstrained datetime")
	}
}

// ======================================================================
// 1.2 BaseSchema Introspection (IsNullable already exists but untested)
// ======================================================================

func TestBaseSchema_IsNullable(t *testing.T) {
	if builders.String().IsNullable() {
		t.Error("expected not nullable by default")
	}
	if !builders.String().Nullable().IsNullable() {
		t.Error("expected nullable after Nullable()")
	}
}

// ======================================================================
// 1.3 AllowAdditional
// ======================================================================

func TestAllowAdditional_Default(t *testing.T) {
	s := builders.Object()
	_, explicit := s.AllowsAdditional()
	if explicit {
		t.Error("expected no explicit policy by default")
	}
}

func TestAllowAdditional_ExplicitTrue(t *testing.T) {
	s := builders.Object().AllowAdditional(true)
	allow, explicit := s.AllowsAdditional()
	if !explicit || !allow {
		t.Error("expected explicit allow=true")
	}
}

func TestAllowAdditional_ExplicitFalse(t *testing.T) {
	s := builders.Object().AllowAdditional(false)
	allow, explicit := s.AllowsAdditional()
	if !explicit || allow {
		t.Error("expected explicit allow=false")
	}
}

func TestAllowAdditional_StrictModeWithAllow(t *testing.T) {
	// Strict mode normally rejects extra fields.
	// AllowAdditional(true) should override that.
	s := builders.Object().
		Field("name", builders.String()).
		AllowAdditional(true)

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	s.Validate(map[string]interface{}{
		"name":  "Alice",
		"extra": "should be allowed",
	}, ctx)
	if ctx.HasErrors() {
		t.Error("AllowAdditional(true) should accept extra fields even in strict mode")
	}
}

func TestAllowAdditional_LooseModeWithReject(t *testing.T) {
	// Loose mode normally accepts extra fields.
	// AllowAdditional(false) should override that.
	s := builders.Object().
		Field("name", builders.String()).
		AllowAdditional(false)

	ctx := queryfy.NewValidationContext(queryfy.Loose)
	s.Validate(map[string]interface{}{
		"name":  "Alice",
		"extra": "should be rejected",
	}, ctx)
	if !ctx.HasErrors() {
		t.Error("AllowAdditional(false) should reject extra fields even in loose mode")
	}
}

func TestAllowAdditional_ValidateAndTransform_StrictAllow(t *testing.T) {
	s := builders.Object().
		Field("name", builders.String()).
		AllowAdditional(true)

	ctx := queryfy.NewValidationContext(queryfy.Strict)
	result, err := s.ValidateAndTransform(map[string]interface{}{
		"name":  "Alice",
		"extra": "preserved",
	}, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]interface{})
	if m["extra"] != "preserved" {
		t.Error("extra field should be preserved when AllowAdditional(true)")
	}
}

func TestAllowAdditional_ValidateAndTransform_LooseReject(t *testing.T) {
	s := builders.Object().
		Field("name", builders.String()).
		AllowAdditional(false)

	ctx := queryfy.NewValidationContext(queryfy.Loose)
	_, err := s.ValidateAndTransform(map[string]interface{}{
		"name":  "Alice",
		"extra": "rejected",
	}, ctx)
	if err == nil {
		t.Error("expected error for extra field with AllowAdditional(false) in loose mode")
	}
}

func TestAllowAdditional_Default_PreservesBackwardCompat(t *testing.T) {
	s := builders.Object().Field("name", builders.String())

	// Strict: rejects extra (backward compat)
	ctx1 := queryfy.NewValidationContext(queryfy.Strict)
	s.Validate(map[string]interface{}{
		"name": "Alice", "extra": "x",
	}, ctx1)
	if !ctx1.HasErrors() {
		t.Error("default strict should reject extra (backward compat)")
	}

	// Loose: accepts extra (backward compat)
	ctx2 := queryfy.NewValidationContext(queryfy.Loose)
	s.Validate(map[string]interface{}{
		"name": "Alice", "extra": "x",
	}, ctx2)
	if ctx2.HasErrors() {
		t.Error("default loose should accept extra (backward compat)")
	}
}
