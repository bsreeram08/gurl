package importers

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sreeram/gurl/pkg/types"
	"gopkg.in/yaml.v3"
)

// OpenAPIImporter handles OpenAPI 3.x specification files
type OpenAPIImporter struct{}

// Name returns the importer name
func (o *OpenAPIImporter) Name() string {
	return "openapi"
}

// Extensions returns supported file extensions
func (o *OpenAPIImporter) Extensions() []string {
	return []string{".yaml", ".yml", ".json"}
}

// OpenAPISpec represents the structure of an OpenAPI 3.x spec
type OpenAPISpec struct {
	OpenAPI    string                 `json:"openapi" yaml:"openapi"`
	Info       OpenAPIInfo            `json:"info" yaml:"info"`
	Paths      map[string]PathItem    `json:"paths" yaml:"paths"`
	Components Components             `json:"components" yaml:"components"`
	Tags       []Tag                  `json:"tags" yaml:"tags"`
}

// OpenAPIInfo contains API metadata
type OpenAPIInfo struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`
	Version     string `json:"version" yaml:"version"`
}

// PathItem represents a path and its operations
type PathItem struct {
	Ref     string      `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Summary string      `json:"summary,omitempty" yaml:"summary,omitempty"`
	Get     *Operation  `json:"get,omitempty" yaml:"get,omitempty"`
	Post    *Operation  `json:"post,omitempty" yaml:"post,omitempty"`
	Put     *Operation  `json:"put,omitempty" yaml:"put,omitempty"`
	Delete  *Operation  `json:"delete,omitempty" yaml:"delete,omitempty"`
	Patch   *Operation  `json:"patch,omitempty" yaml:"patch,omitempty"`
	Options *Operation  `json:"options,omitempty" yaml:"options,omitempty"`
	Head    *Operation  `json:"head,omitempty" yaml:"head,omitempty"`
	Trace   *Operation  `json:"trace,omitempty" yaml:"trace,omitempty"`
}

// Operation represents an HTTP operation
type Operation struct {
	Tags        []string               `json:"tags,omitempty" yaml:"tags,omitempty"`
	Summary     string                 `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string                 `json:"description,omitempty" yaml:"description,omitempty"`
	OperationID string                 `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Parameters  []Parameter           `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody *RequestBody           `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Responses   map[string]Response   `json:"responses,omitempty" yaml:"responses,omitempty"`
}

// Parameter represents an OpenAPI parameter
type Parameter struct {
	Name        string `json:"name" yaml:"name"`
	In          string `json:"in" yaml:"in"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool   `json:"required,omitempty" yaml:"required,omitempty"`
	Schema      Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
}

// RequestBody represents the request body
type RequestBody struct {
	Description string         `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool            `json:"required,omitempty" yaml:"required,omitempty"`
	Content     map[string]MediaType `json:"content,omitempty" yaml:"content,omitempty"`
}

// MediaType represents a media type
type MediaType struct {
	Schema Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
}

// Response represents an API response
type Response struct {
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// Schema represents an OpenAPI schema
type Schema struct {
	Type        string   `json:"type,omitempty" yaml:"type,omitempty"`
	Format      string   `json:"format,omitempty" yaml:"format,omitempty"`
	Example     any      `json:"example,omitempty" yaml:"example,omitempty"`
	Default     any      `json:"default,omitempty" yaml:"default,omitempty"`
	Enum        []any    `json:"enum,omitempty" yaml:"enum,omitempty"`
}

// Components represents reusable components
type Components struct {
	Schemas map[string]Schema `json:"schemas,omitempty" yaml:"schemas,omitempty"`
}

// Tag represents an OpenAPI tag
type Tag struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// Parse reads and parses an OpenAPI specification
func (o *OpenAPIImporter) Parse(path string) ([]*types.SavedRequest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	spec, err := o.parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse spec: %w", err)
	}

	return o.convertToRequests(spec), nil
}

// parse determines the format (YAML/JSON) and unmarshals the spec
func (o *OpenAPIImporter) parse(data []byte) (*OpenAPISpec, error) {
	// Try JSON first
	var spec OpenAPISpec
	if err := json.Unmarshal(data, &spec); err == nil && spec.OpenAPI != "" {
		return &spec, nil
	}

	// Try YAML
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse: %w", err)
	}

	if spec.OpenAPI == "" {
		return nil, fmt.Errorf("not a valid OpenAPI specification")
	}

	return &spec, nil
}

// convertToRequests transforms an OpenAPI spec into SavedRequests
func (o *OpenAPIImporter) convertToRequests(spec *OpenAPISpec) []*types.SavedRequest {
	var requests []*types.SavedRequest

	// Build tag lookup map
	tagMap := make(map[string]string)
	for _, tag := range spec.Tags {
		tagMap[tag.Name] = tag.Description
	}

	// Process each path
	for path, pathItem := range spec.Paths {
		operations := o.getOperations(&pathItem)

		for _, opWithMethod := range operations {
			req := o.operationToRequest(spec, path, opWithMethod, tagMap)
			requests = append(requests, req)
		}
	}

	return requests
}

