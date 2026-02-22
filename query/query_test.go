package query_test

import (
	"testing"

	"github.com/ha1tch/queryfy/query"
)

// Test data used across tests
var testData = map[string]interface{}{
	"name": "Alice",
	"age":  30,
	"address": map[string]interface{}{
		"street": "123 Main St",
		"city":   "Springfield",
		"zip":    "62701",
	},
	"emails": []interface{}{
		"alice@example.com",
		"alice.work@company.com",
	},
	"orders": []interface{}{
		map[string]interface{}{
			"id":    "ord-001",
			"total": 99.50,
			"items": []interface{}{
				map[string]interface{}{"sku": "A1", "qty": 2},
				map[string]interface{}{"sku": "B2", "qty": 1},
			},
		},
		map[string]interface{}{
			"id":    "ord-002",
			"total": 150.00,
		},
	},
	"active": true,
}

// ======================================================================
// Execute — basic field access
// ======================================================================

func TestExecute_SimpleField(t *testing.T) {
	result, err := query.Execute(testData, "name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Alice" {
		t.Errorf("expected 'Alice', got %v", result)
	}
}

func TestExecute_NumericField(t *testing.T) {
	result, err := query.Execute(testData, "age")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 30 {
		t.Errorf("expected 30, got %v", result)
	}
}

func TestExecute_BoolField(t *testing.T) {
	result, err := query.Execute(testData, "active")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

// ======================================================================
// Execute — nested field access
// ======================================================================

func TestExecute_NestedField(t *testing.T) {
	result, err := query.Execute(testData, "address.city")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Springfield" {
		t.Errorf("expected 'Springfield', got %v", result)
	}
}

func TestExecute_DeepNested(t *testing.T) {
	result, err := query.Execute(testData, "address.street")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "123 Main St" {
		t.Errorf("expected '123 Main St', got %v", result)
	}
}

// ======================================================================
// Execute — array indexing
// ======================================================================

func TestExecute_ArrayIndex(t *testing.T) {
	result, err := query.Execute(testData, "emails[0]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "alice@example.com" {
		t.Errorf("expected 'alice@example.com', got %v", result)
	}
}

func TestExecute_ArrayIndexSecond(t *testing.T) {
	result, err := query.Execute(testData, "emails[1]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "alice.work@company.com" {
		t.Errorf("expected 'alice.work@company.com', got %v", result)
	}
}

func TestExecute_ArrayOfObjects(t *testing.T) {
	result, err := query.Execute(testData, "orders[0].id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ord-001" {
		t.Errorf("expected 'ord-001', got %v", result)
	}
}

func TestExecute_ArrayOfObjectsDeep(t *testing.T) {
	result, err := query.Execute(testData, "orders[0].items[1].sku")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "B2" {
		t.Errorf("expected 'B2', got %v", result)
	}
}

func TestExecute_ArrayNumericField(t *testing.T) {
	result, err := query.Execute(testData, "orders[1].total")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 150.00 {
		t.Errorf("expected 150.00, got %v", result)
	}
}

// ======================================================================
// Execute — empty query
// ======================================================================

func TestExecute_EmptyQuery(t *testing.T) {
	result, err := query.Execute(testData, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty query returns the root data
	if result == nil {
		t.Error("expected root data, got nil")
	}
}

// ======================================================================
// Execute — error cases
// ======================================================================

func TestExecute_NonExistentField(t *testing.T) {
	_, err := query.Execute(testData, "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent field")
	}
}

func TestExecute_NonExistentNested(t *testing.T) {
	_, err := query.Execute(testData, "address.country")
	if err == nil {
		t.Error("expected error for non-existent nested field")
	}
}

func TestExecute_ArrayOutOfBounds(t *testing.T) {
	_, err := query.Execute(testData, "emails[5]")
	if err == nil {
		t.Error("expected error for array out of bounds")
	}
}

func TestExecute_FieldOnNonObject(t *testing.T) {
	_, err := query.Execute(testData, "name.sub")
	if err == nil {
		t.Error("expected error for field access on non-object")
	}
}

func TestExecute_IndexOnNonArray(t *testing.T) {
	_, err := query.Execute(testData, "name[0]")
	if err == nil {
		t.Error("expected error for index access on non-array")
	}
}

// ======================================================================
// ParseQuery — edge cases
// ======================================================================

func TestParseQuery_Simple(t *testing.T) {
	q, err := query.ParseQuery("name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q == nil || q.Root == nil {
		t.Fatal("expected parsed query")
	}
}

func TestParseQuery_Dotted(t *testing.T) {
	q, err := query.ParseQuery("a.b.c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	path := query.SimplifyNode(q.Root.Child)
	if len(path) != 3 {
		t.Errorf("expected 3 path segments, got %d", len(path))
	}
}

func TestParseQuery_ArrayAccess(t *testing.T) {
	q, err := query.ParseQuery("items[0]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	path := query.SimplifyNode(q.Root.Child)
	if len(path) != 2 {
		t.Errorf("expected 2 path segments (field + index), got %d", len(path))
	}
	if path[0] != "items" {
		t.Errorf("expected 'items', got %v", path[0])
	}
	if path[1] != 0 {
		t.Errorf("expected index 0, got %v", path[1])
	}
}

func TestParseQuery_Complex(t *testing.T) {
	q, err := query.ParseQuery("orders[0].items[1].sku")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	path := query.SimplifyNode(q.Root.Child)
	// orders, 0, items, 1, sku = 5 segments
	if len(path) != 5 {
		t.Errorf("expected 5 path segments, got %d: %v", len(path), path)
	}
}

func TestParseQuery_EmptyString(t *testing.T) {
	_, err := query.ParseQuery("")
	if err == nil {
		t.Error("expected error for empty query string")
	}
}

func TestParseQuery_InvalidSyntax(t *testing.T) {
	_, err := query.ParseQuery("[0]")
	if err == nil {
		t.Error("expected error for query starting with bracket")
	}
}

func TestParseQuery_UnclosedBracket(t *testing.T) {
	_, err := query.ParseQuery("items[0")
	if err == nil {
		t.Error("expected error for unclosed bracket")
	}
}

// ======================================================================
// PathFromQuery
// ======================================================================

func TestPathFromQuery(t *testing.T) {
	path, err := query.PathFromQuery("user.name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(path) != 2 || path[0] != "user" || path[1] != "name" {
		t.Errorf("unexpected path: %v", path)
	}
}

// ======================================================================
// ExecuteCached
// ======================================================================

func TestExecuteCached(t *testing.T) {
	query.ClearCache()

	// First call — cache miss
	result, err := query.ExecuteCached(testData, "name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Alice" {
		t.Errorf("expected 'Alice', got %v", result)
	}

	// Second call — cache hit
	result2, err := query.ExecuteCached(testData, "name")
	if err != nil {
		t.Fatalf("unexpected error on cached call: %v", err)
	}
	if result2 != "Alice" {
		t.Errorf("expected 'Alice' from cache, got %v", result2)
	}
}

func TestExecuteCached_Nested(t *testing.T) {
	query.ClearCache()

	result, err := query.ExecuteCached(testData, "orders[0].items[0].sku")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "A1" {
		t.Errorf("expected 'A1', got %v", result)
	}
}

// ======================================================================
// Struct field access via reflection
// ======================================================================

type testStruct struct {
	Name  string
	Count int
}

func TestExecute_StructField(t *testing.T) {
	data := map[string]interface{}{
		"item": testStruct{Name: "widget", Count: 5},
	}

	result, err := query.Execute(data, "item.Name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "widget" {
		t.Errorf("expected 'widget', got %v", result)
	}
}

func TestExecute_TypedSlice(t *testing.T) {
	// Use a typed slice (not []interface{}) to exercise reflection path
	data := map[string]interface{}{
		"scores": []int{10, 20, 30},
	}

	result, err := query.Execute(data, "scores[1]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 20 {
		t.Errorf("expected 20, got %v", result)
	}
}

func TestExecute_TypedSliceOutOfBounds(t *testing.T) {
	data := map[string]interface{}{
		"scores": []int{10, 20},
	}

	_, err := query.Execute(data, "scores[5]")
	if err == nil {
		t.Error("expected out of bounds error for typed slice")
	}
}

func TestExecute_TypedMap(t *testing.T) {
	// map[string]int rather than map[string]interface{}
	data := map[string]interface{}{
		"config": map[string]int{"port": 8080, "timeout": 30},
	}

	result, err := query.Execute(data, "config.port")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 8080 {
		t.Errorf("expected 8080, got %v", result)
	}
}

func TestExecute_NonExistentStructField(t *testing.T) {
	data := map[string]interface{}{
		"item": testStruct{Name: "x", Count: 1},
	}

	_, err := query.Execute(data, "item.Missing")
	if err == nil {
		t.Error("expected error for non-existent struct field")
	}
}

func TestExecute_FieldOnScalar(t *testing.T) {
	// Access field on a non-map, non-struct scalar
	_, err := query.Execute(map[string]interface{}{"x": 42}, "x.sub")
	if err == nil {
		t.Error("expected error for field access on int")
	}
}

func TestExecute_IndexOnScalar(t *testing.T) {
	// Access index on a non-slice scalar
	_, err := query.Execute(map[string]interface{}{"x": "hello"}, "x[0]")
	if err == nil {
		t.Error("expected error for index access on string")
	}
}
