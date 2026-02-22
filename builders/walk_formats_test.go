package builders_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/ha1tch/queryfy"
	"github.com/ha1tch/queryfy/builders"
)

// ======================================================================
// 2.1 Custom Format Registry
// ======================================================================

func TestRegisterFormat_Basic(t *testing.T) {
	semverRe := regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	builders.RegisterFormat("semver", func(value interface{}) error {
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string")
		}
		if !semverRe.MatchString(str) {
			return fmt.Errorf("must be valid semver (X.Y.Z)")
		}
		return nil
	})

	s := builders.String().FormatString("semver")

	expectValid(t, s, "1.2.3")
	expectValid(t, s, "0.0.1")
	expectInvalid(t, s, "1.2")
	expectInvalid(t, s, "v1.2.3")
	expectInvalid(t, s, "not-semver")
}

func TestRegisterFormat_FormatType(t *testing.T) {
	builders.RegisterFormat("ipv4", func(value interface{}) error {
		return nil // simplified
	})

	s := builders.String().FormatString("ipv4")
	if s.FormatType() != "ipv4" {
		t.Errorf("expected formatType 'ipv4', got %q", s.FormatType())
	}
}

func TestRegisterFormat_UnknownFormat(t *testing.T) {
	s := builders.String().FormatString("nonexistent_format_xyz")
	expectInvalid(t, s, "anything")
}

func TestRegisterFormat_LateRegistration(t *testing.T) {
	// FormatString used before the format is registered
	s := builders.String().FormatString("late_format")

	// Now register it
	builders.RegisterFormat("late_format", func(value interface{}) error {
		if value.(string) != "valid" {
			return fmt.Errorf("must be 'valid'")
		}
		return nil
	})

	expectValid(t, s, "valid")
	expectInvalid(t, s, "invalid")
}

func TestRegisterFormat_Overwrite(t *testing.T) {
	builders.RegisterFormat("overtest", func(value interface{}) error {
		return fmt.Errorf("old")
	})
	builders.RegisterFormat("overtest", func(value interface{}) error {
		return nil // new: accepts everything
	})

	s := builders.String().FormatString("overtest")
	expectValid(t, s, "anything")
}

func TestLookupFormat(t *testing.T) {
	builders.RegisterFormat("lookup_test", func(value interface{}) error {
		return nil
	})

	if builders.LookupFormat("lookup_test") == nil {
		t.Error("expected non-nil validator")
	}
	if builders.LookupFormat("does_not_exist") != nil {
		t.Error("expected nil for unregistered format")
	}
}

func TestRegisteredFormats(t *testing.T) {
	// We've registered several formats above; just check it returns something
	names := builders.RegisteredFormats()
	if len(names) == 0 {
		t.Error("expected at least one registered format")
	}
}

func TestFormatString_WithOtherConstraints(t *testing.T) {
	builders.RegisterFormat("hex", func(value interface{}) error {
		str := value.(string)
		for _, c := range str {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return fmt.Errorf("must be hex")
			}
		}
		return nil
	})

	// FormatString + MinLength + MaxLength should all work together
	s := builders.String().FormatString("hex").MinLength(2).MaxLength(8)
	expectValid(t, s, "deadBEEF")
	expectValid(t, s, "ff")
	expectInvalid(t, s, "f")       // too short
	expectInvalid(t, s, "xyz")     // not hex
	expectInvalid(t, s, "deadbeef00") // too long
}

// ======================================================================
// 3.2 Field Visitor / Walker
// ======================================================================