// getOperations returns all operations from a PathItem with their HTTP methods
func (o *OpenAPIImporter) getOperations(pi *PathItem) []OpWithMethod {
	var ops []OpWithMethod
	if pi.Get != nil {
		ops = append(ops, OpWithMethod{Method: "GET", Op: pi.Get})
	}
	if pi.Post != nil {
		ops = append(ops, OpWithMethod{Method: "POST", Op: pi.Post})
	}
	if pi.Put != nil {
		ops = append(ops, OpWithMethod{Method: "PUT", Op: pi.Put})
	}
	if pi.Delete != nil {
		ops = append(ops, OpWithMethod{Method: "DELETE", Op: pi.Delete})
	}
	if pi.Patch != nil {
		ops = append(ops, OpWithMethod{Method: "PATCH", Op: pi.Patch})
	}
	if pi.Options != nil {
		ops = append(ops, OpWithMethod{Method: "OPTIONS", Op: pi.Options})
	}
	if pi.Head != nil {
		ops = append(ops, OpWithMethod{Method: "HEAD", Op: pi.Head})
	}
	if pi.Trace != nil {
		ops = append(ops, OpWithMethod{Method: "TRACE", Op: pi.Trace})
	}
	return ops
}

// operationToRequest converts an OpenAPI operation to a SavedRequest
func (o *OpenAPIImporter) operationToRequest(spec *OpenAPISpec, path string, opWithMethod OpWithMethod, tagMap map[string]string) *types.SavedRequest {
	op := opWithMethod.Op
	method := opWithMethod.Method
	name := o.getName(op, path, method)
	url := o.buildURL(spec, path)

	// Build headers
	var headers []types.Header
	for _, param := range op.Parameters {
		if param.In == "header" {
			headers = append(headers, types.Header{
				Key:   param.Name,
				Value: o.getExampleOrDefault(&param.Schema),
			})
		}
	}

	// Build query params into URL
	var queryParams []string
	for _, param := range op.Parameters {
		if param.In == "query" {
			queryParams = append(queryParams, fmt.Sprintf("%s=%s", param.Name, o.getExampleOrDefault(&param.Schema)))
		}
	}
	if len(queryParams) > 0 {
		url += "?" + strings.Join(queryParams, "&")
	}

	// Build body
	var body string
	if op.RequestBody != nil {
		body = o.extractBody(op.RequestBody)
	}

	// Get tags
	var tags []string
	if len(op.Tags) > 0 {
		tags = op.Tags
	} else {
		// Try to find tag from path
		for _, t := range spec.Tags {
			if strings.Contains(path, t.Name) {
				tags = append(tags, t.Name)
				break
			}
		}
	}

	return &types.SavedRequest{
		Name:        name,
		URL:         url,
		Method:      opWithMethod.Method,
		Headers:     headers,
		Body:        body,
		Tags:        tags,
		Collection:  spec.Info.Title,
	}
}

// getMethod determines the HTTP method from an operation
func (o *OpenAPIImporter) getMethod(op *Operation) string {
	switch {
	case op == nil:
		return "GET"
	default:
		// Use struct field inspection via operation reference
		// Since we can't directly check which field we're from, we check summary
		summary := strings.ToLower(op.Summary)
		switch {
		case strings.Contains(summary, "post"):
			return "POST"
		case strings.Contains(summary, "put"):
			return "PUT"
		case strings.Contains(summary, "delete"):
			return "DELETE"
		case strings.Contains(summary, "patch"):
			return "PATCH"
		case strings.Contains(summary, "head"):
			return "HEAD"
		case strings.Contains(summary, "options"):
			return "OPTIONS"
		default:
			return "GET"
		}
	}
}

// getName generates a name for the request
func (o *OpenAPIImporter) getName(op *Operation, path, method string) string {
	if op.OperationID != "" {
		return op.OperationID
	}
	if op.Summary != "" {
		return op.Summary
	}

	// Clean path for name
	name := path
	name = strings.TrimPrefix(name, "/")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "{", "")
	name = strings.ReplaceAll(name, "}", "")

	return method + "_" + name
}

// buildURL constructs the full URL from spec and path
func (o *OpenAPIImporter) buildURL(spec *OpenAPISpec, path string) string {
	// For now, use the path directly
	// In a full implementation, we'd also consider server URLs
	return path
}

// getExampleOrDefault returns an example or default value from schema
func (o *OpenAPIImporter) getExampleOrDefault(schema *Schema) string {
	if schema == nil {
		return ""
	}
	if schema.Example != nil {
		return fmt.Sprintf("%v", schema.Example)
	}
	if schema.Default != nil {
		return fmt.Sprintf("%v", schema.Default)
	}
	if len(schema.Enum) > 0 {
		return fmt.Sprintf("%v", schema.Enum[0])
	}
	return ""
}

// extractBody extracts the request body content
func (o *OpenAPIImporter) extractBody(rb *RequestBody) string {
	if rb == nil || rb.Content == nil {
		return ""
	}

	// Prefer JSON content
	for mediatype, content := range rb.Content {
		if strings.Contains(mediatype, "json") {
			return o.schemaToExample(&content.Schema)
		}
	}

	// Fall back to first available content type
	for _, content := range rb.Content {
		return o.schemaToExample(&content.Schema)
	}

	return ""
}

// schemaToExample converts a schema to an example string
func (o *OpenAPIImporter) schemaToExample(schema *Schema) string {
	if schema == nil {
		return ""
	}

	switch schema.Type {
	case "object":
		return "{ }"
	case "array":
		return "[ ]"
	case "string":
		if schema.Example != nil {
			return fmt.Sprintf("%v", schema.Example)
		}
		return "\"string\""
	case "integer", "number":
		if schema.Example != nil {
			return fmt.Sprintf("%v", schema.Example)
		}
		return "0"
	case "boolean":
		if schema.Example != nil {
			return fmt.Sprintf("%v", schema.Example)
		}
		return "true"
	default:
		return ""
	}
}
