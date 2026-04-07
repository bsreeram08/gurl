package auth

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sreeram/gurl/internal/client"
)

// mockHandler checks for Basic auth header and responds accordingly
func mockAuthHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Expect "Basic <base64>"
		expectedPrefix := "Basic "
		if len(auth) < len(expectedPrefix) || auth[:len(expectedPrefix)] != expectedPrefix {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Decode and check it contains "testuser:testpass"
		encoded := auth[len(expectedPrefix):]
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if string(decoded) == "testuser:testpass" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	})
}

func TestHandlerInterface(t *testing.T) {
	// Verify BasicHandler implements Handler interface
	var h Handler = &BasicHandler{}
	if h.Name() != "basic" {
		t.Errorf("expected name 'basic', got %q", h.Name())
	}
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()

	// Register a handler
	r.Register(&BasicHandler{})

	// Get should return the handler
	h := r.Get("basic")
	if h == nil {
		t.Fatal("expected to get BasicHandler, got nil")
	}
	if h.Name() != "basic" {
		t.Errorf("expected name 'basic', got %q", h.Name())
	}
}

func TestRegistry_GetUnknown(t *testing.T) {
	r := NewRegistry()

	// Get unknown should return nil
	h := r.Get("nonexistent")
	if h != nil {
		t.Errorf("expected nil for unknown handler, got %v", h)
	}
}

func TestRegistry_Apply(t *testing.T) {
	server := httptest.NewServer(mockAuthHandler())
	defer server.Close()

	tests := []struct {
		name       string
		authType   string
		params     map[string]string
		wantStatus int
	}{
		{
			name:       "basic auth with valid credentials",
			authType:   "basic",
			params:     map[string]string{"username": "testuser", "password": "testpass"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "basic auth with invalid credentials",
			authType:   "basic",
			params:     map[string]string{"username": "wrong", "password": "wrong"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "basic auth with missing username",
			authType:   "basic",
			params:     map[string]string{"password": "testpass"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "unknown auth type does nothing",
			authType:   "unknown",
			params:     map[string]string{"username": "testuser", "password": "testpass"},
			wantStatus: http.StatusUnauthorized, // No auth header set
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRegistry()
			r.Register(&BasicHandler{})

			req := &client.Request{
				Method: "GET",
				URL:    server.URL,
			}

			r.Apply(tc.authType, req, tc.params)

			// Build actual request to check headers
			httpReq, _ := http.NewRequest(req.Method, req.URL, nil)
			for _, h := range req.Headers {
				httpReq.Header.Set(h.Key, h.Value)
			}

			// Make the actual HTTP call
			resp, err := http.DefaultClient.Do(httpReq)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			resp.Body.Close()

			if resp.StatusCode != tc.wantStatus {
				t.Errorf("expected status %d, got %d", tc.wantStatus, resp.StatusCode)
			}
		})
	}
}

func TestBasicHandler_Apply(t *testing.T) {
	handler := &BasicHandler{}
	req := &client.Request{
		Method: "GET",
		URL:    "http://example.com",
	}

	params := map[string]string{
		"username": "myuser",
		"password": "mypass",
	}

	handler.Apply(req, params)

	// Check Authorization header was set
	found := false
	for _, h := range req.Headers {
		if h.Key == "Authorization" {
			found = true
			expected := "Basic " + base64.StdEncoding.EncodeToString([]byte("myuser:mypass"))
			if h.Value != expected {
				t.Errorf("expected Authorization %q, got %q", expected, h.Value)
			}
		}
	}
	if !found {
		t.Error("expected Authorization header to be set")
	}
}

func TestBasicHandler_Apply_MissingUsername(t *testing.T) {
	handler := &BasicHandler{}
	req := &client.Request{
		Method: "GET",
		URL:    "http://example.com",
	}

	params := map[string]string{
		"password": "mypass",
	}

	handler.Apply(req, params)

	// Check no Authorization header was set (should not panic)
	if len(req.Headers) != 0 {
		t.Errorf("expected no headers when username missing, got %d headers", len(req.Headers))
	}
}

func TestBasicHandler_Apply_MissingPassword(t *testing.T) {
	handler := &BasicHandler{}
	req := &client.Request{
		Method: "GET",
		URL:    "http://example.com",
	}

	params := map[string]string{
		"username": "myuser",
	}

	handler.Apply(req, params)

	// Check no Authorization header was set (should not panic)
	if len(req.Headers) != 0 {
		t.Errorf("expected no headers when password missing, got %d headers", len(req.Headers))
	}
}

func TestRegistry_Apply_BasicAuthEndToEnd(t *testing.T) {
	server := httptest.NewServer(mockAuthHandler())
	defer server.Close()

	r := NewRegistry()
	r.Register(&BasicHandler{})

	// Create request and apply auth
	req := &client.Request{
		Method: "GET",
		URL:    server.URL,
	}

	r.Apply("basic", req, map[string]string{
		"username": "testuser",
		"password": "testpass",
	})

	// Execute the request
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
