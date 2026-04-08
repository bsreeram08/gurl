package importers

import (
	"os"
	"path/filepath"
	"testing"
)

// Insomnia export JSON with request group hierarchy
const sampleInsomniaExport = `{
  "_version": "2022.4.0",
  "resources": [
    {
      "_id": "grp_1234",
      "type": "request_group",
      "name": "Users",
      "parentId": "wrk_0000"
    },
    {
      "_id": "req_5678",
      "type": "request",
      "name": "List Users",
      "parentId": "grp_1234",
      "method": "GET",
      "url": "https://api.example.com/users",
      "headers": [
        {"id": "h1", "name": "Accept", "value": "application/json"}
      ]
    },
    {
      "_id": "req_9012",
      "type": "request",
      "name": "Create User",
      "parentId": "grp_1234",
      "method": "POST",
      "url": "https://api.example.com/users",
      "body": {
        "type": "json",
        "json": {"name": "John"}
      }
    }
  ]
}`

// Insomnia with bearer auth
const sampleInsomniaBearer = `{
  "_version": "2022.4.0",
  "resources": [
    {
      "_id": "req_auth",
      "type": "request",
      "name": "Authenticated Request",
      "method": "GET",
      "url": "https://api.example.com/private",
      "authentication": {
        "type": "bearer",
        "bearer": {
          "token": "secrettoken123"
        }
      }
    }
  ]
}`

// Insomnia with basic auth
const sampleInsomniaBasicAuth = `{
  "_version": "2022.4.0",
  "resources": [
    {
      "_id": "req_basic",
      "type": "request",
      "name": "Basic Auth Request",
      "method": "GET",
      "url": "https://api.example.com/basic",
      "authentication": {
        "type": "basic",
        "basic": {
          "username": "admin",
          "password": "secret"
        }
      }
    }
  ]
}`

// Insomnia with API key auth (header)
const sampleInsomniaAPIKeyHeader = `{
  "_version": "2022.4.0",
  "resources": [
    {
      "_id": "req_apikey",
      "type": "request",
      "name": "API Key Request",
      "method": "GET",
      "url": "https://api.example.com/apikey",
      "authentication": {
        "type": "apiKey",
        "apiKey": {
          "key": "X-API-Key",
          "value": "myapikey",
          "location": "header"
        }
      }
    }
  ]
}`

// Insomnia with query parameters
const sampleInsomniaQueryParams = `{
  "_version": "2022.4.0",
  "resources": [
    {
      "_id": "req_query",
      "type": "request",
      "name": "Query Params Request",
      "method": "GET",
      "url": "https://api.example.com/search",
      "parameters": [
        {"id": "p1", "name": "q", "value": "searchterm"},
        {"id": "p2", "name": "page", "value": "1"}
      ]
    }
  ]
}`

// Insomnia with graphql body type
const sampleInsomniaGraphQL = `{
  "_version": "2022.4.0",
  "resources": [
    {
      "_id": "req_gql",
      "type": "request",
      "name": "GraphQL Request",
      "method": "POST",
      "url": "https://api.example.com/graphql",
      "body": {
        "type": "graphql",
        "text": "{ user { id name } }"
      }
    }
  ]
}`

// Insomnia with different body types
const sampleInsomniaBodyTypes = `{
  "_version": "2022.4.0",
  "resources": [
    {
      "_id": "req_json",
      "type": "request",
      "name": "JSON Body",
      "method": "POST",
      "url": "https://api.example.com/json",
      "body": {
        "type": "json",
        "json": {"key": "value"}
      }
    },
    {
      "_id": "req_text",
      "type": "request",
      "name": "Text Body",
      "method": "POST",
      "url": "https://api.example.com/text",
      "body": {
        "type": "text",
        "text": "plain text body"
      }
    },
    {
      "_id": "req_form",
      "type": "request",
      "name": "Form Body",
      "method": "POST",
      "url": "https://api.example.com/form",
      "body": {
        "type": "form-urlencoded",
        "text": "user=john&pass=123"
      }
    }
  ]
}`

// Insomnia with tags
const sampleInsomniaTags = `{
  "_version": "2022.4.0",
  "resources": [
    {
      "_id": "req_tags",
      "type": "request",
      "name": "Tagged Request",
      "method": "GET",
      "url": "https://api.example.com/tags",
      "tags": ["important", "vip"]
    }
  ]
}`

