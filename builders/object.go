package builders

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/ha1tch/queryfy"
)

// ObjectSchema validates object/map values.
type ObjectSchema struct {
	queryfy.BaseSchema
	fields         map[string]queryfy.Schema
	requiredFields map[string]bool
	validators     []queryfy.ValidatorFunc
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
func (s *ObjectSchema) Field(name string, schema queryfy.Schema) *ObjectSchema {
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

	// In strict mode, check for extra fields
	if ctx.Mode() == queryfy.Strict {
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

// Type implements the Schema interface.
func (s *ObjectSchema) Type() queryfy.SchemaType {
	return queryfy.TypeObject
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
