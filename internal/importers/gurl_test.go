package importers

import (
	"os"
	"path/filepath"
	"testing"
)

// Sample gurl export file content
const sampleGurlExport = `{
  "version": "1.0",
  "exported_at": "2024-01-15T10:30:00Z",
  "requests": [
    {
      "id": "req-001",
      "name": "List Users",
      "curl_cmd": "curl https://api.example.com/users",
      "url": "https://api.example.com/users",
      "method": "GET",
      "headers": [
        {"key": "Content-Type", "value": "application/json"}
      ],
      "body": "",
      "output_format": "auto",
      "created_at": 1705312200,
      "updated_at": 1705312200
    },
    {
      "id": "req-002",
      "name": "Create User",
      "curl_cmd": "curl -X POST https://api.example.com/users",
      "url": "https://api.example.com/users",
      "method": "POST",
      "headers": [
        {"key": "Content-Type", "value": "application/json"}
      ],
      "body": "{\"name\": \"John\"}",
      "output_format": "auto",
      "created_at": 1705312300,
      "updated_at": 1705312300
    }
  ]
}`

// gurl export with single request
const sampleGurlSingleRequest = `{
  "version": "1.0",
  "exported_at": "2024-01-15T10:30:00Z",
  "requests": [
    {
      "id": "req-003",
      "name": "Get User",
      "curl_cmd": "curl https://api.example.com/users/123",
      "url": "https://api.example.com/users/123",
      "method": "GET",
      "headers": [],
      "body": "",
      "output_format": "auto",
      "created_at": 1705312400,
      "updated_at": 1705312400
    }
  ]
}`

// gurl export with invalid version
const invalidVersionGurl = `{
  "version": "2.0",
  "exported_at": "2024-01-15T10:30:00Z",
  "requests": []
}`

// gurl export with no requests array
const noRequestsGurl = `{
  "version": "1.0",
  "exported_at": "2024-01-15T10:30:00Z"
}`

// gurl export with empty requests
const emptyRequestsGurl = `{
  "version": "1.0",
  "exported_at": "2024-01-15T10:30:00Z",
  "requests": []
}`

// malformed JSON
const malformedGurl = `{
  "version": "1.0",
  "exported_at": "2024-01-15T10:30:00Z
  "requests": []
}`

func TestGurlImporterName(t *testing.T) {
	g := &GurlImporter{}
	if g.Name() != "gurl" {
		t.Errorf("got %q, want %q", g.Name(), "gurl")
	}
}

func TestGurlImporterExtensions(t *testing.T) {
	g := &GurlImporter{}
	exts := g.Extensions()
	if len(exts) != 1 || exts[0] != ".gurl" {
		t.Errorf("got %v, want [.gurl]", exts)
	}
}

func TestGurlParseFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "export.gurl")
	if err := os.WriteFile(tmpFile, []byte(sampleGurlExport), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	g := &GurlImporter{}
	requests, err := g.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 2 {
		t.Errorf("got %d requests, want 2", len(requests))
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

	if requests[1].Name != "Create User" {
		t.Errorf("got name %q, want %q", requests[1].Name, "Create User")
	}

	if requests[1].Method != "POST" {
		t.Errorf("got method %q, want %q", requests[1].Method, "POST")
	}
}

func TestGurlParseSingleRequest(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "single.gurl")
	if err := os.WriteFile(tmpFile, []byte(sampleGurlSingleRequest), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	g := &GurlImporter{}
	requests, err := g.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}

	if requests[0].Name != "Get User" {
		t.Errorf("got name %q, want %q", requests[0].Name, "Get User")
	}
}

func TestGurlInvalidVersion(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.gurl")
	if err := os.WriteFile(tmpFile, []byte(invalidVersionGurl), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	g := &GurlImporter{}
	_, err := g.Parse(tmpFile)
	if err == nil {
		t.Error("expected error for invalid version")
	}
}

func TestGurlNoRequests(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "norequests.gurl")
	if err := os.WriteFile(tmpFile, []byte(noRequestsGurl), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	g := &GurlImporter{}
	requests, err := g.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 0 {
		t.Errorf("got %d requests, want 0", len(requests))
	}
}

func TestGurlEmptyRequests(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "empty.gurl")
	if err := os.WriteFile(tmpFile, []byte(emptyRequestsGurl), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	g := &GurlImporter{}
	requests, err := g.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 0 {
		t.Errorf("got %d requests, want 0", len(requests))
	}
}

func TestGurlMalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "malformed.gurl")
	if err := os.WriteFile(tmpFile, []byte(malformedGurl), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	g := &GurlImporter{}
	_, err := g.Parse(tmpFile)
	if err == nil {
		t.Error("expected error for malformed JSON")
	}
}

func TestGurlInvalidPath(t *testing.T) {
	g := &GurlImporter{}
	_, err := g.Parse("/nonexistent/path/to/file.gurl")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestGurlNonGurlFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(tmpFile, []byte("not a .gurl file"), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	g := &GurlImporter{}
	_, err := g.Parse(tmpFile)
	if err == nil {
		t.Error("expected error for non-.gurl file")
	}
}

func TestGurlDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	g := &GurlImporter{}
	_, err := g.Parse(tmpDir)
	if err == nil {
		t.Error("expected error for directory path")
	}
}

func TestGurlRequestHeaders(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "headers.gurl")
	if err := os.WriteFile(tmpFile, []byte(sampleGurlExport), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	g := &GurlImporter{}
	requests, err := g.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check headers on first request
	if len(requests[0].Headers) != 1 {
		t.Errorf("got %d headers, want 1", len(requests[0].Headers))
	}

	if requests[0].Headers[0].Key != "Content-Type" {
		t.Errorf("got header key %q, want %q", requests[0].Headers[0].Key, "Content-Type")
	}
}

func TestGurlRequestBody(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "body.gurl")
	if err := os.WriteFile(tmpFile, []byte(sampleGurlExport), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	g := &GurlImporter{}
	requests, err := g.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Second request has a body
	if requests[1].Body != "{\"name\": \"John\"}" {
		t.Errorf("got body %q, want %q", requests[1].Body, "{\"name\": \"John\"}")
	}
}

func TestGurlRequestsFieldIsNil(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "nil.gurl")
	if err := os.WriteFile(tmpFile, []byte(noRequestsGurl), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	g := &GurlImporter{}
	requests, err := g.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if requests == nil {
		t.Error("expected empty slice, got nil")
	}
	if len(requests) != 0 {
		t.Errorf("got %d requests, want 0", len(requests))
	}
}
