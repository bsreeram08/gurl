package importers

import (
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

	var collection PostmanCollection
	if err := json.Unmarshal(data, &collection); err != nil {
		return nil, fmt.Errorf("parse postman collection: %w", err)
	}

	return p.convertToRequests(&collection), nil
}

// convertToRequests transforms a Postman collection into SavedRequests
func (p *PostmanImporter) convertToRequests(collection *PostmanCollection) []*types.SavedRequest {
	var requests []*types.SavedRequest

	// Process all items recursively
	p.processItems(collection.Item, collection.Info.Name, "", &requests)

	return requests
}

// processItems recursively processes Postman items
func (p *PostmanImporter) processItems(items []PostmanItem, collectionName, folderPath string, requests *[]*types.SavedRequest) {
	for _, item := range items {
		if item.Request != nil {
			// This is a request
			req := p.requestToSavedRequest(item.Request, item.Name, collectionName, folderPath)
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
			p.processItems(item.Item, collectionName, newFolderPath, requests)
		}
	}
}

// requestToSavedRequest converts a Postman request to a SavedRequest
func (p *PostmanImporter) requestToSavedRequest(req *PostmanRequest, name, collectionName, folderPath string) *types.SavedRequest {
	// Extract URL
	url := p.extractURL(req.URL)

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
				Value: h.Value,
			})
		}
	}

	// Add auth headers
	if req.Auth != nil {
		p.addAuthHeaders(req.Auth, &headers)
	}

	// Extract body
	body := p.extractBody(req.Body)

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
		parts = append(parts, "/"+strings.Join(url.Path, "/"))
	}

	// Build query
	if len(url.Query) > 0 {
		var queryParts []string
		for _, q := range url.Query {
			queryParts = append(queryParts, q.Key+"="+q.Value)
		}
		parts = append(parts, "?"+strings.Join(queryParts, "&"))
	}

	return strings.Join(parts, "")
}

// addAuthHeaders adds authentication headers
func (p *PostmanImporter) addAuthHeaders(auth *PostmanAuth, headers *[]types.Header) {
	switch auth.Type {
	case "bearer":
		for _, param := range auth.Bearer {
			if param.Key == "token" {
				*headers = append(*headers, types.Header{
					Key:   "Authorization",
					Value: "Bearer " + param.Value,
				})
			}
		}
	case "basic":
		var username, password string
		for _, param := range auth.Basic {
			switch param.Key {
			case "username":
				username = param.Value
			case "password":
				password = param.Value
			}
		}
		if username != "" {
			*headers = append(*headers, types.Header{
				Key:   "Authorization",
				Value: "Basic " + basicAuth(username, password),
			})
		}
	case "apikey":
		for _, param := range auth.APIKey {
			if param.Key != "" && param.Value != "" {
				*headers = append(*headers, types.Header{
					Key:   param.Key,
					Value: param.Value,
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
				parts = append(parts, param.Key+"="+param.Value)
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
