package importers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sreeram/gurl/pkg/types"
)

// Importer interface - deterministic, no if-else chains
type Importer interface {
	Name() string
	Extensions() []string
	Parse(path string) ([]*types.SavedRequest, error)
}

// Registry of importers
var importers = make(map[string]Importer)

// Register adds an importer to the registry
func Register(im Importer) {
	for _, ext := range im.Extensions() {
		importers[ext] = im
	}
}

// GetByExtension returns an importer by file extension
func GetByExtension(ext string) Importer {
	return importers[ext]
}

// Import processes a file and returns converted requests
func Import(path string) ([]*types.SavedRequest, error) {
	ext := filepath.Ext(path)
	if ext == "" {
		return nil, fmt.Errorf("no file extension found")
	}

	// Normalize extension (remove leading dot and lowercase)
	ext = "." + ext
	extLower := ext

	importer := GetByExtension(extLower)
	if importer == nil {
		return nil, fmt.Errorf("unsupported file format: %s", ext)
	}

	return importer.Parse(path)
}

// OpWithMethod pairs an HTTP method with its operation
type OpWithMethod struct {
	Method string
	Op     *Operation
}

// AutoDetectImport discovers the appropriate importer based on file content/path
func AutoDetectImport(path string) ([]*types.SavedRequest, error) {
	ext := filepath.Ext(path)
	if ext == "" {
		return nil, fmt.Errorf("no file extension found")
	}

	// Normalize extension
	ext = strings.ToLower(ext)
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	// Deterministic lookup - importers map is keyed by extension
	im := importers[ext]
	if im == nil {
		return nil, fmt.Errorf("unsupported file format: %s", ext)
	}

	return im.Parse(path)
}

// FormatFromExtension returns the format name for an extension
func FormatFromExtension(ext string) string {
	switch ext {
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	case ".har":
		return "har"
	case ".bru":
		return "bru"
	default:
		return "unknown"
	}
}

// IsSupported checks if a file extension is supported
func IsSupported(ext string) bool {
	_, ok := importers[ext]
	return ok
}

// ListSupported returns all supported extensions
func ListSupported() []string {
	var exts []string
	for ext := range importers {
		exts = append(exts, ext)
	}
	return exts
}

// init registers all built-in importers
func init() {
	Register(&OpenAPIImporter{})
	Register(&InsomniaImporter{})
	Register(&BrunoImporter{})
	Register(&PostmanImporter{})
	Register(&HARImporter{})
}

// Helper to read file content
func readFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	return data, nil
}
