package queryfy_test

import (
	"fmt"
	"testing"

	"github.com/ha1tch/queryfy"
	"github.com/ha1tch/queryfy/builders"
)

// ======================================================================
// Compile returns CompiledSchema
// ======================================================================

func TestCompile_ReturnsCompiled(t *testing.T) {
	schema := builders.String().Required()
	compiled := queryfy.Compile(schema)

	if _, ok := compiled.(*queryfy.CompiledSchema); !ok {
		t.Errorf("expected *CompiledSchema, got %T", compiled)
	}
}

func TestCompile_Idempotent(t *testing.T) {
	schema := builders.String()
	compiled := queryfy.Compile(schema)
	recompiled := queryfy.Compile(compiled)

	// Should return the same object, not double-wrap
	if compiled != recompiled {
		t.Error("Compile should be idempotent")
	}
}

func TestCompile_PreservesType(t *testing.T) {
	tests := []struct {
		name   string
		schema queryfy.Schema
		want   queryfy.SchemaType
	}{
		{"string", builders.String(), queryfy.TypeString},
		{"number", builders.Number(), queryfy.TypeNumber},
		{"integer", builders.Number().Integer(), queryfy.TypeNumber},
		{"bool", builders.Bool(), queryfy.TypeBool},
		{"object", builders.Object(), queryfy.TypeObject},
		{"array", builders.Array(), queryfy.TypeArray},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiled := queryfy.Compile(tt.schema)
			if compiled.Type() != tt.want {
				t.Errorf("expected %v, got %v", tt.want, compiled.Type())
			}
		})
	}
}

// ======================================================================
// String: compiled validates identically to uncompiled
// ======================================================================

func TestCompileString_MinLength(t *testing.T) {
	raw := builders.String().MinLength(3)
	compiled := queryfy.Compile(raw)

	assertBothAgree(t, raw, compiled, "abc", true)
	assertBothAgree(t, raw, compiled, "abcdef", true)
	assertBothAgree(t, raw, compiled, "ab", false)
}

func TestCompileString_MaxLength(t *testing.T) {
	raw := builders.String().MaxLength(5)
	compiled := queryfy.Compile(raw)

	assertBothAgree(t, raw, compiled, "hello", true)
	assertBothAgree(t, raw, compiled, "toolong", false)
}

func TestCompileString_Pattern(t *testing.T) {
	raw := builders.String().Pattern("^[A-Z]+$")
	compiled := queryfy.Compile(raw)

	assertBothAgree(t, raw, compiled, "ABC", true)
	assertBothAgree(t, raw, compiled, "abc", false)
}

func TestCompileString_Email(t *testing.T) {
	raw := builders.String().Email()
	compiled := queryfy.Compile(raw)

	assertBothAgree(t, raw, compiled, "user@example.com", true)
	assertBothAgree(t, raw, compiled, "bad", false)
}

func TestCompileString_Enum(t *testing.T) {
	raw := builders.String().Enum("red", "green", "blue")
	compiled := queryfy.Compile(raw)

	assertBothAgree(t, raw, compiled, "red", true)
	assertBothAgree(t, raw, compiled, "yellow", false)
}

func TestCompileString_Required(t *testing.T) {
	raw := builders.String().Required()
	compiled := queryfy.Compile(raw)

	assertBothAgree(t, raw, compiled, "hello", true)
	assertBothAgree(t, raw, compiled, nil, false)
}

func TestCompileString_TypeMismatch(t *testing.T) {
	raw := builders.String()
	compiled := queryfy.Compile(raw)

	assertBothAgree(t, raw, compiled, 42, false)
}

func TestCompileString_CustomValidator(t *testing.T) {
	raw := builders.String().Custom(func(v interface{}) error {
		if v.(string) == "forbidden" {
			return fmt.Errorf("forbidden value")
		}
		return nil
	})
	compiled := queryfy.Compile(raw)

	assertBothAgree(t, raw, compiled, "ok", true)
	assertBothAgree(t, raw, compiled, "forbidden", false)
}

// ======================================================================
// Number: compiled validates identically
// ======================================================================

func TestCompileNumber_Range(t *testing.T) {
	raw := builders.Number().Min(0).Max(100)
	compiled := queryfy.Compile(raw)

	assertBothAgree(t, raw, compiled, 50.0, true)
	assertBothAgree(t, raw, compiled, -1.0, false)
	assertBothAgree(t, raw, compiled, 101.0, false)
}

func TestCompileNumber_Integer(t *testing.T) {
	raw := builders.Number().Integer()
	compiled := queryfy.Compile(raw)

	assertBothAgree(t, raw, compiled, 5.0, true)
	assertBothAgree(t, raw, compiled, 5.5, false)
}

func TestCompileNumber_MultipleOf(t *testing.T) {
	raw := builders.Number().MultipleOf(3)
	compiled := queryfy.Compile(raw)

	assertBothAgree(t, raw, compiled, 9.0, true)
	assertBothAgree(t, raw, compiled, 7.0, false)
}

// ======================================================================
// Bool: compiled validates identically
// ======================================================================

func TestCompileBool_Basic(t *testing.T) {
	raw := builders.Bool().Required()
	compiled := queryfy.Compile(raw)

	assertBothAgree(t, raw, compiled, true, true)
	assertBothAgree(t, raw, compiled, false, true)
	assertBothAgree(t, raw, compiled, nil, false)
	assertBothAgree(t, raw, compiled, "true", false)
}

