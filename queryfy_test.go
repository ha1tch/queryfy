package queryfy_test

import (
	"fmt"
	"strings"
	"testing"

	qf "github.com/ha1tch/queryfy"
	"github.com/ha1tch/queryfy/builders"
)

// Test basic string validation
func TestStringValidation(t *testing.T) {
	tests := []struct {
		name    string
		schema  qf.Schema
		value   interface{}
		wantErr bool
	}{
		{
			name:    "valid string",
			schema:  builders.String(),
			value:   "hello",
			wantErr: false,
		},
		{
			name:    "required string present",
			schema:  builders.String().Required(),
			value:   "hello",
			wantErr: false,
		},
		{
			name:    "required string missing",
			schema:  builders.String().Required(),
			value:   nil,
			wantErr: true,
		},
		{
			name:    "min length valid",
			schema:  builders.String().MinLength(3),
			value:   "hello",
			wantErr: false,
		},
		{
			name:    "min length invalid",
			schema:  builders.String().MinLength(10),
			value:   "hello",
			wantErr: true,
		},
		{
			name:    "max length valid",
			schema:  builders.String().MaxLength(10),
			value:   "hello",
			wantErr: false,
		},
		{
			name:    "max length invalid",
			schema:  builders.String().MaxLength(3),
			value:   "hello",
			wantErr: true,
		},
		{
			name:    "exact length valid",
			schema:  builders.String().Length(5),
			value:   "hello",
			wantErr: false,
		},
		{
			name:    "exact length invalid",
			schema:  builders.String().Length(3),
			value:   "hello",
			wantErr: true,
		},
		{
			name:    "pattern match valid",
			schema:  builders.String().Pattern(`^[a-z]+$`),
			value:   "hello",
			wantErr: false,
		},
		{
			name:    "pattern match invalid",
			schema:  builders.String().Pattern(`^[0-9]+$`),
			value:   "hello",
			wantErr: true,
		},
		{
			name:    "email valid",
			schema:  builders.String().Email(),
			value:   "user@example.com",
			wantErr: false,
		},
		{
			name:    "email invalid",
			schema:  builders.String().Email(),
			value:   "not-an-email",
			wantErr: true,
		},
		{
			name:    "enum valid",
			schema:  builders.String().Enum("red", "green", "blue"),
			value:   "red",
			wantErr: false,
		},
		{
			name:    "enum invalid",
			schema:  builders.String().Enum("red", "green", "blue"),
			value:   "yellow",
			wantErr: true,
		},
		{
			name:    "nullable with nil",
			schema:  builders.String().Nullable(),
			value:   nil,
			wantErr: false,
		},
		{
			name:    "non-nullable with nil",
			schema:  builders.String(),
			value:   nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := qf.Validate(tt.value, tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test number validation
func TestNumberValidation(t *testing.T) {
	tests := []struct {
		name    string
		schema  qf.Schema
		value   interface{}
		wantErr bool
	}{
		{
			name:    "valid int",
			schema:  builders.Number(),
			value:   42,
			wantErr: false,
		},
		{
			name:    "valid float",
			schema:  builders.Number(),
			value:   42.5,
			wantErr: false,
		},
		{
			name:    "min valid",
			schema:  builders.Number().Min(10),
			value:   15,
			wantErr: false,
		},
		{
			name:    "min invalid",
			schema:  builders.Number().Min(10),
			value:   5,
			wantErr: true,
		},
		{
			name:    "max valid",
			schema:  builders.Number().Max(100),
			value:   50,
			wantErr: false,
		},
		{
			name:    "max invalid",
			schema:  builders.Number().Max(100),
			value:   150,
			wantErr: true,
		},
		{
			name:    "range valid",
			schema:  builders.Number().Range(10, 100),
			value:   50,
			wantErr: false,
		},
		{
			name:    "range invalid low",
			schema:  builders.Number().Range(10, 100),
			value:   5,
			wantErr: true,
		},
		{
			name:    "range invalid high",
			schema:  builders.Number().Range(10, 100),
			value:   150,
			wantErr: true,
		},
		{
			name:    "integer valid",
			schema:  builders.Number().Integer(),
			value:   42,
			wantErr: false,
		},
		{
			name:    "integer invalid",
			schema:  builders.Number().Integer(),
			value:   42.5,
			wantErr: true,
		},
		{
			name:    "positive valid",
			schema:  builders.Number().Positive(),
			value:   42,
			wantErr: false,
		},
		{
			name:    "positive invalid",
			schema:  builders.Number().Positive(),
			value:   -5,
			wantErr: true,
		},
		{
			name:    "positive zero invalid",
			schema:  builders.Number().Positive(),
			value:   0,
			wantErr: true,
		},
		{
			name:    "multiple of valid",
			schema:  builders.Number().MultipleOf(5),
			value:   25,
			wantErr: false,
		},
		{
			name:    "multiple of invalid",
			schema:  builders.Number().MultipleOf(5),
			value:   23,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := qf.Validate(tt.value, tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test boolean validation
func TestBooleanValidation(t *testing.T) {
	tests := []struct {
		name    string
		schema  qf.Schema
		value   interface{}
		wantErr bool
	}{
		{
			name:    "valid true",
			schema:  builders.Bool(),
			value:   true,
			wantErr: false,
		},
		{
			name:    "valid false",
			schema:  builders.Bool(),
			value:   false,
			wantErr: false,
		},
		{
			name:    "invalid string",
			schema:  builders.Bool(),
			value:   "true",
			wantErr: true,
		},
		{
			name:    "invalid number",
			schema:  builders.Bool(),
			value:   1,
			wantErr: true,
		},
		{
			name:    "required present",
			schema:  builders.Bool().Required(),
			value:   true,
			wantErr: false,
		},
		{
			name:    "required missing",
			schema:  builders.Bool().Required(),
			value:   nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := qf.Validate(tt.value, tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test array validation
func TestArrayValidation(t *testing.T) {
	tests := []struct {
		name    string
		schema  qf.Schema
		value   interface{}
		wantErr bool
	}{
		{
			name:    "valid array",
			schema:  builders.Array(),
			value:   []interface{}{"a", "b", "c"},
			wantErr: false,
		},
		{
			name:    "valid typed array",
			schema:  builders.Array().Of(builders.String()),
			value:   []interface{}{"a", "b", "c"},
			wantErr: false,
		},
		{
			name:    "invalid typed array",
			schema:  builders.Array().Of(builders.String()),
			value:   []interface{}{"a", 2, "c"},
			wantErr: true,
		},
		{
			name:    "min items valid",
			schema:  builders.Array().MinItems(2),
			value:   []interface{}{"a", "b", "c"},
			wantErr: false,
		},
		{
			name:    "min items invalid",
			schema:  builders.Array().MinItems(5),
			value:   []interface{}{"a", "b", "c"},
			wantErr: true,
		},
		{
			name:    "max items valid",
			schema:  builders.Array().MaxItems(5),
			value:   []interface{}{"a", "b", "c"},
			wantErr: false,
		},
		{
			name:    "max items invalid",
			schema:  builders.Array().MaxItems(2),
			value:   []interface{}{"a", "b", "c"},
			wantErr: true,
		},
		{
			name:    "exact length valid",
			schema:  builders.Array().Length(3),
			value:   []interface{}{"a", "b", "c"},
			wantErr: false,
		},
		{
			name:    "exact length invalid",
			schema:  builders.Array().Length(2),
			value:   []interface{}{"a", "b", "c"},
			wantErr: true,
		},
		{
			name:    "unique items valid",
			schema:  builders.Array().UniqueItems(),
			value:   []interface{}{"a", "b", "c"},
			wantErr: false,
		},
		{
			name:    "unique items invalid",
			schema:  builders.Array().UniqueItems(),
			value:   []interface{}{"a", "b", "a"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := qf.Validate(tt.value, tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test object validation
func TestObjectValidation(t *testing.T) {
	tests := []struct {
		name    string
		schema  qf.Schema
		value   interface{}
		wantErr bool
	}{
		{
			name:    "valid object",
			schema:  builders.Object().Field("name", builders.String()),
			value:   map[string]interface{}{"name": "John"},
			wantErr: false,
		},
		{
			name:    "required field present",
			schema:  builders.Object().Field("name", builders.String().Required()),
			value:   map[string]interface{}{"name": "John"},
			wantErr: false,
		},
		{
			name:    "required field missing",
			schema:  builders.Object().Field("name", builders.String().Required()),
			value:   map[string]interface{}{},
			wantErr: true,
		},
		{
			name:    "nested object valid",
			schema:  builders.Object().Field("user", builders.Object().Field("email", builders.String().Email())),
			value:   map[string]interface{}{"user": map[string]interface{}{"email": "user@example.com"}},
			wantErr: false,
		},
		{
			name:    "nested object invalid",
			schema:  builders.Object().Field("user", builders.Object().Field("email", builders.String().Email())),
			value:   map[string]interface{}{"user": map[string]interface{}{"email": "not-email"}},
			wantErr: true,
		},
		{
			name:    "extra field in strict mode",
			schema:  builders.Object().Field("name", builders.String()),
			value:   map[string]interface{}{"name": "John", "extra": "field"},
			wantErr: true, // Strict mode by default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := qf.Validate(tt.value, tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test AND composite validation
func TestAndValidation(t *testing.T) {
	tests := []struct {
		name    string
		schema  qf.Schema
		value   interface{}
		wantErr bool
	}{
		{
			name: "all conditions pass",
			schema: builders.And(
				builders.Number().Min(10),
				builders.Number().Max(100),
				builders.Number().Integer(),
			),
			value:   50,
			wantErr: false,
		},
		{
			name: "one condition fails",
			schema: builders.And(
				builders.Number().Min(10),
				builders.Number().Max(100),
				builders.Number().Integer(),
			),
			value:   50.5,
			wantErr: true,
		},
		{
			name: "all conditions fail",
			schema: builders.And(
				builders.Number().Min(10),
				builders.Number().Max(100),
			),
			value:   150,
			wantErr: true,
		},
		{
			name: "string with multiple patterns",
			schema: builders.And(
				builders.String().MinLength(8),
				builders.String().Pattern(`[a-z]`),
				builders.String().Pattern(`[0-9]`),
			),
			value:   "hello123",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := qf.Validate(tt.value, tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test OR composite validation
func TestOrValidation(t *testing.T) {
	tests := []struct {
		name    string
		schema  qf.Schema
		value   interface{}
		wantErr bool
	}{
		{
			name: "first condition passes",
			schema: builders.Or(
				builders.String().Email(),
				builders.String().Pattern(`^\+?[1-9]\d{9,14}$`),
			),
			value:   "user@example.com",
			wantErr: false,
		},
		{
			name: "second condition passes",
			schema: builders.Or(
				builders.String().Email(),
				builders.String().Pattern(`^\+?[1-9]\d{9,14}$`),
			),
			value:   "+1234567890",
			wantErr: false,
		},
		{
			name: "no conditions pass",
			schema: builders.Or(
				builders.String().Email(),
				builders.String().Pattern(`^\+?[1-9]\d{9,14}$`),
			),
			value:   "not-email-or-phone",
			wantErr: true,
		},
		{
			name: "number ranges",
			schema: builders.Or(
				builders.Number().Max(10),
				builders.Number().Min(100),
			),
			value:   5,
			wantErr: false,
		},
		{
			name: "number ranges high",
			schema: builders.Or(
				builders.Number().Max(10),
				builders.Number().Min(100),
			),
			value:   150,
			wantErr: false,
		},
		{
			name: "number ranges invalid",
			schema: builders.Or(
				builders.Number().Max(10),
				builders.Number().Min(100),
			),
			value:   50,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := qf.Validate(tt.value, tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test NOT composite validation
func TestNotValidation(t *testing.T) {
	tests := []struct {
		name    string
		schema  qf.Schema
		value   interface{}
		wantErr bool
	}{
		{
			name:    "not empty string - valid",
			schema:  builders.Not(builders.String().Length(0)),
			value:   "hello",
			wantErr: false,
		},
		{
			name:    "not empty string - invalid",
			schema:  builders.Not(builders.String().Length(0)),
			value:   "",
			wantErr: true,
		},
		{
			name:    "not a specific value",
			schema:  builders.Not(builders.String().Enum("admin", "root")),
			value:   "user",
			wantErr: false,
		},
		{
			name:    "not a specific value - invalid",
			schema:  builders.Not(builders.String().Enum("admin", "root")),
			value:   "admin",
			wantErr: true,
		},
		{
			name: "combined with AND",
			schema: builders.And(
				builders.String(),
				builders.Not(builders.String().Pattern(`\s`)), // no whitespace
			),
			value:   "nospaces",
			wantErr: false,
		},
		{
			name: "combined with AND - invalid",
			schema: builders.And(
				builders.String(),
				builders.Not(builders.String().Pattern(`\s`)), // no whitespace
			),
			value:   "has spaces",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := qf.Validate(tt.value, tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test loose mode validation
func TestLooseModeValidation(t *testing.T) {
	tests := []struct {
		name    string
		schema  qf.Schema
		value   interface{}
		mode    qf.ValidationMode
		wantErr bool
	}{
		{
			name:    "string to number conversion",
			schema:  builders.Number(),
			value:   "42",
			mode:    qf.Loose,
			wantErr: false,
		},
		{
			name:    "string to number invalid",
			schema:  builders.Number(),
			value:   "not-a-number",
			mode:    qf.Loose,
			wantErr: true,
		},
		{
			name:    "string to bool true",
			schema:  builders.Bool(),
			value:   "true",
			mode:    qf.Loose,
			wantErr: false,
		},
		{
			name:    "string to bool false",
			schema:  builders.Bool(),
			value:   "false",
			mode:    qf.Loose,
			wantErr: false,
		},
		{
			name:    "string to bool invalid",
			schema:  builders.Bool(),
			value:   "yes",
			mode:    qf.Loose,
			wantErr: true,
		},
		{
			name: "extra fields allowed",
			schema: builders.Object().
				Field("name", builders.String()),
			value: map[string]interface{}{
				"name":  "John",
				"extra": "allowed",
			},
			mode:    qf.Loose,
			wantErr: false,
		},
		{
			name:    "number to string conversion",
			schema:  builders.String(),
			value:   42,
			mode:    qf.Loose,
			wantErr: false,
		},
		{
			name:    "bool to string conversion",
			schema:  builders.String(),
			value:   true,
			mode:    qf.Loose,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := qf.ValidateWithMode(tt.value, tt.schema, tt.mode)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWithMode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test query functionality
func TestQuery(t *testing.T) {
	data := map[string]interface{}{
		"name": "John",
		"age":  30,
		"address": map[string]interface{}{
			"street": "123 Main St",
			"city":   "New York",
		},
		"hobbies": []interface{}{"reading", "gaming", "coding"},
		"scores":  []interface{}{85, 92, 78, 95},
	}

	tests := []struct {
		name      string
		query     string
		want      interface{}
		wantErr   bool
		checkFunc func(got interface{}) bool // Custom check function for complex types
	}{
		{
			name:    "simple field",
			query:   "name",
			want:    "John",
			wantErr: false,
		},
		{
			name:    "nested field",
			query:   "address.city",
			want:    "New York",
			wantErr: false,
		},
		{
			name:    "array index",
			query:   "hobbies[0]",
			want:    "reading",
			wantErr: false,
		},
		{
			name:    "array index nested",
			query:   "scores[1]",
			want:    92,
			wantErr: false,
		},
		{
			name:    "non-existent field",
			query:   "email",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "non-existent nested",
			query:   "address.country",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "array out of bounds",
			query:   "hobbies[10]",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty query",
			query:   "",
			wantErr: false,
			checkFunc: func(got interface{}) bool {
				// For empty query, we expect the entire data back
				// Check it's a map with expected keys
				m, ok := got.(map[string]interface{})
				if !ok {
					return false
				}
				_, hasName := m["name"]
				_, hasAge := m["age"]
				_, hasAddress := m["address"]
				return hasName && hasAge && hasAddress
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := qf.Query(data, tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("Query() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if tt.checkFunc != nil {
					if !tt.checkFunc(got) {
						t.Errorf("Query() custom check failed for result %v", got)
					}
				} else if got != tt.want {
					t.Errorf("Query() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

// Test error messages with paths
func TestErrorPaths(t *testing.T) {
	schema := builders.Object().
		Field("user", builders.Object().
			Field("email", builders.String().Email().Required()).
			Field("age", builders.Number().Min(18).Required())).
		Field("items", builders.Array().Of(
			builders.Object().
				Field("name", builders.String().MinLength(1).Required()). // Add MinLength to reject empty strings
				Field("price", builders.Number().Min(0).Required())))

	data := map[string]interface{}{
		"user": map[string]interface{}{
			"email": "not-an-email",
			"age":   15,
		},
		"items": []interface{}{
			map[string]interface{}{
				"name":  "Item 1",
				"price": 10,
			},
			map[string]interface{}{
				"name":  "", // Empty string should fail with MinLength(1)
				"price": -5,
			},
		},
	}

	err := qf.Validate(data, schema)
	if err == nil {
		t.Fatal("expected validation error")
	}

	validationErr, ok := err.(*qf.ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}

	expectedPaths := map[string]bool{
		"user.email":      true,
		"user.age":        true,
		"items[1].name":   true,
		"items[1].price":  true,
	}

	for _, fieldErr := range validationErr.Errors {
		if !expectedPaths[fieldErr.Path] {
			t.Errorf("unexpected error path: %s", fieldErr.Path)
		}
		delete(expectedPaths, fieldErr.Path)
	}

	for path := range expectedPaths {
		t.Errorf("expected error for path %s but didn't get one", path)
	}
}

// Test custom validators
func TestCustomValidators(t *testing.T) {
	// Custom validator that checks if string contains "test"
	containsTest := builders.Custom(func(value interface{}) error {
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string")
		}
		if !strings.Contains(str, "test") {
			return fmt.Errorf("must contain 'test'")
		}
		return nil
	})

	tests := []struct {
		name    string
		schema  qf.Schema
		value   interface{}
		wantErr bool
	}{
		{
			name:    "custom validator passes",
			schema:  containsTest,
			value:   "this is a test string",
			wantErr: false,
		},
		{
			name:    "custom validator fails",
			schema:  containsTest,
			value:   "this string has no t-word",
			wantErr: true,
		},
		{
			name:    "custom validator wrong type",
			schema:  containsTest,
			value:   123,
			wantErr: true,
		},
		{
			name: "custom with other validators",
			schema: builders.And(
				builders.String().MinLength(10),
				containsTest,
			),
			value:   "short test",
			wantErr: false,
		},
		{
			name: "custom in object",
			schema: builders.Object().
				Field("description", containsTest.Required()),
			value: map[string]interface{}{
				"description": "product test description",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := qf.Validate(tt.value, tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test complex real-world schema
func TestComplexSchema(t *testing.T) {
	// E-commerce order schema
	orderSchema := builders.Object().
		Field("orderId", builders.String().Pattern(`^ORD-\d{6}$`).Required()).
		Field("status", builders.String().Enum("pending", "processing", "shipped", "delivered", "cancelled").Required()).
		Field("customer", builders.Object().
			Field("id", builders.String().Required()).
			Field("email", builders.String().Email().Required()).
			Field("phone", builders.Or(
				builders.String().Pattern(`^\+?[1-9]\d{9,14}$`),
				builders.String().Length(0),
			))).
		Field("items", builders.Array().MinItems(1).Of(
			builders.Object().
				Field("sku", builders.String().Pattern(`^[A-Z]{3}-\d{4}$`).Required()).
				Field("quantity", builders.Number().Min(1).Integer().Required()).
				Field("price", builders.Number().Min(0).Required()).
				Field("discount", builders.Number().Min(0).Max(100)))).
		Field("payment", builders.Object().
			Field("method", builders.String().Enum("card", "paypal", "bank").Required()).
			Field("status", builders.String().Enum("pending", "completed", "failed").Required()).
			Field("amount", builders.Number().Min(0).Required())).
		Field("shipping", builders.Object().
			Field("address", builders.Object().
				Field("street", builders.String().Required()).
				Field("city", builders.String().Required()).
				Field("state", builders.String().Length(2).Required()).
				Field("zip", builders.String().Pattern(`^\d{5}(-\d{4})?$`).Required())).
			Field("method", builders.String().Enum("standard", "express", "overnight")).
			Field("trackingNumber", builders.And(
				builders.String(),
				builders.Not(builders.String().Length(0)),
			).Optional()))

	validOrder := map[string]interface{}{
		"orderId": "ORD-123456",
		"status":  "processing",
		"customer": map[string]interface{}{
			"id":    "CUST-789",
			"email": "customer@example.com",
			"phone": "+1234567890",
		},
		"items": []interface{}{
			map[string]interface{}{
				"sku":      "ABC-1234",
				"quantity": 2,
				"price":    29.99,
				"discount": 10,
			},
			map[string]interface{}{
				"sku":      "XYZ-5678",
				"quantity": 1,
				"price":    49.99,
			},
		},
		"payment": map[string]interface{}{
			"method": "card",
			"status": "completed",
			"amount": 99.97,
		},
		"shipping": map[string]interface{}{
			"address": map[string]interface{}{
				"street": "123 Main St",
				"city":   "New York",
				"state":  "NY",
				"zip":    "10001",
			},
			"method":         "express",
			"trackingNumber": "TRK123456789",
		},
	}

	err := qf.Validate(validOrder, orderSchema)
	if err != nil {
		t.Errorf("valid order failed validation: %v", err)
	}

	// Test invalid order
	invalidOrder := map[string]interface{}{
		"orderId": "INVALID-ID",
		"status":  "unknown",
		"customer": map[string]interface{}{
			"id":    "CUST-789",
			"email": "not-an-email",
			"phone": "invalid-phone",
		},
		"items": []interface{}{}, // Empty items
		"payment": map[string]interface{}{
			"method": "bitcoin", // Invalid method
			"status": "completed",
			"amount": -10, // Negative amount
		},
		"shipping": map[string]interface{}{
			"address": map[string]interface{}{
				"street": "", // Empty required field
				"city":   "New York",
				"state":  "NEW YORK", // Wrong length
				"zip":    "invalid",  // Wrong pattern
			},
		},
	}

	err = qf.Validate(invalidOrder, orderSchema)
	if err == nil {
		t.Error("invalid order should have failed validation")
	}
}