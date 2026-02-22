package jsonschema

import (
	"encoding/json"
	"sort"

	"github.com/ha1tch/queryfy"
	"github.com/ha1tch/queryfy/builders"
)

// ExportOptions controls the export behaviour.
type ExportOptions struct {
	// SchemaURI sets the $schema keyword. Empty string omits it.
	SchemaURI string

	// ID sets the $id keyword. Empty string omits it.
	ID string

	// IncludeMeta includes stored metadata as extension keywords in the
	// output (e.g., "x-custom": "value").
	IncludeMeta bool
}

// ToJSON converts a queryfy schema to a JSON Schema document.
// Returns the JSON bytes and any errors encountered during conversion.
// A nil opts uses default settings.
func ToJSON(schema queryfy.Schema, opts *ExportOptions) ([]byte, error) {
	if opts == nil {
		opts = &ExportOptions{}
	}

	raw := exportNode(schema, opts)

	if opts.SchemaURI != "" {
		raw["$schema"] = opts.SchemaURI
	}
	if opts.ID != "" {
		raw["$id"] = opts.ID
	}

	return json.MarshalIndent(raw, "", "  ")
}

// ToMap converts a queryfy schema to a map representation of JSON Schema.
// Useful when you need to manipulate the output before serialising.
func ToMap(schema queryfy.Schema, opts *ExportOptions) map[string]interface{} {
	if opts == nil {
		opts = &ExportOptions{}
	}

	raw := exportNode(schema, opts)

	if opts.SchemaURI != "" {
		raw["$schema"] = opts.SchemaURI
	}
	if opts.ID != "" {
		raw["$id"] = opts.ID
	}

	return raw
}

// exportNode converts a single schema node to its JSON Schema map form.
func exportNode(schema queryfy.Schema, opts *ExportOptions) map[string]interface{} {
	switch s := schema.(type) {
	case *builders.StringSchema:
		return exportString(s, opts)
	case *builders.NumberSchema:
		return exportNumber(s, opts)
	case *builders.BoolSchema:
		return exportBool(s, opts)
	case *builders.ObjectSchema:
		return exportObject(s, opts)
	case *builders.ArraySchema:
		return exportArray(s, opts)
	case *builders.TransformSchema:
		// Export the inner schema — transforms are a queryfy concept
		return exportNode(s.InnerSchema(), opts)
	default:
		return map[string]interface{}{}
	}
}

func exportString(s *builders.StringSchema, opts *ExportOptions) map[string]interface{} {
	out := makeBase(s, "string")

	minLen, maxLen := s.LengthConstraints()
	if minLen != nil {
		out["minLength"] = *minLen
	}
	if maxLen != nil {
		out["maxLength"] = *maxLen
	}

	if p := s.PatternString(); p != "" {
		out["pattern"] = p
	}

	if vals := s.EnumValues(); len(vals) > 0 {
		ivals := make([]interface{}, len(vals))
		for i, v := range vals {
			ivals[i] = v
		}
		out["enum"] = ivals
	}

	if f := s.FormatType(); f != "" {
		out["format"] = f
	}

	includeMeta(s, out, opts)
	return out
}

func exportNumber(s *builders.NumberSchema, opts *ExportOptions) map[string]interface{} {
	typeName := "number"
	if s.IsInteger() {
		typeName = "integer"
	}
	out := makeBase(s, typeName)

	min, max := s.RangeConstraints()

	// Check for exclusiveMinimum/Maximum stored as metadata
	if meta, ok := s.GetMeta("exclusiveMinimum"); ok {
		out["exclusiveMinimum"] = meta
	} else if min != nil {
		out["minimum"] = *min
	}

	if meta, ok := s.GetMeta("exclusiveMaximum"); ok {
		out["exclusiveMaximum"] = meta
	} else if max != nil {
		out["maximum"] = *max
	}

	if mul := s.MultipleOfValue(); mul != nil {
		out["multipleOf"] = *mul
	}

	includeMeta(s, out, opts)
	return out
}

func exportBool(s *builders.BoolSchema, opts *ExportOptions) map[string]interface{} {
	out := makeBase(s, "boolean")
	includeMeta(s, out, opts)
	return out
}

