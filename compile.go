package queryfy

import (
	"fmt"
	"strings"
)

// checkFunc is a single validation check. It adds errors to ctx if the
// value is invalid. The value has already been type-asserted by the
// compiled schema's outer Validate method.
type checkFunc func(value interface{}, ctx *ValidationContext)

// CompiledSchema wraps a Schema with a pre-built flat slice of check
// functions. All constraint checks (length, range, pattern, enum, custom
// validators, etc.) are resolved once at compile time and stored as
// function pointers. At validation time, the compiled schema executes
// one loop with no nil-checks or conditional branching on which
// constraints are configured.
type CompiledSchema struct {
	BaseSchema
	inner      Schema
	checks     []checkFunc
	schemaType SchemaType
}

// compileInfo is extracted from concrete schema types via interface
// assertions so the compiler doesn't import the builders package.
type stringIntrospection interface {
	LengthConstraints() (min, max *int)
	PatternString() string
	EnumValues() []string
	FormatType() string
}

type numberIntrospection interface {
	RangeConstraints() (min, max *float64)
	IsInteger() bool
	MultipleOfValue() *float64
}

type objectIntrospection interface {
	FieldNames() []string
	GetField(string) (Schema, bool)
	RequiredFieldNames() []string
	AllowsAdditional() (bool, bool)
}

type arrayIntrospection interface {
	ElementSchema() Schema
	ItemCountConstraints() (min, max *int)
	IsUniqueItems() bool
}

type patternMatcher interface {
	PatternMatch(string) bool
}

type validatorProvider interface {
	Validators() []ValidatorFunc
}

// Compile pre-compiles a schema into a flat function-call chain.
// The returned CompiledSchema validates with a single loop over
// pre-resolved check functions, eliminating per-call nil checks
// and constraint branching.
//
// If the schema is already compiled, it is returned as-is.
func Compile(schema Schema) Schema {
	if _, already := schema.(*CompiledSchema); already {
		return schema
	}

	cs := &CompiledSchema{
		inner:      schema,
		schemaType: schema.Type(),
	}
	cs.BaseSchema = extractBase(schema)

	switch schema.Type() {
	case TypeString:
		cs.compileString(schema)
	case TypeNumber:
		cs.compileNumber(schema)
	case TypeBool:
		cs.compileBool(schema)
	case TypeObject:
		cs.compileObject(schema)
	case TypeArray:
		cs.compileArray(schema)
	default:
		// For types we can't optimise (composite, dependent, datetime,
		// transform, custom), delegate to the original schema's Validate.
		cs.checks = append(cs.checks, func(value interface{}, ctx *ValidationContext) {
			schema.Validate(value, ctx)
		})
	}

	return cs
}

// Validate runs the pre-compiled check chain.
func (cs *CompiledSchema) Validate(value interface{}, ctx *ValidationContext) error {
	if !cs.CheckRequired(value, ctx) {
		return nil
	}

	for _, check := range cs.checks {
		check(value, ctx)
	}
	return nil
}

// Type returns the underlying schema type.
func (cs *CompiledSchema) Type() SchemaType {
	return cs.schemaType
}

// Inner returns the original uncompiled schema.
func (cs *CompiledSchema) Inner() Schema {
	return cs.inner
}

// ---- string compilation ----

func (cs *CompiledSchema) compileString(schema Schema) {
	// Type check (mode-aware)
	cs.checks = append(cs.checks, func(value interface{}, ctx *ValidationContext) {
		if ctx.Mode() == Loose {
			if _, ok := ConvertToString(value); !ok {
				ctx.AddError(fmt.Sprintf("cannot convert %T to string", value), value)
			}
		} else {
			if _, ok := value.(string); !ok {
				ctx.AddError(fmt.Sprintf("expected string, got %T", value), value)
			}
		}
	})

	si, ok := schema.(stringIntrospection)
	if !ok {
		return
	}

	minLen, maxLen := si.LengthConstraints()
	if minLen != nil {
		min := *minLen
		cs.checks = append(cs.checks, func(value interface{}, ctx *ValidationContext) {
			str := toString(value, ctx)
			if str != "" || value != nil {
				if len(str) < min {
					ctx.AddError(fmt.Sprintf("length must be at least %d, got %d", min, len(str)), str)
				}
			}
		})
	}
	if maxLen != nil {
		max := *maxLen
		cs.checks = append(cs.checks, func(value interface{}, ctx *ValidationContext) {
			str := toString(value, ctx)
			if len(str) > max {
				ctx.AddError(fmt.Sprintf("length must be at most %d, got %d", max, len(str)), str)
			}
		})
	}

	patStr := si.PatternString()
	if patStr != "" {
		formatType := si.FormatType()
		// We use the original schema's pattern matching to avoid re-compiling
		if pm, ok := schema.(patternMatcher); ok {
			cs.checks = append(cs.checks, func(value interface{}, ctx *ValidationContext) {
				str := toString(value, ctx)
				if !pm.PatternMatch(str) {
					var msg string
					switch formatType {
					case "email":
						msg = "must be a valid email address"
					case "url":
						msg = "must be a valid URL"
					case "uuid":
						msg = "must be a valid UUID"
					default:
						msg = fmt.Sprintf("must match pattern %s", patStr)
					}
					ctx.AddError(msg, str)
				}
			})
		}
	}

	if vals := si.EnumValues(); len(vals) > 0 {
		// Pre-build set for O(1) lookup
		set := make(map[string]bool, len(vals))
		for _, v := range vals {
			set[v] = true
		}
		enumStr := strings.Join(vals, ", ")
		cs.checks = append(cs.checks, func(value interface{}, ctx *ValidationContext) {
			str := toString(value, ctx)
			if !set[str] {
				ctx.AddError(fmt.Sprintf("must be one of: %s", enumStr), str)
			}
		})
	}

	// Custom validators from the original schema
	if vp, ok := schema.(validatorProvider); ok {
		for _, v := range vp.Validators() {
			validator := v // capture
			cs.checks = append(cs.checks, func(value interface{}, ctx *ValidationContext) {
				str := toString(value, ctx)
				if err := validator(str); err != nil {
					ctx.AddError(err.Error(), str)
				}
			})
		}
	}
}