func TestWalk_SimpleObject(t *testing.T) {
	schema := builders.Object().
		Field("name", builders.String().Required()).
		Field("age", builders.Number())

	var paths []string
	err := builders.Walk(schema, func(path string, s queryfy.Schema) error {
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		t.Fatalf("Walk error: %v", err)
	}

	// Should visit: "" (root), "age", "name" (sorted)
	expected := []string{"", "age", "name"}
	if len(paths) != len(expected) {
		t.Fatalf("expected %d visits, got %d: %v", len(expected), len(paths), paths)
	}
	for i, p := range expected {
		if paths[i] != p {
			t.Errorf("path[%d]: expected %q, got %q", i, p, paths[i])
		}
	}
}

func TestWalk_NestedObject(t *testing.T) {
	schema := builders.Object().
		Field("user", builders.Object().
			Field("name", builders.String()).
			Field("address", builders.Object().
				Field("city", builders.String()).
				Field("zip", builders.String())))

	var paths []string
	builders.Walk(schema, func(path string, s queryfy.Schema) error {
		paths = append(paths, path)
		return nil
	})

	// Root, user, user.address, user.address.city, user.address.zip, user.name
	if !contains(paths, "user.address.city") {
		t.Errorf("expected 'user.address.city' in paths: %v", paths)
	}
	if !contains(paths, "user.name") {
		t.Errorf("expected 'user.name' in paths: %v", paths)
	}
}

func TestWalk_Array(t *testing.T) {
	schema := builders.Object().
		Field("tags", builders.Array().Of(builders.String()))

	var paths []string
	builders.Walk(schema, func(path string, s queryfy.Schema) error {
		paths = append(paths, path)
		return nil
	})

	if !contains(paths, "tags") {
		t.Errorf("expected 'tags' in paths: %v", paths)
	}
	if !contains(paths, "tags[*]") {
		t.Errorf("expected 'tags[*]' in paths: %v", paths)
	}
}

func TestWalk_ArrayOfObjects(t *testing.T) {
	schema := builders.Object().
		Field("items", builders.Array().Of(
			builders.Object().
				Field("sku", builders.String()).
				Field("qty", builders.Number())))

	var paths []string
	builders.Walk(schema, func(path string, s queryfy.Schema) error {
		paths = append(paths, path)
		return nil
	})

	if !contains(paths, "items[*].sku") {
		t.Errorf("expected 'items[*].sku' in paths: %v", paths)
	}
	if !contains(paths, "items[*].qty") {
		t.Errorf("expected 'items[*].qty' in paths: %v", paths)
	}
}

func TestWalk_TransformSchema(t *testing.T) {
	schema := builders.Object().
		Field("email", builders.Transform(builders.String().Email()))

	var types []queryfy.SchemaType
	builders.Walk(schema, func(path string, s queryfy.Schema) error {
		if path == "email" {
			types = append(types, s.Type())
		}
		return nil
	})

	// Should visit the TransformSchema at "email"
	if len(types) == 0 {
		t.Error("expected at least one visit at 'email'")
	}
}

func TestWalk_ErrorPropagation(t *testing.T) {
	schema := builders.Object().
		Field("a", builders.String()).
		Field("b", builders.String()).
		Field("c", builders.String())

	sentinel := fmt.Errorf("stop")
	err := builders.Walk(schema, func(path string, s queryfy.Schema) error {
		if path == "b" {
			return sentinel
		}
		return nil
	})

	if err != sentinel {
		t.Errorf("expected sentinel error, got %v", err)
	}
}

func TestWalk_LeafOnly(t *testing.T) {
	// Collect only leaf nodes (non-object, non-array)
	schema := builders.Object().
		Field("name", builders.String()).
		Field("scores", builders.Array().Of(builders.Number())).
		Field("meta", builders.Object().
			Field("tag", builders.String()))

	var leaves []string
	builders.Walk(schema, func(path string, s queryfy.Schema) error {
		typ := s.Type()
		if typ != queryfy.TypeObject && typ != queryfy.TypeArray && path != "" {
			leaves = append(leaves, path)
		}
		return nil
	})

	expected := []string{"meta.tag", "name", "scores[*]"}
	if len(leaves) != len(expected) {
		t.Fatalf("expected leaves %v, got %v", expected, leaves)
	}
	for i, l := range expected {
		if leaves[i] != l {
			t.Errorf("leaf[%d]: expected %q, got %q", i, l, leaves[i])
		}
	}
}

func TestWalk_CollectRequired(t *testing.T) {
	schema := builders.Object().
		Field("id", builders.String().Required()).
		Field("name", builders.String().Required()).
		Field("bio", builders.String()).
		Field("address", builders.Object().
			Field("street", builders.String().Required()).
			Field("city", builders.String()))

	var required []string
	builders.Walk(schema, func(path string, s queryfy.Schema) error {
		if path != "" {
			if req, ok := s.(interface{ IsRequired() bool }); ok && req.IsRequired() {
				required = append(required, path)
			}
		}
		return nil
	})

	// id, name, address.street
	if len(required) != 3 {
		t.Errorf("expected 3 required fields, got %d: %v", len(required), required)
	}
	if !contains(required, "address.street") {
		t.Errorf("expected 'address.street' in required: %v", required)
	}
}

func TestWalk_EmptyObject(t *testing.T) {
	schema := builders.Object()
	count := 0
	builders.Walk(schema, func(path string, s queryfy.Schema) error {
		count++
		return nil
	})
	if count != 1 {
		t.Errorf("expected 1 visit (root only), got %d", count)
	}
}

func TestWalk_PlainString(t *testing.T) {
	// Walk on a non-object schema should just visit the root
	schema := builders.String()
	count := 0
	builders.Walk(schema, func(path string, s queryfy.Schema) error {
		count++
		return nil
	})
	if count != 1 {
		t.Errorf("expected 1 visit, got %d", count)
	}
}

func TestWalk_CompositeAnd(t *testing.T) {
	schema := builders.And(
		builders.String().MinLength(1),
		builders.String().MaxLength(100),
	)

	var paths []string
	builders.Walk(schema, func(path string, s queryfy.Schema) error {
		paths = append(paths, path)
		return nil
	})

	// Root "", <and[0]>, <and[1]>
	if len(paths) != 3 {
		t.Errorf("expected 3 visits, got %d: %v", len(paths), paths)
	}
}

// ======================================================================
// Composite introspection
// ======================================================================

func TestAndSchema_Schemas(t *testing.T) {
	s := builders.And(builders.String(), builders.Number())
	schemas := s.Schemas()
	if len(schemas) != 2 {
		t.Errorf("expected 2 sub-schemas, got %d", len(schemas))
	}
}

func TestOrSchema_Schemas(t *testing.T) {
	s := builders.Or(builders.String(), builders.Number(), builders.Bool())
	schemas := s.Schemas()
	if len(schemas) != 3 {
		t.Errorf("expected 3 sub-schemas, got %d", len(schemas))
	}
}

func TestNotSchema_InnerSchema(t *testing.T) {
	inner := builders.String().Email()
	s := builders.Not(inner)
	if s.InnerSchema() != inner {
		t.Error("expected same inner schema back")
	}
}

func TestTransformSchema_InnerSchema(t *testing.T) {
	inner := builders.String().Required()
	s := builders.Transform(inner)
	if s.InnerSchema().Type() != queryfy.TypeString {
		t.Errorf("expected TypeString, got %v", s.InnerSchema().Type())
	}
}

// ======================================================================
// helpers
// ======================================================================

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// collectPaths is a convenience for tests that want all visited paths.
func collectPaths(t *testing.T, schema queryfy.Schema) []string {
	t.Helper()
	var paths []string
	err := builders.Walk(schema, func(path string, s queryfy.Schema) error {
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		t.Fatalf("Walk error: %v", err)
	}
	return paths
}

// dumpPaths prints all paths for debugging.
func dumpPaths(paths []string) string {
	return "[" + strings.Join(paths, ", ") + "]"
}
