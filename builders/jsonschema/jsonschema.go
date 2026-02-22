// Package jsonschema converts JSON Schema documents into queryfy schemas.
//
// This package supports a practical subset of JSON Schema (Draft 2020-12 /
// Draft 7 compatible) focused on the features that map cleanly to queryfy's
// validation model. Unsupported features produce clear errors rather than
// silent data loss.
//
// Usage:
//
//	schema, errs := jsonschema.FromJSON(data, nil)
//	if len(errs) > 0 {
//	    // handle conversion errors
//	}
//	// schema is a queryfy.Schema ready for validation
package jsonschema

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"

	"github.com/ha1tch/queryfy"
	"github.com/ha1tch/queryfy/builders"
)

// Options controls the conversion behaviour.
type Options struct {
	// StrictMode causes unsupported JSON Schema features to produce errors.
	// When false, unsupported features are skipped with a warning in the
	// returned error slice.
	StrictMode bool

	// StoreUnknown causes unrecognised keywords to be stored as schema
	// metadata via Meta(). This is useful for round-tripping or for
	// downstream consumers that need access to custom extensions.
	StoreUnknown bool
}

// ConversionError describes a single problem encountered during conversion.
type ConversionError struct {
	// Path is the dot-separated location in the JSON Schema document
	// (e.g., "properties.address.properties.city").
	Path string

	// Keyword is the JSON Schema keyword that caused the error
	// (e.g., "$ref", "oneOf").
	Keyword string

	// Message describes the problem.
	Message string

	// IsWarning is true when the error is non-fatal (StrictMode=false
	// and the feature was skipped).
	IsWarning bool
}

func (e *ConversionError) Error() string {
	prefix := "error"
	if e.IsWarning {
		prefix = "warning"
	}
	if e.Path == "" {
		return fmt.Sprintf("%s: %s: %s", prefix, e.Keyword, e.Message)
	}
	return fmt.Sprintf("%s at %s: %s: %s", prefix, e.Path, e.Keyword, e.Message)
}

// unsupported keywords that we explicitly reject
var unsupportedKeywords = map[string]string{
	"$ref":                    "schema references are not supported",
	"$defs":                   "schema definitions are not supported",
	"definitions":             "schema definitions are not supported",
	"oneOf":                   "composite schemas are not supported; use queryfy builders directly",
	"anyOf":                   "composite schemas are not supported; use queryfy builders directly",
	"allOf":                   "composite schemas are not supported; use queryfy builders directly",
	"not":                     "composite schemas are not supported; use queryfy builders directly",
	"if":                      "conditional schemas are not supported",
	"then":                    "conditional schemas are not supported",
	"else":                    "conditional schemas are not supported",
	"dependentRequired":       "dependent schemas are not supported; use queryfy builders directly",
	"dependentSchemas":        "dependent schemas are not supported; use queryfy builders directly",
	"patternProperties":       "pattern properties are not supported",
	"unevaluatedProperties":   "unevaluated properties are not supported",
	"unevaluatedItems":        "unevaluated items are not supported",
	"prefixItems":             "tuple validation is not supported",
	"contains":                "contains is not supported",
	"additionalItems":         "additionalItems is not supported; use items",
}

// recognised keywords that we handle (not flagged as unknown)
var recognisedKeywords = map[string]bool{
	"type": true, "properties": true, "required": true,
	"additionalProperties": true, "items": true,
	"minLength": true, "maxLength": true, "pattern": true,
	"minimum": true, "maximum": true, "exclusiveMinimum": true,
	"exclusiveMaximum": true, "multipleOf": true,
	"enum": true, "format": true, "nullable": true,
	"minItems": true, "maxItems": true, "uniqueItems": true,
	"title": true, "description": true, "default": true,
	"examples": true, "const": true,
	"$schema": true, "$id": true, "$comment": true,
}

// FromJSON parses a JSON Schema document and returns a queryfy schema.
// The second return value is a slice of conversion errors/warnings.
// A nil opts uses default settings (non-strict, no unknown storage).
func FromJSON(data []byte, opts *Options) (queryfy.Schema, []ConversionError) {
	if opts == nil {
		opts = &Options{}
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, []ConversionError{{
			Keyword: "json",
			Message: fmt.Sprintf("invalid JSON: %s", err.Error()),
		}}
	}

	c := &converter{opts: opts}
	schema := c.convertNode(raw, "")
	return schema, c.errors
}

