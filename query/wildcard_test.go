package query_test

import (
	"testing"

	"github.com/ha1tch/queryfy/query"
)

// Test data
var orderData = map[string]interface{}{
	"items": []interface{}{
		map[string]interface{}{
			"name":  "Widget",
			"price": 9.99,
			"tags":  []interface{}{"sale", "popular"},
		},
		map[string]interface{}{
			"name":  "Gadget",
			"price": 24.99,
			"tags":  []interface{}{"new"},
		},
		map[string]interface{}{
			"name":  "Doohickey",
			"price": 4.50,
			"tags":  []interface{}{"sale", "clearance"},
		},
	},
	"customers": []interface{}{
		map[string]interface{}{
			"name": "Alice",
			"orders": []interface{}{
				map[string]interface{}{"id": 1.0, "total": 100.0},
				map[string]interface{}{"id": 2.0, "total": 200.0},
			},
		},
		map[string]interface{}{
			"name": "Bob",
			"orders": []interface{}{
				map[string]interface{}{"id": 3.0, "total": 50.0},
			},
		},
	},
}

// ======================================================================
// Wildcard: items[*].field
// ======================================================================

func TestWildcard_BasicField(t *testing.T) {
	result, err := query.Execute(orderData, "items[*].name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	names, ok := result.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", result)
	}
	if len(names) != 3 {
		t.Fatalf("expected 3 names, got %d", len(names))
	}
	if names[0] != "Widget" || names[1] != "Gadget" || names[2] != "Doohickey" {
		t.Errorf("unexpected names: %v", names)
	}
}

func TestWildcard_NumericField(t *testing.T) {
	result, err := query.Execute(orderData, "items[*].price")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	prices := result.([]interface{})
	if len(prices) != 3 {
		t.Fatalf("expected 3 prices, got %d", len(prices))
	}
	if prices[0] != 9.99 || prices[1] != 24.99 || prices[2] != 4.50 {
		t.Errorf("unexpected prices: %v", prices)
	}
}

func TestWildcard_Standalone(t *testing.T) {
	// items[*] without further path — returns all elements
	result, err := query.Execute(orderData, "items[*]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := result.([]interface{})
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
}

// ======================================================================
// Nested wildcards: customers[*].orders[*].total
// ======================================================================

func TestWildcard_Nested(t *testing.T) {
	result, err := query.Execute(orderData, "customers[*].orders[*].total")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	totals := result.([]interface{})
	// Alice: 100, 200; Bob: 50 → flattened: [100, 200, 50]
	if len(totals) != 3 {
		t.Fatalf("expected 3 totals, got %d: %v", len(totals), totals)
	}
	if totals[0] != 100.0 || totals[1] != 200.0 || totals[2] != 50.0 {
		t.Errorf("unexpected totals: %v", totals)
	}
}

func TestWildcard_NestedNames(t *testing.T) {
	result, err := query.Execute(orderData, "customers[*].name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	names := result.([]interface{})
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
	if names[0] != "Alice" || names[1] != "Bob" {
		t.Errorf("unexpected names: %v", names)
	}
}

// ======================================================================
// Wildcard + index: items[*].tags[0]
// ======================================================================

func TestWildcard_FollowedByIndex(t *testing.T) {
	result, err := query.Execute(orderData, "items[*].tags[0]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	firstTags := result.([]interface{})
	if len(firstTags) != 3 {
		t.Fatalf("expected 3 first tags, got %d", len(firstTags))
	}
	if firstTags[0] != "sale" || firstTags[1] != "new" || firstTags[2] != "sale" {
		t.Errorf("unexpected first tags: %v", firstTags)
	}
}

// ======================================================================
// Error cases
// ======================================================================

func TestWildcard_OnNonArray(t *testing.T) {
	data := map[string]interface{}{"name": "Alice"}
	_, err := query.Execute(data, "name[*]")
	if err == nil {
		t.Error("expected error for wildcard on string")
	}
}

func TestWildcard_OnNilValue(t *testing.T) {
	data := map[string]interface{}{"items": nil}
	_, err := query.Execute(data, "items[*]")
	if err == nil {
		t.Error("expected error for wildcard on nil")
	}
}

func TestWildcard_NestedFieldMissing(t *testing.T) {
	data := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"name": "A"},
			map[string]interface{}{"other": "B"},
		},
	}
	_, err := query.Execute(data, "items[*].name")
	if err == nil {
		t.Error("expected error for missing field in wildcard expansion")
	}
}

// ======================================================================
// Empty array
// ======================================================================

func TestWildcard_EmptyArray(t *testing.T) {
	data := map[string]interface{}{"items": []interface{}{}}
	result, err := query.Execute(data, "items[*].name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := result.([]interface{})
	if len(items) != 0 {
		t.Errorf("expected empty result, got %v", items)
	}
}

// ======================================================================
// Cached execution with wildcards
// ======================================================================

func TestWildcard_Cached(t *testing.T) {
	query.ClearCache()

	// First call — cache miss
	result1, err := query.ExecuteCached(orderData, "items[*].name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second call — cache hit
	result2, err := query.ExecuteCached(orderData, "items[*].name")
	if err != nil {
		t.Fatalf("unexpected error on cached: %v", err)
	}

	names1 := result1.([]interface{})
	names2 := result2.([]interface{})
	if len(names1) != len(names2) {
		t.Errorf("cached result differs: %v vs %v", names1, names2)
	}
}

// ======================================================================
// Parse verification
// ======================================================================

func TestWildcard_Parse(t *testing.T) {
	path, err := query.PathFromQuery("items[*].name")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if len(path) != 3 {
		t.Fatalf("expected 3 segments, got %d: %v", len(path), path)
	}
	if path[0] != "items" {
		t.Errorf("expected 'items', got %v", path[0])
	}
	if _, ok := path[1].(query.Wildcard); !ok {
		t.Errorf("expected Wildcard, got %T", path[1])
	}
	if path[2] != "name" {
		t.Errorf("expected 'name', got %v", path[2])
	}
}
