package formatter

import (
	"strings"
	"testing"
)

func TestFormatTableSingleObject(t *testing.T) {
	data := map[string]interface{}{
		"status":  200,
		"content": "application/json",
	}

	result := FormatTable(data)

	if result == "" {
		t.Fatal("expected non-empty result for single object")
	}

	if !containsString(result, "status") {
		t.Error("expected table to contain 'status' key")
	}
	if !containsString(result, "200") {
		t.Error("expected table to contain '200' value")
	}
}

func TestFormatTableArrayOfObjects(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{"id": float64(1), "name": "Alice", "email": "alice@example.com"},
		map[string]interface{}{"id": float64(2), "name": "Bob", "email": "bob@example.com"},
	}

	result := FormatTable(data)

	if result == "" {
		t.Fatal("expected non-empty result for array of objects")
	}

	if !containsString(result, "id") {
		t.Error("expected table to contain 'id' column")
	}
	if !containsString(result, "name") {
		t.Error("expected table to contain 'name' column")
	}
	if !containsString(result, "Alice") {
		t.Error("expected table to contain 'Alice' value")
	}
	if !containsString(result, "Bob") {
		t.Error("expected table to contain 'Bob' value")
	}
}

func TestFormatTableEmptyArray(t *testing.T) {
	data := []interface{}{}

	result := FormatTable(data)

	if result != "" {
		t.Errorf("expected empty string for empty array, got %q", result)
	}
}

func TestFormatTableNonJSONPassthrough(t *testing.T) {
	data := "not json data"

	result := FormatTable(data)

	if result != "" {
		t.Errorf("expected empty string for non-JSON interface{}, got %q", result)
	}
}

func TestFormatTableFromBytesSingleObject(t *testing.T) {
	data := []byte(`{"status": 200, "content": "application/json"}`)

	result := FormatTableFromBytes(data)

	if result == "" {
		t.Fatal("expected non-empty result for valid JSON object")
	}

	if !containsString(result, "status") {
		t.Error("expected table to contain 'status' key")
	}
}

func TestFormatTableFromBytesArray(t *testing.T) {
	data := []byte(`[{"id": 1, "name": "Alice"}, {"id": 2, "name": "Bob"}]`)

	result := FormatTableFromBytes(data)

	if result == "" {
		t.Fatal("expected non-empty result for valid JSON array")
	}

	if !containsString(result, "id") {
		t.Error("expected table to contain 'id' column")
	}
	if !containsString(result, "Alice") {
		t.Error("expected table to contain 'Alice' value")
	}
}

func TestFormatTableFromBytesEmptyArray(t *testing.T) {
	data := []byte(`[]`)

	result := FormatTableFromBytes(data)

	if result != "" {
		t.Errorf("expected empty string for empty array, got %q", result)
	}
}

func TestFormatTableFromBytesNonJSONPassthrough(t *testing.T) {
	data := []byte(`not json at all`)

	result := FormatTableFromBytes(data)

	if result != "" {
		t.Errorf("expected empty string for non-JSON bytes, got %q", result)
	}
}

func TestFormatTableFromBytesInvalidJSON(t *testing.T) {
	data := []byte(`{"broken json`)

	result := FormatTableFromBytes(data)

	if result != "" {
		t.Errorf("expected empty string for invalid JSON, got %q", result)
	}
}

func TestFormatTableFromBytesEmpty(t *testing.T) {
	data := []byte{}

	result := FormatTableFromBytes(data)

	if result != "" {
		t.Errorf("expected empty string for empty bytes, got %q", result)
	}
}

func TestFormatTableNilData(t *testing.T) {
	result := FormatTable(nil)

	if result != "" {
		t.Errorf("expected empty string for nil data, got %q", result)
	}
}

func TestFormatTableNestedValues(t *testing.T) {
	data := map[string]interface{}{
		"user": map[string]interface{}{
			"name": "Alice",
			"age":  float64(30),
		},
		"active": true,
	}

	result := FormatTable(data)

	if result == "" {
		t.Fatal("expected non-empty result")
	}

	if !containsString(result, "user") {
		t.Error("expected table to contain 'user' key")
	}
	if !containsString(result, "active") {
		t.Error("expected table to contain 'active' key")
	}
}

func TestFormatTableNumericValues(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{"id": float64(1), "score": float64(95.5)},
		map[string]interface{}{"id": float64(2), "score": float64(87.25)},
	}

	result := FormatTable(data)

	if result == "" {
		t.Fatal("expected non-empty result")
	}

	if !containsString(result, "1") {
		t.Error("expected table to contain integer 1")
	}
	if !containsString(result, "95.5") && !containsString(result, "95") {
		t.Error("expected table to contain score value")
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && strings.Contains(s, substr))
}
