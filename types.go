package queryfy

// SchemaType represents the type of a schema.
type SchemaType string

const (
	// TypeString represents a string schema
	TypeString SchemaType = "string"
	// TypeNumber represents a number schema
	TypeNumber SchemaType = "number"
	// TypeBool represents a boolean schema
	TypeBool SchemaType = "boolean"
	// TypeObject represents an object schema
	TypeObject SchemaType = "object"
	// TypeArray represents an array schema
	TypeArray SchemaType = "array"
	// TypeAny represents a schema that accepts any type
	TypeAny SchemaType = "any"
	// TypeCustom represents a custom validator
	TypeCustom SchemaType = "custom"
)

// ValidationMode determines how strict the validation is.
type ValidationMode int

const (
	// Strict mode requires exact schema compliance.
	// Extra fields in objects will cause validation to fail.
	Strict ValidationMode = iota

	// Loose mode allows extra fields and safe type coercion.
	// Extra fields in objects are ignored.
	// Safe type coercions are applied (e.g., "123" -> 123).
	Loose
)

// ValidatorFunc is a function that validates a value.
// It should return an error if validation fails, nil otherwise.
type ValidatorFunc func(value interface{}) error

// Option represents a configuration option for validators.
type Option func(interface{})

// String returns the string representation of a SchemaType.
func (t SchemaType) String() string {
	return string(t)
}

// String returns the string representation of a ValidationMode.
func (m ValidationMode) String() string {
	switch m {
	case Strict:
		return "strict"
	case Loose:
		return "loose"
	default:
		return "unknown"
	}
}
