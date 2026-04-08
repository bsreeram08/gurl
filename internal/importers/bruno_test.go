package importers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sreeram/gurl/pkg/types"
)

// Sample .bru file content
const sampleBrunoFile = `meta {
  name: List Users
  method: GET
  url: https://api.example.com/users
}

headers {
  Content-Type: application/json
  Accept: application/json
}

body {
  none
}
`

// Bruno file with bearer auth
const sampleBrunoBearerAuth = `meta {
  name: Protected Endpoint
  method: GET
  url: https://api.example.com/private
}

headers {
}

auth {
  type: bearer
  token: secrettoken123
}
`

// Bruno file with basic auth
const sampleBrunoBasicAuth = `meta {
  name: Basic Auth Endpoint
  method: GET
  url: https://api.example.com/basic
}

auth {
  type: basic
  username: admin
  password: secretpass
}
`

// Bruno file with body content
const sampleBrunoWithBody = `meta {
  name: Create User
  method: POST
  url: https://api.example.com/users
}

headers {
  Content-Type: application/json
}

body {
  {
    "name": "John Doe",
    "email": "john@example.com"
  }
}
`

// Bruno file with query params in URL
const sampleBrunoQueryParams = `meta {
  name: Search Users
  method: GET
  url: https://api.example.com/users?page=1&limit=10
}

headers {
}
`

// Bruno file with vars
const sampleBrunoWithVars = `meta {
  name: User Request
  method: GET
  url: https://api.example.com/users
}

vars {
  baseUrl: https://api.example.com
  userId: 123
}
`

// Bruno file with minimal content
const sampleBrunoMinimal = `meta {
  name: Minimal Request
  method: GET
  url: https://example.com
}
`

// Bruno file with no name (should use filename)
const sampleBrunoNoName = `meta {
  method: POST
  url: https://api.example.com/test
}
`

// Bruno file with script section
const sampleBrunoWithScript = `meta {
  name: Scripted Request
  method: POST
  url: https://api.example.com/scripted
}

script {
  console.log("Before request");
}

body {
  {"action": "run"}
}
`

// Invalid .bru file (not a directory or .bru file)
const invalidBrunoPath = "/some/path.txt"

func TestBrunoImporterName(t *testing.T) {
	b := &BrunoImporter{}
	if b.Name() != "bruno" {
		t.Errorf("got %q, want %q", b.Name(), "bruno")
	}
}

func TestBrunoImporterExtensions(t *testing.T) {
	b := &BrunoImporter{}
	exts := b.Extensions()
	if len(exts) != 1 || exts[0] != ".bru" {
		t.Errorf("got %v, want [.bru]", exts)
	}
}

func TestBrunoParseFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "list-users.bru")
	if err := os.WriteFile(tmpFile, []byte(sampleBrunoFile), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	b := &BrunoImporter{}
	requests, err := b.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}

	if requests[0].Name != "List Users" {
		t.Errorf("got name %q, want %q", requests[0].Name, "List Users")
	}

	if requests[0].Method != "GET" {
		t.Errorf("got method %q, want %q", requests[0].Method, "GET")
	}

	if requests[0].URL != "https://api.example.com/users" {
		t.Errorf("got URL %q, want %q", requests[0].URL, "https://api.example.com/users")
	}

	// Should have headers
	if len(requests[0].Headers) != 2 {
		t.Errorf("got %d headers, want 2", len(requests[0].Headers))
	}
}

func TestBrunoParseDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple .bru files
	files := map[string]string{
		"get-users.bru":   sampleBrunoMinimal,
		"create-user.bru": sampleBrunoWithBody,
	}

	for name, content := range files {
		fpath := filepath.Join(tmpDir, name)
		if err := os.WriteFile(fpath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write temp file: %v", err)
		}
	}

	// Create a non-.bru file that should be ignored
	ignoredFile := filepath.Join(tmpDir, "readme.txt")
	if err := os.WriteFile(ignoredFile, []byte("this should be ignored"), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	b := &BrunoImporter{}
	requests, err := b.Parse(tmpDir)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 2 {
		t.Errorf("got %d requests, want 2", len(requests))
	}
}

