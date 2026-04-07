package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sreeram/gurl/internal/client"
)

func TestAPIKeyHandler_Name(t *testing.T) {
	h := &APIKeyHandler{}
	if h.Name() != "apikey" {
		t.Errorf("expected name 'apikey', got %q", h.Name())
	}
}

func TestAPIKeyHandler_Apply_HeaderMode(t *testing.T) {
	handler := &APIKeyHandler{}
	req := &client.Request{
		Method: "GET",
		URL:    "http://example.com",
	}

	params := map[string]string{
		"in":          "header",
		"header_name": "X-API-Key",
		"key":         "my-secret-key",
	}

	handler.Apply(req, params)

	// Check header was set
	found := false
	for _, h := range req.Headers {
		if h.Key == "X-API-Key" && h.Value == "my-secret-key" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected X-API-Key header to be set")
	}
}

func TestAPIKeyHandler_Apply_QueryMode(t *testing.T) {
	handler := &APIKeyHandler{}
	req := &client.Request{
		Method: "GET",
		URL:    "http://example.com",
	}

	params := map[string]string{
		"in":         "query",
		"param_name": "api_key",
		"key":        "my-secret-key",
	}

	handler.Apply(req, params)

	// Check URL was modified with query param
	expected := "http://example.com?api_key=my-secret-key"
	if req.URL != expected {
		t.Errorf("expected URL %q, got %q", expected, req.URL)
	}
}

func TestAPIKeyHandler_Apply_HeaderMode_NonDefaultHeaderName(t *testing.T) {
	handler := &APIKeyHandler{}
	req := &client.Request{
		Method: "GET",
		URL:    "http://example.com",
	}

	params := map[string]string{
		"in":          "header",
		"header_name": "Authorization",
		"key":         "Token abc123",
	}

	handler.Apply(req, params)

	found := false
	for _, h := range req.Headers {
		if h.Key == "Authorization" && h.Value == "Token abc123" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected Authorization header to be set")
	}
}

func TestAPIKeyHandler_Apply_QueryMode_NonDefaultParamName(t *testing.T) {
	handler := &APIKeyHandler{}
	req := &client.Request{
		Method: "GET",
		URL:    "http://example.com",
	}

	params := map[string]string{
		"in":         "query",
		"param_name": "key",
		"key":        "secret",
	}

	handler.Apply(req, params)

	expected := "http://example.com?key=secret"
	if req.URL != expected {
		t.Errorf("expected URL %q, got %q", expected, req.URL)
	}
}

func TestAPIKeyHandler_Apply_MissingKey(t *testing.T) {
	handler := &APIKeyHandler{}
	req := &client.Request{
		Method: "GET",
		URL:    "http://example.com",
	}

	params := map[string]string{
		"in":          "header",
		"header_name": "X-API-Key",
	}

	handler.Apply(req, params)

	// Should not panic, no headers should be set
	if len(req.Headers) != 0 {
		t.Errorf("expected no headers when key missing, got %d headers", len(req.Headers))
	}
}

func TestAPIKeyHandler_Apply_UnknownMode(t *testing.T) {
	handler := &APIKeyHandler{}
	req := &client.Request{
		Method: "GET",
		URL:    "http://example.com",
	}

	params := map[string]string{
		"in":  "cookies",
		"key": "my-key",
	}

	handler.Apply(req, params)

	// Should not modify anything
	if len(req.Headers) != 0 {
		t.Errorf("expected no headers for unknown mode, got %d headers", len(req.Headers))
	}
}

func TestAPIKeyHandler_Apply_QueryModeWithExistingQuery(t *testing.T) {
	handler := &APIKeyHandler{}
	req := &client.Request{
		Method: "GET",
		URL:    "http://example.com?page=1",
	}

	params := map[string]string{
		"in":         "query",
		"param_name": "api_key",
		"key":        "my-key",
	}

	handler.Apply(req, params)

	// Should append to existing query params
	expected := "http://example.com?page=1&api_key=my-key"
	if req.URL != expected {
		t.Errorf("expected URL %q, got %q", expected, req.URL)
	}
}

// mockAPIKeyHandler creates a test server that checks for API Key auth
func mockAPIKeyHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check header mode
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			// Check query mode
			apiKey = r.URL.Query().Get("api_key")
		}
		if apiKey == "test-api-key" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	})
}

func TestRegistry_Apply_APIKey_HeaderMode(t *testing.T) {
	server := httptest.NewServer(mockAPIKeyHandler())
	defer server.Close()

	r := NewRegistry()
	r.Register(&APIKeyHandler{})

	req := &client.Request{
		Method: "GET",
		URL:    server.URL,
	}

	r.Apply("apikey", req, map[string]string{
		"in":          "header",
		"header_name": "X-API-Key",
		"key":         "test-api-key",
	})

	httpReq, _ := http.NewRequest(req.Method, req.URL, nil)
	for _, h := range req.Headers {
		httpReq.Header.Set(h.Key, h.Value)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestRegistry_Apply_APIKey_QueryMode(t *testing.T) {
	server := httptest.NewServer(mockAPIKeyHandler())
	defer server.Close()

	r := NewRegistry()
	r.Register(&APIKeyHandler{})

	req := &client.Request{
		Method: "GET",
		URL:    server.URL,
	}

	r.Apply("apikey", req, map[string]string{
		"in":         "query",
		"param_name": "api_key",
		"key":        "test-api-key",
	})

	httpReq, _ := http.NewRequest(req.Method, req.URL, nil)
	for _, h := range req.Headers {
		httpReq.Header.Set(h.Key, h.Value)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}
