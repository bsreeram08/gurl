package curl

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sreeram/gurl/pkg/types"
)

func TestExecuteCurlUsesClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	req := &types.SavedRequest{
		ID:      "test-req-1",
		Name:    "test request",
		Method:  "GET",
		URL:     server.URL,
		Headers: []types.Header{{Key: "X-Test-Header", Value: "test-value"}},
	}

	history, err := ExecuteCurl(req, nil)
	if err != nil {
		t.Fatalf("ExecuteCurl failed: %v", err)
	}

	if history.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", history.StatusCode)
	}

	if history.Response == "" {
		t.Error("Expected non-empty response")
	}
}

func TestExecuteCurlWithOutputUsesClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": 123}`))
	}))
	defer server.Close()

	req := &types.SavedRequest{
		ID:      "test-req-2",
		Name:    "test post",
		Method:  "POST",
		URL:     server.URL,
		Headers: []types.Header{{Key: "Content-Type", Value: "application/json"}},
		Body:    `{"name": "test"}`,
	}

	output, statusCode, durationMs, err := ExecuteCurlWithOutput(req, nil)
	if err != nil {
		t.Fatalf("ExecuteCurlWithOutput failed: %v", err)
	}

	if statusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", statusCode)
	}

	if output == "" {
		t.Error("Expected non-empty output")
	}

	if durationMs < 0 {
		t.Error("Expected non-negative duration")
	}
}

func TestExecuteCurlWithVariables(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"var": "value"}`))
	}))
	defer server.Close()

	req := &types.SavedRequest{
		ID:      "test-req-3",
		Name:    "variable test",
		Method:  "GET",
		URL:     server.URL + "?key={{api_key}}",
		Headers: []types.Header{},
	}

	vars := map[string]string{"api_key": "secret123"}
	history, err := ExecuteCurl(req, vars)
	if err != nil {
		t.Fatalf("ExecuteCurl with vars failed: %v", err)
	}

	if history.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", history.StatusCode)
	}

	_ = history
}

func TestBuildCurlCommandStillWorks(t *testing.T) {
	req := &types.SavedRequest{
		ID:     "test-req-4",
		Name:   "build test",
		Method: "POST",
		URL:    "https://example.com/api",
		Headers: []types.Header{
			{Key: "Authorization", Value: "Bearer token123"},
			{Key: "Content-Type", Value: "application/json"},
		},
		Body: `{"data": "test"}`,
	}

	args, err := BuildCurlCommand(req, nil)
	if err != nil {
		t.Fatalf("BuildCurlCommand failed: %v", err)
	}

	if len(args) == 0 {
		t.Error("Expected non-empty args")
	}

	expected := []string{
		"-s",
		"-w", "\n%{http_code}",
		"-o", "-",
		"-X", "POST",
		"-H", "Authorization: Bearer token123",
		"-H", "Content-Type: application/json",
		"-d", `{"data": "test"}`,
		"https://example.com/api",
	}

	for i, exp := range expected {
		if i >= len(args) || args[i] != exp {
			t.Errorf("Expected args[%d]=%q, got %q (args: %v)", i, exp, args[i], args)
			break
		}
	}
}

func TestParseStatusCodeFromResponse(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected int
	}{
		{"standard output", "response body\n200", 200},
		{"no status", "response only", 0},
		{"invalid status", "response\nabc", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseStatusCode(tt.output)
			if got != tt.expected {
				t.Errorf("parseStatusCode(%q) = %d, want %d", tt.output, got, tt.expected)
			}
		})
	}
}
