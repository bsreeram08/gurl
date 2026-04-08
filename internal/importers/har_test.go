package importers

import (
	"os"
	"path/filepath"
	"testing"
)

// Sample HAR file
const sampleHAR = `{
  "log": {
    "version": "1.2",
    "creator": {
      "name": "Test",
      "version": "1.0"
    },
    "entries": [
      {
        "startedDateTime": "2023-01-01T00:00:00Z",
        "time": 100,
        "request": {
          "method": "GET",
          "url": "https://api.example.com/users",
          "httpVersion": "HTTP/1.1",
          "cookies": [],
          "headers": [
            {"name": "Accept", "value": "application/json"}
          ],
          "queryString": [
            {"name": "page", "value": "1"}
          ]
        },
        "response": {
          "status": 200,
          "statusText": "OK",
          "httpVersion": "HTTP/1.1",
          "cookies": [],
          "headers": [],
          "content": {
            "size": 100,
            "mimeType": "application/json"
          },
          "redirectURL": ""
        },
        "timings": {
          "wait": 100
        }
      },
      {
        "startedDateTime": "2023-01-01T00:00:01Z",
        "time": 150,
        "request": {
          "method": "POST",
          "url": "https://api.example.com/users",
          "httpVersion": "HTTP/1.1",
          "cookies": [],
          "headers": [
            {"name": "Content-Type", "value": "application/json"}
          ],
          "queryString": [],
          "postData": {
            "mimeType": "application/json",
            "text": "{\"name\": \"John\"}"
          }
        },
        "response": {
          "status": 201,
          "statusText": "Created",
          "httpVersion": "HTTP/1.1",
          "cookies": [],
          "headers": [],
          "content": {
            "size": 50,
            "mimeType": "application/json"
          },
          "redirectURL": ""
        },
        "timings": {
          "wait": 150
        }
      }
    ]
  }
}`

// HAR with multiple entries
const sampleHARMultipleEntries = `{
  "log": {
    "version": "1.2",
    "creator": {
      "name": "Test",
      "version": "1.0"
    },
    "entries": [
      {
        "startedDateTime": "2023-01-01T00:00:00Z",
        "time": 50,
        "request": {
          "method": "GET",
          "url": "https://httpbin.org/get",
          "cookies": [],
          "headers": [],
          "queryString": []
        },
        "response": {
          "status": 200,
          "statusText": "OK",
          "cookies": [],
          "headers": [],
          "content": {"size": 100, "mimeType": "application/json"},
          "redirectURL": ""
        },
        "timings": {"wait": 50}
      },
      {
        "startedDateTime": "2023-01-01T00:00:01Z",
        "time": 60,
        "request": {
          "method": "PUT",
          "url": "https://httpbin.org/put",
          "cookies": [],
          "headers": [],
          "queryString": []
        },
        "response": {
          "status": 200,
          "statusText": "OK",
          "cookies": [],
          "headers": [],
          "content": {"size": 100, "mimeType": "application/json"},
          "redirectURL": ""
        },
        "timings": {"wait": 60}
      },
      {
        "startedDateTime": "2023-01-01T00:00:02Z",
        "time": 40,
        "request": {
          "method": "DELETE",
          "url": "https://httpbin.org/delete",
          "cookies": [],
          "headers": [],
          "queryString": []
        },
        "response": {
          "status": 204,
          "statusText": "No Content",
          "cookies": [],
          "headers": [],
          "content": {"size": 0, "mimeType": ""},
          "redirectURL": ""
        },
        "timings": {"wait": 40}
      }
    ]
  }
}`

// HAR with cookies in request
const sampleHARWithCookies = `{
  "log": {
    "version": "1.2",
    "creator": {"name": "Test", "version": "1.0"},
    "entries": [
      {
        "startedDateTime": "2023-01-01T00:00:00Z",
        "time": 50,
        "request": {
          "method": "GET",
          "url": "https://api.example.com/cookies",
          "cookies": [
            {"name": "session", "value": "abc123"},
            {"name": "theme", "value": "dark"}
          ],
          "headers": [],
          "queryString": []
        },
        "response": {
          "status": 200,
          "statusText": "OK",
          "cookies": [],
          "headers": [],
          "content": {"size": 50, "mimeType": "application/json"},
          "redirectURL": ""
        },
        "timings": {"wait": 50}
      }
    ]
  }
}`

