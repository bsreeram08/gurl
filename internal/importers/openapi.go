package importers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/bsreeram08/valetkey/pkg/types"
	"gopkg.in/yaml.v3"
)

// OpenAPIImporter handles OpenAPI 3.x specification files
type OpenAPIImporter struct {
	// BaseURL, when set, replaces the spec's first server URL in every
	// imported request, for example "{{BASE_URL}}" so one import runs against
	// every environment.
	BaseURL string
}

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
	OpenAPI    string                `json:"openapi" yaml:"openapi"`
	Info       OpenAPIInfo           `json:"info" yaml:"info"`
	Paths      map[string]PathItem   `json:"paths" yaml:"paths"`
	Components Components            `json:"components" yaml:"components"`
	Tags       []Tag                 `json:"tags" yaml:"tags"`
	Servers    []Server              `json:"servers" yaml:"servers"`
	Security   []map[string][]string `json:"security" yaml:"security"`
}

// Server represents a server object
type Server struct {
	URL         string                    `json:"url" yaml:"url"`
	Description string                    `json:"description,omitempty" yaml:"description,omitempty"`
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
	Ref     string     `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Summary string     `json:"summary,omitempty" yaml:"summary,omitempty"`
	Get     *Operation `json:"get,omitempty" yaml:"get,omitempty"`
	Post    *Operation `json:"post,omitempty" yaml:"post,omitempty"`
	Put     *Operation `json:"put,omitempty" yaml:"put,omitempty"`
	Delete  *Operation `json:"delete,omitempty" yaml:"delete,omitempty"`
	Patch   *Operation `json:"patch,omitempty" yaml:"patch,omitempty"`
	Options *Operation `json:"options,omitempty" yaml:"options,omitempty"`
	Head    *Operation `json:"head,omitempty" yaml:"head,omitempty"`
	Trace   *Operation `json:"trace,omitempty" yaml:"trace,omitempty"`
}

// Operation represents an HTTP operation
type Operation struct {
	Tags        []string              `json:"tags,omitempty" yaml:"tags,omitempty"`
	Summary     string                `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string                `json:"description,omitempty" yaml:"description,omitempty"`
	OperationID string                `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Parameters  []Parameter           `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody *RequestBody          `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Responses   map[string]Response   `json:"responses,omitempty" yaml:"responses,omitempty"`
	Security    []map[string][]string `json:"security,omitempty" yaml:"security,omitempty"`
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
	Description string               `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool                 `json:"required,omitempty" yaml:"required,omitempty"`
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
	Ref              string            `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Type             string            `json:"type,omitempty" yaml:"type,omitempty"`
	Format           string            `json:"format,omitempty" yaml:"format,omitempty"`
	Example          any               `json:"example,omitempty" yaml:"example,omitempty"`
	Examples         []any             `json:"examples,omitempty" yaml:"examples,omitempty"`
	Default          any               `json:"default,omitempty" yaml:"default,omitempty"`
	Const            any               `json:"const,omitempty" yaml:"const,omitempty"`
	Enum             []any             `json:"enum,omitempty" yaml:"enum,omitempty"`
	Properties       map[string]Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	Items            *Schema           `json:"items,omitempty" yaml:"items,omitempty"`
	AllOf            []Schema          `json:"allOf,omitempty" yaml:"allOf,omitempty"`
	AnyOf            []Schema          `json:"anyOf,omitempty" yaml:"anyOf,omitempty"`
	OneOf            []Schema          `json:"oneOf,omitempty" yaml:"oneOf,omitempty"`
	Minimum          *float64          `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	ExclusiveMinimum any               `json:"exclusiveMinimum,omitempty" yaml:"exclusiveMinimum,omitempty"`
	MinLength        *int              `json:"minLength,omitempty" yaml:"minLength,omitempty"`
	MaxLength        *int              `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
}

// rawSchema is Schema with the type left raw: OpenAPI 3.1 allows a list of
// types (["string", "null"]) where 3.0 has a single string.
type rawSchema struct {
	Ref              string            `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Type             any               `json:"type,omitempty" yaml:"type,omitempty"`
	Format           string            `json:"format,omitempty" yaml:"format,omitempty"`
	Example          any               `json:"example,omitempty" yaml:"example,omitempty"`
	Examples         any               `json:"examples,omitempty" yaml:"examples,omitempty"`
	Default          any               `json:"default,omitempty" yaml:"default,omitempty"`
	Const            any               `json:"const,omitempty" yaml:"const,omitempty"`
	Enum             []any             `json:"enum,omitempty" yaml:"enum,omitempty"`
	Properties       map[string]Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	Items            *Schema           `json:"items,omitempty" yaml:"items,omitempty"`
	AllOf            []Schema          `json:"allOf,omitempty" yaml:"allOf,omitempty"`
	AnyOf            []Schema          `json:"anyOf,omitempty" yaml:"anyOf,omitempty"`
	OneOf            []Schema          `json:"oneOf,omitempty" yaml:"oneOf,omitempty"`
	Minimum          *float64          `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	ExclusiveMinimum any               `json:"exclusiveMinimum,omitempty" yaml:"exclusiveMinimum,omitempty"`
	MinLength        *int              `json:"minLength,omitempty" yaml:"minLength,omitempty"`
	MaxLength        *int              `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
}

