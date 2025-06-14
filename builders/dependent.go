// dependent.go - Dependent fields validation for Queryfy
package builders

import (
	"fmt"
	"reflect"

	"github.com/ha1tch/queryfy"
)

// DependentSchema validates fields based on conditions from other fields.
type DependentSchema struct {
	queryfy.BaseSchema
	fieldName     string              // The field this schema validates
	dependsOn     []string            // Fields this validation depends on
	conditionFunc DependencyCondition // Function that determines if validation should run
	schema        queryfy.Schema      // The schema to apply when condition is met
	elseSchema    queryfy.Schema      // Optional schema to apply when condition is not met
	validators    []queryfy.ValidatorFunc
}

// DependencyCondition is a function that receives the parent object and returns whether
// the dependent validation should be applied.
type DependencyCondition func(parentData map[string]interface{}) bool

// Dependent creates a new dependent field schema.
func Dependent(fieldName string) *DependentSchema {
	return &DependentSchema{
		BaseSchema: queryfy.BaseSchema{
			SchemaType: queryfy.TypeDependent,
		},
		fieldName: fieldName,
		dependsOn: []string{},
	}
}

// On specifies which fields this validation depends on.
func (s *DependentSchema) On(fields ...string) *DependentSchema {
	s.dependsOn = append(s.dependsOn, fields...)
	return s
}

// When sets the condition function that determines when validation applies.
func (s *DependentSchema) When(condition DependencyCondition) *DependentSchema {
	s.conditionFunc = condition
	return s
}

// Then sets the schema to apply when the condition is true.
func (s *DependentSchema) Then(schema queryfy.Schema) *DependentSchema {
	s.schema = schema
	return s
}

// Else sets the schema to apply when the condition is false.
func (s *DependentSchema) Else(elseSchema queryfy.Schema) *DependentSchema {
	s.elseSchema = elseSchema
	return s
}

// Required marks the field as required (when the condition is met).
func (s *DependentSchema) Required() *DependentSchema {
	s.SetRequired(true)
	return s
}

// Optional marks the field as optional.
func (s *DependentSchema) Optional() *DependentSchema {
	s.SetRequired(false)
	return s
}

// Custom adds a custom validator function.
func (s *DependentSchema) Custom(fn queryfy.ValidatorFunc) *DependentSchema {
	s.validators = append(s.validators, fn)
	return s
}

// Validate implements the Schema interface.
func (s *DependentSchema) Validate(value interface{}, ctx *queryfy.ValidationContext) error {
	// For dependent validation, we need access to the parent object
	// This is typically called from within an object schema

	// The value here is the field value, but we need the parent object
	// This will be handled by the enhanced object schema

	if s.schema != nil {
		return s.schema.Validate(value, ctx)
	}

	return nil
}

// ValidateWithParent validates the field considering the parent object context.
func (s *DependentSchema) ValidateWithParent(value interface{}, parentData map[string]interface{}, ctx *queryfy.ValidationContext) error {
	// Check if condition is met
	conditionMet := false
	if s.conditionFunc != nil {
		conditionMet = s.conditionFunc(parentData)
	}

	// Apply appropriate schema based on condition
	if conditionMet && s.schema != nil {
		return s.schema.Validate(value, ctx)
	} else if !conditionMet && s.elseSchema != nil {
		return s.elseSchema.Validate(value, ctx)
	}

	// Run custom validators
	for _, validator := range s.validators {
		if err := validator(value); err != nil {
			ctx.AddError(err.Error(), value)
		}
	}

	return nil
}

// Type implements the Schema interface.
func (s *DependentSchema) Type() queryfy.SchemaType {
	return queryfy.TypeDependent
}

// Common dependency conditions

// WhenEquals creates a condition that checks if a field equals a specific value.
func WhenEquals(field string, value interface{}) DependencyCondition {
	return func(data map[string]interface{}) bool {
		if fieldValue, exists := data[field]; exists {
			return reflect.DeepEqual(fieldValue, value)
		}
		return false
	}
}

// WhenNotEquals creates a condition that checks if a field does not equal a specific value.
func WhenNotEquals(field string, value interface{}) DependencyCondition {
	return func(data map[string]interface{}) bool {
		if fieldValue, exists := data[field]; exists {
			return !reflect.DeepEqual(fieldValue, value)
		}
		return true
	}
}

