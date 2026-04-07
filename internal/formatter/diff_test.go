package formatter

import (
	"testing"

	"github.com/sreeram/gurl/pkg/types"
)

func TestDiffJSON_FieldAdded(t *testing.T) {
	a := []byte(`{"name":"test","value":"old"}`)
	b := []byte(`{"name":"test","value":"old","newField":"added"}`)

	result, err := DiffJSON(a, b)
	if err != nil {
		t.Fatalf("DiffJSON returned error: %v", err)
	}

	// Should show newField as added (green indicator)
	if result == "" {
		t.Fatal("DiffJSON returned empty result")
	}
	// Result should contain indication of the added field
	if !containsSubstring(result, "newField") {
		t.Errorf("expected result to mention newField, got: %s", result)
	}
}

func TestDiffJSON_FieldRemoved(t *testing.T) {
	a := []byte(`{"name":"test","value":"old","extra":"removed"}`)
	b := []byte(`{"name":"test","value":"old"}`)

	result, err := DiffJSON(a, b)
	if err != nil {
		t.Fatalf("DiffJSON returned error: %v", err)
	}

	if result == "" {
		t.Fatal("DiffJSON returned empty result")
	}
	// Should mention the removed field
	if !containsSubstring(result, "extra") {
		t.Errorf("expected result to mention removed field 'extra', got: %s", result)
	}
}

func TestDiffJSON_FieldChanged(t *testing.T) {
	a := []byte(`{"name":"test","status":"old"}`)
	b := []byte(`{"name":"test","status":"new"}`)

	result, err := DiffJSON(a, b)
	if err != nil {
		t.Fatalf("DiffJSON returned error: %v", err)
	}

	if result == "" {
		t.Fatal("DiffJSON returned empty result")
	}
	if !containsSubstring(result, "status") {
		t.Errorf("expected result to mention status field, got: %s", stripANSI(result))
	}
}

func TestDiffJSON_DeepNested(t *testing.T) {
	a := []byte(`{"user":{"profile":{"name":"alice","age":30}}}`)
	b := []byte(`{"user":{"profile":{"name":"alice","age":31}}}`)

	result, err := DiffJSON(a, b)
	if err != nil {
		t.Fatalf("DiffJSON returned error: %v", err)
	}

	if result == "" {
		t.Fatal("DiffJSON returned empty result")
	}
	// Should detect the nested change
	if !containsSubstring(result, "age") {
		t.Errorf("expected result to mention nested 'age' field, got: %s", result)
	}
}

func TestDiffJSON_ArrayReorder(t *testing.T) {
	a := []byte(`{"items":["a","b","c"]}`)
	b := []byte(`{"items":["a","c","b"]}`)

	result, err := DiffJSON(a, b)
	if err != nil {
		t.Fatalf("DiffJSON returned error: %v", err)
	}

	if result == "" {
		t.Fatal("DiffJSON returned empty result")
	}
	// Should detect array change
	if !containsSubstring(result, "items") {
		t.Errorf("expected result to mention 'items' array, got: %s", result)
	}
}

func TestDiffText_LineDiff(t *testing.T) {
	a := []byte("line1\nline2\nline3\n")
	b := []byte("line1\nline2 modified\nline3\n")

	result := DiffText(a, b)
	if result == "" {
		t.Fatal("DiffText returned empty result")
	}
	// Should show unified diff format
	if !containsSubstring(result, "line2") {
		t.Errorf("expected result to show line2 diff, got: %s", result)
	}
}

func TestDiffIdentical(t *testing.T) {
	a := []byte(`{"name":"test","value":"same"}`)
	b := []byte(`{"name":"test","value":"same"}`)

	result, err := DiffJSON(a, b)
	if err != nil {
		t.Fatalf("DiffJSON returned error: %v", err)
	}

	// Identical content should return "no differences" message
	if !containsSubstring(result, "no diff") && !containsSubstring(result, "identical") && !containsSubstring(result, "same") {
		t.Errorf("expected 'no differences' message for identical content, got: %s", result)
	}
}

func TestDiffResponses(t *testing.T) {
	histA := types.ExecutionHistory{
		ID:         "id-a",
		RequestID:  "req-1",
		Response:   `{"status":"ok","count":5}`,
		StatusCode: 200,
	}
	histB := types.ExecutionHistory{
		ID:         "id-b",
		RequestID:  "req-1",
		Response:   `{"status":"ok","count":10}`,
		StatusCode: 200,
	}

	result, err := DiffResponses(histA, histB)
	if err != nil {
		t.Fatalf("DiffResponses returned error: %v", err)
	}

	if result == "" {
		t.Fatal("DiffResponses returned empty result")
	}
	// Should show the diff between responses
	if !containsSubstring(result, "count") {
		t.Errorf("expected result to mention 'count' field diff, got: %s", result)
	}
}

func stripANSI(s string) string {
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\033' && i+1 < len(s) && s[i+1] == '[' {
			i += 2
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		result = append(result, s[i])
	}
	return string(result)
}

func TestDiffJSON_InvalidJSON(t *testing.T) {
	a := []byte(`not json`)
	b := []byte(`{"valid":true}`)

	_, err := DiffJSON(a, b)
	if err == nil {
		t.Error("expected error for invalid JSON input")
	}
}

func TestDiffText_Identical(t *testing.T) {
	a := []byte("same content\n")
	b := []byte("same content\n")

	result := DiffText(a, b)
	// Should indicate no differences
	if !containsSubstring(result, "no diff") && !containsSubstring(result, "identical") && result == "" {
		t.Errorf("expected no differences indicator, got: %s", result)
	}
}

// containsSubstring is a simple helper to check if a string contains a substring
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
