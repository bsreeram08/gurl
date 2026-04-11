package importers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sreeram/gurl/pkg/types"
)

// InsomniaImporter handles Insomnia JSON export files
type InsomniaImporter struct{}

// Name returns the importer name
func (i *InsomniaImporter) Name() string {
	return "insomnia"
}

// Extensions returns supported file extensions
func (i *InsomniaImporter) Extensions() []string {
	return []string{".json"}
}

// InsomniaExport represents an Insomnia export structure
type InsomniaExport struct {
	Version string       `json:"_version"`
	XML     *InsomniaXML `json:"_xml,omitempty"`
	Resources []InsomniaResource `json:"resources"`
}

// InsomniaXML contains XML export info
type InsomniaXML struct {
	ExportAngle  string `json:"exportAngle"`
	Extension    string `json:"extension"`
	ExportedWith string `json:"exportedWith"`
}

// InsomniaResource represents any resource in Insomnia
type InsomniaResource struct {
	ID      string `json:"_id"`
	Type    string `json:"type"`

	// Common fields
	Name        string              `json:"name,omitempty"`
	Description string              `json:"description,omitempty"`
	Tags        []string            `json:"tags,omitempty"`

	// Request/RequestGroup fields
	URL         string                 `json:"url,omitempty"`
	Method      string                 `json:"method,omitempty"`
	Headers     []InsomniaHeader       `json:"headers,omitempty"`
	Body        *InsomniaBody          `json:"body,omitempty"`
	Parameters  []InsomniaParameter    `json:"parameters,omitempty"`

	// Authentication
	Authentication *InsomniaAuth       `json:"authentication,omitempty"`

	// For folders/collections
	ParentID string `json:"parentId,omitempty"`
}

// InsomniaHeader represents a header
type InsomniaHeader struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

// InsomniaBody represents the request body
type InsomniaBody struct {
	Type        string `json:"type,omitempty"` // json, file, form-urlencoded, form-multipart, graphql
	Text        string `json:"text,omitempty"`
	JSON        any    `json:"json,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// InsomniaParameter represents a query parameter
type InsomniaParameter struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

// InsomniaAuth represents authentication config
type InsomniaAuth struct {
	Type        string            `json:"type,omitempty"`
	Basic       *AuthBasic        `json:"basic,omitempty"`
	Bearer      *AuthBearer       `json:"bearer,omitempty"`
	APIKey      *AuthAPIKey       `json:"apiKey,omitempty"`
}

type AuthBasic struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type AuthBearer struct {
	Token string `json:"token,omitempty"`
}

type AuthAPIKey struct {
	Value  string `json:"value,omitempty"`
	Key    string `json:"key,omitempty"`
	Location string `json:"location,omitempty"` // header, query
}

// Parse reads and parses an Insomnia export file
func (i *InsomniaImporter) Parse(path string) ([]*types.SavedRequest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var export InsomniaExport
	if err := json.Unmarshal(data, &export); err != nil {
		return nil, fmt.Errorf("parse insomnia export: %w", err)
	}

	return i.convertToRequests(&export), nil
}

// convertToRequests transforms Insomnia resources into SavedRequests
func (i *InsomniaImporter) convertToRequests(export *InsomniaExport) []*types.SavedRequest {
	var requests []*types.SavedRequest

	// Build folder lookup for collection names
	folders := make(map[string]string)
	for _, res := range export.Resources {
		if res.Type == "request_group" {
			folders[res.ID] = res.Name
		}
	}

	// Process each request
	for _, res := range export.Resources {
		if res.Type != "request" {
			continue
		}

		req := i.resourceToRequest(&res, folders)
		requests = append(requests, req)
	}

	return requests
}

// resourceToRequest converts an Insomnia resource to a SavedRequest
func (i *InsomniaImporter) resourceToRequest(res *InsomniaResource, folders map[string]string) *types.SavedRequest {
	// Build headers
	var headers []types.Header
	for _, h := range res.Headers {
		headers = append(headers, types.Header{
			Key:   h.Name,
			Value: h.Value,
		})
	}

	// Add auth headers if present
	if res.Authentication != nil {
		switch {
		case res.Authentication.Basic != nil:
			// Basic auth - encode credentials
			encoded := base64.StdEncoding.EncodeToString([]byte(res.Authentication.Basic.Username + ":" + res.Authentication.Basic.Password))
			headers = append(headers, types.Header{
				Key:   "Authorization",
				Value: "Basic " + encoded,
			})
		case res.Authentication.Bearer != nil:
			headers = append(headers, types.Header{
				Key:   "Authorization",
				Value: "Bearer " + res.Authentication.Bearer.Token,
			})
		case res.Authentication.APIKey != nil && res.Authentication.APIKey.Location == "header":
			headers = append(headers, types.Header{
				Key:   res.Authentication.APIKey.Key,
				Value: res.Authentication.APIKey.Value,
			})
		}
	}

	// Build URL with query parameters
	url := res.URL
	if len(res.Parameters) > 0 {
		var params []string
		for _, p := range res.Parameters {
			params = append(params, fmt.Sprintf("%s=%s", p.Name, p.Value))
		}
		if len(params) > 0 {
			url += "?" + strings.Join(params, "&")
		}
	}

	// Extract body
	body := i.extractBody(res.Body)

	// Determine collection
	collection := ""
	if res.ParentID != "" {
		collection = folders[res.ParentID]
	}

	// Determine tags
	var tags []string
	if collection != "" {
		tags = append(tags, collection)
	}
	tags = append(tags, res.Tags...)

	// Determine method
	method := res.Method
	if method == "" {
		method = "GET"
	}

	return &types.SavedRequest{
		Name:       res.Name,
		URL:        url,
		Method:     method,
		Headers:    headers,
		Body:       body,
		Collection: collection,
		Tags:       tags,
	}
}

// extractBody extracts the body content from Insomnia body
func (i *InsomniaImporter) extractBody(body *InsomniaBody) string {
	if body == nil {
		return ""
	}

	switch body.Type {
	case "json":
		if body.JSON != nil {
			switch v := body.JSON.(type) {
			case string:
				return v
			default:
				data, _ := json.Marshal(v)
				return string(data)
			}
		}
		return body.Text
	case "graphql":
		// GraphQL body text
		return body.Text
	default:
		return body.Text
	}
}