func TestBrunoBearerAuth(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "bearer.bru")
	if err := os.WriteFile(tmpFile, []byte(sampleBrunoBearerAuth), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	b := &BrunoImporter{}
	requests, err := b.Parse(tmpFile)
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

func TestBrunoBasicAuth(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "basic.bru")
	if err := os.WriteFile(tmpFile, []byte(sampleBrunoBasicAuth), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	b := &BrunoImporter{}
	requests, err := b.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}

	// Check for Authorization header with basic auth
	found := false
	for _, h := range requests[0].Headers {
		if h.Key == "Authorization" {
			found = true
			// Value should be admin:secretpass
			break
		}
	}
	if !found {
		t.Error("expected to find Authorization header with basic auth")
	}
}

func TestBrunoWithBody(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "body.bru")
	if err := os.WriteFile(tmpFile, []byte(sampleBrunoWithBody), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	b := &BrunoImporter{}
	requests, err := b.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}

	if requests[0].Body == "" {
		t.Error("expected non-empty body")
	}
}

func TestBrunoQueryParams(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "query.bru")
	if err := os.WriteFile(tmpFile, []byte(sampleBrunoQueryParams), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	b := &BrunoImporter{}
	requests, err := b.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}

	if requests[0].URL != "https://api.example.com/users?page=1&limit=10" {
		t.Errorf("got URL %q", requests[0].URL)
	}
}

func TestBrunoWithVars(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "vars.bru")
	if err := os.WriteFile(tmpFile, []byte(sampleBrunoWithVars), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	b := &BrunoImporter{}
	requests, err := b.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}

	// Vars are parsed but currently not used in SavedRequest
}

func TestBrunoMinimal(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "minimal.bru")
	if err := os.WriteFile(tmpFile, []byte(sampleBrunoMinimal), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	b := &BrunoImporter{}
	requests, err := b.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}

	if requests[0].Name != "Minimal Request" {
		t.Errorf("got name %q, want %q", requests[0].Name, "Minimal Request")
	}
}

func TestBrunoNoName(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "unnamed-request.bru")
	if err := os.WriteFile(tmpFile, []byte(sampleBrunoNoName), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	b := &BrunoImporter{}
	requests, err := b.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}

	// Should fall back to filename
	if requests[0].Name != "unnamed-request" {
		t.Errorf("got name %q, want %q", requests[0].Name, "unnamed-request")
	}
}

func TestBrunoWithScript(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "scripted.bru")
	if err := os.WriteFile(tmpFile, []byte(sampleBrunoWithScript), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	b := &BrunoImporter{}
	requests, err := b.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}
}

func TestBrunoInvalidPath(t *testing.T) {
	b := &BrunoImporter{}
	_, err := b.Parse("/nonexistent/path/to/file.bru")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestBrunoNonBruFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(tmpFile, []byte("not a .bru file"), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	b := &BrunoImporter{}
	_, err := b.Parse(tmpFile)
	if err == nil {
		t.Error("expected error for non-.bru file")
	}
}

func TestBrunoParseMetaLine(t *testing.T) {
	b := &BrunoImporter{}
	req := &BrunoRequest{}

	tests := []struct {
		name  string
		line  string
		check func(*types.SavedRequest)
	}{
		{
			name: "name",
			line: "  name: Test Request",
			check: func(sr *types.SavedRequest) {
				if req.Name != "Test Request" {
					t.Errorf("got name %q, want %q", req.Name, "Test Request")
				}
			},
		},
		{
			name: "method get",
			line: "  method: get",
			check: func(sr *types.SavedRequest) {
				if req.Method != "GET" {
					t.Errorf("got method %q, want %q", req.Method, "GET")
				}
			},
		},
		{
			name: "method post",
			line: "  method: post",
			check: func(sr *types.SavedRequest) {
				if req.Method != "POST" {
					t.Errorf("got method %q, want %q", req.Method, "POST")
				}
			},
		},
		{
			name: "url",
			line: "  url: https://api.example.com",
			check: func(sr *types.SavedRequest) {
				if req.URL != "https://api.example.com" {
					t.Errorf("got URL %q, want %q", req.URL, "https://api.example.com")
				}
			},
		},
		{
			name:  "invalid line (no colon)",
			line:  "invalid",
			check: func(sr *types.SavedRequest) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req = &BrunoRequest{} // Reset
			b.parseMetaLine(req, tt.line)
			tt.check(nil)
		})
	}
}

func TestBrunoParseHeaderLine(t *testing.T) {
	b := &BrunoImporter{}
	req := &BrunoRequest{}

	b.parseHeaderLine(req, "  Content-Type: application/json")

	if len(req.Headers) != 1 {
		t.Errorf("got %d headers, want 1", len(req.Headers))
	}

	if req.Headers[0].Key != "Content-Type" || req.Headers[0].Value != "application/json" {
		t.Errorf("got header %q: %q, want %q: %q", req.Headers[0].Key, req.Headers[0].Value, "Content-Type", "application/json")
	}
}

