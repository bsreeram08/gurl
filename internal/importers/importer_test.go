package importers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bsreeram08/apigent/pkg/types"
)

// TestImporterRegistry tests the importer registry functionality
func TestImporterRegistry(t *testing.T) {
	// Test that built-in importers are registered
	importers := ListSupported()

	if len(importers) == 0 {
		t.Error("expected importers to be registered")
	}

	// Verify OpenAPI is registered
	found := false
	for _, ext := range importers {
		if ext == ".yaml" || ext == ".yml" || ext == ".json" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected yaml/json extensions to be registered")
	}
}

// TestGetByExtension tests retrieving importers by file extension
func TestGetByExtension(t *testing.T) {
	tests := []struct {
		name    string
		ext     string
		wantNil bool
	}{
		{
			name:    "yaml extension",
			ext:     ".yaml",
			wantNil: false,
		},
		{
			name:    "yml extension",
			ext:     ".yml",
			wantNil: false,
		},
		{
			name:    "json extension",
			ext:     ".json",
			wantNil: false,
		},
		{
			name:    "unknown extension",
			ext:     ".txt",
			wantNil: true,
		},
		{
			name:    "empty extension",
			ext:     "",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			importer := GetByExtension(tt.ext)
			if tt.wantNil && importer != nil {
				t.Errorf("expected nil importer for %q", tt.ext)
			}
			if !tt.wantNil && importer == nil {
				t.Errorf("expected non-nil importer for %q", tt.ext)
			}
		})
	}
}

// TestIsSupported tests checking if an extension is supported
func TestIsSupported(t *testing.T) {
	tests := []struct {
		name string
		ext  string
		want bool
	}{
		{
			name: "supported yaml",
			ext:  ".yaml",
			want: true,
		},
		{
			name: "supported yml",
			ext:  ".yml",
			want: true,
		},
		{
			name: "supported json",
			ext:  ".json",
			want: true,
		},
		{
			name: "unsupported txt",
			ext:  ".txt",
			want: false,
		},
		{
			name: "unsupported md",
			ext:  ".md",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSupported(tt.ext)
			if got != tt.want {
				t.Errorf("IsSupported(%q) = %v, want %v", tt.ext, got, tt.want)
			}
		})
	}
}

// TestFormatFromExtension tests format detection from extension
func TestFormatFromExtension(t *testing.T) {
	tests := []struct {
		name string
		ext  string
		want string
	}{
		{
			name: "yaml format",
			ext:  ".yaml",
			want: "yaml",
		},
		{
			name: "yml format",
			ext:  ".yml",
			want: "yaml",
		},
		{
			name: "json format",
			ext:  ".json",
			want: "json",
		},
		{
			name: "har format",
			ext:  ".har",
			want: "har",
		},
		{
			name: "bru format",
			ext:  ".bru",
			want: "bru",
		},
		{
			name: "unknown format",
			ext:  ".xyz",
			want: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatFromExtension(tt.ext)
			if got != tt.want {
				t.Errorf("FormatFromExtension(%q) = %v, want %v", tt.ext, got, tt.want)
			}
		})
	}
}

// TestListSupported tests listing all supported extensions
func TestListSupported(t *testing.T) {
	exts := ListSupported()
	if len(exts) == 0 {
		t.Error("expected at least one supported extension")
	}

	// Check for duplicates
	seen := make(map[string]bool)
	for _, ext := range exts {
		if seen[ext] {
			t.Errorf("duplicate extension found: %s", ext)
		}
		seen[ext] = true
	}
}

// TestCustomImporterRegistration tests registering a custom importer
func TestCustomImporterRegistration(t *testing.T) {
	// Create a mock importer
	mock := &mockImporter{}

	Register(mock)

	// Verify it was registered
	importer := GetByExtension(".mock")
	if importer == nil {
		t.Error("expected custom importer to be registered")
	}

	if importer.Name() != "mock" {
		t.Errorf("got name %q, want %q", importer.Name(), "mock")
	}
}

// mockImporter for testing
type mockImporter struct{}

