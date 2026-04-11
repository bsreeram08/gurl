package codegen

import (
	"fmt"
	"strings"

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
	IncludeImports   bool
	AsyncPython      bool // For Python: generate async httpx code instead of sync requests
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

	// Check for unsupported protocols
	if isUnsupportedProtocol(req) {
		return "", fmt.Errorf("Protocol not supported for codegen: %s", getProtocol(req))
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

// isUnsupportedProtocol checks if the request uses a protocol not supported for codegen
func isUnsupportedProtocol(req *types.SavedRequest) bool {
	protocol := getProtocol(req)
	return protocol == "grpc" || protocol == "websocket" || protocol == "sse"
}

// getProtocol determines the protocol from the URL or request settings
func getProtocol(req *types.SavedRequest) string {
	url := strings.ToLower(req.URL)
	if strings.HasPrefix(url, "grpc://") || strings.HasPrefix(url, "grpcs://") {
		return "grpc"
	}
	if strings.HasPrefix(url, "ws://") || strings.HasPrefix(url, "wss://") {
		return "websocket"
	}
	if strings.HasPrefix(url, "sse://") || strings.HasPrefix(url, "eventsource://") {
		return "sse"
	}
	return "http"
}

// ListLanguages returns all supported languages
func ListLanguages() []string {
	return []string{"go", "python", "javascript", "curl"}
}

// substituteVariables replaces {{var}} patterns with values from the vars map
func substituteVariables(input string, vars []types.Var) string {
	result := input
	for _, v := range vars {
		pattern := "{{" + v.Name + "}}"
		result = strings.ReplaceAll(result, pattern, v.Example)
	}
	return result
}

// isBinaryContentType checks if the content type indicates binary data
func isBinaryContentType(contentType string) bool {
	ct := strings.ToLower(contentType)
	binaryPrefixes := []string{
		"application/octet-stream",
		"image/",
		"audio/",
		"video/",
		"application/pdf",
		"application/zip",
		"application/gzip",
		"application/x-tar",
		"application/x-rar-compressed",
	}
	for _, prefix := range binaryPrefixes {
		if strings.HasPrefix(ct, prefix) {
			return true
		}
	}
	return false
}

// isMultipartContentType checks if the content type indicates multipart form data
func isMultipartContentType(contentType string) bool {
	return strings.HasPrefix(strings.ToLower(contentType), "multipart/")
}

// getContentTypeFromHeaders extracts the content type from headers
func getContentTypeFromHeaders(headers []types.Header) string {
	for _, h := range headers {
		if strings.ToLower(h.Key) == "content-type" {
			return h.Value
		}
	}
	return ""
}