// WhenExists creates a condition that checks if a field exists and is not nil.
func WhenExists(field string) DependencyCondition {
	return func(data map[string]interface{}) bool {
		value, exists := data[field]
		return exists && value != nil
	}
}

// WhenNotExists creates a condition that checks if a field does not exist or is nil.
func WhenNotExists(field string) DependencyCondition {
	return func(data map[string]interface{}) bool {
		value, exists := data[field]
		return !exists || value == nil
	}
}

// WhenIn creates a condition that checks if a field's value is in a list.
func WhenIn(field string, values ...interface{}) DependencyCondition {
	return func(data map[string]interface{}) bool {
		if fieldValue, exists := data[field]; exists {
			for _, v := range values {
				if reflect.DeepEqual(fieldValue, v) {
					return true
				}
			}
		}
		return false
	}
}

// WhenGreaterThan creates a condition for numeric comparisons.
func WhenGreaterThan(field string, threshold float64) DependencyCondition {
	return func(data map[string]interface{}) bool {
		if fieldValue, exists := data[field]; exists {
			if num, ok := dependentToFloat64(fieldValue); ok {
				return num > threshold
			}
		}
		return false
	}
}

// WhenLessThan creates a condition for numeric comparisons.
func WhenLessThan(field string, threshold float64) DependencyCondition {
	return func(data map[string]interface{}) bool {
		if fieldValue, exists := data[field]; exists {
			if num, ok := dependentToFloat64(fieldValue); ok {
				return num < threshold
			}
		}
		return false
	}
}

// WhenTrue creates a condition that checks if a boolean field is true.
func WhenTrue(field string) DependencyCondition {
	return func(data map[string]interface{}) bool {
		if fieldValue, exists := data[field]; exists {
			if boolVal, ok := fieldValue.(bool); ok {
				return boolVal
			}
		}
		return false
	}
}

// WhenFalse creates a condition that checks if a boolean field is false.
func WhenFalse(field string) DependencyCondition {
	return func(data map[string]interface{}) bool {
		if fieldValue, exists := data[field]; exists {
			if boolVal, ok := fieldValue.(bool); ok {
				return !boolVal
			}
		}
		return false
	}
}

// WhenAll creates a condition that requires all sub-conditions to be true.
func WhenAll(conditions ...DependencyCondition) DependencyCondition {
	return func(data map[string]interface{}) bool {
		for _, condition := range conditions {
			if !condition(data) {
				return false
			}
		}
		return true
	}
}

// WhenAny creates a condition that requires at least one sub-condition to be true.
func WhenAny(conditions ...DependencyCondition) DependencyCondition {
	return func(data map[string]interface{}) bool {
		for _, condition := range conditions {
			if condition(data) {
				return true
			}
		}
		return false
	}
}

// dependentToFloat64 converts to float64 for dependent field validation
func dependentToFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint64:
		return float64(v), true
	case uint32:
		return float64(v), true
	default:
		return 0, false
	}
}

// ObjectSchemaWithDependencies extends the regular ObjectSchema to support dependent fields.
type ObjectSchemaWithDependencies struct {
	*ObjectSchema
	dependentFields map[string]*DependentSchema
}

// WithDependencies converts an ObjectSchema to support dependent field validation.
func (s *ObjectSchema) WithDependencies() *ObjectSchemaWithDependencies {
	return &ObjectSchemaWithDependencies{
		ObjectSchema:    s,
		dependentFields: make(map[string]*DependentSchema),
	}
}

// Field adds a regular field to the schema (override to return correct type).
func (s *ObjectSchemaWithDependencies) Field(name string, schema queryfy.Schema) *ObjectSchemaWithDependencies {
	s.ObjectSchema.Field(name, schema)
	return s
}

// Fields adds multiple fields at once (override to return correct type).
func (s *ObjectSchemaWithDependencies) Fields(fields map[string]queryfy.Schema) *ObjectSchemaWithDependencies {
	s.ObjectSchema.Fields(fields)
	return s
}

// RequiredFields marks specific fields as required (override to return correct type).
func (s *ObjectSchemaWithDependencies) RequiredFields(names ...string) *ObjectSchemaWithDependencies {
	s.ObjectSchema.RequiredFields(names...)
	return s
}