func (s *Schema) fromRaw(raw rawSchema) {
	*s = Schema{
		Ref: raw.Ref, Format: raw.Format, Example: raw.Example, Default: raw.Default,
		Const: raw.Const, Enum: raw.Enum, Properties: raw.Properties, Items: raw.Items,
		AllOf: raw.AllOf, AnyOf: raw.AnyOf, OneOf: raw.OneOf,
		Minimum: raw.Minimum, ExclusiveMinimum: raw.ExclusiveMinimum,
		MinLength: raw.MinLength, MaxLength: raw.MaxLength,
	}
	switch t := raw.Type.(type) {
	case string:
		s.Type = t
	case []any:
		// The first non-null type; "null" alone stays "null".
		for _, v := range t {
			if name, ok := v.(string); ok && (s.Type == "" || s.Type == "null") {
				s.Type = name
			}
		}
	}
	// "examples" is a list in a 3.1 schema; a map elsewhere is ignored.
	if list, ok := raw.Examples.([]any); ok {
		s.Examples = list
	}
}

// UnmarshalJSON accepts both the 3.0 and 3.1 forms of "type".
func (s *Schema) UnmarshalJSON(data []byte) error {
	var raw rawSchema
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	s.fromRaw(raw)
	return nil
}

// UnmarshalYAML accepts both the 3.0 and 3.1 forms of "type".
func (s *Schema) UnmarshalYAML(node *yaml.Node) error {
	var raw rawSchema
	if err := node.Decode(&raw); err != nil {
		return err
	}
	s.fromRaw(raw)
	return nil
}