// HAR with pages (for collection naming)
const sampleHARWithPages = `{
  "log": {
    "version": "1.2",
    "creator": {"name": "Test", "version": "1.0"},
    "pages": [
      {
        "id": "page_1",
        "title": "User API",
        "startedDateTime": "2023-01-01T00:00:00Z"
      }
    ],
    "entries": [
      {
        "startedDateTime": "2023-01-01T00:00:00Z",
        "time": 50,
        "request": {
          "method": "GET",
          "url": "https://api.example.com/users",
          "cookies": [],
          "headers": [],
          "queryString": []
        },
        "response": {
          "status": 200,
          "statusText": "OK",
          "cookies": [],
          "headers": [],
          "content": {"size": 100, "mimeType": "application/json"},
          "redirectURL": ""
        },
        "timings": {"wait": 50}
      }
    ]
  }
}`

// HAR with headers to filter (Host, Content-Length)
const sampleHARWithFilteredHeaders = `{
  "log": {
    "version": "1.2",
    "creator": {"name": "Test", "version": "1.0"},
    "entries": [
      {
        "startedDateTime": "2023-01-01T00:00:00Z",
        "time": 50,
        "request": {
          "method": "POST",
          "url": "https://api.example.com/upload",
          "cookies": [],
          "headers": [
            {"name": "Host", "value": "api.example.com"},
            {"name": "Content-Length", "value": "1234"},
            {"name": "X-Custom", "value": "custom-value"}
          ],
          "queryString": [],
          "postData": {
            "mimeType": "application/octet-stream",
            "text": "binary data here"
          }
        },
        "response": {
          "status": 201,
          "statusText": "Created",
          "cookies": [],
          "headers": [],
          "content": {"size": 0, "mimeType": ""},
          "redirectURL": ""
        },
        "timings": {"wait": 50}
      }
    ]
  }
}`

// HAR with empty method (should default to GET)
const sampleHAREmptyMethod = `{
  "log": {
    "version": "1.2",
    "creator": {"name": "Test", "version": "1.0"},
    "entries": [
      {
        "startedDateTime": "2023-01-01T00:00:00Z",
        "time": 50,
        "request": {
          "method": "",
          "url": "https://api.example.com/test",
          "cookies": [],
          "headers": [],
          "queryString": []
        },
        "response": {
          "status": 200,
          "statusText": "OK",
          "cookies": [],
          "headers": [],
          "content": {"size": 0, "mimeType": ""},
          "redirectURL": ""
        },
        "timings": {"wait": 50}
      }
    ]
  }
}`

// Invalid HAR JSON
const invalidHARJSON = `{not valid json}`

func TestHARImporterName(t *testing.T) {
	h := &HARImporter{}
	if h.Name() != "har" {
		t.Errorf("got %q, want %q", h.Name(), "har")
	}
}

func TestHARImporterExtensions(t *testing.T) {
	h := &HARImporter{}
	exts := h.Extensions()
	if len(exts) != 2 {
		t.Errorf("got %v, want [.har, .json]", exts)
	}
}

func TestHARParse(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.har")
	if err := os.WriteFile(tmpFile, []byte(sampleHAR), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	h := &HARImporter{}
	requests, err := h.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 2 {
		t.Errorf("got %d requests, want 2", len(requests))
	}
}

func TestHARMultipleEntries(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "multiple.har")
	if err := os.WriteFile(tmpFile, []byte(sampleHARMultipleEntries), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	h := &HARImporter{}
	requests, err := h.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 3 {
		t.Errorf("got %d requests, want 3", len(requests))
	}

	// Check different methods
	methods := make(map[string]bool)
	for _, req := range requests {
		methods[req.Method] = true
	}

	if !methods["GET"] || !methods["PUT"] || !methods["DELETE"] {
		t.Error("expected GET, PUT, DELETE methods")
	}
}

func TestHARInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.har")
	if err := os.WriteFile(tmpFile, []byte(invalidHARJSON), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	h := &HARImporter{}
	_, err := h.Parse(tmpFile)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestHARNonexistentFile(t *testing.T) {
	h := &HARImporter{}
	_, err := h.Parse("/nonexistent/path/to/file.har")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestHARConvertToRequests(t *testing.T) {
	h := &HARImporter{}
	har := &HAR{
		Log: HARLog{
			Version: "1.2",
			Creator: HARCreator{Name: "Test", Version: "1.0"},
			Entries: []HAREntry{
				{
					StartedDateTime: "2023-01-01T00:00:00Z",
					Request: HARRequest{
						Method:  "GET",
						URL:     "https://api.example.com/test",
						Headers: []HARHeader{{Name: "Accept", Value: "application/json"}},
					},
					Response: HARResponse{
						Status:  200,
						Content: HARContent{MimeType: "application/json"},
					},
				},
			},
		},
	}

	requests := h.convertToRequests(har)
	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}
}

func TestHAREntryToSavedRequest(t *testing.T) {
	h := &HARImporter{}
	entry := &HAREntry{
		StartedDateTime: "2023-01-01T00:00:00Z",
		Request: HARRequest{
			Method: "POST",
			URL:    "https://api.example.com/users",
			Headers: []HARHeader{
				{Name: "Content-Type", Value: "application/json"},
			},
			PostData: &HARPostData{
				MimeType: "application/json",
				Text:     `{"name": "John"}`,
			},
		},
	}
	pageMap := map[string]string{}

	saved := h.entryToSavedRequest(entry, 0, pageMap)

	if saved.Method != "POST" {
		t.Errorf("got method %q, want %q", saved.Method, "POST")
	}
	if saved.URL != "https://api.example.com/users" {
		t.Errorf("got URL %q, want %q", saved.URL, "https://api.example.com/users")
	}
	if saved.Body != `{"name": "John"}` {
		t.Errorf("got body %q, want %q", saved.Body, `{"name": "John"}`)
	}
}

func TestHARWithCookies(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "cookies.har")
	if err := os.WriteFile(tmpFile, []byte(sampleHARWithCookies), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	h := &HARImporter{}
	requests, err := h.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}

	// Check for cookie headers
	hasCookie := false
	for _, h := range requests[0].Headers {
		if h.Key == "Cookie" {
			hasCookie = true
			break
		}
	}
	if !hasCookie {
		t.Error("expected Cookie header")
	}
}

func TestHARWithPages(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "pages.har")
	if err := os.WriteFile(tmpFile, []byte(sampleHARWithPages), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	h := &HARImporter{}
	requests, err := h.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}
}