// DependentField adds a dependent field to the schema.
func (s *ObjectSchemaWithDependencies) DependentField(name string, dependent *DependentSchema) *ObjectSchemaWithDependencies {
	s.dependentFields[name] = dependent
	// Also add it as a regular field so it appears in the schema
	s.fields[name] = dependent
	return s
}

// Custom adds a custom validator (override to return correct type).
func (s *ObjectSchemaWithDependencies) Custom(fn queryfy.ValidatorFunc) *ObjectSchemaWithDependencies {
	s.ObjectSchema.Custom(fn)
	return s
}

// Required marks the object as required (override to return correct type).
func (s *ObjectSchemaWithDependencies) Required() *ObjectSchemaWithDependencies {
	s.ObjectSchema.Required()
	return s
}

// Optional marks the object as optional (override to return correct type).
func (s *ObjectSchemaWithDependencies) Optional() *ObjectSchemaWithDependencies {
	s.ObjectSchema.Optional()
	return s
}

// Nullable allows the object to be null (override to return correct type).
func (s *ObjectSchemaWithDependencies) Nullable() *ObjectSchemaWithDependencies {
	s.ObjectSchema.Nullable()
	return s
}

// Validate overrides the base validate to handle dependent fields.
func (s *ObjectSchemaWithDependencies) Validate(value interface{}, ctx *queryfy.ValidationContext) error {
	// First convert to map
	objMap, ok := convertToMap(value)
	if !ok {
		ctx.AddError(fmt.Sprintf("cannot convert %T to map", value), value)
		return nil
	}

	// Validate regular fields first
	if err := s.ObjectSchema.Validate(value, ctx); err != nil {
		return err
	}

	// Now validate dependent fields with parent context
	for fieldName, depSchema := range s.dependentFields {
		fieldValue, exists := objMap[fieldName]

		ctx.WithPath(fieldName, func() {
			// If field doesn't exist, check if it's required based on condition
			if !exists {
				if depSchema.conditionFunc != nil && depSchema.conditionFunc(objMap) {
					if depSchema.IsRequired() || (depSchema.schema != nil && isRequired(depSchema.schema)) {
						ctx.AddError("field is required", nil)
					}
				}
			} else {
				// Validate with parent context
				depSchema.ValidateWithParent(fieldValue, objMap, ctx)
			}
		})
	}

	return nil
}

// Helper functions for common patterns

// RequiredWhen creates a schema that makes a field required when a condition is met.
func RequiredWhen(condition DependencyCondition, schema queryfy.Schema) *DependentSchema {
	return Dependent("").
		When(condition).
		Then(markAsRequired(schema))
}

// RequiredUnless creates a schema that makes a field required unless a condition is met.
func RequiredUnless(condition DependencyCondition, schema queryfy.Schema) *DependentSchema {
	return Dependent("").
		When(func(data map[string]interface{}) bool {
			return !condition(data)
		}).
		Then(markAsRequired(schema))
}

// markAsRequired ensures a schema is marked as required.
func markAsRequired(schema queryfy.Schema) queryfy.Schema {
	if setter, ok := schema.(interface{ SetRequired(bool) }); ok {
		setter.SetRequired(true)
	}
	return schema
}

// Example usage patterns:
/*
// Basic dependent field
schema := builders.Object().WithDependencies().
    Field("accountType", builders.String().Enum("personal", "business")).
    DependentField("companyName",
        builders.Dependent("companyName").
            On("accountType").
            When(builders.WhenEquals("accountType", "business")).
            Then(builders.String().Required()).
            Else(builders.String().Optional()))

// Multiple dependencies
schema := builders.Object().WithDependencies().
    Field("country", builders.String()).
    Field("hasAddress", builders.Bool()).
    DependentField("zipCode",
        builders.Dependent("zipCode").
            On("country", "hasAddress").
            When(builders.WhenAll(
                builders.WhenEquals("country", "US"),
                builders.WhenTrue("hasAddress"),
            )).
            Then(builders.String().Pattern(`^\d{5}(-\d{4})?$`).Required()))

// Complex conditions
schema := builders.Object().WithDependencies().
    Field("orderTotal", builders.Number()).
    Field("customerType", builders.String()).
    DependentField("approvalRequired",
        builders.Dependent("approvalRequired").
            When(builders.WhenAny(
                builders.WhenGreaterThan("orderTotal", 10000),
                builders.WhenEquals("customerType", "new"),
            )).
            Then(builders.Bool().Required()).
            Else(builders.Bool().Optional()))
*/
