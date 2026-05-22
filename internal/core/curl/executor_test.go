package curl

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sreeram/gurl/internal/client"
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

func TestBuildClientRequestAppliesBearerAuthAfterTemplates(t *testing.T) {
	req := &types.SavedRequest{
		Name:   "templated bearer",
		Method: "POST",
		URL:    "https://{{host}}/users/{{id}}",
		Headers: []types.Header{
			{Key: "X-Tenant", Value: "{{tenant}}"},
		},
		Body: `{"name":"{{name}}"}`,
		AuthConfig: &types.AuthConfig{
			Type: "bearer",
			Params: map[string]string{
				"token": "{{token}}",
			},
		},
	}

	clientReq, err := BuildClientRequest(req, map[string]string{
		"host":   "api.example.com",
		"id":     "42",
		"tenant": "acme",
		"name":   "sreeram",
		"token":  "abc123",
	})
	if err != nil {
		t.Fatalf("BuildClientRequest returned error: %v", err)
	}

	if clientReq.URL != "https://api.example.com/users/42" {
		t.Fatalf("unexpected URL %q", clientReq.URL)
	}
	if clientReq.Body != `{"name":"sreeram"}` {
		t.Fatalf("unexpected body %q", clientReq.Body)
	}
	assertHeader(t, clientReq.Headers, "X-Tenant", "acme")
	assertHeader(t, clientReq.Headers, "Authorization", "Bearer abc123")
}

func TestBuildClientRequestAppliesAPIKeyAuth(t *testing.T) {
	req := &types.SavedRequest{
		Name:   "templated apikey",
		Method: "GET",
		URL:    "https://api.example.com/widgets",
		AuthConfig: &types.AuthConfig{
			Type: "apikey",
			Params: map[string]string{
				"header": "X-{{tenant}}-Key",
				"value":  "{{api_key}}",
			},
		},
	}

	clientReq, err := BuildClientRequest(req, map[string]string{
		"tenant":  "Acme",
		"api_key": "secret123",
	})
	if err != nil {
		t.Fatalf("BuildClientRequest returned error: %v", err)
	}

	assertHeader(t, clientReq.Headers, "X-Acme-Key", "secret123")
}

func TestBuildClientRequestUnknownAuthTypeReturnsError(t *testing.T) {
	req := &types.SavedRequest{
		Name:   "unknown auth",
		Method: "GET",
		URL:    "https://api.example.com/widgets",
		AuthConfig: &types.AuthConfig{
			Type:   "made-up",
			Params: map[string]string{"token": "abc123"},
		},
	}

	_, err := BuildClientRequest(req, nil)
	if err == nil {
		t.Fatal("expected unknown auth type error")
	}
	if !strings.Contains(err.Error(), `unknown auth type "made-up"`) {
		t.Fatalf("expected unknown auth type error, got %v", err)
	}
}

func assertHeader(t *testing.T, headers []client.Header, key, value string) {
	t.Helper()
	for _, h := range headers {
		if h.Key == key && h.Value == value {
			return
		}
	}
	t.Fatalf("expected header %s: %s in %#v", key, value, headers)
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
