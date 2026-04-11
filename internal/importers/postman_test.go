package importers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sreeram/gurl/pkg/types"
)

// Postman v2.1 collection with nested structure
const samplePostmanCollection = `{
  "info": {
    "name": "Test API Collection",
    "description": "A test collection",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "item": [
    {
      "name": "Users Folder",
      "item": [
        {
          "name": "List Users",
          "request": {
            "method": "GET",
            "header": [],
            "url": {
              "raw": "https://api.example.com/users?page=1",
              "host": ["api", "example", "com"],
              "path": ["users"],
              "query": [{"key": "page", "value": "1"}]
            }
          }
        },
        {
          "name": "Create User",
          "request": {
            "method": "POST",
            "header": [
              {"key": "Content-Type", "value": "application/json"}
            ],
            "url": "https://api.example.com/users",
            "body": {
              "mode": "raw",
              "raw": "{\"name\": \"John\", \"email\": \"john@example.com\"}"
            }
          }
        }
      ]
    },
    {
      "name": "Get Single User",
      "request": {
        "method": "GET",
        "url": "https://api.example.com/users/123"
      }
    },
    {
      "name": "Update User",
      "request": {
        "method": "PUT",
        "url": "https://api.example.com/users/123",
        "body": {
          "mode": "raw",
          "raw": "{\"name\": \"Jane\"}"
        }
      }
    },
    {
      "name": "Delete User",
      "request": {
        "method": "DELETE",
        "url": "https://api.example.com/users/123"
      }
    }
  ]
}`

// Postman v2.0 collection format
const samplePostmanV2Collection = `{
  "info": {
    "name": "V2 Collection",
    "schema": "https://schema.getpostman.com/json/collection/v2.0.0/collection.json"
  },
  "item": [
    {
      "name": "Simple GET",
      "request": {
        "method": "GET",
        "url": "https://httpbin.org/get"
      }
    }
  ]
}`

// Postman collection with auth (bearer, basic, apikey)
const samplePostmanAuthCollection = `{
  "info": {
    "name": "Auth Collection",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "item": [
    {
      "name": "Bearer Auth Request",
      "request": {
        "method": "GET",
        "url": "https://api.example.com/private",
        "auth": {
          "type": "bearer",
          "bearer": [{"key": "token", "value": "abc123"}]
        }
      }
    },
    {
      "name": "Basic Auth Request",
      "request": {
        "method": "GET",
        "url": "https://api.example.com/basic",
        "auth": {
          "type": "basic",
          "basic": [
            {"key": "username", "value": "admin"},
            {"key": "password", "value": "secret"}
          ]
        }
      }
    },
    {
      "name": "API Key Request",
      "request": {
        "method": "GET",
        "url": "https://api.example.com/apikey",
        "auth": {
          "type": "apikey",
          "apikey": [
            {"key": "X-API-Key", "value": "myapikey"}
          ]
        }
      }
    }
  ]
}`

// Postman collection with URL-encoded body
const samplePostmanURLEncodedCollection = `{
  "info": {
    "name": "URL Encoded Collection",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "item": [
    {
      "name": "URL Encoded POST",
      "request": {
        "method": "POST",
        "url": "https://api.example.com/login",
        "body": {
          "mode": "urlencoded",
          "urlencoded": [
            {"key": "username", "value": "john"},
            {"key": "password", "value": "pass123"}
          ]
        }
      }
    }
  ]
}`

// Postman collection with GraphQL body
const samplePostmanGraphQLCollection = `{
  "info": {
    "name": "GraphQL Collection",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "item": [
    {
      "name": "GraphQL Query",
      "request": {
        "method": "POST",
        "url": "https://api.example.com/graphql",
        "body": {
          "mode": "graphql",
          "graphql": {
            "query": "query { user(id: 1) { name } }",
            "variables": "{\"id\": 1}"
          }
        }
      }
    }
  ]
}`

// Postman with disabled header
const samplePostmanDisabledHeader = `{
  "info": {
    "name": "Disabled Header Collection",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "item": [
    {
      "name": "Request with Disabled Header",
      "request": {
        "method": "GET",
        "header": [
          {"key": "X-Enabled", "value": "yes"},
          {"key": "X-Disabled", "value": "no", "disabled": true}
        ],
        "url": "https://api.example.com/test"
      }
    }
  ]
}`

