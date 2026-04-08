package codegen

import (
	"fmt"

	"github.com/sreeram/gurl/pkg/types"
)

// Generator interface for code generation
type Generator interface {
	Language() string
	Generate(req *types.SavedRequest, opts *GenOptions) (string, error)
}

// GenOptions controls code generation output
type GenOptions struct {
	IncludeComments bool
	IncludeImports  bool
}

// generators holds all registered generators
var generators = map[string]Generator{
	"go":         &GoGenerator{},
	"python":     &PythonGenerator{},
	"javascript": &JavaScriptGenerator{},
	"curl":       &CurlGenerator{},
}

// Generate generates code for the specified language from a SavedRequest
func Generate(lang string, req *types.SavedRequest, opts *GenOptions) (string, error) {
	if opts == nil {
		opts = &GenOptions{}
	}

	switch lang {
	case "go":
		return generators["go"].Generate(req, opts)
	case "python":
		return generators["python"].Generate(req, opts)
	case "javascript":
		return generators["javascript"].Generate(req, opts)
	case "curl":
		return generators["curl"].Generate(req, opts)
	default:
		return "", fmt.Errorf("unsupported language '%s', available: go, python, javascript, curl", lang)
	}
}

// ListLanguages returns all supported languages
func ListLanguages() []string {
	return []string{"go", "python", "javascript", "curl"}
}