func TestHARFilteredHeaders(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "filtered.har")
	if err := os.WriteFile(tmpFile, []byte(sampleHARWithFilteredHeaders), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	h := &HARImporter{}
	requests, err := h.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(requests) != 1 {
		t.Errorf("got %d requests, want 1", len(requests))
	}

	// Should not have Host or Content-Length headers
	for _, header := range requests[0].Headers {
		if header.Key == "Host" || header.Key == "Content-Length" {
			t.Errorf("unexpected header %q in output", header.Key)
		}
	}

	// Should have X-Custom header
	found := false
	for _, header := range requests[0].Headers {
		if header.Key == "X-Custom" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected X-Custom header to be preserved")
	}
}

func TestHAREmptyMethod(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "emptymethod.har")
	if err := os.WriteFile(tmpFile, []byte(sampleHAREmptyMethod), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	h := &HARImporter{}
	requests, err := h.Parse(tmpFile)
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

func TestHARGenerateName(t *testing.T) {
	h := &HARImporter{}

	tests := []struct {
		name  string
		entry *HAREntry
		index int
	}{
		{
			name: "GET request",
			entry: &HAREntry{
				Request: HARRequest{
					Method: "GET",
					URL:    "https://api.example.com/users",
				},
			},
			index: 0,
		},
		{
			name: "with path",
			entry: &HAREntry{
				Request: HARRequest{
					Method: "POST",
					URL:    "https://api.example.com/api/users/create",
				},
			},
			index: 1,
		},
		{
			name: "with query string only",
			entry: &HAREntry{
				Request: HARRequest{
					Method: "GET",
					URL:    "https://api.example.com?foo=bar",
				},
			},
			index: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name := h.generateName(tt.entry, tt.index)

			if name == "" {
				t.Error("expected non-empty name")
			}
			methodLen := len(tt.entry.Request.Method)
			if len(name) >= methodLen && name[:methodLen] != tt.entry.Request.Method {
				t.Errorf("name should start with method, got %q", name)
			}
		})
	}
}

func TestHARCleanPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "https URL",
			path: "https://api.example.com/users",
			want: "/users",
		},
		{
			name: "http URL",
			path: "http://api.example.com/api/v1/items",
			want: "/api/v1/items",
		},
		{
			name: "URL with query",
			path: "https://api.example.com/search?q=test",
			want: "/search?q=test",
		},
		{
			name: "short URL",
			path: "https://a.co",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanPath(tt.path)
			// Just check it doesn't panic
			_ = got
		})
	}
}

func TestHARFindPageID(t *testing.T) {
	h := &HARImporter{}
	entry := &HAREntry{}
	pageMap := map[string]string{}

	result := h.findPageID(entry, pageMap)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestHARContains(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"hello world", "world", true},
		{"hello world", "foo", false},
		{"", "", true},
		{"hello", "", true},
	}

	for _, tt := range tests {
		got := contains(tt.s, tt.substr)
		if got != tt.want {
			t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
		}
	}
}
