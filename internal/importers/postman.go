package importers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sreeram/gurl/pkg/types"
)

// PostmanImporter handles Postman collection v2.1 files
type PostmanImporter struct{}

// Name returns the importer name
func (p *PostmanImporter) Name() string {
	return "postman"
}

// Extensions returns supported file extensions
func (p *PostmanImporter) Extensions() []string {
	return []string{".json"}
}

// PostmanCollection represents a Postman collection v2.1
type PostmanCollection struct {
	Info      PostmanInfo   `json:"info"`
	Item      []PostmanItem `json:"item"`
	Variable  []PostmanVar  `json:"variable,omitempty"`
	Auth      *PostmanAuth  `json:"auth,omitempty"`
}

// PostmanInfo contains collection metadata
type PostmanInfo struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Schema      string `json:"schema"`
}

// PostmanItem represents a request or folder in a collection
type PostmanItem struct {
	Name        string          `json:"name,omitempty"`
	Item        []PostmanItem   `json:"item,omitempty"` // Nested items (folders)
	Request     *PostmanRequest  `json:"request,omitempty"`
	Response    []any           `json:"response,omitempty"` // Ignore responses
}

// PostmanRequest represents a request in Postman
type PostmanRequest struct {
	Method   string              `json:"method,omitempty"`
	Header   []PostmanHeader     `json:"header,omitempty"`
	URL      interface{}         `json:"url,omitempty"` // Can be string or object
	Body     *PostmanBody         `json:"body,omitempty"`
	Auth     *PostmanAuth        `json:"auth,omitempty"`
}

// PostmanHeader represents a header
type PostmanHeader struct {
	Key        string `json:"key"`
	Value      string `json:"value"`
	Disabled   bool   `json:"disabled,omitempty"`
	Description string `json:"description,omitempty"`
}

// PostmanBody represents the request body
type PostmanBody struct {
	Mode       string `json:"mode,omitempty"` // raw, formdata, urlencoded, binary, graphql
	Raw        string `json:"raw,omitempty"`
	FormData   []PostmanFormParam `json:"formdata,omitempty"`
	URLEncoded []PostmanFormParam `json:"urlencoded,omitempty"`
	GraphQL    *PostmanGraphQL   `json:"graphql,omitempty"`
}

// PostmanFormParam represents a form parameter
type PostmanFormParam struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Type   string `json:"type,omitempty"`
	Disabled bool `json:"disabled,omitempty"`
}

// PostmanGraphQL represents GraphQL body
type PostmanGraphQL struct {
	Query     string `json:"query,omitempty"`
	Variables string `json:"variables,omitempty"`
}

// PostmanAuth represents authentication
type PostmanAuth struct {
	Type        string                 `json:"type,omitempty"`
	Bearer      []PostmanAuthParam     `json:"bearer,omitempty"`
	Basic       []PostmanAuthParam     `json:"basic,omitempty"`
	APIKey      []PostmanAuthParam     `json:"apikey,omitempty"`
	Digest      []PostmanAuthParam     `json:"digest,omitempty"`
}

// PostmanAuthParam represents an auth parameter
type PostmanAuthParam struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// PostmanVar represents a collection variable
type PostmanVar struct {
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
}

// PostmanURL represents a URL object
type PostmanURL struct {
	Raw   string `json:"raw,omitempty"`
	Host  []string `json:"host,omitempty"`
	Path  []string `json:"path,omitempty"`
	Query []PostmanQueryParam `json:"query,omitempty"`
}

// PostmanQueryParam represents a query parameter
type PostmanQueryParam struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Parse reads and parses a Postman collection file
func (p *PostmanImporter) Parse(path string) ([]*types.SavedRequest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	// Strip BOM if present
	data = stripBOM(data)

	var collection PostmanCollection
	if err := json.Unmarshal(data, &collection); err != nil {
		return nil, fmt.Errorf("parse postman collection: %w", err)
	}

	return p.convertToRequests(&collection), nil
}

// stripBOM removes UTF-8 BOM if present
func stripBOM(data []byte) []byte {
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return data[3:]
	}
	return data
}

// toUTF8 converts data to UTF-8
func toUTF8(data []byte) []byte {
	// Simple implementation - in production would use charset detection
	return data
}

// convertToRequests transforms a Postman collection into SavedRequests
func (p *PostmanImporter) convertToRequests(collection *PostmanCollection) []*types.SavedRequest {
	var requests []*types.SavedRequest

	// Process all items recursively
	p.processItems(collection.Item, collection.Info.Name, "", collection.Variable, collection.Auth, &requests)

	return requests
}

// processItems recursively processes Postman items
func (p *PostmanImporter) processItems(items []PostmanItem, collectionName, folderPath string, variables []PostmanVar, collectionAuth *PostmanAuth, requests *[]*types.SavedRequest) {
	for _, item := range items {
		if item.Request != nil {
			// Merge collection auth with request auth (request auth takes precedence)
			var auth *PostmanAuth
			if item.Request.Auth != nil {
				auth = item.Request.Auth
			} else if collectionAuth != nil {
				auth = collectionAuth
			}
			// Merge collection variables with item variables
			req := p.requestToSavedRequest(item.Request, item.Name, collectionName, folderPath, variables, auth)
			*requests = append(*requests, req)
		}

		// Process nested items (folders)
		if len(item.Item) > 0 {
			newFolderPath := folderPath
			if item.Name != "" {
				if newFolderPath != "" {
					newFolderPath += "/"
				}
				newFolderPath += item.Name
			}
			p.processItems(item.Item, collectionName, newFolderPath, variables, collectionAuth, requests)
		}
	}
}

