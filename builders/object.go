package builders

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/ha1tch/queryfy"
)

// ObjectSchema validates object/map values.
type ObjectSchema struct {
	queryfy.BaseSchema
	fields            map[string]queryfy.Schema
	requiredFields    map[string]bool
	allowAdditional   *bool // nil = use mode default, true/false = explicit override
	validators        []queryfy.ValidatorFunc
	asyncValidators   []queryfy.AsyncValidatorFunc
}

// Object creates a new object schema builder.
func Object() *ObjectSchema {
	return &ObjectSchema{
		BaseSchema: queryfy.BaseSchema{
			SchemaType: queryfy.TypeObject,
		},
		fields:         make(map[string]queryfy.Schema),
		requiredFields: make(map[string]bool),
	}
}

// Required marks the object itself as required.
func (s *ObjectSchema) Required() *ObjectSchema {
	s.SetRequired(true)
	return s
}

// Optional marks the object as optional (default).
func (s *ObjectSchema) Optional() *ObjectSchema {
	s.SetRequired(false)
	return s
}

// Nullable allows the field to be null.
func (s *ObjectSchema) Nullable() *ObjectSchema {
	s.SetNullable(true)
	return s
}

// Field adds a field schema to the object.
//
// Note: Do not pass a *DependentSchema directly to Field — it will be
// treated as a regular schema and the dependency condition will be ignored.
// Use ObjectSchemaWithDependencies (via WithDependencies()) and its
// DependentField method instead.
func (s *ObjectSchema) Field(name string, schema queryfy.Schema) *ObjectSchema {
	// Guard against accidentally passing DependentSchema as a regular field.
	if _, ok := schema.(*DependentSchema); ok {
		panic(fmt.Sprintf(
			"queryfy: DependentSchema passed to Field(%q) — use "+
				"Object().WithDependencies().DependentField(%q, ...) instead",
			name, name,
		))
	}
	s.fields[name] = schema
	// Check if the field's schema marks it as required
	if requirer, ok := schema.(interface{ IsRequired() bool }); ok && requirer.IsRequired() {
		s.requiredFields[name] = true
	}
	return s
}

// Fields adds multiple field schemas at once.
func (s *ObjectSchema) Fields(fields map[string]queryfy.Schema) *ObjectSchema {
	for name, schema := range fields {
		s.Field(name, schema)
	}
	return s
}

// RequiredFields marks specific fields as required.
// This overrides the required status set on individual field schemas.
func (s *ObjectSchema) RequiredFields(names ...string) *ObjectSchema {
	for _, name := range names {
		s.requiredFields[name] = true
		// Also update the field schema if possible
		if schema, ok := s.fields[name]; ok {
			if setter, ok := schema.(interface{ SetRequired(bool) }); ok {
				setter.SetRequired(true)
			}
		}
	}
	return s
}

// Custom adds a custom validator function.
func (s *ObjectSchema) Custom(fn queryfy.ValidatorFunc) *ObjectSchema {
	s.validators = append(s.validators, fn)
	return s
}

// Validate implements the Schema interface.
func (s *ObjectSchema) Validate(value interface{}, ctx *queryfy.ValidationContext) error {
	if !s.CheckRequired(value, ctx) {
		return nil
	}

	// Type validation
	if !queryfy.ValidateValue(value, queryfy.TypeObject, ctx) {
		return nil
	}

	// Convert to map for validation
	objMap, ok := convertToMap(value)
	if !ok {
		ctx.AddError(fmt.Sprintf("cannot convert %T to map", value), value)
		return nil
	}

	// Check required fields first
	for fieldName, required := range s.requiredFields {
		if required {
			if _, exists := objMap[fieldName]; !exists {
				ctx.WithPath(fieldName, func() {
					ctx.AddError("field is required", nil)
				})
			}
		}
	}

	// Validate each defined field
	for fieldName, fieldSchema := range s.fields {
		fieldValue, exists := objMap[fieldName]

		ctx.WithPath(fieldName, func() {
			if exists {
				// Validate the field value
				fieldSchema.Validate(fieldValue, ctx)
			} else if s.requiredFields[fieldName] {
				// Already handled above, skip
			} else if isRequired(fieldSchema) {
				// Field schema itself says it's required
				ctx.AddError("field is required", nil)
			}
		})
	}

	// Check for extra fields based on AllowAdditional policy and mode
	if s.rejectsExtra(ctx) {
		for key := range objMap {
			if _, defined := s.fields[key]; !defined {
				ctx.WithPath(key, func() {
					ctx.AddError("unexpected field", objMap[key])
				})
			}
		}
	}

	// Custom validators
	for _, validator := range s.validators {
		if err := validator(objMap); err != nil {
			ctx.AddError(err.Error(), objMap)
		}
	}

	return nil
}