// Insomnia with empty method (should default to GET)
const sampleInsomniaEmptyMethod = `{
  "_version": "2022.4.0",
  "resources": [
    {
      "_id": "req_empty",
      "type": "request",
      "name": "Empty Method",
      "url": "https://api.example.com/implicit"
    }
  ]
}`

// Invalid Insomnia JSON
const invalidInsomniaJSON = `{not valid json}`

// Insomnia with nested request groups
const sampleInsomniaNestedGroups = `{
  "_version": "2022.4.0",
  "resources": [
    {
      "_id": "wrk_0001",
      "type": "request_group",
      "name": "Root Folder",
      "parentId": null
    },
    {
      "_id": "grp_nested",
      "type": "request_group",
      "name": "Nested Folder",
      "parentId": "wrk_0001"
    },
    {
      "_id": "req_nested",
      "type": "request",
      "name": "Nested Request",
      "parentId": "grp_nested",
      "method": "POST",
      "url": "https://api.example.com/nested"
    }
  ]
}`

func TestInsomniaImporterName(t *testing.T) {
	i := &InsomniaImporter{}
	if i.Name() != "insomnia" {
		t.Errorf("got %q, want %q", i.Name(), "insomnia")
	}
}

func TestInsomniaImporterExtensions(t *testing.T) {
	i := &InsomniaImporter{}
	exts := i.Extensions()
	if len(exts) != 1 || exts[0] != ".json" {
		t.Errorf("got %v, want [.json]", exts)
	}
}

func TestInsomniaParseExport(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "insomnia.json")
	if err := os.WriteFile(tmpFile, []byte(sampleInsomniaExport), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	i := &InsomniaImporter{}
	requests, err := i.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 2 {
		t.Errorf("got %d requests, want 2", len(requests))
	}
}

func TestInsomniaInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.json")
	if err := os.WriteFile(tmpFile, []byte(invalidInsomniaJSON), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	i := &InsomniaImporter{}
	_, err := i.Parse(tmpFile)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestInsomniaNonexistentFile(t *testing.T) {
	i := &InsomniaImporter{}
	_, err := i.Parse("/nonexistent/path/insomnia.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestInsomniaConvertToRequests(t *testing.T) {
	i := &InsomniaImporter{}
	export := &InsomniaExport{
		Version: "2022.4.0",
		Resources: []InsomniaResource{
			{
				ID:   "grp_1",
				Type: "request_group",
				Name: "Folder",
			},
			{
				ID:       "req_1",
				Type:     "request",
				Name:     "Test Request",
				ParentID: "grp_1",
				Method:   "GET",
				URL:      "https://api.example.com/test",
			},
		},
	}

	requests := i.convertToRequests(export)
	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}
}

func TestInsomniaResourceToRequest(t *testing.T) {
	i := &InsomniaImporter{}
	res := &InsomniaResource{
		ID:       "req_1",
		Type:     "request",
		Name:     "Test Request",
		ParentID: "grp_1",
		Method:   "POST",
		URL:      "https://api.example.com/test",
		Headers: []InsomniaHeader{
			{ID: "h1", Name: "Content-Type", Value: "application/json"},
		},
	}
	folders := map[string]string{"grp_1": "Test Folder"}

	saved := i.resourceToRequest(res, folders)

	if saved.Name != "Test Request" {
		t.Errorf("got name %q, want %q", saved.Name, "Test Request")
	}
	if saved.Method != "POST" {
		t.Errorf("got method %q, want %q", saved.Method, "POST")
	}
	if saved.Collection != "Test Folder" {
		t.Errorf("got collection %q, want %q", saved.Collection, "Test Folder")
	}
	if len(saved.Headers) != 1 {
		t.Errorf("got %d headers, want 1", len(saved.Headers))
	}
}

func TestInsomniaBearerAuth(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "bearer.json")
	if err := os.WriteFile(tmpFile, []byte(sampleInsomniaBearer), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	i := &InsomniaImporter{}
	requests, err := i.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}

	// Check for Authorization header
	found := false
	for _, h := range requests[0].Headers {
		if h.Key == "Authorization" && h.Value == "Bearer secrettoken123" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find Authorization header with bearer token")
	}
}

func TestInsomniaBasicAuth(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "basic.json")
	if err := os.WriteFile(tmpFile, []byte(sampleInsomniaBasicAuth), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	i := &InsomniaImporter{}
	requests, err := i.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}
}

