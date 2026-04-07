package formatter

import (
	"testing"
)

func TestJSONPath_SimpleKey(t *testing.T) {
	input := `{"name": "Alice", "age": 30}`
	result, err := FilterJSON([]byte(input), "$.name")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != `"Alice"` {
		t.Fatalf("expected \"Alice\", got %s", result)
	}
}

func TestJSONPath_NestedPath(t *testing.T) {
	input := `{"data": {"users": [{"email": "alice@example.com"}]}}`
	result, err := FilterJSON([]byte(input), "$.data.users[0].email")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != `"alice@example.com"` {
		t.Fatalf("expected \"alice@example.com\", got %s", result)
	}
}

func TestJSONPath_ArrayIndex(t *testing.T) {
	input := `{"items": [1, 2, 3, 4, 5]}`
	result, err := FilterJSON([]byte(input), "$.items[0]")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != `1` {
		t.Fatalf("expected 1, got %s", result)
	}
}

func TestJSONPath_Wildcard(t *testing.T) {
	input := `{"users": [{"name": "Alice"}, {"name": "Bob"}, {"name": "Charlie"}]}`
	result, err := FilterJSON([]byte(input), "$.users[*]")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// Result should be array of all users
	if result == "" {
		t.Fatalf("expected non-empty result, got empty")
	}
}

func TestJSONPath_InvalidPath(t *testing.T) {
	input := `{"name": "Alice"}`
	_, err := FilterJSON([]byte(input), "$.name[")
	if err == nil {
		t.Fatalf("expected error for invalid path, got nil")
	}
}

func TestJSONPath_NoMatch(t *testing.T) {
	input := `{"name": "Alice"}`
	_, err := FilterJSON([]byte(input), "$.nonexistent")
	// PaesslerAG/jsonpath returns error for unknown keys
	if err == nil {
		t.Fatalf("expected error for no match, got nil")
	}
}

func TestXPath_SimpleElement(t *testing.T) {
	input := `<catalog><book><title>1984</title></book></catalog>`
	result, err := FilterXML([]byte(input), "//title")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == "" {
		t.Fatalf("expected non-empty result")
	}
}

func TestXPath_Attribute(t *testing.T) {
	input := `<catalog><book category="fiction"><title>1984</title></book><book category="tech"><title>Go Programming</title></book></catalog>`
	result, err := FilterXML([]byte(input), "//book[@category='fiction']")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == "" {
		t.Fatalf("expected non-empty result")
	}
}

func TestXPath_NoMatch(t *testing.T) {
	input := `<catalog><book><title>1984</title></book></catalog>`
	result, err := FilterXML([]byte(input), "//author")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != "" {
		t.Fatalf("expected empty result for no match, got %s", result)
	}
}