// GetField returns the schema for a named field and whether it exists.
func (s *ObjectSchema) GetField(name string) (queryfy.Schema, bool) {
	schema, ok := s.fields[name]
	return schema, ok
}

// RequiredFieldNames returns the names of all required fields, sorted.
func (s *ObjectSchema) RequiredFieldNames() []string {
	var names []string
	for name, req := range s.requiredFields {
		if req {
			names = append(names, name)
		}
	}
	// Also check field schemas that declare themselves required
	for name, schema := range s.fields {
		if s.requiredFields[name] {
			continue // already counted
		}
		if requirer, ok := schema.(interface{ IsRequired() bool }); ok && requirer.IsRequired() {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// AllowAdditional controls whether fields not declared in the schema are
// accepted, independent of the validation mode. When set to true, extra
// fields are always accepted; when false, always rejected. When not called
// (nil), the behaviour follows the validation mode (Strict rejects, Loose
// accepts).
func (s *ObjectSchema) AllowAdditional(allow bool) *ObjectSchema {
	s.allowAdditional = &allow
	return s
}

// AllowsAdditional reports the current additional-properties policy.
// Returns (value, explicit) where explicit is false if no policy has been
// set (nil — mode default applies).
func (s *ObjectSchema) AllowsAdditional() (allow bool, explicit bool) {
	if s.allowAdditional == nil {
		return false, false
	}
	return *s.allowAdditional, true
}

// rejectsExtra reports whether extra fields should be rejected given
// the schema's AllowAdditional setting and the current validation mode.
func (s *ObjectSchema) rejectsExtra(ctx *queryfy.ValidationContext) bool {
	if s.allowAdditional != nil {
		return !*s.allowAdditional
	}
	return ctx.Mode() == queryfy.Strict
}

// Meta attaches a key-value metadata pair to the schema.
func (s *ObjectSchema) Meta(key string, value interface{}) *ObjectSchema {
	s.SetMeta(key, value)
	return s
}

// Validators returns the custom validator functions.
func (s *ObjectSchema) Validators() []queryfy.ValidatorFunc {
	return s.validators
}

// Type implements the Schema interface.
func (s *ObjectSchema) Type() queryfy.SchemaType {
	return queryfy.TypeObject
}

// ValidateAndTransform validates the object and returns a new map with all
// transformations applied. Fields whose schemas implement TransformableSchema
// (e.g. TransformSchema) will have their transformed values in the result.
// Non-transformable fields are validated normally and their original values
// are preserved. Nested objects and arrays are handled recursively.
func (s *ObjectSchema) ValidateAndTransform(value interface{}, ctx *queryfy.ValidationContext) (interface{}, error) {
	if !s.CheckRequired(value, ctx) {
		return value, ctx.Error()
	}

	// Type validation
	if !queryfy.ValidateValue(value, queryfy.TypeObject, ctx) {
		return value, ctx.Error()
	}

	// Convert to map
	objMap, ok := convertToMap(value)
	if !ok {
		ctx.AddError(fmt.Sprintf("cannot convert %T to map", value), value)
		return value, ctx.Error()
	}

	// Build result map with transformed values
	result := make(map[string]interface{})

	// Copy all original values first (preserves extra fields when allowed)
	if !s.rejectsExtra(ctx) {
		for k, v := range objMap {
			result[k] = v
		}
	}

	// Check required fields
	for fieldName, required := range s.requiredFields {
		if required {
			if _, exists := objMap[fieldName]; !exists {
				ctx.WithPath(fieldName, func() {
					ctx.AddError("field is required", nil)
				})
			}
		}
	}

	// Validate and transform each defined field
	for fieldName, fieldSchema := range s.fields {
		fieldValue, exists := objMap[fieldName]

		if !exists {
			if s.requiredFields[fieldName] {
				// Already reported above
			} else if isRequired(fieldSchema) {
				ctx.WithPath(fieldName, func() {
					ctx.AddError("field is required", nil)
				})
			}
			continue
		}

		ctx.WithPath(fieldName, func() {
			switch ts := fieldSchema.(type) {
			case interface {
				ValidateAndTransform(interface{}, *queryfy.ValidationContext) (interface{}, error)
			}:
				// Schema supports ValidateAndTransform — use it
				transformed, _ := ts.ValidateAndTransform(fieldValue, ctx)
				result[fieldName] = transformed
			default:
				// Plain schema — validate and keep original value
				fieldSchema.Validate(fieldValue, ctx)
				result[fieldName] = fieldValue
			}
		})
	}

	// Check for extra fields based on AllowAdditional policy and mode
	if s.rejectsExtra(ctx) {
		for key := range objMap {
			if _, defined := s.fields[key]; !defined {
				ctx.WithPath(key, func() {
					ctx.AddError("unexpected field", objMap[key])
				})
			}
		}
	}

	// Custom validators run against the result map
	for _, validator := range s.validators {
		if err := validator(result); err != nil {
			ctx.AddError(err.Error(), result)
		}
	}

	return result, ctx.Error()
}

// AsyncCustom adds an async validator to the object schema. Async validators
// are only invoked by ValidateAndTransformAsync; sync methods ignore them.
// Object-level async validators receive the full result map and are useful
// for cross-field checks that require I/O (e.g. uniqueness across fields).
func (s *ObjectSchema) AsyncCustom(fn queryfy.AsyncValidatorFunc) *ObjectSchema {
	s.asyncValidators = append(s.asyncValidators, fn)
	return s
}

// HasAsyncValidators returns true if any async validators are registered
// on this schema or on any of its field schemas.
func (s *ObjectSchema) HasAsyncValidators() bool {
	if len(s.asyncValidators) > 0 {
		return true
	}
	for _, fieldSchema := range s.fields {
		if checker, ok := fieldSchema.(interface{ HasAsyncValidators() bool }); ok {
			if checker.HasAsyncValidators() {
				return true
			}
		}
	}
	return false
}

// ValidateAndTransformAsync runs sync validation and transformations first.
// If sync validation passes, it then runs async validators on fields and
// on the object itself, sequentially with the provided context.
func (s *ObjectSchema) ValidateAndTransformAsync(goCtx context.Context, value interface{}, ctx *queryfy.ValidationContext) (interface{}, error) {
	if !s.CheckRequired(value, ctx) {
		return value, ctx.Error()
	}

	// Type validation
	if !queryfy.ValidateValue(value, queryfy.TypeObject, ctx) {
		return value, ctx.Error()
	}

	// Convert to map
	objMap, ok := convertToMap(value)
	if !ok {
		ctx.AddError(fmt.Sprintf("cannot convert %T to map", value), value)
		return value, ctx.Error()
	}

	// Build result map with transformed values (same as sync)
	result := make(map[string]interface{})

	if !s.rejectsExtra(ctx) {
		for k, v := range objMap {
			result[k] = v
		}
	}

	// Check required fields
	for fieldName, required := range s.requiredFields {
		if required {
			if _, exists := objMap[fieldName]; !exists {
				ctx.WithPath(fieldName, func() {
					ctx.AddError("field is required", nil)
				})
			}
		}
	}

	// Validate and transform each field (sync pass)
	for fieldName, fieldSchema := range s.fields {
		fieldValue, exists := objMap[fieldName]

		if !exists {
			if s.requiredFields[fieldName] {
				// Already reported above
			} else if isRequired(fieldSchema) {
				ctx.WithPath(fieldName, func() {
					ctx.AddError("field is required", nil)
				})
			}
			continue
		}

		ctx.WithPath(fieldName, func() {
			switch ts := fieldSchema.(type) {
			case interface {
				ValidateAndTransform(interface{}, *queryfy.ValidationContext) (interface{}, error)
			}:
				transformed, _ := ts.ValidateAndTransform(fieldValue, ctx)
				result[fieldName] = transformed
			default:
				fieldSchema.Validate(fieldValue, ctx)
				result[fieldName] = fieldValue
			}
		})
	}

	// Check for extra fields based on AllowAdditional policy and mode
	if s.rejectsExtra(ctx) {
		for key := range objMap {
			if _, defined := s.fields[key]; !defined {
				ctx.WithPath(key, func() {
					ctx.AddError("unexpected field", objMap[key])
				})
			}
		}
	}

	// Sync custom validators
	for _, validator := range s.validators {
		if err := validator(result); err != nil {
			ctx.AddError(err.Error(), result)
		}
	}

	// If sync validation failed, do not run async validators
	if ctx.HasErrors() {
		return result, ctx.Error()
	}

	// Check context cancellation before async phase
	if goCtx.Err() != nil {
		ctx.AddError(fmt.Sprintf("validation cancelled: %s", goCtx.Err()), result)
		return result, ctx.Error()
	}

	// Run async validators on individual fields (sequentially)
	for fieldName, fieldSchema := range s.fields {
		fieldValue, exists := result[fieldName]
		if !exists {
			continue
		}

		if goCtx.Err() != nil {
			ctx.AddError(fmt.Sprintf("validation cancelled: %s", goCtx.Err()), result)
			return result, ctx.Error()
		}

		ctx.WithPath(fieldName, func() {
			switch ts := fieldSchema.(type) {
			case interface {
				ValidateAndTransformAsync(context.Context, interface{}, *queryfy.ValidationContext) (interface{}, error)
			}:
				ts.ValidateAndTransformAsync(goCtx, fieldValue, ctx)
			}
		})
	}

	// Run object-level async validators
	for _, asyncValidator := range s.asyncValidators {
		if goCtx.Err() != nil {
			ctx.AddError(fmt.Sprintf("validation cancelled: %s", goCtx.Err()), result)
			return result, ctx.Error()
		}

		if err := asyncValidator(goCtx, result); err != nil {
			ctx.AddError(err.Error(), result)
		}
	}

	return result, ctx.Error()
}

// convertToMap attempts to convert a value to map[string]interface{}.
func convertToMap(value interface{}) (map[string]interface{}, bool) {
	// Direct type assertion
	if m, ok := value.(map[string]interface{}); ok {
		return m, true
	}

	// Use reflection for other map types
	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Map {
		return nil, false
	}

	// Check if keys are strings
	if rv.Type().Key().Kind() != reflect.String {
		return nil, false
	}

	// Convert to map[string]interface{}
	result := make(map[string]interface{})
	for _, key := range rv.MapKeys() {
		result[key.String()] = rv.MapIndex(key).Interface()
	}

	return result, true
}

// isRequired checks if a schema marks its field as required.
func isRequired(schema queryfy.Schema) bool {
	if requirer, ok := schema.(interface{ IsRequired() bool }); ok {
		return requirer.IsRequired()
	}
	return false
}

// FieldNames returns the names of all defined fields.
func (s *ObjectSchema) FieldNames() []string {
	names := make([]string, 0, len(s.fields))
	for name := range s.fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// String returns a string representation of the object schema.
func (s *ObjectSchema) String() string {
	var parts []string
	for _, name := range s.FieldNames() {
		parts = append(parts, fmt.Sprintf("%s: %v", name, s.fields[name].Type()))
	}
	return fmt.Sprintf("Object{%s}", strings.Join(parts, ", "))
}