func (m *mockImporter) Name() string {
	return "mock"
}

func (m *mockImporter) Extensions() []string {
	return []string{".mock"}
}

func (m *mockImporter) Parse(path string) ([]*types.SavedRequest, error) {
	return []*types.SavedRequest{}, nil
}

// TestImportWithEmptyPath tests import with empty path
func TestImportWithEmptyPath(t *testing.T) {
	_, err := Import("")
	if err == nil {
		t.Error("expected error for empty path")
	}
}

func TestImportUsesNormalizedExtension(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "requests.gurl")
	data := []byte(`{
  "version": "1.0",
  "requests": [
    {
      "id": "req-1",
      "name": "health",
      "url": "https://example.com/health"
    }
  ]
}`)
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		t.Fatalf("failed to write import file: %v", err)
	}

	requests, err := Import(tmpFile)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if len(requests) != 1 || requests[0].Name != "health" {
		t.Fatalf("expected native gurl request export, got %+v", requests)
	}
}

// TestAutoDetectImportWithNoExtension tests auto detection with no extension
func TestAutoDetectImportWithNoExtension(t *testing.T) {
	_, err := AutoDetectImport("/path/to/file")
	if err == nil {
		t.Error("expected error for file without extension")
	}
}

// TestAutoDetectImportWithUnsupported tests auto detection with unsupported format
func TestAutoDetectImportWithUnsupported(t *testing.T) {
	_, err := AutoDetectImport("/path/to/file.unsupported")
	if err == nil {
		t.Error("expected error for unsupported format")
	}
}

// TestDetectPicksImporterFromContent covers the formats that share .json:
// before detection, every .json file went to whichever importer registered
// the extension last.
func TestDetectPicksImporterFromContent(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		content string
		want    string
	}{
		{"openapi json", "spec.json", `{"openapi":"3.1.0","info":{"title":"x"},"paths":{}}`, "openapi"},
		{"openapi yaml", "spec.yaml", "openapi: 3.0.0\npaths: {}\n", "openapi"},
		{"postman", "coll.json", `{"info":{"name":"x","schema":"https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},"item":[]}`, "postman"},
		{"insomnia", "export.json", `{"_type":"export","resources":[]}`, "insomnia"},
		{"har", "trace.json", `{"log":{"version":"1.2","entries":[]}}`, "har"},
		{"yaak", "yaak.json", `{"version":"1","collections":[]}`, "yaak"},
		{"bruno by extension", "req.bru", "meta {\n  name: x\n}\n", "bruno"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), tt.file)
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}
			im, err := Detect(path)
			if err != nil {
				t.Fatalf("Detect: %v", err)
			}
			if im.Name() != tt.want {
				t.Errorf("Detect(%s) = %s, want %s", tt.file, im.Name(), tt.want)
			}
		})
	}
}

func TestDetectRejectsUnknownJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.json")
	if err := os.WriteFile(path, []byte(`{"hello":"world"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Detect(path); err == nil {
		t.Fatal("expected an error for a JSON document of unknown format")
	}
}

func TestImportOpenAPIJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "spec.json")
	spec := `{"openapi":"3.1.0","info":{"title":"Pets"},"paths":{"/pets":{"get":{"operationId":"listPets"}}}}`
	if err := os.WriteFile(path, []byte(spec), 0644); err != nil {
		t.Fatal(err)
	}
	reqs, err := Import(path)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(reqs) != 1 || reqs[0].Name != "listPets" {
		t.Fatalf("Import returned %+v, want the listPets operation", reqs)
	}
}

func TestImportAsUsesNamedFormat(t *testing.T) {
	path := filepath.Join(t.TempDir(), "spec.txt.json")
	if err := os.WriteFile(path, []byte(`{"hello":"world"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ImportAs(path, "postman"); err != nil {
		t.Fatalf("ImportAs postman: %v", err)
	}
	if _, err := ImportAs(path, "nope"); err == nil {
		t.Fatal("expected an error for an unknown format")
	}
}