// converter holds state during a single conversion.
type converter struct {
	opts   *Options
	errors []ConversionError
}

func (c *converter) addError(path, keyword, message string) {
	c.errors = append(c.errors, ConversionError{
		Path:    path,
		Keyword: keyword,
		Message: message,
	})
}

func (c *converter) addWarning(path, keyword, message string) {
	c.errors = append(c.errors, ConversionError{
		Path:      path,
		Keyword:   keyword,
		Message:   message,
		IsWarning: true,
	})
}

// checkUnsupported scans for unsupported keywords and emits errors/warnings.
// Returns true if any fatal errors were added (StrictMode).
func (c *converter) checkUnsupported(raw map[string]interface{}, path string) bool {
	fatal := false
	for key, reason := range unsupportedKeywords {
		if _, exists := raw[key]; exists {
			if c.opts.StrictMode {
				c.addError(path, key, reason)
				fatal = true
			} else {
				c.addWarning(path, key, reason+" (skipped)")
			}
		}
	}
	return fatal
}

// convertNode converts a single JSON Schema node to a queryfy schema.
func (c *converter) convertNode(raw map[string]interface{}, path string) queryfy.Schema {
	if c.checkUnsupported(raw, path) && c.opts.StrictMode {
		// In strict mode with unsupported features, we still try to convert
		// the supported parts so the caller gets maximum information.
	}

	typeName := c.resolveType(raw, path)

	var schema queryfy.Schema
	switch typeName {
	case "string":
		schema = c.convertString(raw, path)
	case "number":
		schema = c.convertNumber(raw, path, false)
	case "integer":
		schema = c.convertNumber(raw, path, true)
	case "boolean":
		schema = c.convertBool(raw, path)
	case "object":
		schema = c.convertObject(raw, path)
	case "array":
		schema = c.convertArray(raw, path)
	default:
		c.addError(path, "type", fmt.Sprintf("unsupported or missing type: %q", typeName))
		// Return a minimal string schema as fallback
		schema = builders.String()
	}

	return schema
}

// resolveType determines the JSON Schema type, handling nullable arrays.
func (c *converter) resolveType(raw map[string]interface{}, path string) string {
	t, ok := raw["type"]
	if !ok {
		// No type specified — try to infer from properties/items
		if _, hasProps := raw["properties"]; hasProps {
			return "object"
		}
		if _, hasItems := raw["items"]; hasItems {
			return "array"
		}
		return ""
	}

	switch v := t.(type) {
	case string:
		return v
	case []interface{}:
		// type: ["string", "null"] → nullable string
		var nonNull string
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				continue
			}
			if s == "null" {
				raw["_nullable"] = true
				continue
			}
			nonNull = s
		}
		return nonNull
	default:
		c.addError(path, "type", fmt.Sprintf("unexpected type value: %T", t))
		return ""
	}
}

// convertString handles string schemas.
func (c *converter) convertString(raw map[string]interface{}, path string) queryfy.Schema {
	s := builders.String()

	if minLen, ok := getFloat(raw, "minLength"); ok {
		s.MinLength(int(minLen))
	}
	if maxLen, ok := getFloat(raw, "maxLength"); ok {
		s.MaxLength(int(maxLen))
	}
	if pattern, ok := getString(raw, "pattern"); ok {
		if _, err := regexp.Compile(pattern); err != nil {
			c.addError(path, "pattern", fmt.Sprintf("invalid regex: %s", err.Error()))
		} else {
			s.Pattern(pattern)
		}
	}
	if enumVals, ok := getStringSlice(raw, "enum"); ok {
		s.Enum(enumVals...)
	}
	if format, ok := getString(raw, "format"); ok {
		c.applyStringFormat(s, format, path)
	}

	c.applyNullable(raw, s)
	c.storeUnknownOnSchema(raw, path, s)
	return s
}

