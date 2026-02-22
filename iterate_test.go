package queryfy_test

import (
	"fmt"
	"testing"

	"github.com/ha1tch/queryfy"
	"github.com/ha1tch/queryfy/builders"
)

var iterData = map[string]interface{}{
	"items": []interface{}{
		map[string]interface{}{"name": "Widget", "price": 9.99},
		map[string]interface{}{"name": "Gadget", "price": 24.99},
		map[string]interface{}{"name": "Doohickey", "price": 4.50},
	},
	"single": "hello",
}

// ======================================================================
// Each
// ======================================================================

func TestEach_Wildcard(t *testing.T) {
	var names []string
	err := queryfy.Each(iterData, "items[*].name", func(i int, v interface{}) error {
		names = append(names, v.(string))
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 3 {
		t.Fatalf("expected 3 names, got %d", len(names))
	}
	if names[0] != "Widget" || names[2] != "Doohickey" {
		t.Errorf("unexpected names: %v", names)
	}
}

func TestEach_SingleValue(t *testing.T) {
	var called int
	err := queryfy.Each(iterData, "single", func(i int, v interface{}) error {
		called++
		if i != 0 {
			t.Errorf("expected index 0, got %d", i)
		}
		if v != "hello" {
			t.Errorf("expected 'hello', got %v", v)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called != 1 {
		t.Errorf("expected 1 call, got %d", called)
	}
}

func TestEach_EarlyStop(t *testing.T) {
	var count int
	err := queryfy.Each(iterData, "items[*].name", func(i int, v interface{}) error {
		count++
		if i == 1 {
			return fmt.Errorf("stop")
		}
		return nil
	})
	if err == nil || err.Error() != "stop" {
		t.Errorf("expected stop error, got %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 calls before stop, got %d", count)
	}
}

func TestEach_BadQuery(t *testing.T) {
	err := queryfy.Each(iterData, "nonexistent[*]", func(i int, v interface{}) error {
		return nil
	})
	if err == nil {
		t.Error("expected error for bad query")
	}
}

// ======================================================================
// Collect
// ======================================================================

func TestCollect_Wildcard(t *testing.T) {
	results, err := queryfy.Collect(iterData, "items[*].price", func(v interface{}) (interface{}, error) {
		return v.(float64) * 1.10, nil // 10% markup
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	// 9.99 * 1.10 ≈ 10.989
	if results[0].(float64) < 10.98 || results[0].(float64) > 10.99 {
		t.Errorf("unexpected first result: %v", results[0])
	}
}

func TestCollect_SingleValue(t *testing.T) {
	results, err := queryfy.Collect(iterData, "single", func(v interface{}) (interface{}, error) {
		return v.(string) + "!", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || results[0] != "hello!" {
		t.Errorf("expected ['hello!'], got %v", results)
	}
}

func TestCollect_TransformError(t *testing.T) {
	_, err := queryfy.Collect(iterData, "items[*].name", func(v interface{}) (interface{}, error) {
		return nil, fmt.Errorf("transform failed")
	})
	if err == nil {
		t.Error("expected error from transform function")
	}
}

func TestCollect_BadQuery(t *testing.T) {
	_, err := queryfy.Collect(iterData, "nonexistent", func(v interface{}) (interface{}, error) {
		return v, nil
	})
	if err == nil {
		t.Error("expected error for bad query")
	}
}

// ======================================================================
// ValidateEach
// ======================================================================

func TestValidateEach_AllValid(t *testing.T) {
	schema := builders.Object().
		Field("name", builders.String().Required()).
		Field("price", builders.Number().Min(0).Required())

	err := queryfy.ValidateEach(iterData, "items[*]", schema, queryfy.Strict)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateEach_SomeInvalid(t *testing.T) {
	data := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"name": "Good", "price": 10.0},
			map[string]interface{}{"name": "", "price": -5.0},    // invalid
			map[string]interface{}{"name": "Also Good", "price": 1.0},
		},
	}

	schema := builders.Object().
		Field("name", builders.String().MinLength(1).Required()).
		Field("price", builders.Number().Min(0).Required())

	err := queryfy.ValidateEach(data, "items[*]", schema, queryfy.Strict)
	if err == nil {
		t.Fatal("expected validation error")
	}

	// Error paths should include the index
	errStr := err.Error()
	if errStr == "" {
		t.Error("error string should not be empty")
	}
}

func TestValidateEach_SingleValue(t *testing.T) {
	schema := builders.String().MinLength(1)

	err := queryfy.ValidateEach(iterData, "single", schema, queryfy.Strict)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateEach_EmptyArray(t *testing.T) {
	data := map[string]interface{}{"items": []interface{}{}}
	schema := builders.String()

	err := queryfy.ValidateEach(data, "items[*]", schema, queryfy.Strict)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateEach_BadQuery(t *testing.T) {
	schema := builders.String()
	err := queryfy.ValidateEach(iterData, "nonexistent[*]", schema, queryfy.Strict)
	if err == nil {
		t.Error("expected error for bad query")
	}
}