func exportObject(s *builders.ObjectSchema, opts *ExportOptions) map[string]interface{} {
	out := makeBase(s, "object")

	fieldNames := s.FieldNames()
	if len(fieldNames) > 0 {
		sort.Strings(fieldNames)

		properties := make(map[string]interface{})
		var required []interface{}

		for _, name := range fieldNames {
			fieldSchema, _ := s.GetField(name)
			properties[name] = exportNode(fieldSchema, opts)

			if isRequired(fieldSchema) {
				required = append(required, name)
			}
		}

		out["properties"] = properties
		if len(required) > 0 {
			out["required"] = required
		}
	}

	// Also check RequiredFieldNames for fields marked via RequiredFields()
	reqNames := s.RequiredFieldNames()
	if len(reqNames) > 0 {
		existing := make(map[string]bool)
		if req, ok := out["required"]; ok {
			for _, r := range req.([]interface{}) {
				existing[r.(string)] = true
			}
		}
		var merged []interface{}
		for k := range existing {
			merged = append(merged, k)
		}
		for _, name := range reqNames {
			if !existing[name] {
				merged = append(merged, name)
				existing[name] = true
			}
		}
		if len(merged) > 0 {
			// Sort for deterministic output
			strMerged := make([]string, len(merged))
			for i, m := range merged {
				strMerged[i] = m.(string)
			}
			sort.Strings(strMerged)
			sortedMerged := make([]interface{}, len(strMerged))
			for i, s := range strMerged {
				sortedMerged[i] = s
			}
			out["required"] = sortedMerged
		}
	}

	if allow, explicit := s.AllowsAdditional(); explicit {
		out["additionalProperties"] = allow
	}

	includeMeta(s, out, opts)
	return out
}

func exportArray(s *builders.ArraySchema, opts *ExportOptions) map[string]interface{} {
	out := makeBase(s, "array")

	minItems, maxItems := s.ItemCountConstraints()
	if minItems != nil {
		out["minItems"] = *minItems
	}
	if maxItems != nil {
		out["maxItems"] = *maxItems
	}

	if s.IsUniqueItems() {
		out["uniqueItems"] = true
	}

	if elem := s.ElementSchema(); elem != nil {
		out["items"] = exportNode(elem, opts)
	}

	includeMeta(s, out, opts)
	return out
}

// makeBase creates the base map with type and nullable handling.
func makeBase(schema queryfy.Schema, typeName string) map[string]interface{} {
	out := map[string]interface{}{}

	if isNullable(schema) {
		out["type"] = []interface{}{typeName, "null"}
	} else {
		out["type"] = typeName
	}

	return out
}

// includeMeta copies stored metadata into the output map.
// It skips keys that are already set (to avoid overwriting structural fields)
// and skips internal keys like exclusiveMinimum/exclusiveMaximum that are
// handled explicitly.
func includeMeta(schema queryfy.Schema, out map[string]interface{}, opts *ExportOptions) {
	if !opts.IncludeMeta {
		return
	}

	type metaProvider interface {
		AllMeta() map[string]interface{}
	}

	mp, ok := schema.(metaProvider)
	if !ok {
		return
	}

	skip := map[string]bool{
		"exclusiveMinimum": true,
		"exclusiveMaximum": true,
	}

	for key, value := range mp.AllMeta() {
		if skip[key] {
			continue
		}
		if _, exists := out[key]; exists {
			continue
		}
		out[key] = value
	}
}

// isRequired checks if a schema is required via BaseSchema.
func isRequired(schema queryfy.Schema) bool {
	type requiredChecker interface {
		IsRequired() bool
	}
	if rc, ok := schema.(requiredChecker); ok {
		return rc.IsRequired()
	}
	return false
}

// isNullable checks if a schema is nullable via BaseSchema.
func isNullable(schema queryfy.Schema) bool {
	type nullableChecker interface {
		IsNullable() bool
	}
	if nc, ok := schema.(nullableChecker); ok {
		return nc.IsNullable()
	}
	return false
}