// ---- number compilation ----

func (cs *CompiledSchema) compileNumber(schema Schema) {
	// Type check + coercion
	cs.checks = append(cs.checks, func(value interface{}, ctx *ValidationContext) {
		if !ValidateValue(value, TypeNumber, ctx) {
			return
		}
	})

	ni, ok := schema.(numberIntrospection)
	if !ok {
		return
	}

	if ni.IsInteger() {
		cs.checks = append(cs.checks, func(value interface{}, ctx *ValidationContext) {
			f := toFloat(value)
			if f != float64(int64(f)) {
				ctx.AddError("must be an integer", value)
			}
		})
	}

	min, max := ni.RangeConstraints()
	if min != nil {
		minVal := *min
		cs.checks = append(cs.checks, func(value interface{}, ctx *ValidationContext) {
			if toFloat(value) < minVal {
				ctx.AddError(fmt.Sprintf("must be at least %v", minVal), value)
			}
		})
	}
	if max != nil {
		maxVal := *max
		cs.checks = append(cs.checks, func(value interface{}, ctx *ValidationContext) {
			if toFloat(value) > maxVal {
				ctx.AddError(fmt.Sprintf("must be at most %v", maxVal), value)
			}
		})
	}

	if mul := ni.MultipleOfValue(); mul != nil {
		mulVal := *mul
		cs.checks = append(cs.checks, func(value interface{}, ctx *ValidationContext) {
			f := toFloat(value)
			rem := f / mulVal
			if rem != float64(int64(rem)) {
				ctx.AddError(fmt.Sprintf("must be a multiple of %v", mulVal), value)
			}
		})
	}

	if vp, ok := schema.(validatorProvider); ok {
		for _, v := range vp.Validators() {
			validator := v
			cs.checks = append(cs.checks, func(value interface{}, ctx *ValidationContext) {
				if err := validator(value); err != nil {
					ctx.AddError(err.Error(), value)
				}
			})
		}
	}
}

// ---- bool compilation ----

func (cs *CompiledSchema) compileBool(schema Schema) {
	cs.checks = append(cs.checks, func(value interface{}, ctx *ValidationContext) {
		if !ValidateValue(value, TypeBool, ctx) {
			return
		}
	})

	if vp, ok := schema.(validatorProvider); ok {
		for _, v := range vp.Validators() {
			validator := v
			cs.checks = append(cs.checks, func(value interface{}, ctx *ValidationContext) {
				if err := validator(value); err != nil {
					ctx.AddError(err.Error(), value)
				}
			})
		}
	}
}

// ---- object compilation ----

