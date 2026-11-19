package importers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

// importersByName holds every registered importer by its Name.
var importersByName = make(map[string]Importer)

// Register adds an importer to the registry
func Register(im Importer) {
	for _, ext := range im.Extensions() {
		importers[ext] = im
	}
	importersByName[im.Name()] = im
}

// GetByName returns the importer with the given name (openapi, postman, ...).
func GetByName(name string) Importer {
	return importersByName[strings.ToLower(strings.TrimSpace(name))]
}

// Names returns the names of all registered importers, sorted.
func Names() []string {
	names := make([]string, 0, len(importersByName))
	for name := range importersByName {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetByExtension returns an importer by file extension
func GetByExtension(ext string) Importer {
	return importers[ext]
}

// Import processes a file and returns converted requests. Several formats
// share the .json and .yaml extensions, so the format is detected from the
// file content (see Detect).
func Import(path string) ([]*types.SavedRequest, error) {
	importer, err := Detect(path)
	if err != nil {
		return nil, err
	}
	return importer.Parse(path)
}

// ImportAs parses path with the named importer, skipping detection. An empty
// format behaves like Import.
func ImportAs(path, format string) ([]*types.SavedRequest, error) {
	return ImportWithOptions(path, Options{Format: format})
}

// Options adjust an import.
type Options struct {
	// Format names the importer to use; empty means detect it.
	Format string
	// BaseURL replaces the server URL of an OpenAPI spec (only OpenAPI
	// imports have one), for example "{{BASE_URL}}".
	BaseURL string
}

// ImportWithOptions parses path with the importer picked by opts.Format or
// detected from the file, applying opts.
func ImportWithOptions(path string, opts Options) ([]*types.SavedRequest, error) {
	var importer Importer
	if strings.TrimSpace(opts.Format) != "" {
		importer = GetByName(opts.Format)
		if importer == nil {
			return nil, fmt.Errorf("unknown import format %q (supported: %s)", opts.Format, strings.Join(Names(), ", "))
		}
	} else {
		detected, err := Detect(path)
		if err != nil {
			return nil, err
		}
		importer = detected
	}
	if opts.BaseURL != "" {
		if _, ok := importer.(*OpenAPIImporter); !ok {
			return nil, fmt.Errorf("--base-url applies to OpenAPI imports only, not %s", importer.Name())
		}
		importer = &OpenAPIImporter{BaseURL: opts.BaseURL}
	}
	return importer.Parse(path)
}

// Detect picks the importer for path. An extension used by one format (.bru,
// .har, .gurl) decides on its own; for .json and .yaml the top-level keys of
// the document decide, because OpenAPI, Postman, Insomnia, HAR and Yaak
// exports all use .json.
func Detect(path string) (Importer, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return nil, fmt.Errorf("no file extension found")
	}
	switch ext {
	case ".json":
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read file: %w", err)
		}
		name := detectJSONFormat(stripBOM(data))
		if name == "" {
			return nil, fmt.Errorf("could not detect the format of %s; pass --format (%s)", path, strings.Join(Names(), ", "))
		}
		return GetByName(name), nil
	case ".yaml", ".yml":
		return GetByName("openapi"), nil
	}
	importer := GetByExtension(ext)
	if importer == nil {
		return nil, fmt.Errorf("unsupported file format: %s", ext)
	}
	return importer, nil
}

// detectJSONFormat returns the importer name for a JSON export, or "" when
// the document matches no known format.
func detectJSONFormat(data []byte) string {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		return ""
	}
	has := func(key string) bool { _, ok := doc[key]; return ok }
	switch {
	case has("openapi"):
		return "openapi"
	case has("log"):
		return "har"
	case has("resources") || has("_type"):
		return "insomnia"
	case has("item") && has("info"):
		return "postman"
	case has("collections"):
		return "yaak"
	}
	return ""
}

// OpWithMethod pairs an HTTP method with its operation
type OpWithMethod struct {
	Method string
	Op     *Operation
}

// AutoDetectImport discovers the appropriate importer based on file content/path
func AutoDetectImport(path string) ([]*types.SavedRequest, error) {
	return Import(path)
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
	Register(&GurlImporter{})
	Register(&YaakImporter{})
}

// Helper to read file content
func readFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	return data, nil
}