// Components represents reusable components
type Components struct {
	Schemas         map[string]Schema         `json:"schemas,omitempty" yaml:"schemas,omitempty"`
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes,omitempty" yaml:"securitySchemes,omitempty"`
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

// convertToRequests transforms an OpenAPI spec into SavedRequests, in path
// order so repeated imports produce the same requests in the same order.
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
	if o.BaseURL != "" {
		baseURL = o.BaseURL
	}

	paths := make([]string, 0, len(spec.Paths))
	for path := range spec.Paths {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		pathItem := spec.Paths[path]
		// Resolve PathItem refs
		resolvedPI := o.resolvePathItem(path, spec, nil)
		if resolvedPI == nil {
			resolvedPI = &pathItem
		}

		operations := o.getOperations(resolvedPI)

		for _, opWithMethod := range operations {
			req := o.operationToRequest(spec, baseURL, path, opWithMethod, tagMap, securitySchemeMap)
			req.SortOrder = len(requests) + 1
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

// operationToRequest converts an OpenAPI operation to a SavedRequest.
//
// The request is meant to run as imported: path parameters become {{name}}
// placeholders (a missing value stops the run instead of sending "{name}"),
// a required query or header parameter
// without an example becomes a {{name}} placeholder, an optional one without
// an example is left out, and credentials are {{token}}-style placeholders.
func (o *OpenAPIImporter) operationToRequest(spec *OpenAPISpec, baseURL, path string, opWithMethod OpWithMethod, tagMap map[string]string, securitySchemeMap map[string]SecurityScheme) *types.SavedRequest {
	op := opWithMethod.Op
	method := opWithMethod.Method
	name := o.getName(op, path, method)

	url := o.buildURL(baseURL, pathParamsToPlaceholders(path))

	// Build headers and query params
	var headers []types.Header
	var queryParams []string
	for _, param := range op.Parameters {
		value, ok := o.paramValue(param)
		if !ok {
			continue
		}
		switch param.In {
		case "header":
			headers = append(headers, types.Header{Key: param.Name, Value: value})
		case "query":
			queryParams = append(queryParams, queryEscape(param.Name)+"="+queryValue(value))
		}
	}

	// Build body, with the Content-Type it was generated for
	var body string
	if op.RequestBody != nil {
		var mediaType string
		body, mediaType = o.extractBodyFor(op.RequestBody, spec.Components.Schemas)
		if body != "" && mediaType != "" && !hasHeader(headers, "Content-Type") {
			headers = append(headers, types.Header{Key: "Content-Type", Value: mediaType})
		}
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
	var folder string
	if len(tags) > 0 {
		folder = tags[0]
	}

	// Apply security
	authConfig, authQuery := o.applySecurity(op.Security, spec.Security, securitySchemeMap, &headers)
	queryParams = append(queryParams, authQuery...)
	if len(queryParams) > 0 {
		url += "?" + strings.Join(queryParams, "&")
	}

	return &types.SavedRequest{
		Name:       name,
		URL:        url,
		Method:     opWithMethod.Method,
		Headers:    headers,
		Body:       body,
		Tags:       tags,
		Folder:     folder,
		AuthConfig: authConfig,
		Collection: spec.Info.Title,
	}
}

// pathParamsToPlaceholders turns OpenAPI {name} path templates into gurl
// {{name}} placeholders.
func pathParamsToPlaceholders(path string) string {
	var b strings.Builder
	for {
		start := strings.IndexByte(path, '{')
		if start < 0 {
			b.WriteString(path)
			return b.String()
		}
		end := strings.IndexByte(path[start:], '}')
		if end < 0 {
			b.WriteString(path)
			return b.String()
		}
		end += start
		b.WriteString(path[:start])
		b.WriteString("{{" + path[start+1:end] + "}}")
		path = path[end+1:]
	}
}

// paramValue returns the value to send for a query or header parameter, and
// false when an optional parameter without an example should be left out.
func (o *OpenAPIImporter) paramValue(param Parameter) (string, bool) {
	if value := o.getExampleOrDefault(&param.Schema); value != "" {
		return value, true
	}
	if param.Required {
		return "{{" + param.Name + "}}", true
	}
	return "", false
}

// queryValue escapes a query value but keeps a {{placeholder}} intact.
func queryValue(value string) string {
	if strings.HasPrefix(value, "{{") && strings.HasSuffix(value, "}}") {
		return value
	}
	return queryEscape(value)
}

func hasHeader(headers []types.Header, key string) bool {
	for _, h := range headers {
		if strings.EqualFold(h.Key, key) {
			return true
		}
	}
	return false
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

// applySecurity turns the operation's security requirement into gurl auth.
// Bearer and basic schemes become an auth config with {{token}} or
// {{username}}/{{password}} placeholders, so the run fails loudly until the
// environment provides them; an API key becomes a {{apiKey}} header or
// query parameter. An operation with "security: []" is public and gets none.
func (o *OpenAPIImporter) applySecurity(opSecurity, specSecurity []map[string][]string, securitySchemeMap map[string]SecurityScheme, headers *[]types.Header) (*types.AuthConfig, []string) {
	security := opSecurity
	if security == nil {
		security = specSecurity
	}

	var authConfig *types.AuthConfig
	var query []string
	for _, sec := range security {
		names := make([]string, 0, len(sec))
		for schemeName := range sec {
			names = append(names, schemeName)
		}
		sort.Strings(names)
		for _, schemeName := range names {
			scheme, ok := securitySchemeMap[schemeName]
			if !ok {
				continue
			}
			switch scheme.Type {
			case "http":
				switch strings.ToLower(scheme.Scheme) {
				case "bearer":
					if authConfig == nil {
						authConfig = &types.AuthConfig{Type: "bearer", Params: map[string]string{"token": "{{token}}"}}
					}
				case "basic":
					if authConfig == nil {
						authConfig = &types.AuthConfig{Type: "basic", Params: map[string]string{"username": "{{username}}", "password": "{{password}}"}}
					}
				}
			case "apiKey":
				switch scheme.In {
				case "header":
					if !hasHeader(*headers, scheme.Name) {
						*headers = append(*headers, types.Header{Key: scheme.Name, Value: "{{apiKey}}"})
					}
				case "query":
					query = append(query, queryEscape(scheme.Name)+"={{apiKey}}")
				}
			}
		}
		// The first requirement that applies is enough; alternatives
		// (several entries) are not combined.
		if authConfig != nil || len(query) > 0 {
			break
		}
	}
	return authConfig, query
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
	body, _ := o.extractBodyFor(rb, nil)
	return body
}

// extractBodyFor returns an example body and the media type it was built
// for. JSON is preferred over other media types, then text.
func (o *OpenAPIImporter) extractBodyFor(rb *RequestBody, schemas map[string]Schema) (string, string) {
	if rb == nil || rb.Content == nil {
		return "", ""
	}

	mediaTypes := make([]string, 0, len(rb.Content))
	for mediaType := range rb.Content {
		mediaTypes = append(mediaTypes, mediaType)
	}
	sort.Strings(mediaTypes)

	for _, mediaType := range mediaTypes {
		if strings.Contains(mediaType, "json") {
			content := rb.Content[mediaType]
			return o.schemaToExampleWith(&content.Schema, schemas), mediaType
		}
	}

	for _, mediaType := range mediaTypes {
		if strings.Contains(mediaType, "text") {
			content := rb.Content[mediaType]
			return o.getExampleOrDefault(&content.Schema), mediaType
		}
	}

	for _, mediaType := range mediaTypes {
		content := rb.Content[mediaType]
		return o.schemaToExampleWith(&content.Schema, schemas), mediaType
	}

	return "", ""
}

// schemaToExample converts a schema to an example JSON document
func (o *OpenAPIImporter) schemaToExample(schema *Schema) string {
	return o.schemaToExampleWith(schema, nil)
}

// maxExampleDepth bounds recursion through nested and self-referencing schemas.
const maxExampleDepth = 12

// schemaToExampleWith renders the example value of schema as indented JSON,
// resolving $refs against schemas. It returns "" when the schema says nothing
// about the value.
func (o *OpenAPIImporter) schemaToExampleWith(schema *Schema, schemas map[string]Schema) string {
	if schema == nil {
		return ""
	}
	value, ok := exampleValue(schema, schemas, 0)
	if !ok {
		return ""
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		return ""
	}
	return strings.TrimSuffix(buf.String(), "\n")
}

// exampleValue builds an example for schema: its example, const, default or
// first enum value when it has one, otherwise a placeholder of its type. The
// first alternative of anyOf/oneOf is used, preferring a non-null one, and
// allOf members are merged. ok is false when nothing is known.
func exampleValue(schema *Schema, schemas map[string]Schema, depth int) (any, bool) {
	if schema == nil || depth > maxExampleDepth {
		return nil, false
	}
	if schema.Ref != "" {
		resolved, ok := lookupSchemaRef(schema.Ref, schemas)
		if !ok {
			return nil, false
		}
		return exampleValue(&resolved, schemas, depth+1)
	}
	switch {
	case schema.Example != nil:
		return schema.Example, true
	case len(schema.Examples) > 0:
		return schema.Examples[0], true
	case schema.Const != nil:
		return schema.Const, true
	case schema.Default != nil:
		return schema.Default, true
	case len(schema.Enum) > 0:
		return schema.Enum[0], true
	}

	if len(schema.AllOf) > 0 {
		merged := map[string]any{}
		var last any
		found := false
		for i := range schema.AllOf {
			value, ok := exampleValue(&schema.AllOf[i], schemas, depth+1)
			if !ok {
				continue
			}
			found = true
			if obj, isObj := value.(map[string]any); isObj {
				for k, v := range obj {
					merged[k] = v
				}
			} else {
				last = value
			}
		}
		if len(merged) > 0 {
			return merged, true
		}
		if found {
			return last, true
		}
	}
	for _, alternatives := range [][]Schema{schema.AnyOf, schema.OneOf} {
		if len(alternatives) == 0 {
			continue
		}
		for i := range alternatives {
			if alternatives[i].Type == "null" {
				continue
			}
			if value, ok := exampleValue(&alternatives[i], schemas, depth+1); ok {
				return atLeastMinimum(schema, value), true
			}
		}
		return nil, true
	}

	schemaType := schema.Type
	if schemaType == "" && len(schema.Properties) > 0 {
		schemaType = "object"
	}
	switch schemaType {
	case "object":
		obj := make(map[string]any, len(schema.Properties))
		for name, prop := range schema.Properties {
			value, ok := exampleValue(&prop, schemas, depth+1)
			if !ok {
				value = ""
			}
			obj[name] = value
		}
		return obj, true
	case "array":
		if schema.Items != nil {
			if item, ok := exampleValue(schema.Items, schemas, depth+1); ok {
				return []any{item}, true
			}
		}
		return []any{}, true
	case "string":
		return fitLength(stringExample(schema.Format), schema.MinLength, schema.MaxLength), true
	case "integer", "number":
		return numberExample(schema), true
	case "boolean":
		return true, true
	case "null":
		return nil, true
	}
	return nil, false
}

// numberExample returns 0, or the smallest value the schema's minimum allows.
func numberExample(schema *Schema) any {
	value := 0.0
	if schema.Minimum != nil && *schema.Minimum > value {
		value = *schema.Minimum
	}
	switch m := schema.ExclusiveMinimum.(type) {
	case float64: // OpenAPI 3.1: the bound itself
		if m >= value {
			value = m + 1
		}
	case int:
		if float64(m) >= value {
			value = float64(m) + 1
		}
	case bool: // OpenAPI 3.0: minimum is exclusive
		if m && schema.Minimum != nil && value == *schema.Minimum {
			value++
		}
	}
	if value == float64(int64(value)) {
		return int64(value)
	}
	return value
}

// atLeastMinimum raises a numeric value picked from a union to the bound the
// union itself declares, as in {"minimum": 1, "anyOf": [{"type": "string",
// "default": 0}, {"type": "integer"}]}.
func atLeastMinimum(schema *Schema, value any) any {
	if schema.Minimum == nil && schema.ExclusiveMinimum == nil {
		return value
	}
	var number float64
	switch v := value.(type) {
	case float64:
		number = v
	case int:
		number = float64(v)
	case int64:
		number = float64(v)
	default:
		return value
	}
	floor := numberExample(schema)
	var floorValue float64
	switch f := floor.(type) {
	case int64:
		floorValue = float64(f)
	case float64:
		floorValue = f
	}
	if number < floorValue {
		return floor
	}
	return value
}

// fitLength pads or trims a placeholder string to the schema's length bounds.
func fitLength(value string, minLength, maxLength *int) string {
	if minLength != nil && len(value) < *minLength {
		value += strings.Repeat("x", *minLength-len(value))
	}
	if maxLength != nil && *maxLength >= 0 && len(value) > *maxLength {
		value = value[:*maxLength]
	}
	return value
}

// stringExample returns a placeholder string that satisfies common formats.
func stringExample(format string) string {
	switch format {
	case "date-time":
		return "2024-01-01T00:00:00Z"
	case "date":
		return "2024-01-01"
	case "email":
		return "user@example.com"
	case "uuid":
		return "00000000-0000-0000-0000-000000000000"
	case "uri", "url":
		return "https://example.com"
	}
	return "string"
}

// lookupSchemaRef resolves a local "#/components/schemas/<name>" reference.
func lookupSchemaRef(ref string, schemas map[string]Schema) (Schema, bool) {
	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(ref, prefix) || schemas == nil {
		return Schema{}, false
	}
	name := strings.ReplaceAll(strings.TrimPrefix(ref, prefix), "~1", "/")
	name = strings.ReplaceAll(name, "~0", "~")
	resolved, ok := schemas[name]
	return resolved, ok
}