// substituteVars replaces {{var}} patterns with variable values
func (p *PostmanImporter) substituteVars(s string, variables []PostmanVar) string {
	for _, v := range variables {
		s = strings.ReplaceAll(s, "{{"+v.Key+"}}", v.Value)
	}
	return s
}

// requestToSavedRequest converts a Postman request to a SavedRequest
func (p *PostmanImporter) requestToSavedRequest(req *PostmanRequest, name, collectionName, folderPath string, variables []PostmanVar, auth *PostmanAuth) *types.SavedRequest {
	// Extract URL and substitute variables
	url := p.extractURL(req.URL)
	url = p.substituteVars(url, variables)

	// Extract method
	method := req.Method
	if method == "" {
		method = "GET"
	}

	// Build headers
	var headers []types.Header
	for _, h := range req.Header {
		if !h.Disabled {
			headers = append(headers, types.Header{
				Key:   h.Key,
				Value: p.substituteVars(h.Value, variables),
			})
		}
	}

	// Add auth headers
	if auth != nil {
		p.addAuthHeaders(auth, &headers, variables)
	}

	// Extract body and substitute variables
	body := p.extractBody(req.Body)
	body = p.substituteVars(body, variables)

	// Build full name
	fullName := name
	if fullName == "" {
		fullName = method + " " + url
	}

	// Determine tags and collection
	var tags []string
	if folderPath != "" {
		tags = append(tags, strings.Split(folderPath, "/")...)
	}

	return &types.SavedRequest{
		Name:       fullName,
		URL:        url,
		Method:     method,
		Headers:    headers,
		Body:       body,
		Collection: collectionName,
		Tags:       tags,
	}
}

// extractURL extracts the URL from various URL representations
func (p *PostmanImporter) extractURL(url interface{}) string {
	switch v := url.(type) {
	case string:
		return v
	case map[string]any:
		// Try to construct URL from parts
		if raw, ok := v["raw"].(string); ok {
			return raw
		}
	case PostmanURL:
		return p.urlToString(&v)
	}
	return ""
}

// urlToString converts a PostmanURL to a string
func (p *PostmanImporter) urlToString(url *PostmanURL) string {
	if url == nil {
		return ""
	}

	if url.Raw != "" {
		return url.Raw
	}

	var parts []string

	// Build host
	if len(url.Host) > 0 {
		parts = append(parts, strings.Join(url.Host, "."))
	}

	// Build path
	if len(url.Path) > 0 {
		parts = append(parts, "/ "+strings.Join(url.Path, "/"))
	}

	// Build query
	if len(url.Query) > 0 {
		var queryParts []string
		for _, q := range url.Query {
			queryParts = append(queryParts, postmanQueryEscape(q.Key)+"="+postmanQueryEscape(q.Value))
		}
		parts = append(parts, "?"+strings.Join(queryParts, "&"))
	}

	return strings.Join(parts, "")
}

// postmanQueryEscape URL-encodes a string for use in query parameters
func postmanQueryEscape(s string) string {
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

// addAuthHeaders adds authentication headers
func (p *PostmanImporter) addAuthHeaders(auth *PostmanAuth, headers *[]types.Header, variables []PostmanVar) {
	switch auth.Type {
	case "bearer":
		for _, param := range auth.Bearer {
			if param.Key == "token" {
				*headers = append(*headers, types.Header{
					Key:   "Authorization",
					Value: "Bearer " + p.substituteVars(param.Value, variables),
				})
			}
		}
	case "basic":
		var username, password string
		for _, param := range auth.Basic {
			switch param.Key {
			case "username":
				username = p.substituteVars(param.Value, variables)
			case "password":
				password = p.substituteVars(param.Value, variables)
			}
		}
		if username != "" {
			encoded := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
			*headers = append(*headers, types.Header{
				Key:   "Authorization",
				Value: "Basic " + encoded,
			})
		}
	case "apikey":
		for _, param := range auth.APIKey {
			if param.Key != "" && param.Value != "" {
				*headers = append(*headers, types.Header{
					Key:   p.substituteVars(param.Key, variables),
					Value: p.substituteVars(param.Value, variables),
				})
			}
		}
	}
}

// extractBody extracts the body content from Postman body
func (p *PostmanImporter) extractBody(body *PostmanBody) string {
	if body == nil {
		return ""
	}

	switch body.Mode {
	case "raw":
		return body.Raw
	case "graphql":
		if body.GraphQL != nil {
			if body.GraphQL.Variables != "" {
				return fmt.Sprintf(`{"query": %s, "variables": %s}`,
					jsonString(body.GraphQL.Query),
					body.GraphQL.Variables)
			}
			return body.GraphQL.Query
		}
		return body.Raw
	case "urlencoded":
		var parts []string
		for _, param := range body.URLEncoded {
			if !param.Disabled {
				parts = append(parts, postmanQueryEscape(param.Key)+"="+postmanQueryEscape(param.Value))
			}
		}
		return strings.Join(parts, "&")
	case "formdata":
		var parts []string
		for _, param := range body.FormData {
			if !param.Disabled {
				parts = append(parts, postmanQueryEscape(param.Key)+"="+postmanQueryEscape(param.Value))
			}
		}
		return strings.Join(parts, "&")
	default:
		return body.Raw
	}
}

// jsonString wraps a string in quotes for JSON
func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
