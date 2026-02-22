package builders_test

import (
	"testing"

	"github.com/ha1tch/queryfy/builders"
)

// ======================================================================
// 2.4 Schema Hashing
// ======================================================================

func TestHash_Deterministic(t *testing.T) {
	s1 := builders.String().Required().MinLength(3).MaxLength(50)
	s2 := builders.String().Required().MinLength(3).MaxLength(50)

	h1 := builders.Hash(s1)
	h2 := builders.Hash(s2)

	if h1 != h2 {
		t.Errorf("identical schemas should produce same hash:\n  %s\n  %s", h1, h2)
	}
	if len(h1) != 64 {
		t.Errorf("expected 64-char hex SHA-256, got %d chars", len(h1))
	}
}

func TestHash_DifferentConstraints(t *testing.T) {
	s1 := builders.String().MinLength(3)
	s2 := builders.String().MinLength(5)

	if builders.Hash(s1) == builders.Hash(s2) {
		t.Error("different constraints should produce different hashes")
	}
}

func TestHash_DifferentTypes(t *testing.T) {
	s := builders.String()
	n := builders.Number()

	if builders.Hash(s) == builders.Hash(n) {
		t.Error("different types should produce different hashes")
	}
}

func TestHash_RequiredVsOptional(t *testing.T) {
	s1 := builders.String().Required()
	s2 := builders.String()

	if builders.Hash(s1) == builders.Hash(s2) {
		t.Error("required vs optional should produce different hashes")
	}
}

func TestHash_ComplexObject(t *testing.T) {
	makeSchema := func() *builders.ObjectSchema {
		return builders.Object().
			Field("name", builders.String().Required().MinLength(1).MaxLength(255)).
			Field("age", builders.Number().Integer().Min(0).Max(150)).
			Field("tags", builders.Array().Of(builders.String()).MinItems(0).MaxItems(20)).
			AllowAdditional(false)
	}

	h1 := builders.Hash(makeSchema())
	h2 := builders.Hash(makeSchema())

	if h1 != h2 {
		t.Errorf("identical complex schemas should hash equal:\n  %s\n  %s", h1, h2)
	}
}

func TestHash_FieldOrder(t *testing.T) {
	// Fields should be sorted, so declaration order doesn't matter
	s1 := builders.Object().
		Field("b", builders.String()).
		Field("a", builders.Number())

	s2 := builders.Object().
		Field("a", builders.Number()).
		Field("b", builders.String())

	if builders.Hash(s1) != builders.Hash(s2) {
		t.Error("field declaration order should not affect hash")
	}
}

func TestHash_MetadataMatters(t *testing.T) {
	s1 := builders.String().Meta("db_type", "VARCHAR")
	s2 := builders.String().Meta("db_type", "TEXT")

	if builders.Hash(s1) == builders.Hash(s2) {
		t.Error("different metadata values should produce different hashes")
	}
}

func TestHash_MetadataAbsence(t *testing.T) {
	s1 := builders.String()
	s2 := builders.String().Meta("tag", "x")

	if builders.Hash(s1) == builders.Hash(s2) {
		t.Error("presence of metadata should affect hash")
	}
}

func TestHash_CustomValidatorIgnored(t *testing.T) {
	// Custom validators are excluded — schemas with different custom
	// validators but same structure should hash equal
	s1 := builders.String().Required().Custom(func(v interface{}) error {
		return nil
	})
	s2 := builders.String().Required()

	if builders.Hash(s1) != builders.Hash(s2) {
		t.Error("custom validators should not affect hash")
	}
}

func TestHash_TransformSchema(t *testing.T) {
	s1 := builders.Transform(builders.String().Required())
	s2 := builders.Transform(builders.String().Required())

	if builders.Hash(s1) != builders.Hash(s2) {
		t.Error("identical transform schemas should hash equal")
	}
}

func TestHash_Composites(t *testing.T) {
	s1 := builders.And(builders.String().MinLength(1), builders.String().MaxLength(100))
	s2 := builders.And(builders.String().MinLength(1), builders.String().MaxLength(100))

	if builders.Hash(s1) != builders.Hash(s2) {
		t.Error("identical And composites should hash equal")
	}
}

func TestHash_DateTime(t *testing.T) {
	s1 := builders.DateTime().DateOnly().StrictFormat()
	s2 := builders.DateTime().DateOnly().StrictFormat()
	s3 := builders.DateTime().DateOnly() // not strict

	if builders.Hash(s1) != builders.Hash(s2) {
		t.Error("identical datetime schemas should hash equal")
	}
	if builders.Hash(s1) == builders.Hash(s3) {
		t.Error("strict vs non-strict should differ")
	}
}

// ======================================================================
// 2.4 Schema Equality
// ======================================================================

func TestEqual_Identical(t *testing.T) {
	s1 := builders.String().Required().Email()
	s2 := builders.String().Required().Email()

	if !builders.Equal(s1, s2) {
		t.Error("identical schemas should be equal")
	}
}

func TestEqual_Different(t *testing.T) {
	s1 := builders.String().Required().Email()
	s2 := builders.String().Required().URL()

	if builders.Equal(s1, s2) {
		t.Error("different format types should not be equal")
	}
}

func TestEqual_ComplexObject(t *testing.T) {
	makeSchema := func() *builders.ObjectSchema {
		return builders.Object().
			Field("id", builders.String().Required()).
			Field("value", builders.Number().Min(0)).
			AllowAdditional(false)
	}

	if !builders.Equal(makeSchema(), makeSchema()) {
		t.Error("identical complex schemas should be equal")
	}
}

func TestEqual_FieldDifference(t *testing.T) {
	s1 := builders.Object().
		Field("a", builders.String()).
		Field("b", builders.Number())

	s2 := builders.Object().
		Field("a", builders.String()).
		Field("c", builders.Number())

	if builders.Equal(s1, s2) {
		t.Error("different field names should not be equal")
	}
}

func TestEqual_NullableMatters(t *testing.T) {
	s1 := builders.String().Nullable()
	s2 := builders.String()

	if builders.Equal(s1, s2) {
		t.Error("nullable vs non-nullable should not be equal")
	}
}

func TestEqual_AllowAdditionalMatters(t *testing.T) {
	s1 := builders.Object().AllowAdditional(true)
	s2 := builders.Object().AllowAdditional(false)
	s3 := builders.Object() // default (nil)

	if builders.Equal(s1, s2) {
		t.Error("allow(true) vs allow(false) should differ")
	}
	if builders.Equal(s1, s3) {
		t.Error("allow(true) vs default should differ")
	}
}