// Postman with URL as raw string
const samplePostmanRawURL = `{
  "info": {
    "name": "Raw URL Collection",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "item": [
    {
      "name": "Raw URL Request",
      "request": {
        "method": "GET",
        "url": "https://api.example.com/users?id=123&filter=active"
      }
    }
  ]
}`

// Invalid Postman JSON
const invalidPostmanJSON = `{not valid json at all}`

// Postman with empty name (should use method + url)
const samplePostmanEmptyName = `{
  "info": {
    "name": "Empty Name Collection",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "item": [
    {
      "request": {
        "method": "POST",
        "url": "https://api.example.com/test"
      }
    }
  ]
}`

// Postman with missing method (should default to GET)
const samplePostmanMissingMethod = `{
  "info": {
    "name": "Missing Method Collection",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "item": [
    {
      "name": "No Method Specified",
      "request": {
        "url": "https://api.example.com/implicit-get"
      }
    }
  ]
}`

func TestPostmanImporterName(t *testing.T) {
	p := &PostmanImporter{}
	if p.Name() != "postman" {
		t.Errorf("got %q, want %q", p.Name(), "postman")
	}
}

func TestPostmanImporterExtensions(t *testing.T) {
	p := &PostmanImporter{}
	exts := p.Extensions()
	if len(exts) != 1 || exts[0] != ".json" {
		t.Errorf("got %v, want [.json]", exts)
	}
}

func TestPostmanParseCollection(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "collection.json")
	if err := os.WriteFile(tmpFile, []byte(samplePostmanCollection), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	p := &PostmanImporter{}
	requests, err := p.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 5 {
		t.Errorf("got %d requests, want 5", len(requests))
	}
}

func TestPostmanParseV2Collection(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "collection.json")
	if err := os.WriteFile(tmpFile, []byte(samplePostmanV2Collection), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	p := &PostmanImporter{}
	requests, err := p.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}
}

func TestPostmanInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.json")
	if err := os.WriteFile(tmpFile, []byte(invalidPostmanJSON), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	p := &PostmanImporter{}
	_, err := p.Parse(tmpFile)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestPostmanNonexistentFile(t *testing.T) {
	p := &PostmanImporter{}
	_, err := p.Parse("/nonexistent/path/collection.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestPostmanConvertToRequests(t *testing.T) {
	p := &PostmanImporter{}
	collection := &PostmanCollection{
		Info: PostmanInfo{Name: "Test", Schema: "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},
		Item: []PostmanItem{
			{
				Name: "Test Request",
				Request: &PostmanRequest{
					Method: "GET",
					URL:    "https://api.example.com/test",
				},
			},
		},
	}

	requests := p.convertToRequests(collection)
	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}
}

func TestPostmanProcessItems(t *testing.T) {
	p := &PostmanImporter{}
	var requests []*types.SavedRequest

	items := []PostmanItem{
		{
			Name: "Folder 1",
			Item: []PostmanItem{
				{
					Name: "Nested Request",
					Request: &PostmanRequest{
						Method: "POST",
						URL:    "https://api.example.com/nested",
					},
				},
			},
		},
	}

	p.processItems(items, "Collection", "path/to/folder", nil, nil, &requests)

	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}

	if requests[0].Name != "Nested Request" {
		t.Errorf("got name %q, want %q", requests[0].Name, "Nested Request")
	}

	if len(requests[0].Tags) == 0 {
		t.Error("expected tags to be populated")
	}
}

func TestPostmanRequestToSavedRequest(t *testing.T) {
	p := &PostmanImporter{}
	req := &PostmanRequest{
		Method: "POST",
		URL:    "https://api.example.com/test",
		Header: []PostmanHeader{
			{Key: "Content-Type", Value: "application/json"},
		},
	}

	saved := p.requestToSavedRequest(req, "Test Request", "Test Collection", "folder/subfolder", nil, nil)

	if saved.Name != "Test Request" {
		t.Errorf("got name %q, want %q", saved.Name, "Test Request")
	}
	if saved.Method != "POST" {
		t.Errorf("got method %q, want %q", saved.Method, "POST")
	}
	if saved.Collection != "Test Collection" {
		t.Errorf("got collection %q, want %q", saved.Collection, "Test Collection")
	}
}

func TestPostmanExtractURL(t *testing.T) {
	p := &PostmanImporter{}

	tests := []struct {
		name string
		url  interface{}
		want string
	}{
		{
			name: "string URL",
			url:  "https://api.example.com/string",
			want: "https://api.example.com/string",
		},
		{
			name: "map with raw",
			url: map[string]any{
				"raw": "https://api.example.com/map",
			},
			want: "https://api.example.com/map",
		},
		{
			name: "PostmanURL struct",
			url: PostmanURL{
				Raw:  "https://api.example.com/raw",
				Host: []string{"api", "example", "com"},
				Path: []string{"path"},
			},
			want: "https://api.example.com/raw",
		},
		{
			name: "nil URL",
			url:  nil,
			want: "",
		},
		{
			name: "empty map",
			url:  map[string]any{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.extractURL(tt.url)
			if got != tt.want {
				t.Errorf("extractURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPostmanURLToString(t *testing.T) {
	p := &PostmanImporter{}

	tests := []struct {
		name string
		url  *PostmanURL
		want string
	}{
		{
			name: "nil URL",
			url:  nil,
			want: "",
		},
		{
			name: "raw URL",
			url:  &PostmanURL{Raw: "https://api.example.com/raw"},
			want: "https://api.example.com/raw",
		},
		{
			name: "host and path",
			url: &PostmanURL{
				Host: []string{"api", "example", "com"},
				Path: []string{"users", "123"},
			},
			want: "api.example.com/users/123", // path has leading /
		},
		{
			name: "with query",
			url: &PostmanURL{
				Host:  []string{"api", "example", "com"},
				Path:  []string{"users"},
				Query: []PostmanQueryParam{{Key: "page", Value: "1"}},
			},
			want: "api.example.com/users?page=1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.urlToString(tt.url)
			if got != tt.want {
				t.Errorf("urlToString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPostmanAddAuthHeaders(t *testing.T) {
	p := &PostmanImporter{}

	tests := []struct {
		name           string
		auth           *PostmanAuth
		wantAuthHeader bool
		wantKey        string
		wantValue      string
	}{
		{
			name: "bearer token",
			auth: &PostmanAuth{
				Type:   "bearer",
				Bearer: []PostmanAuthParam{{Key: "token", Value: "abc123"}},
			},
			wantAuthHeader: true,
			wantKey:        "Authorization",
			wantValue:      "Bearer abc123",
		},
		{
			name: "basic auth",
			auth: &PostmanAuth{
				Type:  "basic",
				Basic: []PostmanAuthParam{{Key: "username", Value: "user"}, {Key: "password", Value: "pass"}},
			},
			wantAuthHeader: true,
			wantKey:        "Authorization",
			wantValue:      "Basic user:pass", // simple concat, not base64
		},
		{
			name: "apikey",
			auth: &PostmanAuth{
				Type:   "apikey",
				APIKey: []PostmanAuthParam{{Key: "X-API-Key", Value: "key123"}},
			},
			wantAuthHeader: true,
			wantKey:        "X-API-Key",
			wantValue:      "key123",
		},
		{
			name:           "nil auth",
			auth:           nil,
			wantAuthHeader: false,
		},
		{
			name: "unsupported type",
			auth: &PostmanAuth{
				Type: "digest",
			},
			wantAuthHeader: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := []types.Header{}
			if tt.auth != nil {
				p.addAuthHeaders(tt.auth, &headers, nil)
			}

			if tt.wantAuthHeader {
				if len(headers) == 0 {
					t.Error("expected headers but got none")
				} else if headers[0].Key != tt.wantKey {
					t.Errorf("got key %q, want %q", headers[0].Key, tt.wantKey)
				} else if headers[0].Value != tt.wantValue {
					t.Errorf("got value %q, want %q", headers[0].Value, tt.wantValue)
				}
			} else {
				if len(headers) != 0 {
					t.Errorf("expected no headers but got %d", len(headers))
				}
			}
		})
	}
}

func TestPostmanExtractBody(t *testing.T) {
	p := &PostmanImporter{}

	tests := []struct {
		name string
		body *PostmanBody
		want string
	}{
		{
			name: "nil body",
			body: nil,
			want: "",
		},
		{
			name: "raw body",
			body: &PostmanBody{Mode: "raw", Raw: `{"key": "value"}`},
			want: `{"key": "value"}`,
		},
		{
			name: "graphql body",
			body: &PostmanBody{
				Mode: "graphql",
				GraphQL: &PostmanGraphQL{
					Query:     "{ user { id } }",
					Variables: `{"id": 1}`,
				},
			},
			want: `{"query": "{ user { id } }", "variables": {"id": 1}}`,
		},
		{
			name: "urlencoded body",
			body: &PostmanBody{
				Mode:       "urlencoded",
				URLEncoded: []PostmanFormParam{{Key: "user", Value: "john"}, {Key: "pass", Value: "123"}},
			},
			want: "user=john&pass=123",
		},
		{
			name: "urlencoded with disabled",
			body: &PostmanBody{
				Mode:       "urlencoded",
				URLEncoded: []PostmanFormParam{{Key: "enbled", Value: "yes", Disabled: false}, {Key: "disabled", Value: "no", Disabled: true}},
			},
			want: "enbled=yes",
		},
		{
			name: "graphql with empty variables",
			body: &PostmanBody{
				Mode:    "graphql",
				GraphQL: &PostmanGraphQL{Query: "{ user { id } }"},
			},
			want: "{ user { id } }",
		},
		{
			name: "default case",
			body: &PostmanBody{Mode: "unknown", Raw: "something"},
			want: "something",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.extractBody(tt.body)
			if got != tt.want {
				t.Errorf("extractBody() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPostmanJsonString(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`hello`, `"hello"`},
		{`"quoted"`, `"\"quoted\""`},
		{``, `""`},
	}

	for _, tt := range tests {
		got := jsonString(tt.input)
		if got != tt.want {
			t.Errorf("jsonString(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestPostmanCollectionWithAuth(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "auth.json")
	if err := os.WriteFile(tmpFile, []byte(samplePostmanAuthCollection), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	p := &PostmanImporter{}
	requests, err := p.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 3 {
		t.Errorf("got %d requests, want 3", len(requests))
	}

	// Check that auth headers were added
	for _, req := range requests {
		hasAuth := false
		for _, h := range req.Headers {
			if h.Key == "Authorization" {
				hasAuth = true
				break
			}
		}
		if !hasAuth && req.Name != "API Key Request" {
			// API key adds header with key name, not Authorization
		}
	}
}

func TestPostmanURLEncodedBody(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "urlencoded.json")
	if err := os.WriteFile(tmpFile, []byte(samplePostmanURLEncodedCollection), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	p := &PostmanImporter{}
	requests, err := p.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}

	if requests[0].Body != "username=john&password=pass123" {
		t.Errorf("got body %q, want %q", requests[0].Body, "username=john&password=pass123")
	}
}

func TestPostmanGraphQLBody(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "graphql.json")
	if err := os.WriteFile(tmpFile, []byte(samplePostmanGraphQLCollection), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	p := &PostmanImporter{}
	requests, err := p.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}
}

func TestPostmanDisabledHeader(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "disabled.json")
	if err := os.WriteFile(tmpFile, []byte(samplePostmanDisabledHeader), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	p := &PostmanImporter{}
	requests, err := p.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}

	// Should only have the enabled header
	if len(requests[0].Headers) != 1 {
		t.Errorf("got %d headers, want 1", len(requests[0].Headers))
	}

	if requests[0].Headers[0].Key != "X-Enabled" {
		t.Errorf("got header key %q, want %q", requests[0].Headers[0].Key, "X-Enabled")
	}
}

func TestPostmanRawURL(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "rawurl.json")
	if err := os.WriteFile(tmpFile, []byte(samplePostmanRawURL), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	p := &PostmanImporter{}
	requests, err := p.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}

	if requests[0].URL != "https://api.example.com/users?id=123&filter=active" {
		t.Errorf("got URL %q", requests[0].URL)
	}
}

func TestPostmanEmptyName(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "empty.json")
	if err := os.WriteFile(tmpFile, []byte(samplePostmanEmptyName), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	p := &PostmanImporter{}
	requests, err := p.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}

	// Empty name should fall back to method + URL
	if requests[0].Name == "" {
		t.Error("expected non-empty name fallback")
	}
}

func TestPostmanMissingMethod(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "missingmethod.json")
	if err := os.WriteFile(tmpFile, []byte(samplePostmanMissingMethod), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	p := &PostmanImporter{}
	requests, err := p.Parse(tmpFile)
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