// ======================================================================
// Object: compiled validates identically
// ======================================================================

func TestCompileObject_Fields(t *testing.T) {
	raw := builders.Object().
		Field("name", builders.String().Required().MinLength(1)).
		Field("age", builders.Number().Min(0))

	compiled := queryfy.Compile(raw)

	valid := map[string]interface{}{"name": "Alice", "age": 25.0}
	assertBothAgree(t, raw, compiled, valid, true)

	missingReq := map[string]interface{}{"age": 25.0}
	assertBothAgree(t, raw, compiled, missingReq, false)

	badAge := map[string]interface{}{"name": "Alice", "age": -1.0}
	assertBothAgree(t, raw, compiled, badAge, false)
}

func TestCompileObject_AdditionalFalse(t *testing.T) {
	raw := builders.Object().
		Field("name", builders.String()).
		AllowAdditional(false)

	compiled := queryfy.Compile(raw)

	clean := map[string]interface{}{"name": "Alice"}
	assertBothAgree(t, raw, compiled, clean, true)

	extra := map[string]interface{}{"name": "Alice", "extra": "bad"}
	assertBothAgreeMode(t, raw, compiled, extra, false, queryfy.Loose)
}

func TestCompileObject_Nested(t *testing.T) {
	raw := builders.Object().
		Field("address", builders.Object().
			Field("city", builders.String().Required()).
			Field("zip", builders.String()))

	compiled := queryfy.Compile(raw)

	valid := map[string]interface{}{
		"address": map[string]interface{}{"city": "Houston"},
	}
	assertBothAgree(t, raw, compiled, valid, true)

	invalid := map[string]interface{}{
		"address": map[string]interface{}{},
	}
	assertBothAgree(t, raw, compiled, invalid, false)
}

// ======================================================================
// Array: compiled validates identically
// ======================================================================

func TestCompileArray_Items(t *testing.T) {
	raw := builders.Array().Of(builders.String().MinLength(1)).MinItems(1).MaxItems(5)
	compiled := queryfy.Compile(raw)

	valid := []interface{}{"a", "bc"}
	assertBothAgree(t, raw, compiled, valid, true)

	tooFew := []interface{}{}
	assertBothAgree(t, raw, compiled, tooFew, false)

	badItem := []interface{}{"a", ""}
	assertBothAgree(t, raw, compiled, badItem, false)
}

func TestCompileArray_UniqueItems(t *testing.T) {
	raw := builders.Array().Of(builders.String()).UniqueItems()
	compiled := queryfy.Compile(raw)

	unique := []interface{}{"a", "b", "c"}
	assertBothAgree(t, raw, compiled, unique, true)

	dupe := []interface{}{"a", "b", "a"}
	assertBothAgree(t, raw, compiled, dupe, false)
}

// ======================================================================
// Inner access
// ======================================================================

func TestCompiledSchema_Inner(t *testing.T) {
	raw := builders.String().Email()
	compiled := queryfy.Compile(raw)

	cs, ok := compiled.(*queryfy.CompiledSchema)
	if !ok {
		t.Fatal("expected *CompiledSchema")
	}
	if cs.Inner() != raw {
		t.Error("Inner() should return the original schema")
	}
}

// ======================================================================
// Fallback for unsupported types
// ======================================================================

func TestCompile_Composite(t *testing.T) {
	raw := builders.Or(builders.String(), builders.Number())
	compiled := queryfy.Compile(raw)

	// Should still work — falls back to delegating
	assertBothAgree(t, raw, compiled, "hello", true)
	assertBothAgree(t, raw, compiled, 42.0, true)
	assertBothAgree(t, raw, compiled, true, false)
}

// ======================================================================
// helpers
// ======================================================================

func assertBothAgree(t *testing.T, raw, compiled queryfy.Schema, value interface{}, expectValid bool) {
	t.Helper()
	assertBothAgreeMode(t, raw, compiled, value, expectValid, queryfy.Strict)
}

func assertBothAgreeMode(t *testing.T, raw, compiled queryfy.Schema, value interface{}, expectValid bool, mode queryfy.ValidationMode) {
	t.Helper()

	ctxRaw := queryfy.NewValidationContext(mode)
	raw.Validate(value, ctxRaw)
	rawHasErrors := ctxRaw.HasErrors()

	ctxCompiled := queryfy.NewValidationContext(mode)
	compiled.Validate(value, ctxCompiled)
	compiledHasErrors := ctxCompiled.HasErrors()

	if rawHasErrors != compiledHasErrors {
		t.Errorf("disagreement for %v: raw errors=%v, compiled errors=%v",
			value, rawHasErrors, compiledHasErrors)
		if rawHasErrors {
			t.Errorf("  raw: %v", ctxRaw.Error())
		}
		if compiledHasErrors {
			t.Errorf("  compiled: %v", ctxCompiled.Error())
		}
	}

	if expectValid && compiledHasErrors {
		t.Errorf("expected %v to be valid (compiled), got: %v", value, ctxCompiled.Error())
	}
	if !expectValid && !compiledHasErrors {
		t.Errorf("expected %v to be invalid (compiled)", value)
	}
}
