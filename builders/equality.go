// equality.go - Schema equality and hashing
package builders

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/ha1tch/queryfy"
)

// Hash returns a deterministic SHA-256 hash of the schema's structure
// and constraints. Two schemas that are structurally identical produce
// the same hash.
//
// Custom validators and async validators are excluded from the hash
// because Go functions are not comparable or serialisable. Two schemas
// that differ only in custom validators will produce the same hash.
//
// The hash is stable across process restarts.
func Hash(schema queryfy.Schema) string {
	h := sha256.New()
	h.Write([]byte(canonicalise(schema)))
	return hex.EncodeToString(h.Sum(nil))
}

// Equal reports whether two schemas are structurally equivalent.
//
// Like Hash, this excludes custom validators and async validators.
// Two schemas that differ only in custom validators are considered equal.
func Equal(a, b queryfy.Schema) bool {
	return canonicalise(a) == canonicalise(b)
}

// canonicalise produces a deterministic string representation of a
// schema's structure and constraints.
func canonicalise(schema queryfy.Schema) string {
	var b strings.Builder
	canonicaliseNode(&b, schema)
	return b.String()
}

func canonicaliseNode(b *strings.Builder, schema queryfy.Schema) {
	if schema == nil {
		b.WriteString("null")
		return
	}

	switch s := schema.(type) {
	case *StringSchema:
		canonicaliseString(b, s)
	case *NumberSchema:
		canonicaliseNumber(b, s)
	case *BoolSchema:
		canonicaliseBool(b, s)
	case *DateTimeSchema:
		canonicaliseDateTime(b, s)
	case *ObjectSchema:
		canonicaliseObject(b, s)
	case *ObjectSchemaWithDependencies:
		// Dependent schemas canonicalise as their base object
		// (dependency conditions involve functions — not comparable)
		canonicaliseObject(b, s.ObjectSchema)
		b.WriteString(";deps")
	case *ArraySchema:
		canonicaliseArray(b, s)
	case *CustomSchema:
		b.WriteString("custom")
		canonicaliseBase(b, &s.BaseSchema)
	case *TransformSchema:
		b.WriteString("transform(")
		canonicaliseNode(b, s.InnerSchema())
		b.WriteString(")")
		canonicaliseBase(b, &s.BaseSchema)
	case *AndSchema:
		b.WriteString("and(")
		for i, sub := range s.Schemas() {
			if i > 0 {
				b.WriteString(",")
			}
			canonicaliseNode(b, sub)
		}
		b.WriteString(")")
	case *OrSchema:
		b.WriteString("or(")
		for i, sub := range s.Schemas() {
			if i > 0 {
				b.WriteString(",")
			}
			canonicaliseNode(b, sub)
		}
		b.WriteString(")")
	case *NotSchema:
		b.WriteString("not(")
		canonicaliseNode(b, s.InnerSchema())
		b.WriteString(")")
	default:
		// Unknown schema type — use type name
		b.WriteString(fmt.Sprintf("unknown<%T>", schema))
		b.WriteString(fmt.Sprintf(";type=%v", schema.Type()))
	}
}

func canonicaliseBase(b *strings.Builder, base *queryfy.BaseSchema) {
	if base.IsRequired() {
		b.WriteString(";req")
	}
	if base.IsNullable() {
		b.WriteString(";null")
	}
	meta := base.AllMeta()
	if len(meta) > 0 {
		canonicaliseMeta(b, meta)
	}
}

func canonicaliseMeta(b *strings.Builder, meta map[string]interface{}) {
	// Sort keys for determinism
	keys := make([]string, 0, len(meta))
	for k := range meta {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	b.WriteString(";meta{")
	for i, k := range keys {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf("%s=%v", k, meta[k]))
	}
	b.WriteString("}")
}

func canonicaliseString(b *strings.Builder, s *StringSchema) {
	b.WriteString("string")
	canonicaliseBase(b, &s.BaseSchema)

	if ft := s.FormatType(); ft != "" {
		b.WriteString(fmt.Sprintf(";fmt=%s", ft))
	}
	min, max := s.LengthConstraints()
	if min != nil {
		b.WriteString(fmt.Sprintf(";minLen=%d", *min))
	}
	if max != nil {
		b.WriteString(fmt.Sprintf(";maxLen=%d", *max))
	}
	if p := s.PatternString(); p != "" {
		b.WriteString(fmt.Sprintf(";pat=%s", p))
	}
	if vals := s.EnumValues(); len(vals) > 0 {
		b.WriteString(fmt.Sprintf(";enum=[%s]", strings.Join(vals, ",")))
	}
}

func canonicaliseNumber(b *strings.Builder, s *NumberSchema) {
	b.WriteString("number")
	canonicaliseBase(b, &s.BaseSchema)

	if s.IsInteger() {
		b.WriteString(";int")
	}
	min, max := s.RangeConstraints()
	if min != nil {
		b.WriteString(fmt.Sprintf(";min=%g", *min))
	}
	if max != nil {
		b.WriteString(fmt.Sprintf(";max=%g", *max))
	}
	if m := s.MultipleOfValue(); m != nil {
		b.WriteString(fmt.Sprintf(";mul=%g", *m))
	}
}

func canonicaliseBool(b *strings.Builder, s *BoolSchema) {
	b.WriteString("bool")
	canonicaliseBase(b, &s.BaseSchema)
}

func canonicaliseDateTime(b *strings.Builder, s *DateTimeSchema) {
	b.WriteString("datetime")
	canonicaliseBase(b, &s.BaseSchema)

	b.WriteString(fmt.Sprintf(";fmt=%s", s.FormatString()))
	if s.IsStrictFormat() {
		b.WriteString(";strict")
	}
	min, max := s.TimeConstraints()
	if min != nil {
		b.WriteString(fmt.Sprintf(";min=%s", min.UTC().Format("2006-01-02T15:04:05Z")))
	}
	if max != nil {
		b.WriteString(fmt.Sprintf(";max=%s", max.UTC().Format("2006-01-02T15:04:05Z")))
	}
}

func canonicaliseObject(b *strings.Builder, s *ObjectSchema) {
	b.WriteString("object")
	canonicaliseBase(b, &s.BaseSchema)

	allow, explicit := s.AllowsAdditional()
	if explicit {
		b.WriteString(fmt.Sprintf(";additional=%v", allow))
	}

	// Fields in sorted order
	names := s.FieldNames()
	b.WriteString("{")
	for i, name := range names {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(name)
		b.WriteString(":")
		field, _ := s.GetField(name)
		canonicaliseNode(b, field)
	}
	b.WriteString("}")
}

func canonicaliseArray(b *strings.Builder, s *ArraySchema) {
	b.WriteString("array")
	canonicaliseBase(b, &s.BaseSchema)

	min, max := s.ItemCountConstraints()
	if min != nil {
		b.WriteString(fmt.Sprintf(";minItems=%d", *min))
	}
	if max != nil {
		b.WriteString(fmt.Sprintf(";maxItems=%d", *max))
	}

	elem := s.ElementSchema()
	if elem != nil {
		b.WriteString(";of=")
		canonicaliseNode(b, elem)
	}
}
