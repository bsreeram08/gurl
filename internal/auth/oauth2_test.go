package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sreeram/gurl/internal/client"
)

// mockTokenHandler simulates an OAuth2 token endpoint
func mockTokenHandler(tokenResponse map[string]interface{}) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check content type
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/x-www-form-urlencoded" && contentType != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse)
	})
}

func TestOAuth2Handler_Name(t *testing.T) {
	h := &OAuth2Handler{}
	if h.Name() != "oauth2" {
		t.Errorf("expected name 'oauth2', got %q", h.Name())
	}
}

func TestOAuth2Handler_AuthCodeFlow(t *testing.T) {
	tokenResponse := map[string]interface{}{
		"access_token": "test-access-token-123",
		"token_type":   "Bearer",
		"expires_in":   3600,
	}

	server := httptest.NewServer(mockTokenHandler(tokenResponse))
	defer server.Close()

	handler := &OAuth2Handler{}
	req := &client.Request{
		Method: "GET",
		URL:    "http://example.com",
	}

	params := map[string]string{
		"client_id":     "my-client-id",
		"client_secret": "my-client-secret",
		"token_url":     server.URL,
		"auth_code":     "test-auth-code-456",
		"flow":          "auth_code",
		"redirect_uri":  "http://localhost:8080/callback",
		"scope":         "read write",
	}

	handler.Apply(req, params)

	// Check Authorization header was set with Bearer token
	found := false
	for _, h := range req.Headers {
		if h.Key == "Authorization" {
			found = true
			expected := "Bearer test-access-token-123"
			if h.Value != expected {
				t.Errorf("expected Authorization %q, got %q", expected, h.Value)
			}
		}
	}
	if !found {
		t.Error("expected Authorization header to be set")
	}
}

func TestOAuth2Handler_ClientCredentialsFlow(t *testing.T) {
	tokenResponse := map[string]interface{}{
		"access_token": "client-creds-token-xyz",
		"token_type":   "Bearer",
		"expires_in":   7200,
	}

	server := httptest.NewServer(mockTokenHandler(tokenResponse))
	defer server.Close()

	handler := &OAuth2Handler{}
	req := &client.Request{
		Method: "GET",
		URL:    "http://example.com",
	}

	params := map[string]string{
		"client_id":     "my-client-id",
		"client_secret": "my-client-secret",
		"token_url":     server.URL,
		"flow":          "client_credentials",
		"scope":         "api:read api:write",
	}

	handler.Apply(req, params)

	// Check Authorization header was set with Bearer token
	found := false
	for _, h := range req.Headers {
		if h.Key == "Authorization" {
			found = true
			expected := "Bearer client-creds-token-xyz"
			if h.Value != expected {
				t.Errorf("expected Authorization %q, got %q", expected, h.Value)
			}
		}
	}
	if !found {
		t.Error("expected Authorization header to be set")
	}
}

func TestOAuth2Handler_TokenCaching(t *testing.T) {
	tokenResponse := map[string]interface{}{
		"access_token": "fresh-token",
		"token_type":   "Bearer",
		"expires_in":   3600,
	}

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse)
	}))
	defer server.Close()

	handler := &OAuth2Handler{}

	// First request - should fetch token
	req1 := &client.Request{Method: "GET", URL: "http://example.com"}
	handler.Apply(req1, map[string]string{
		"client_id":     "my-client-id",
		"client_secret": "my-client-secret",
		"token_url":     server.URL,
		"flow":          "client_credentials",
	})

	// Second request - should use cached token
	req2 := &client.Request{Method: "GET", URL: "http://example.com"}
	handler.Apply(req2, map[string]string{
		"client_id":     "my-client-id",
		"client_secret": "my-client-secret",
		"token_url":     server.URL,
		"flow":          "client_credentials",
	})

	if requestCount != 1 {
		t.Errorf("expected 1 token request (cached), got %d", requestCount)
	}

	// Both should have same token
	token1 := ""
	token2 := ""
	for _, h := range req1.Headers {
		if h.Key == "Authorization" {
			token1 = h.Value
		}
	}
	for _, h := range req2.Headers {
		if h.Key == "Authorization" {
			token2 = h.Value
		}
	}
	if token1 != token2 {
		t.Errorf("expected same token from cache, got %q and %q", token1, token2)
	}
}

