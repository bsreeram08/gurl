package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sreeram/gurl/internal/client"
)

func TestBearerHandler_ImplementsHandler(t *testing.T) {
	var h Handler = &BearerHandler{}
	if h.Name() != "bearer" {
		t.Errorf("expected name 'bearer', got %q", h.Name())
	}
}

func TestBearerHandler_Apply_SetsCorrectHeader(t *testing.T) {
	handler := &BearerHandler{}
	req := &client.Request{
		Method: "GET",
		URL:    "http://example.com",
	}

	token := "my-secret-token"
	handler.Apply(req, map[string]string{"token": token})

	found := false
	for _, h := range req.Headers {
		if h.Key == "Authorization" {
			found = true
			expected := "Bearer " + token
			if h.Value != expected {
				t.Errorf("expected Authorization %q, got %q", expected, h.Value)
			}
		}
	}
	if !found {
		t.Error("expected Authorization header to be set")
	}
}

func TestBearerHandler_Apply_MissingToken(t *testing.T) {
	handler := &BearerHandler{}
	req := &client.Request{
		Method: "GET",
		URL:    "http://example.com",
	}

	handler.Apply(req, map[string]string{})

	if len(req.Headers) != 0 {
		t.Errorf("expected no headers when token missing, got %d headers", len(req.Headers))
	}
}

func TestBearerHandler_Apply_EmptyToken(t *testing.T) {
	handler := &BearerHandler{}
	req := &client.Request{
		Method: "GET",
		URL:    "http://example.com",
	}

	handler.Apply(req, map[string]string{"token": ""})

	if len(req.Headers) != 0 {
		t.Errorf("expected no headers when token empty, got %d headers", len(req.Headers))
	}
}

func TestRegistry_Apply_BearerAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if auth != "Bearer my-token-123" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	r := NewRegistry()
	r.Register(&BearerHandler{})

	req := &client.Request{
		Method: "GET",
		URL:    server.URL,
	}

	r.Apply("bearer", req, map[string]string{"token": "my-token-123"})

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

func TestRegistry_Apply_BearerAuth_MissingToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "" {
			t.Errorf("expected no Authorization header, got %q", auth)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	r := NewRegistry()
	r.Register(&BearerHandler{})

	req := &client.Request{
		Method: "GET",
		URL:    server.URL,
	}

	r.Apply("bearer", req, map[string]string{})

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