// applyStringFormat maps JSON Schema format to queryfy format methods.
func (c *converter) applyStringFormat(s *builders.StringSchema, format, path string) {
	switch format {
	case "email":
		s.Email()
	case "uri", "url":
		s.URL()
	case "uuid":
		s.UUID()
	case "date-time", "date", "time":
		// date-time/date/time are valid but don't map to StringSchema
		// Store as metadata for downstream consumers
		s.Meta("format", format)
	default:
		// Check the format registry
		if builders.LookupFormat(format) != nil {
			s.FormatString(format)
		} else {
			// Unknown format — store as metadata
			s.Meta("format", format)
			c.addWarning(path, "format", fmt.Sprintf("unknown format %q stored as metadata", format))
		}
	}
}

// convertNumber handles number and integer schemas.
func (c *converter) convertNumber(raw map[string]interface{}, path string, integer bool) queryfy.Schema {
	s := builders.Number()

	if integer {
		s.Integer()
	}

	if min, ok := getFloat(raw, "minimum"); ok {
		s.Min(min)
	}
	if max, ok := getFloat(raw, "maximum"); ok {
		s.Max(max)
	}
	if exMin, ok := getFloat(raw, "exclusiveMinimum"); ok {
		// JSON Schema exclusiveMinimum means > value; approximate with min+epsilon
		// Store exact value as metadata for precise round-tripping
		s.Min(exMin)
		s.Meta("exclusiveMinimum", exMin)
	}
	if exMax, ok := getFloat(raw, "exclusiveMaximum"); ok {
		s.Max(exMax)
		s.Meta("exclusiveMaximum", exMax)
	}
	if mul, ok := getFloat(raw, "multipleOf"); ok {
		s.MultipleOf(mul)
	}

	c.applyNullable(raw, s)
	c.storeUnknownOnSchema(raw, path, s)
	return s
}

// convertBool handles boolean schemas.
func (c *converter) convertBool(raw map[string]interface{}, path string) queryfy.Schema {
	s := builders.Bool()
	c.applyNullable(raw, s)
	c.storeUnknownOnSchema(raw, path, s)
	return s
}

