package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDataDriven_CSV(t *testing.T) {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "users.csv")

	csvContent := "name,email,age\nAlice,alice@example.com,30\nBob,bob@example.com,25\nCharlie,charlie@example.com,35\n"

	if err := os.WriteFile(csvPath, []byte(csvContent), 0644); err != nil {
		t.Fatalf("failed to write CSV file: %v", err)
	}

	loader, err := NewDataLoader(csvPath)
	if err != nil {
		t.Fatalf("NewDataLoader failed: %v", err)
	}

	rows, err := loader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if len(rows) != 3 {
		t.Errorf("expected 3 rows, got %d", len(rows))
	}

	if rows[0]["name"] != "Alice" {
		t.Errorf("expected name=Alice, got %s", rows[0]["name"])
	}
	if rows[0]["email"] != "alice@example.com" {
		t.Errorf("expected email=alice@example.com, got %s", rows[0]["email"])
	}
	if rows[0]["age"] != "30" {
		t.Errorf("expected age=30, got %s", rows[0]["age"])
	}

	if rows[1]["name"] != "Bob" {
		t.Errorf("expected name=Bob, got %s", rows[1]["name"])
	}

	if rows[2]["name"] != "Charlie" {
		t.Errorf("expected name=Charlie, got %s", rows[2]["name"])
	}
}

func TestDataDriven_JSON(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "users.json")

	jsonContent := `[
		{"name": "Alice", "email": "alice@example.com", "age": 30},
		{"name": "Bob", "email": "bob@example.com", "age": 25},
		{"name": "Charlie", "email": "charlie@example.com", "age": 35}
	]`

	if err := os.WriteFile(jsonPath, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("failed to write JSON file: %v", err)
	}

	loader, err := NewDataLoader(jsonPath)
	if err != nil {
		t.Fatalf("NewDataLoader failed: %v", err)
	}

	rows, err := loader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if len(rows) != 3 {
		t.Errorf("expected 3 rows, got %d", len(rows))
	}

	if rows[0]["name"] != "Alice" {
		t.Errorf("expected name=Alice, got %s", rows[0]["name"])
	}
	if rows[0]["email"] != "alice@example.com" {
		t.Errorf("expected email=alice@example.com, got %s", rows[0]["email"])
	}
	if rows[0]["age"] != "30" {
		t.Errorf("expected age=30, got %s", rows[0]["age"])
	}

	if rows[1]["name"] != "Bob" {
		t.Errorf("expected name=Bob, got %s", rows[1]["name"])
	}
}

func TestDataDriven_CSVHeaders(t *testing.T) {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "headers.csv")

	csvContent := "name,email,role\nAlice,alice@example.com,admin\nBob,bob@example.com,user\n"

	if err := os.WriteFile(csvPath, []byte(csvContent), 0644); err != nil {
		t.Fatalf("failed to write CSV file: %v", err)
	}

	loader, err := NewDataLoader(csvPath)
	if err != nil {
		t.Fatalf("NewDataLoader failed: %v", err)
	}

	rows, err := loader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if len(rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(rows))
	}

	expectedHeaders := []string{"name", "email", "role"}
	headers := loader.Headers()

	if len(headers) != len(expectedHeaders) {
		t.Errorf("expected %d headers, got %d", len(expectedHeaders), len(headers))
	}

	for i, h := range expectedHeaders {
		if headers[i] != h {
			t.Errorf("expected header[%d]=%s, got %s", i, h, headers[i])
		}
	}
}

func TestDataDriven_VariableSubstitution(t *testing.T) {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "vars.csv")

	csvContent := "name,email\nAlice,alice@example.com\n"

	if err := os.WriteFile(csvPath, []byte(csvContent), 0644); err != nil {
		t.Fatalf("failed to write CSV file: %v", err)
	}

	loader, err := NewDataLoader(csvPath)
	if err != nil {
		t.Fatalf("NewDataLoader failed: %v", err)
	}

	rows, err := loader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	template := "User {{name}} with email {{email}}"
	expected := "User Alice with email alice@example.com"

	result := substituteTemplate(template, rows[0])
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestDataDriven_Iteration(t *testing.T) {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "iter.csv")

	csvContent := "name,value\nFirst,1\nSecond,2\nThird,3\n"

	if err := os.WriteFile(csvPath, []byte(csvContent), 0644); err != nil {
		t.Fatalf("failed to write CSV file: %v", err)
	}

	loader, err := NewDataLoader(csvPath)
	if err != nil {
		t.Fatalf("NewDataLoader failed: %v", err)
	}

	count := 0
	expectedValues := []string{"1", "2", "3"}

	err = loader.Iterate(func(row map[string]string) error {
		if row["name"] == "First" && row["value"] != expectedValues[0] {
			t.Errorf("row 1: expected value=%s, got %s", expectedValues[0], row["value"])
		}
		if row["name"] == "Second" && row["value"] != expectedValues[1] {
			t.Errorf("row 2: expected value=%s, got %s", expectedValues[1], row["value"])
		}
		if row["name"] == "Third" && row["value"] != expectedValues[2] {
			t.Errorf("row 3: expected value=%s, got %s", expectedValues[2], row["value"])
		}
		count++
		return nil
	})

	if err != nil {
		t.Fatalf("Iterate failed: %v", err)
	}

	if count != 3 {
		t.Errorf("expected 3 iterations, got %d", count)
	}
}

func TestDataDriven_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "empty.csv")

	if err := os.WriteFile(csvPath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write empty file: %v", err)
	}

	loader, err := NewDataLoader(csvPath)
	if err != nil {
		t.Fatalf("NewDataLoader failed: %v", err)
	}

	rows, err := loader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if len(rows) != 0 {
		t.Errorf("expected 0 rows for empty file, got %d", len(rows))
	}
}

func TestDataDriven_MissingColumn(t *testing.T) {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "missing.csv")

	csvContent := "name,email\nAlice,alice@example.com\n"

	if err := os.WriteFile(csvPath, []byte(csvContent), 0644); err != nil {
		t.Fatalf("failed to write CSV file: %v", err)
	}

	loader, err := NewDataLoader(csvPath)
	if err != nil {
		t.Fatalf("NewDataLoader failed: %v", err)
	}

	rows, err := loader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	template := "User {{name}} with phone {{phone}}"

	_, err = SubstituteTemplateWithVars(template, nil, rows[0])
	if err == nil {
		t.Error("expected error for missing column, got nil")
	}

	missingErr, ok := err.(*MissingColumnError)
	if !ok {
		t.Fatalf("expected *MissingColumnError, got %T", err)
	}

	if missingErr.Column != "phone" {
		t.Errorf("expected missing column 'phone', got %q", missingErr.Column)
	}
}
