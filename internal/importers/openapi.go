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
	OpenAPI    string                  `json:"openapi" yaml:"openapi"`
	Info       OpenAPIInfo             `json:"info" yaml:"info"`
	Paths      map[string]PathItem     `json:"paths" yaml:"paths"`
	Components Components              `json:"components" yaml:"components"`
	Tags       []Tag                   `json:"tags" yaml:"tags"`
	Servers    []Server                `json:"servers" yaml:"servers"`
	Security   []map[string][]string   `json:"security" yaml:"security"`
}

// Server represents a server object
type Server struct {
	URL         string                  `json:"url" yaml:"url"`
	Description string                  `json:"description,omitempty" yaml:"description,omitempty"`
	Variables   map[string]ServerVariable `json:"variables,omitempty" yaml:"variables,omitempty"`
}

// ServerVariable represents a server variable
type ServerVariable struct {
	Default     string   `json:"default,omitempty" yaml:"default,omitempty"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Enum        []string `json:"enum,omitempty" yaml:"enum,omitempty"`
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
	Tags        []string                `json:"tags,omitempty" yaml:"tags,omitempty"`
	Summary     string                  `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string                  `json:"description,omitempty" yaml:"description,omitempty"`
	OperationID string                  `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Parameters  []Parameter             `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody *RequestBody            `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Responses   map[string]Response     `json:"responses,omitempty" yaml:"responses,omitempty"`
	Security    []map[string][]string   `json:"security,omitempty" yaml:"security,omitempty"`
}

// Parameter represents an OpenAPI parameter
type Parameter struct {
	Ref         string `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Name        string `json:"name" yaml:"name"`
	In          string `json:"in" yaml:"in"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool   `json:"required,omitempty" yaml:"required,omitempty"`
	Schema      Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
}

// RequestBody represents the request body
type RequestBody struct {
	Description string                  `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool                    `json:"required,omitempty" yaml:"required,omitempty"`
	Content     map[string]MediaType    `json:"content,omitempty" yaml:"content,omitempty"`
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
	Ref       string   `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Type      string   `json:"type,omitempty" yaml:"type,omitempty"`
	Format    string   `json:"format,omitempty" yaml:"format,omitempty"`
	Example   any      `json:"example,omitempty" yaml:"example,omitempty"`
	Default   any      `json:"default,omitempty" yaml:"default,omitempty"`
	Enum      []any    `json:"enum,omitempty" yaml:"enum,omitempty"`
	Properties map[string]Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	Items     *Schema  `json:"items,omitempty" yaml:"items,omitempty"`
}

// Components represents reusable components
type Components struct {
	Schemas         map[string]Schema              `json:"schemas,omitempty" yaml:"schemas,omitempty"`
	SecuritySchemes map[string]SecurityScheme      `json:"securitySchemes,omitempty" yaml:"securitySchemes,omitempty"`
}

// SecurityScheme represents a security scheme
type SecurityScheme struct {
	Type         string `json:"type,omitempty" yaml:"type,omitempty"`
	Scheme       string `json:"scheme,omitempty" yaml:"scheme,omitempty"`
	BearerFormat string `json:"bearerFormat,omitempty" yaml:"bearerFormat,omitempty"`
	Name         string `json:"name,omitempty" yaml:"name,omitempty"`
	In           string `json:"in,omitempty" yaml:"in,omitempty"`
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

	// Strip BOM if present
	data = stripBOM(data)

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

// resolvePathItem resolves $ref in PathItem
func (o *OpenAPIImporter) resolvePathItem(path string, spec *OpenAPISpec, visited map[string]bool) *PathItem {
	if visited == nil {
		visited = make(map[string]bool)
	}

	// Check for circular reference
	if visited[path] {
		return nil
	}

	pi, ok := spec.Paths[path]
	if !ok {
		return nil
	}

	if pi.Ref != "" {
		visited[path] = true
		// Resolve $ref - format: #/paths/pathname
		parts := strings.Split(pi.Ref, "/")
		if len(parts) >= 3 && parts[0] == "#" {
			resolvedPath := strings.Join(parts[2:], "/")
			resolvedPath = strings.ReplaceAll(resolvedPath, "~1", "/")
			resolvedPath = strings.ReplaceAll(resolvedPath, "~0", "~")
			if resolvedPI, ok := spec.Paths[resolvedPath]; ok {
				pi = resolvedPI
				if pi.Ref != "" {
					return o.resolvePathItem("/"+resolvedPath, spec, visited)
				}
			}
		}
	}

	// Resolve refs in parameters for each operation
	if pi.Get != nil {
		for i := range pi.Get.Parameters {
			o.resolveParameterRef(&pi.Get.Parameters[i], spec, visited)
		}
	}
	if pi.Post != nil {
		for i := range pi.Post.Parameters {
			o.resolveParameterRef(&pi.Post.Parameters[i], spec, visited)
		}
	}
	if pi.Put != nil {
		for i := range pi.Put.Parameters {
			o.resolveParameterRef(&pi.Put.Parameters[i], spec, visited)
		}
	}
	if pi.Delete != nil {
		for i := range pi.Delete.Parameters {
			o.resolveParameterRef(&pi.Delete.Parameters[i], spec, visited)
		}
	}
	if pi.Patch != nil {
		for i := range pi.Patch.Parameters {
			o.resolveParameterRef(&pi.Patch.Parameters[i], spec, visited)
		}
	}
	if pi.Options != nil {
		for i := range pi.Options.Parameters {
			o.resolveParameterRef(&pi.Options.Parameters[i], spec, visited)
		}
	}
	if pi.Head != nil {
		for i := range pi.Head.Parameters {
			o.resolveParameterRef(&pi.Head.Parameters[i], spec, visited)
		}
	}
	if pi.Trace != nil {
		for i := range pi.Trace.Parameters {
			o.resolveParameterRef(&pi.Trace.Parameters[i], spec, visited)
		}
	}

	return &pi
}

// resolveParameterRef resolves $ref in parameter
func (o *OpenAPIImporter) resolveParameterRef(param *Parameter, spec *OpenAPISpec, visited map[string]bool) {
	if param.Ref != "" {
		// Resolve $ref - format: #/components/parameters/paramname
		parts := strings.Split(param.Ref, "/")
		if len(parts) >= 4 && parts[0] == "#" {
			resolvedRef := strings.Join(parts[3:], "/")
			resolvedRef = strings.ReplaceAll(resolvedRef, "~1", "/")
			resolvedRef = strings.ReplaceAll(resolvedRef, "~0", "~")
			if spec.Components.Schemas != nil {
				if resolvedSchema, ok := spec.Components.Schemas[resolvedRef]; ok {
					param.Name = resolvedSchema.Type
					param.Schema = resolvedSchema
				}
			}
		}
	}

	// Resolve schema ref if present
	o.resolveSchemaRef(&param.Schema, spec, visited)
}

// resolveSchemaRef resolves $ref in schema
func (o *OpenAPIImporter) resolveSchemaRef(schema *Schema, spec *OpenAPISpec, visited map[string]bool) {
	if schema == nil || schema.Ref == "" {
		return
	}

	if visited == nil {
		visited = make(map[string]bool)
	}

	// Check for circular reference
	if visited[schema.Ref] {
		return
	}
	visited[schema.Ref] = true

	// Resolve $ref - format: #/components/schemas/typename
	parts := strings.Split(schema.Ref, "/")
	if len(parts) >= 4 && parts[0] == "#" {
		resolvedRef := strings.Join(parts[3:], "/")
		resolvedRef = strings.ReplaceAll(resolvedRef, "~1", "/")
		resolvedRef = strings.ReplaceAll(resolvedRef, "~0", "~")
		if spec.Components.Schemas != nil {
			if resolvedSchema, ok := spec.Components.Schemas[resolvedRef]; ok {
				*schema = resolvedSchema
				// Continue resolving if the resolved schema also has a ref
				if schema.Ref != "" {
					o.resolveSchemaRef(schema, spec, visited)
				}
			}
		}
	}
}

// convertToRequests transforms an OpenAPI spec into SavedRequests
func (o *OpenAPIImporter) convertToRequests(spec *OpenAPISpec) []*types.SavedRequest {
	var requests []*types.SavedRequest

	// Build tag lookup map
	tagMap := make(map[string]string)
	for _, tag := range spec.Tags {
		tagMap[tag.Name] = tag.Description
	}

	// Build security schemes map
	securitySchemeMap := make(map[string]SecurityScheme)
	if spec.Components.SecuritySchemes != nil {
		for name, scheme := range spec.Components.SecuritySchemes {
			securitySchemeMap[name] = scheme
		}
	}

	// Get base server URL
	baseURL := o.getServerURL(spec)

	// Process each path
	for path, pathItem := range spec.Paths {
		// Resolve PathItem refs
		resolvedPI := o.resolvePathItem(path, spec, nil)
		if resolvedPI == nil {
			resolvedPI = &pathItem
		}

		operations := o.getOperations(resolvedPI)

		for _, opWithMethod := range operations {
			req := o.operationToRequest(spec, baseURL, path, opWithMethod, tagMap, securitySchemeMap)
			requests = append(requests, req)
		}
	}

	return requests
}

// getServerURL returns the base server URL with resolved variables
func (o *OpenAPIImporter) getServerURL(spec *OpenAPISpec) string {
	if len(spec.Servers) == 0 {
		return ""
	}

	server := spec.Servers[0]
	url := server.URL

	// Resolve server variables
	for name, variable := range server.Variables {
		defaultVal := variable.Default
		if defaultVal == "" && len(variable.Enum) > 0 {
			defaultVal = variable.Enum[0]
		}
		url = strings.ReplaceAll(url, "{"+name+"}", defaultVal)
	}

	return url
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
func (o *OpenAPIImporter) operationToRequest(spec *OpenAPISpec, baseURL, path string, opWithMethod OpWithMethod, tagMap map[string]string, securitySchemeMap map[string]SecurityScheme) *types.SavedRequest {
	op := opWithMethod.Op
	method := opWithMethod.Method
	name := o.getName(op, path, method)

	url := o.buildURL(baseURL, path)

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
			queryParams = append(queryParams, fmt.Sprintf("%s=%s", queryEscape(param.Name), queryEscape(o.getExampleOrDefault(&param.Schema))))
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

	// Apply security
	o.applySecurity(op.Security, spec.Security, securitySchemeMap, &headers)

	return &types.SavedRequest{
		Name:       name,
		URL:        url,
		Method:     opWithMethod.Method,
		Headers:    headers,
		Body:       body,
		Tags:       tags,
		Collection: spec.Info.Title,
	}
}

// queryEscape URL-encodes a query parameter value
func queryEscape(s string) string {
	// Simple URL encoding for query params
	var result strings.Builder
	for _, c := range s {
		switch {
		case c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.' || c == '~':
			result.WriteRune(c)
		case c == ' ':
			result.WriteString("%20")
		default:
			result.WriteString(fmt.Sprintf("%%%X", c))
		}
	}
	return result.String()
}

// substitutePathParams replaces {param} placeholders with example/default values
func (o *OpenAPIImporter) substitutePathParams(path string, params []Parameter) string {
	for _, param := range params {
		if param.In == "path" {
			replacement := o.getExampleOrDefault(&param.Schema)
			if replacement == "" {
				replacement = "example"
			}
			path = strings.ReplaceAll(path, "{"+param.Name+"}", replacement)
		}
	}
	return path
}

// applySecurity applies security schemes to headers
func (o *OpenAPIImporter) applySecurity(opSecurity, specSecurity []map[string][]string, securitySchemeMap map[string]SecurityScheme, headers *[]types.Header) {
	security := opSecurity
	if len(security) == 0 {
		security = specSecurity
	}

	for _, sec := range security {
		for schemeName := range sec {
			if scheme, ok := securitySchemeMap[schemeName]; ok {
				switch scheme.Type {
				case "http":
					if scheme.Scheme == "bearer" {
						*headers = append(*headers, types.Header{
							Key:   "Authorization",
							Value: "Bearer <token>",
						})
					} else if scheme.Scheme == "basic" {
						*headers = append(*headers, types.Header{
							Key:   "Authorization",
							Value: "Basic <credentials>",
						})
					}
				case "apiKey":
					if scheme.In == "header" {
						*headers = append(*headers, types.Header{
							Key:   scheme.Name,
							Value: "<api_key>",
						})
					}
				}
			}
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

// getMethod extracts the HTTP method from an operation's Summary field
func (o *OpenAPIImporter) getMethod(op *Operation) string {
	if op == nil {
		return "GET"
	}
	summary := strings.ToUpper(op.Summary)
	switch {
	case strings.Contains(summary, "POST"):
		return "POST"
	case strings.Contains(summary, "PUT"):
		return "PUT"
	case strings.Contains(summary, "DELETE"):
		return "DELETE"
	case strings.Contains(summary, "PATCH"):
		return "PATCH"
	case strings.Contains(summary, "HEAD"):
		return "HEAD"
	case strings.Contains(summary, "OPTIONS"):
		return "OPTIONS"
	default:
		return "GET"
	}
}

// buildURL constructs the full URL from spec and path
func (o *OpenAPIImporter) buildURL(baseURL, path string) string {
	if baseURL == "" {
		return path
	}
	// Ensure proper joining
	baseURL = strings.TrimSuffix(baseURL, "/")
	path = strings.TrimPrefix(path, "/")
	return baseURL + "/" + path
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

	for mediatype, content := range rb.Content {
		if strings.Contains(mediatype, "json") {
			return o.schemaToExample(&content.Schema)
		}
	}

	for mediatype, content := range rb.Content {
		if strings.Contains(mediatype, "text") {
			return o.getExampleOrDefault(&content.Schema)
		}
	}

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
		if len(schema.Properties) > 0 {
			var pairs []string
			for name, prop := range schema.Properties {
				value := o.schemaToExample(&prop)
				if value == "" {
					value = "\"\""
				}
				pairs = append(pairs, fmt.Sprintf("%q: %s", name, value))
			}
			return "{" + strings.Join(pairs, ", ") + "}"
		}
		return "{ }"
	case "array":
		if schema.Items != nil {
			item := o.schemaToExample(schema.Items)
			if item == "" {
				item = "null"
			}
			return "[" + item + "]"
		}
		return "[ ]"
	case "string":
		if schema.Example != nil {
			return fmt.Sprintf("%v", schema.Example)
		}
		if schema.Default != nil {
			return fmt.Sprintf("%v", schema.Default)
		}
		return "\"string\""
	case "integer", "number":
		if schema.Example != nil {
			return fmt.Sprintf("%v", schema.Example)
		}
		if schema.Default != nil {
			return fmt.Sprintf("%v", schema.Default)
		}
		return "0"
	case "boolean":
		if schema.Example != nil {
			return fmt.Sprintf("%v", schema.Example)
		}
		if schema.Default != nil {
			return fmt.Sprintf("%v", schema.Default)
		}
		return "true"
	default:
		return ""
	}
}