// convertObject handles object schemas.
func (c *converter) convertObject(raw map[string]interface{}, path string) queryfy.Schema {
	s := builders.Object()

	// required fields set
	requiredSet := make(map[string]bool)
	if reqArr, ok := raw["required"]; ok {
		if arr, ok := reqArr.([]interface{}); ok {
			for _, item := range arr {
				if name, ok := item.(string); ok {
					requiredSet[name] = true
				}
			}
		}
	}

	// properties
	if props, ok := raw["properties"]; ok {
		propsMap, ok := props.(map[string]interface{})
		if !ok {
			c.addError(path, "properties", "expected object")
		} else {
			// Sort keys for deterministic output
			keys := make([]string, 0, len(propsMap))
			for k := range propsMap {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, fieldName := range keys {
				fieldDef := propsMap[fieldName]
				fieldPath := appendPath(path, "properties."+fieldName)

				fieldRaw, ok := fieldDef.(map[string]interface{})
				if !ok {
					c.addError(fieldPath, "properties", "expected object for field definition")
					continue
				}

				fieldSchema := c.convertNode(fieldRaw, fieldPath)
				if requiredSet[fieldName] {
					fieldSchema = markRequired(fieldSchema)
				}
				s.Field(fieldName, fieldSchema)
			}
		}
	}

	// additionalProperties
	if ap, ok := raw["additionalProperties"]; ok {
		switch v := ap.(type) {
		case bool:
			s.AllowAdditional(v)
		case map[string]interface{}:
			// additionalProperties as a schema — we can't validate against
			// it, but we allow additional properties and store the schema
			s.AllowAdditional(true)
			s.Meta("additionalPropertiesSchema", v)
			c.addWarning(path, "additionalProperties",
				"schema-valued additionalProperties stored as metadata; validation not enforced")
		}
	}

	c.applyNullable(raw, s)
	c.storeUnknownOnSchema(raw, path, s)
	return s
}

// convertArray handles array schemas.
func (c *converter) convertArray(raw map[string]interface{}, path string) queryfy.Schema {
	s := builders.Array()

	if minItems, ok := getFloat(raw, "minItems"); ok {
		s.MinItems(int(minItems))
	}
	if maxItems, ok := getFloat(raw, "maxItems"); ok {
		s.MaxItems(int(maxItems))
	}
	if unique, ok := getBool(raw, "uniqueItems"); ok && unique {
		s.UniqueItems()
	}

	// items — element schema
	if items, ok := raw["items"]; ok {
		itemsRaw, ok := items.(map[string]interface{})
		if !ok {
			c.addError(appendPath(path, "items"), "items", "expected object")
		} else {
			elemSchema := c.convertNode(itemsRaw, appendPath(path, "items"))
			s.Of(elemSchema)
		}
	}

	c.applyNullable(raw, s)
	c.storeUnknownOnSchema(raw, path, s)
	return s
}

// applyNullable sets Nullable() if the schema was marked nullable.
func (c *converter) applyNullable(raw map[string]interface{}, schema interface{}) {
	isNullable := false
	if _, ok := raw["_nullable"]; ok {
		isNullable = true
	}
	if nullable, ok := getBool(raw, "nullable"); ok && nullable {
		isNullable = true
	}
	if !isNullable {
		return
	}

	type nullableSetter interface {
		Nullable() interface{}
	}

	// Each schema type has its own Nullable() with a concrete return type,
	// so we need to type-switch.
	switch s := schema.(type) {
	case *builders.StringSchema:
		s.Nullable()
	case *builders.NumberSchema:
		s.Nullable()
	case *builders.BoolSchema:
		s.Nullable()
	case *builders.ObjectSchema:
		s.Nullable()
	case *builders.ArraySchema:
		s.Nullable()
	}
}

// storeUnknownOnSchema stores unrecognised keywords as metadata.
func (c *converter) storeUnknownOnSchema(raw map[string]interface{}, path string, schema interface{}) {
	if !c.opts.StoreUnknown {
		return
	}

	type metaSetter interface {
		Meta(string, interface{})
	}

	// Store title/description as metadata always (they're recognised but informational)
	setMeta := func(key string, value interface{}) {
		switch s := schema.(type) {
		case *builders.StringSchema:
			s.Meta(key, value)
		case *builders.NumberSchema:
			s.Meta(key, value)
		case *builders.BoolSchema:
			s.Meta(key, value)
		case *builders.ObjectSchema:
			s.Meta(key, value)
		case *builders.ArraySchema:
			s.Meta(key, value)
		}
	}

	for key, value := range raw {
		if recognisedKeywords[key] {
			continue
		}
		if _, isUnsupported := unsupportedKeywords[key]; isUnsupported {
			continue
		}
		if key == "_nullable" {
			continue // internal marker
		}
		setMeta(key, value)
	}
}

// markRequired marks a schema as required.
func markRequired(schema queryfy.Schema) queryfy.Schema {
	switch s := schema.(type) {
	case *builders.StringSchema:
		return s.Required()
	case *builders.NumberSchema:
		return s.Required()
	case *builders.BoolSchema:
		return s.Required()
	case *builders.ObjectSchema:
		return s.Required()
	case *builders.ArraySchema:
		return s.Required()
	case *builders.DateTimeSchema:
		return s.Required()
	case *builders.CustomSchema:
		return s.Required()
	default:
		return schema
	}
}

// ---- helpers ----

func getString(raw map[string]interface{}, key string) (string, bool) {
	v, ok := raw[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

func getFloat(raw map[string]interface{}, key string) (float64, bool) {
	v, ok := raw[key]
	if !ok {
		return 0, false
	}
	f, ok := v.(float64)
	return f, ok
}

func getBool(raw map[string]interface{}, key string) (bool, bool) {
	v, ok := raw[key]
	if !ok {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}

func getStringSlice(raw map[string]interface{}, key string) ([]string, bool) {
	v, ok := raw[key]
	if !ok {
		return nil, false
	}
	arr, ok := v.([]interface{})
	if !ok {
		return nil, false
	}
	result := make([]string, 0, len(arr))
	for _, item := range arr {
		s, ok := item.(string)
		if !ok {
			return nil, false
		}
		result = append(result, s)
	}
	return result, true
}

func appendPath(base, suffix string) string {
	if base == "" {
		return suffix
	}
	return base + "." + suffix
}