func TestInsomniaAPIKeyHeader(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "apikey.json")
	if err := os.WriteFile(tmpFile, []byte(sampleInsomniaAPIKeyHeader), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	i := &InsomniaImporter{}
	requests, err := i.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}

	// Check for X-API-Key header
	found := false
	for _, h := range requests[0].Headers {
		if h.Key == "X-API-Key" && h.Value == "myapikey" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find X-API-Key header")
	}
}

func TestInsomniaQueryParams(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "query.json")
	if err := os.WriteFile(tmpFile, []byte(sampleInsomniaQueryParams), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	i := &InsomniaImporter{}
	requests, err := i.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}

	// URL should include query params
	if requests[0].URL != "https://api.example.com/search?q=searchterm&page=1" {
		t.Errorf("got URL %q", requests[0].URL)
	}
}

func TestInsomniaGraphQLBody(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "gql.json")
	if err := os.WriteFile(tmpFile, []byte(sampleInsomniaGraphQL), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	i := &InsomniaImporter{}
	requests, err := i.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}

	if requests[0].Body != "{ user { id name } }" {
		t.Errorf("got body %q", requests[0].Body)
	}
}

func TestInsomniaBodyTypes(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "bodies.json")
	if err := os.WriteFile(tmpFile, []byte(sampleInsomniaBodyTypes), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	i := &InsomniaImporter{}
	requests, err := i.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 3 {
		t.Errorf("got %d requests, want 3", len(requests))
	}
}

func TestInsomniaExtractBody(t *testing.T) {
	i := &InsomniaImporter{}

	tests := []struct {
		name string
		body *InsomniaBody
		want string
	}{
		{
			name: "nil body",
			body: nil,
			want: "",
		},
		{
			name: "json body with map",
			body: &InsomniaBody{Type: "json", JSON: map[string]any{"key": "value"}},
			want: `{"key":"value"}`,
		},
		{
			name: "json body with string",
			body: &InsomniaBody{Type: "json", JSON: "raw json string"},
			want: "raw json string",
		},
		{
			name: "json body with nil json, has text",
			body: &InsomniaBody{Type: "json", Text: "fallback text"},
			want: "fallback text",
		},
		{
			name: "graphql body",
			body: &InsomniaBody{Type: "graphql", Text: "{ query { user } }"},
			want: "{ query { user } }",
		},
		{
			name: "text body",
			body: &InsomniaBody{Type: "text", Text: "plain text"},
			want: "plain text",
		},
		{
			name: "unknown type",
			body: &InsomniaBody{Type: "binary", Text: "binary data"},
			want: "binary data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := i.extractBody(tt.body)
			if got != tt.want {
				t.Errorf("extractBody() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInsomniaTags(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "tags.json")
	if err := os.WriteFile(tmpFile, []byte(sampleInsomniaTags), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	i := &InsomniaImporter{}
	requests, err := i.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}

	if len(requests[0].Tags) != 2 {
		t.Errorf("got %d tags, want 2", len(requests[0].Tags))
	}
}

func TestInsomniaEmptyMethod(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "emptymethod.json")
	if err := os.WriteFile(tmpFile, []byte(sampleInsomniaEmptyMethod), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	i := &InsomniaImporter{}
	requests, err := i.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}

	// Should default to GET
	if requests[0].Method != "GET" {
		t.Errorf("got method %q, want %q", requests[0].Method, "GET")
	}
}

func TestInsomniaNestedGroups(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "nested.json")
	if err := os.WriteFile(tmpFile, []byte(sampleInsomniaNestedGroups), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	i := &InsomniaImporter{}
	requests, err := i.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}

	if requests[0].Collection != "Nested Folder" {
		t.Errorf("got collection %q, want %q", requests[0].Collection, "Nested Folder")
	}
}

func TestInsomniaOnlyRequestsProcessed(t *testing.T) {
	i := &InsomniaImporter{}
	export := &InsomniaExport{
		Version: "2022.4.0",
		Resources: []InsomniaResource{
			{ID: "env_1", Type: "environment", Name: "Dev Env"},
			{ID: "spc_1", Type: "space", Name: "Workspace"},
			{ID: "req_1", Type: "request", Name: "My Request", Method: "GET", URL: "https://example.com"},
			{ID: "grpc_1", Type: "grpc_request", Name: "gRPC Request"},
		},
	}

	requests := i.convertToRequests(export)

	// Only "request" type should be processed
	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}
}
