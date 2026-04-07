package importers

import (
	"testing"

	"github.com/sreeram/gurl/pkg/types"
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
		name     string
		ext      string
		wantNil  bool
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
