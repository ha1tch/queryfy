package queryfy_test

import (
	"fmt"
	"testing"

	qf "github.com/ha1tch/queryfy"
	"github.com/ha1tch/queryfy/builders"
)

// benchObjectSchema builds a realistic nested object schema for benchmarking.
func benchObjectSchema() qf.Schema {
	return builders.Object().
		Field("name", builders.String().MinLength(1).MaxLength(100).Required()).
		Field("email", builders.String().Email().Required()).
		Field("age", builders.Number().Min(0).Max(150).Required()).
		Field("role", builders.String().Enum("admin", "editor", "viewer", "guest").Required()).
		Field("address", builders.Object().
			Field("street", builders.String().MinLength(1).Required()).
			Field("city", builders.String().MinLength(1).Required()).
			Field("zip", builders.String().Pattern(`^\d{5}$`).Required()))
}

var benchValidObject = map[string]interface{}{
	"name":  "Alice Johnson",
	"email": "alice@example.com",
	"age":   float64(30),
	"role":  "editor",
	"address": map[string]interface{}{
		"street": "123 Main St",
		"city":   "Springfield",
		"zip":    "62704",
	},
}

var benchInvalidObject = map[string]interface{}{
	"name":  "",
	"email": "not-an-email",
	"age":   float64(200),
	"role":  "superadmin",
	"address": map[string]interface{}{
		"street": "",
		"city":   "",
		"zip":    "ABCDE",
	},
}

func BenchmarkValidate_Object_Raw(b *testing.B) {
	schema := benchObjectSchema()
	ctx := qf.NewValidationContext(qf.Strict)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Reset()
		schema.Validate(benchValidObject, ctx)
	}
}

func BenchmarkValidate_Object_Compiled(b *testing.B) {
	schema := qf.Compile(benchObjectSchema())
	ctx := qf.NewValidationContext(qf.Strict)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Reset()
		schema.Validate(benchValidObject, ctx)
	}
}

func BenchmarkValidate_Object_Invalid_Raw(b *testing.B) {
	schema := benchObjectSchema()
	ctx := qf.NewValidationContext(qf.Strict)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Reset()
		schema.Validate(benchInvalidObject, ctx)
	}
}

func BenchmarkValidate_Object_Invalid_Compiled(b *testing.B) {
	schema := qf.Compile(benchObjectSchema())
	ctx := qf.NewValidationContext(qf.Strict)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Reset()
		schema.Validate(benchInvalidObject, ctx)
	}
}

func BenchmarkValidate_String_Raw(b *testing.B) {
	schema := builders.String().MinLength(1).MaxLength(255).Email()
	ctx := qf.NewValidationContext(qf.Strict)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Reset()
		schema.Validate("alice@example.com", ctx)
	}
}

func BenchmarkValidate_String_Compiled(b *testing.B) {
	schema := qf.Compile(builders.String().MinLength(1).MaxLength(255).Email())
	ctx := qf.NewValidationContext(qf.Strict)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Reset()
		schema.Validate("alice@example.com", ctx)
	}
}

func BenchmarkValidate_Enum10_Raw(b *testing.B) {
	schema := builders.String().Enum("a", "b", "c", "d", "e", "f", "g", "h", "i", "j")
	ctx := qf.NewValidationContext(qf.Strict)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Reset()
		schema.Validate("j", ctx)
	}
}

func BenchmarkValidate_Enum10_Compiled(b *testing.B) {
	schema := qf.Compile(builders.String().Enum("a", "b", "c", "d", "e", "f", "g", "h", "i", "j"))
	ctx := qf.NewValidationContext(qf.Strict)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Reset()
		schema.Validate("j", ctx)
	}
}

func BenchmarkValidate_Number_Raw(b *testing.B) {
	schema := builders.Number().Min(0).Max(1000).MultipleOf(0.5)
	ctx := qf.NewValidationContext(qf.Strict)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Reset()
		schema.Validate(float64(42), ctx)
	}
}

func BenchmarkValidate_Number_Compiled(b *testing.B) {
	schema := qf.Compile(builders.Number().Min(0).Max(1000).MultipleOf(0.5))
	ctx := qf.NewValidationContext(qf.Strict)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Reset()
		schema.Validate(float64(42), ctx)
	}
}

func BenchmarkValidate_Array20_Raw(b *testing.B) {
	schema := builders.Array().Of(builders.String().MinLength(1).MaxLength(50)).MinItems(1).MaxItems(100)
	items := make([]interface{}, 20)
	for i := range items {
		items[i] = fmt.Sprintf("item-%d", i)
	}
	ctx := qf.NewValidationContext(qf.Strict)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Reset()
		schema.Validate(items, ctx)
	}
}

func BenchmarkValidate_Array20_Compiled(b *testing.B) {
	schema := qf.Compile(builders.Array().Of(builders.String().MinLength(1).MaxLength(50)).MinItems(1).MaxItems(100))
	items := make([]interface{}, 20)
	for i := range items {
		items[i] = fmt.Sprintf("item-%d", i)
	}
	ctx := qf.NewValidationContext(qf.Strict)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Reset()
		schema.Validate(items, ctx)
	}
}

func BenchmarkCompile_Object(b *testing.B) {
	schema := benchObjectSchema()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qf.Compile(schema)
	}
}