func TestBrunoParseAuthLine(t *testing.T) {
	b := &BrunoImporter{}

	tests := []struct {
		name  string
		line  string
		check func(*BrunoRequest)
	}{
		{
			name: "type bearer",
			line: "  type: bearer",
			check: func(req *BrunoRequest) {
				if req.Auth == nil || req.Auth.Type != "bearer" {
					t.Error("expected auth type bearer")
				}
			},
		},
		{
			name: "token",
			line: "  token: mytoken",
			check: func(req *BrunoRequest) {
				if req.Auth == nil || req.Auth.Bearer != "mytoken" {
					t.Error("expected bearer token")
				}
			},
		},
		{
			name: "username",
			line: "  username: admin",
			check: func(req *BrunoRequest) {
				if req.Auth == nil || req.Auth.Basic == nil || req.Auth.Basic.Username != "admin" {
					t.Error("expected basic username")
				}
			},
		},
		{
			name: "password",
			line: "  password: secret",
			check: func(req *BrunoRequest) {
				if req.Auth == nil || req.Auth.Basic == nil || req.Auth.Basic.Password != "secret" {
					t.Error("expected basic password")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &BrunoRequest{}
			b.parseAuthLine(req, tt.line)
			tt.check(req)
		})
	}
}

func TestBrunoParseVarLine(t *testing.T) {
	b := &BrunoImporter{}
	req := &BrunoRequest{}

	b.parseVarLine(req, "  baseUrl: https://api.example.com")

	if len(req.Vars) != 1 {
		t.Errorf("got %d vars, want 1", len(req.Vars))
	}

	if req.Vars[0].Name != "baseUrl" || req.Vars[0].Value != "https://api.example.com" {
		t.Errorf("got var %q: %q", req.Vars[0].Name, req.Vars[0].Value)
	}
}

func TestBrunoToSavedRequest(t *testing.T) {
	b := &BrunoImporter{}
	req := &BrunoRequest{
		Name:    "Test Request",
		Method:  "POST",
		URL:     "https://api.example.com/test",
		Headers: []types.Header{{Key: "Content-Type", Value: "application/json"}},
		Body:    `{"key": "value"}`,
		Auth: &BrunoAuth{
			Type:   "bearer",
			Bearer: "token123",
		},
	}

	saved := b.toSavedRequest(req, "/path/to/test.bru")

	if saved.Name != "Test Request" {
		t.Errorf("got name %q, want %q", saved.Name, "Test Request")
	}
	if saved.Method != "POST" {
		t.Errorf("got method %q, want %q", saved.Method, "POST")
	}
	if saved.URL != "https://api.example.com/test" {
		t.Errorf("got URL %q, want %q", saved.URL, "https://api.example.com/test")
	}
	// Auth header should be added
	if len(saved.Headers) != 2 {
		t.Errorf("got %d headers, want 2 (including auth)", len(saved.Headers))
	}
}

func TestBrunoToSavedRequestWithBasicAuth(t *testing.T) {
	b := &BrunoImporter{}
	req := &BrunoRequest{
		Name:   "Basic Auth Request",
		Method: "GET",
		URL:    "https://api.example.com",
		Auth: &BrunoAuth{
			Type: "basic",
			Basic: &BrunoBasicAuth{
				Username: "user",
				Password: "pass",
			},
		},
	}

	saved := b.toSavedRequest(req, "/test/request.bru")

	found := false
	for _, h := range saved.Headers {
		if h.Key == "Authorization" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected Authorization header")
	}
}

func TestBrunoToSavedRequestCollection(t *testing.T) {
	b := &BrunoImporter{}
	req := &BrunoRequest{
		Name:   "Request",
		Method: "GET",
		URL:    "https://api.example.com",
	}

	// Path like /collection-name/request.bru
	saved := b.toSavedRequest(req, "/mycollection/myrequest.bru")

	if saved.Collection != "mycollection" {
		t.Errorf("got collection %q, want %q", saved.Collection, "mycollection")
	}
}

func TestBrunoBasicAuthFunc(t *testing.T) {
	result := basicAuth("user", "pass")
	if result != "user:pass" {
		t.Errorf("got %q, want %q", result, "user:pass")
	}
}
