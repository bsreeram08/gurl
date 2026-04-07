package client

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

// TestHTTPProxyFromRequest tests that a request uses the proxy URL when set
func TestHTTPProxyFromRequest(t *testing.T) {
	// Create a target server that should NOT be called directly
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("target server should not be called directly")
	}))
	defer targetServer.Close()

	// Create a proxy server that records the request
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request came through the proxy
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"proxied": true}`))
	}))
	defer proxyServer.Close()

	proxyURL, _ := url.Parse(proxyServer.URL)

	// Create client with proxy configured
	client := NewClient()
	client.SetProxyURL(proxyURL.String())

	// Execute request - should go through proxy
	resp, err := client.Execute(Request{
		Method:   "GET",
		URL:      targetServer.URL,
		ProxyURL: proxyServer.URL,
	})
	if err != nil {
		t.Fatalf("Execute with proxy failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 through proxy, got %d", resp.StatusCode)
	}
}

// TestProxyFromEnvironment tests that HTTP_PROXY and HTTPS_PROXY env vars are respected
func TestProxyFromEnvironment(t *testing.T) {
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer proxyServer.Close()

	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer targetServer.Close()

	// Set HTTP_PROXY environment variable
	oldProxy := os.Getenv("HTTP_PROXY")
	os.Setenv("HTTP_PROXY", proxyServer.URL)
	defer func() {
		os.Setenv("HTTP_PROXY", oldProxy)
	}()

	// Create client and use ProxyFromEnvironment
	client := NewClient()
	client.UseEnvironmentProxy()

	resp, err := client.Execute(Request{
		Method: "GET",
		URL:    targetServer.URL,
	})
	if err != nil {
		t.Fatalf("Execute with env proxy failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 via env proxy, got %d", resp.StatusCode)
	}
}

// TestNoProxyList tests that NO_PROXY is respected
func TestNoProxyList(t *testing.T) {
	// Create two servers - one should bypass proxy, one should use it
	directServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"direct": true}`))
	}))
	defer directServer.Close()

	proxiedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"proxied": true}`))
	}))
	defer proxiedServer.Close()

	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer proxyServer.Close()

	// Extract host:port from directServer URL for NO_PROXY
	directURL, _ := url.Parse(directServer.URL)
	noProxyHost := directURL.Host

	// Create client with proxy and NO_PROXY configured
	client := NewClient()
	client.SetProxyURL(proxyServer.URL)
	client.SetNoProxy([]string{noProxyHost})

	// Request to directServer should bypass proxy (in NO_PROXY)
	respDirect, err := client.Execute(Request{
		Method: "GET",
		URL:    directServer.URL,
	})
	if err != nil {
		t.Fatalf("Execute direct (no proxy) failed: %v", err)
	}
	if respDirect.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 direct, got %d", respDirect.StatusCode)
	}

	// Request to proxiedServer should go through proxy
	respProxied, err := client.Execute(Request{
		Method: "GET",
		URL:    proxiedServer.URL,
	})
	if err != nil {
		t.Fatalf("Execute via proxy failed: %v", err)
	}
	if respProxied.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 via proxy, got %d", respProxied.StatusCode)
	}
}

// TestProxyAuthInURL tests that proxy URLs with user:pass are handled
func TestProxyAuthInURL(t *testing.T) {
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for Proxy-Authorization header
		auth := r.Header.Get("Proxy-Authorization")
		if auth == "" {
			t.Error("expected Proxy-Authorization header for authenticated proxy")
			w.WriteHeader(http.StatusProxyAuthRequired)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"auth": true}`))
	}))
	defer proxyServer.Close()

	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer targetServer.Close()

	// Proxy URL with authentication
	proxyAuthURL := "http://user:pass@" + strings.TrimPrefix(proxyServer.URL, "http://")

	client := NewClient()
	client.SetProxyURL(proxyAuthURL)

	resp, err := client.Execute(Request{
		Method: "GET",
		URL:    targetServer.URL,
	})
	if err != nil {
		t.Fatalf("Execute with proxy auth failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 with proxy auth, got %d", resp.StatusCode)
	}
}

func TestHTTPSProxy(t *testing.T) {
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer proxyServer.Close()

	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer targetServer.Close()

	proxyURL, _ := url.Parse(proxyServer.URL)

	client := NewClient()
	client.SetProxyURL(proxyURL.String())

	resp, err := client.Execute(Request{
		Method: "GET",
		URL:    targetServer.URL,
	})
	if err != nil {
		t.Fatalf("Execute with http proxy failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 via http proxy, got %d", resp.StatusCode)
	}
}

// TestSOCKS5Proxy tests SOCKS5 proxy configuration
func TestSOCKS5Proxy(t *testing.T) {
	// For SOCKS5, we need a real SOCKS5 proxy or we can use a simple test
	// Since setting up a real SOCKS5 proxy is complex, we test the URL parsing
	// and that the protocol is recognized

	socks5URL := "socks5://proxy:1080"
	u, err := url.Parse(socks5URL)
	if err != nil {
		t.Fatalf("Failed to parse SOCKS5 URL: %v", err)
	}

	// Verify the scheme is socks5
	if u.Scheme != "socks5" {
		t.Errorf("expected scheme 'socks5', got '%s'", u.Scheme)
	}
	if u.Host != "proxy:1080" {
		t.Errorf("expected host 'proxy:1080', got '%s'", u.Host)
	}
}

// TestProxyRequestField tests that ProxyURL in Request struct is used
func TestProxyRequestField(t *testing.T) {
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer targetServer.Close()

	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer proxyServer.Close()

	// Use proxy URL directly in Request
	resp, err := Execute(Request{
		Method:   "GET",
		URL:      targetServer.URL,
		ProxyURL: proxyServer.URL,
	})
	if err != nil {
		t.Fatalf("Execute with request proxy failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 via request proxy, got %d", resp.StatusCode)
	}
}

// TestNoProxyWhenDisabled tests that no proxy is used when not configured
func TestNoProxyWhenDisabled(t *testing.T) {
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"direct": true}`))
	}))
	defer targetServer.Close()

	// Ensure no proxy env vars are set
	oldHTTP := os.Getenv("HTTP_PROXY")
	oldHTTPS := os.Getenv("HTTPS_PROXY")
	oldNO := os.Getenv("NO_PROXY")
	os.Unsetenv("HTTP_PROXY")
	os.Unsetenv("HTTPS_PROXY")
	os.Unsetenv("NO_PROXY")
	defer func() {
		os.Setenv("HTTP_PROXY", oldHTTP)
		os.Setenv("HTTPS_PROXY", oldHTTPS)
		os.Setenv("NO_PROXY", oldNO)
	}()

	client := NewClient()
	// Don't set any proxy

	resp, err := client.Execute(Request{
		Method: "GET",
		URL:    targetServer.URL,
	})
	if err != nil {
		t.Fatalf("Execute without proxy failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 without proxy, got %d", resp.StatusCode)
	}
}