func TestOAuth2Handler_TokenNearExpiryRefreshes(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		// Return a token that expires in 1 second (near expiry)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "token-" + string(rune('a'+requestCount-1)),
			"token_type":   "Bearer",
			"expires_in":   1, // 1 second - will be considered near expiry
		})
	}))
	defer server.Close()

	handler := &OAuth2Handler{}

	// First request - should fetch token
	req1 := &client.Request{Method: "GET", URL: "http://example.com"}
	handler.Apply(req1, map[string]string{
		"client_id":     "my-client-id",
		"client_secret": "my-client-secret",
		"token_url":     server.URL,
		"flow":          "client_credentials",
	})

	// Wait for token to be near expiry
	time.Sleep(500 * time.Millisecond)

	// Second request - should refresh since token is near expiry
	req2 := &client.Request{Method: "GET", URL: "http://example.com"}
	handler.Apply(req2, map[string]string{
		"client_id":     "my-client-id",
		"client_secret": "my-client-secret",
		"token_url":     server.URL,
		"flow":          "client_credentials",
	})

	if requestCount != 2 {
		t.Errorf("expected 2 token requests (token should have refreshed), got %d", requestCount)
	}
}

func TestOAuth2Handler_UnsupportedFlow(t *testing.T) {
	handler := &OAuth2Handler{}
	req := &client.Request{
		Method: "GET",
		URL:    "http://example.com",
	}

	params := map[string]string{
		"client_id":     "my-client-id",
		"client_secret": "my-client-secret",
		"token_url":     "http://example.com/token",
		"flow":          "unsupported_flow",
	}

	handler.Apply(req, params)

	// No Authorization header should be set for unsupported flow
	for _, h := range req.Headers {
		if h.Key == "Authorization" {
			t.Error("expected no Authorization header for unsupported flow")
		}
	}
}

func TestOAuth2Handler_MissingClientID(t *testing.T) {
	handler := &OAuth2Handler{}
	req := &client.Request{
		Method: "GET",
		URL:    "http://example.com",
	}

	params := map[string]string{
		"client_secret": "my-client-secret",
		"token_url":     "http://example.com/token",
		"flow":          "client_credentials",
	}

	handler.Apply(req, params)

	// No Authorization header should be set
	for _, h := range req.Headers {
		if h.Key == "Authorization" {
			t.Error("expected no Authorization header when client_id missing")
		}
	}
}

func TestOAuth2Handler_MissingTokenURL(t *testing.T) {
	handler := &OAuth2Handler{}
	req := &client.Request{
		Method: "GET",
		URL:    "http://example.com",
	}

	params := map[string]string{
		"client_id":     "my-client-id",
		"client_secret": "my-client-secret",
		"flow":          "client_credentials",
	}

	handler.Apply(req, params)

	// No Authorization header should be set
	for _, h := range req.Headers {
		if h.Key == "Authorization" {
			t.Error("expected no Authorization header when token_url missing")
		}
	}
}

func TestOAuth2Handler_RegistryIntegration(t *testing.T) {
	server := httptest.NewServer(mockTokenHandler(map[string]interface{}{
		"access_token": "registry-test-token",
		"token_type":   "Bearer",
		"expires_in":   3600,
	}))
	defer server.Close()

	r := NewRegistry()
	r.Register(&OAuth2Handler{})

	req := &client.Request{
		Method: "GET",
		URL:    server.URL,
	}

	r.Apply("oauth2", req, map[string]string{
		"client_id":     "my-client-id",
		"client_secret": "my-client-secret",
		"token_url":     server.URL,
		"flow":          "client_credentials",
	})

	// Execute the request
	httpReq, _ := http.NewRequest(req.Method, req.URL, nil)
	for _, h := range req.Headers {
		httpReq.Header.Set(h.Key, h.Value)
	}

	// Verify Authorization header is set correctly
	authHeader := httpReq.Header.Get("Authorization")
	expected := "Bearer registry-test-token"
	if authHeader != expected {
		t.Errorf("expected Authorization %q, got %q", expected, authHeader)
	}
}
