// diff.go - Schema structural diff
package builders

import (
	"fmt"

	"github.com/ha1tch/queryfy"
)

// FieldChange describes a change between two versions of a field.
type FieldChange struct {
	Path    string          // dot-notation path
	OldType queryfy.SchemaType
	NewType queryfy.SchemaType
	Details string          // human-readable description
}

// SchemaDiff describes the structural differences between two schemas.
type SchemaDiff struct {
	Added   []string       // field paths present in new but not old
	Removed []string       // field paths present in old but not new
	Changed []FieldChange  // fields present in both but structurally different
}

// HasChanges reports whether any differences were found.
func (d *SchemaDiff) HasChanges() bool {
	return len(d.Added) > 0 || len(d.Removed) > 0 || len(d.Changed) > 0
}

// Diff compares two schemas and returns a structured diff describing
// what changed: added fields, removed fields, and changed fields.
//
// Both schemas must be ObjectSchema (or ObjectSchemaWithDependencies).
// For non-object schemas, Diff compares them as single values.
func Diff(old, new queryfy.Schema) (*SchemaDiff, error) {
	result := &SchemaDiff{}

	oldFields := collectFields(old)
	newFields := collectFields(new)

	// If neither is an object, compare directly
	if len(oldFields) == 0 && len(newFields) == 0 {
		if !Equal(old, new) {
			result.Changed = append(result.Changed, FieldChange{
				Path:    "",
				OldType: old.Type(),
				NewType: new.Type(),
				Details: "schema changed",
			})
		}
		return result, nil
	}

	// Build maps for lookup
	oldMap := make(map[string]queryfy.Schema, len(oldFields))
	for _, f := range oldFields {
		oldMap[f.path] = f.schema
	}
	newMap := make(map[string]queryfy.Schema, len(newFields))
	for _, f := range newFields {
		newMap[f.path] = f.schema
	}

	// Find removed (in old, not in new)
	for _, f := range oldFields {
		if _, exists := newMap[f.path]; !exists {
			result.Removed = append(result.Removed, f.path)
		}
	}

	// Find added (in new, not in old)
	for _, f := range newFields {
		if _, exists := oldMap[f.path]; !exists {
			result.Added = append(result.Added, f.path)
		}
	}

	// Find changed (in both, but different)
	for _, f := range newFields {
		oldSchema, exists := oldMap[f.path]
		if !exists {
			continue // already in Added
		}
		if !Equal(oldSchema, f.schema) {
			change := FieldChange{
				Path:    f.path,
				OldType: oldSchema.Type(),
				NewType: f.schema.Type(),
			}
			change.Details = describeChange(oldSchema, f.schema)
			result.Changed = append(result.Changed, change)
		}
	}

	return result, nil
}

// fieldEntry pairs a path with its schema.
type fieldEntry struct {
	path   string
	schema queryfy.Schema
}

// collectFields walks a schema and returns all leaf fields with paths.
// Intermediate containers (objects, arrays) are excluded — their changes
// are expressed through their children.
func collectFields(schema queryfy.Schema) []fieldEntry {
	var fields []fieldEntry
	Walk(schema, func(path string, s queryfy.Schema) error {
		if path == "" {
			return nil // skip root
		}
		// Skip intermediate containers — their structural changes are
		// captured by added/removed/changed children
		switch s.(type) {
		case *ObjectSchema, *ObjectSchemaWithDependencies:
			return nil
		case *ArraySchema:
			// Include arrays only if they have no element schema
			// (i.e., they're leaves). Arrays with elements are
			// represented by their [*] child.
			arr := s.(*ArraySchema)
			if arr.ElementSchema() != nil {
				return nil
			}
		}
		fields = append(fields, fieldEntry{path: path, schema: s})
		return nil
	})
	return fields
}

// describeChange produces a human-readable description of what changed.
func describeChange(old, new queryfy.Schema) string {
	if old.Type() != new.Type() {
		return fmt.Sprintf("type changed from %v to %v", old.Type(), new.Type())
	}

	var details []string

	// Check required
	oldReq := isRequired(old)
	newReq := isRequired(new)
	if oldReq != newReq {
		if newReq {
			details = append(details, "became required")
		} else {
			details = append(details, "became optional")
		}
	}

	// Check nullable
	oldNull := isNullable(old)
	newNull := isNullable(new)
	if oldNull != newNull {
		if newNull {
			details = append(details, "became nullable")
		} else {
			details = append(details, "became non-nullable")
		}
	}

	// Type-specific changes
	switch o := old.(type) {
	case *StringSchema:
		n := new.(*StringSchema)
		describeStringChange(o, n, &details)
	case *NumberSchema:
		n := new.(*NumberSchema)
		describeNumberChange(o, n, &details)
	}

	if len(details) == 0 {
		return "constraints changed"
	}
	result := ""
	for i, d := range details {
		if i > 0 {
			result += "; "
		}
		result += d
	}
	return result
}

func describeStringChange(old, new *StringSchema, details *[]string) {
	if old.FormatType() != new.FormatType() {
		*details = append(*details, fmt.Sprintf("format: %q -> %q", old.FormatType(), new.FormatType()))
	}
	oldMin, oldMax := old.LengthConstraints()
	newMin, newMax := new.LengthConstraints()
	if !ptrEqInt(oldMin, newMin) {
		*details = append(*details, fmt.Sprintf("minLength: %s -> %s", fmtPtrInt(oldMin), fmtPtrInt(newMin)))
	}
	if !ptrEqInt(oldMax, newMax) {
		*details = append(*details, fmt.Sprintf("maxLength: %s -> %s", fmtPtrInt(oldMax), fmtPtrInt(newMax)))
	}
	if old.PatternString() != new.PatternString() {
		*details = append(*details, fmt.Sprintf("pattern: %q -> %q", old.PatternString(), new.PatternString()))
	}
}

func describeNumberChange(old, new *NumberSchema, details *[]string) {
	if old.IsInteger() != new.IsInteger() {
		*details = append(*details, fmt.Sprintf("integer: %v -> %v", old.IsInteger(), new.IsInteger()))
	}
	oldMin, oldMax := old.RangeConstraints()
	newMin, newMax := new.RangeConstraints()
	if !ptrEqFloat(oldMin, newMin) {
		*details = append(*details, fmt.Sprintf("min: %s -> %s", fmtPtrFloat(oldMin), fmtPtrFloat(newMin)))
	}
	if !ptrEqFloat(oldMax, newMax) {
		*details = append(*details, fmt.Sprintf("max: %s -> %s", fmtPtrFloat(oldMax), fmtPtrFloat(newMax)))
	}
}

func isNullable(s queryfy.Schema) bool {
	if n, ok := s.(interface{ IsNullable() bool }); ok {
		return n.IsNullable()
	}
	return false
}

func ptrEqInt(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func ptrEqFloat(a, b *float64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func fmtPtrInt(p *int) string {
	if p == nil {
		return "none"
	}
	return fmt.Sprintf("%d", *p)
}

func fmtPtrFloat(p *float64) string {
	if p == nil {
		return "none"
	}
	return fmt.Sprintf("%g", *p)
}