func (cs *CompiledSchema) compileObject(schema Schema) {
	// Type check
	cs.checks = append(cs.checks, func(value interface{}, ctx *ValidationContext) {
		if !ValidateValue(value, TypeObject, ctx) {
			return
		}
	})

	oi, ok := schema.(objectIntrospection)
	if !ok {
		// Can't introspect — fall back to original
		cs.checks = append(cs.checks, func(value interface{}, ctx *ValidationContext) {
			schema.Validate(value, ctx)
		})
		return
	}

	// Pre-compute: required field set, compiled field schemas
	fieldNames := oi.FieldNames()
	type compiledField struct {
		name     string
		schema   Schema
		required bool
	}

	reqSet := make(map[string]bool)
	for _, name := range oi.RequiredFieldNames() {
		reqSet[name] = true
	}

	fields := make([]compiledField, 0, len(fieldNames))
	fieldSet := make(map[string]bool, len(fieldNames))
	for _, name := range fieldNames {
		fieldSchema, _ := oi.GetField(name)
		required := reqSet[name] || isSchemaRequired(fieldSchema)
		fields = append(fields, compiledField{
			name:     name,
			schema:   Compile(fieldSchema), // recursively compile
			required: required,
		})
		fieldSet[name] = true
	}

	allow, explicit := oi.AllowsAdditional()

	// Single check for the whole object — this replaces the entire
	// ObjectSchema.Validate method with pre-resolved field iteration
	cs.checks = append(cs.checks, func(value interface{}, ctx *ValidationContext) {
		objMap, ok := value.(map[string]interface{})
		if !ok {
			ctx.AddError(fmt.Sprintf("cannot convert %T to map", value), value)
			return
		}

		// Validate each pre-compiled field
		for i := range fields {
			f := &fields[i]
			fieldValue, exists := objMap[f.name]

			ctx.WithPath(f.name, func() {
				if exists {
					f.schema.Validate(fieldValue, ctx)
				} else if f.required {
					ctx.AddError("field is required", nil)
				}
			})
		}

		// Extra field check
		rejectExtra := false
		if explicit {
			rejectExtra = !allow
		} else {
			rejectExtra = ctx.Mode() == Strict
		}

		if rejectExtra {
			for key := range objMap {
				if !fieldSet[key] {
					ctx.WithPath(key, func() {
						ctx.AddError("unexpected field", objMap[key])
					})
				}
			}
		}
	})

	// Object-level custom validators
	if vp, ok := schema.(validatorProvider); ok {
		for _, v := range vp.Validators() {
			validator := v
			cs.checks = append(cs.checks, func(value interface{}, ctx *ValidationContext) {
				if err := validator(value); err != nil {
					ctx.AddError(err.Error(), value)
				}
			})
		}
	}
}

// ---- array compilation ----

func (cs *CompiledSchema) compileArray(schema Schema) {
	cs.checks = append(cs.checks, func(value interface{}, ctx *ValidationContext) {
		if !ValidateValue(value, TypeArray, ctx) {
			return
		}
	})

	ai, ok := schema.(arrayIntrospection)
	if !ok {
		cs.checks = append(cs.checks, func(value interface{}, ctx *ValidationContext) {
			schema.Validate(value, ctx)
		})
		return
	}

	minItems, maxItems := ai.ItemCountConstraints()
	elemSchema := ai.ElementSchema()
	unique := ai.IsUniqueItems()

	// Compile element schema if present
	var compiledElem Schema
	if elemSchema != nil {
		compiledElem = Compile(elemSchema)
	}

	// Single check for the whole array
	cs.checks = append(cs.checks, func(value interface{}, ctx *ValidationContext) {
		arr, ok := value.([]interface{})
		if !ok {
			return // type check already reported error
		}

		length := len(arr)
		if minItems != nil && length < *minItems {
			ctx.AddError(fmt.Sprintf("must have at least %d items, got %d", *minItems, length), value)
		}
		if maxItems != nil && length > *maxItems {
			ctx.AddError(fmt.Sprintf("must have at most %d items, got %d", *maxItems, length), value)
		}

		if unique && length > 1 {
			seen := make(map[interface{}]bool, length)
			for _, item := range arr {
				if seen[item] {
					ctx.AddError("items must be unique", value)
					break
				}
				seen[item] = true
			}
		}

		if compiledElem != nil {
			for i, item := range arr {
				ctx.WithPath(fmt.Sprintf("[%d]", i), func() {
					compiledElem.Validate(item, ctx)
				})
			}
		}
	})

	if vp, ok := schema.(validatorProvider); ok {
		for _, v := range vp.Validators() {
			validator := v
			cs.checks = append(cs.checks, func(value interface{}, ctx *ValidationContext) {
				if err := validator(value); err != nil {
					ctx.AddError(err.Error(), value)
				}
			})
		}
	}
}

// ---- helpers ----

// extractBase copies BaseSchema fields from the original.
func extractBase(schema Schema) BaseSchema {
	type baseProvider interface {
		IsRequired() bool
		IsNullable() bool
	}
	var bs BaseSchema
	if bp, ok := schema.(baseProvider); ok {
		bs.SetRequired(bp.IsRequired())
		bs.SetNullable(bp.IsNullable())
	}
	return bs
}

func isSchemaRequired(schema Schema) bool {
	type requiredChecker interface {
		IsRequired() bool
	}
	if rc, ok := schema.(requiredChecker); ok {
		return rc.IsRequired()
	}
	return false
}

// toString extracts a string from a value (handling loose mode conversion).
func toString(value interface{}, ctx *ValidationContext) string {
	if s, ok := value.(string); ok {
		return s
	}
	if ctx.Mode() == Loose {
		if s, ok := ConvertToString(value); ok {
			return s
		}
	}
	return ""
}

// toFloat extracts a float64 from a numeric value.
func toFloat(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case float32:
		return float64(v)
	default:
		return 0
	}
}
