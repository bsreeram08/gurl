package runner

import (
	"errors"
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

func TestNewDataLoader_UnsupportedFileType(t *testing.T) {
	_, err := NewDataLoader("/tmp/data.txt")
	if err == nil {
		t.Error("expected error for unsupported file type, got nil")
	}
	expected := "unsupported data file type: .txt (supported: .csv, .json)"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestNewDataLoader_CaseInsensitive(t *testing.T) {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "test.CSV")
	if err := os.WriteFile(csvPath, []byte("a,b\n1,2"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	_, err := NewDataLoader(csvPath)
	if err != nil {
		t.Fatalf("expected no error for .CSV extension, got %v", err)
	}

	jsonPath := filepath.Join(tmpDir, "test.JSON")
	if err := os.WriteFile(jsonPath, []byte(`[{"a":"1"}]`), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	_, err = NewDataLoader(jsonPath)
	if err != nil {
		t.Fatalf("expected no error for .JSON extension, got %v", err)
	}
}

func TestDataLoader_Headers(t *testing.T) {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "headers.csv")
	if err := os.WriteFile(csvPath, []byte("name,email\nAlice,alice@example.com"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	loader, err := NewDataLoader(csvPath)
	if err != nil {
		t.Fatalf("NewDataLoader failed: %v", err)
	}

	headers := loader.Headers()
	if len(headers) != 0 {
		t.Errorf("expected empty headers before read, got %d", len(headers))
	}

	_, err = loader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	headers = loader.Headers()
	if len(headers) != 2 {
		t.Errorf("expected 2 headers, got %d", len(headers))
	}
	if headers[0] != "name" || headers[1] != "email" {
		t.Errorf("unexpected headers: %v", headers)
	}
}

func TestDataLoader_ReadAll_UnsupportedType(t *testing.T) {
	loader := &DataLoader{fileType: "xml"}
	_, err := loader.ReadAll()
	if err == nil {
		t.Error("expected error for unsupported file type, got nil")
	}
}

func TestDataLoader_Iterate_UnsupportedType(t *testing.T) {
	loader := &DataLoader{fileType: "xml"}
	err := loader.Iterate(func(row map[string]string) error { return nil })
	if err == nil {
		t.Error("expected error for unsupported file type, got nil")
	}
}

func TestDataDriven_CSVEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "empty.csv")
	// Write just headers, no data rows
	if err := os.WriteFile(csvPath, []byte("name,email\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
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
		t.Errorf("expected 0 rows for header-only CSV, got %d", len(rows))
	}
}

func TestDataDriven_CSVFileNotFound(t *testing.T) {
	loader, err := NewDataLoader("/nonexistent/path/data.csv")
	if err != nil {
		t.Fatalf("NewDataLoader should not fail for nonexistent file: %v", err)
	}

	_, err = loader.ReadAll()
	if err == nil {
		t.Error("expected error reading nonexistent file, got nil")
	}
}

func TestDataDriven_CSVBadFormat(t *testing.T) {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "bad.csv")
	// Write an unquoted CSV with quotes that will cause parse error
	if err := os.WriteFile(csvPath, []byte("name\n\"unclosed"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	loader, err := NewDataLoader(csvPath)
	if err != nil {
		t.Fatalf("NewDataLoader failed: %v", err)
	}

	_, err = loader.ReadAll()
	if err == nil {
		t.Error("expected error for malformed CSV, got nil")
	}
}

func TestDataDriven_IterateCSVFileNotFound(t *testing.T) {
	loader, err := NewDataLoader("/nonexistent/path/data.csv")
	if err != nil {
		t.Fatalf("NewDataLoader should not fail for nonexistent file: %v", err)
	}

	err = loader.Iterate(func(row map[string]string) error { return nil })
	if err == nil {
		t.Error("expected error iterating nonexistent file, got nil")
	}
}

func TestDataDriven_IterateCSVWithIteratorError(t *testing.T) {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "iter_error.csv")
	if err := os.WriteFile(csvPath, []byte("name,value\nAlice,1\nBob,2"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	loader, err := NewDataLoader(csvPath)
	if err != nil {
		t.Fatalf("NewDataLoader failed: %v", err)
	}

	expectedErr := errors.New("iterator stopped")
	err = loader.Iterate(func(row map[string]string) error {
		if row["name"] == "Bob" {
			return expectedErr
		}
		return nil
	})

	if err == nil {
		t.Fatal("expected error from iterator, got nil")
	}
}

func TestDataDriven_JSONFileNotFound(t *testing.T) {
	loader, err := NewDataLoader("/nonexistent/path/data.json")
	if err != nil {
		t.Fatalf("NewDataLoader should not fail for nonexistent file: %v", err)
	}

	_, err = loader.ReadAll()
	if err == nil {
		t.Error("expected error reading nonexistent file, got nil")
	}
}

func TestDataDriven_JSONBadFormat(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "bad.json")
	if err := os.WriteFile(jsonPath, []byte(`[{"name":}`), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	loader, err := NewDataLoader(jsonPath)
	if err != nil {
		t.Fatalf("NewDataLoader failed: %v", err)
	}

	_, err = loader.ReadAll()
	if err == nil {
		t.Error("expected error for malformed JSON, got nil")
	}
}

func TestDataDriven_IterateJSONFileNotFound(t *testing.T) {
	loader, err := NewDataLoader("/nonexistent/path/data.json")
	if err != nil {
		t.Fatalf("NewDataLoader should not fail for nonexistent file: %v", err)
	}

	err = loader.Iterate(func(row map[string]string) error { return nil })
	if err == nil {
		t.Error("expected error iterating nonexistent file, got nil")
	}
}

func TestDataDriven_IterateJSONWithIteratorError(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "iter_error.json")
	if err := os.WriteFile(jsonPath, []byte(`[{"name":"Alice"},{"name":"Bob"}]`), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	loader, err := NewDataLoader(jsonPath)
	if err != nil {
		t.Fatalf("NewDataLoader failed: %v", err)
	}

	expectedErr := errors.New("iterator stopped")
	err = loader.Iterate(func(row map[string]string) error {
		if row["name"] == "Bob" {
			return expectedErr
		}
		return nil
	})

	if err == nil {
		t.Fatal("expected error from iterator, got nil")
	}
}

func TestExtractTemplateVars(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"no vars here", nil},
		{"{{name}}", []string{"name"}},
		{"{{name}} and {{email}}", []string{"name", "email"}},
		{"{{a}}{{b}}{{c}}", []string{"a", "b", "c"}},
		{"prefix {{name}} suffix", []string{"name"}},
		{"{{na}} and {{na}}", []string{"na", "na"}},
		{"unclosed {{name", nil},
		{"{{name}} extra {{email", []string{"name"}},
	}

	for _, tt := range tests {
		result := extractTemplateVars(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("extractTemplateVars(%q): expected %v, got %v", tt.input, tt.expected, result)
			continue
		}
		for i, v := range result {
			if v != tt.expected[i] {
				t.Errorf("extractTemplateVars(%q): expected %v, got %v", tt.input, tt.expected, result)
				break
			}
		}
	}
}

func TestMissingColumnError_Error(t *testing.T) {
	err := &MissingColumnError{Column: "missing_col", Row: 5}
	expected := "missing column 'missing_col' in data row 5"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestSubstituteTemplateWithVars_MissingColumn(t *testing.T) {
	template := "User {{name}} with phone {{phone}}"
	baseVars := map[string]string{"name": "Alice"}
	rowVars := map[string]string{}

	_, err := SubstituteTemplateWithVars(template, baseVars, rowVars)
	if err == nil {
		t.Fatal("expected error for missing column, got nil")
	}

	missingErr, ok := err.(*MissingColumnError)
	if !ok {
		t.Fatalf("expected *MissingColumnError, got %T", err)
	}

	if missingErr.Column != "phone" {
		t.Errorf("expected missing column 'phone', got %q", missingErr.Column)
	}
	if missingErr.Row != 0 {
		t.Errorf("expected row 0, got %d", missingErr.Row)
	}
}

func TestSubstituteTemplateWithVars_BothBaseAndRow(t *testing.T) {
	template := "User {{name}} at {{domain}}"
	baseVars := map[string]string{"domain": "example.com"}
	rowVars := map[string]string{"name": "Alice"}

	result, err := SubstituteTemplateWithVars(template, baseVars, rowVars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "User Alice at example.com"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSubstituteTemplateWithVars_RowOverridesBase(t *testing.T) {
	template := "Value: {{val}}"
	baseVars := map[string]string{"val": "base"}
	rowVars := map[string]string{"val": "row"}

	result, err := SubstituteTemplateWithVars(template, baseVars, rowVars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Value: row"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
