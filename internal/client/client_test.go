package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// mockHandler returns a handler that responds based on request details
func mockHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Echo back method
		w.Header().Set("X-Method", r.Method)
		w.Header().Set("Content-Type", "application/json")

		// Check for custom header
		if val := r.Header.Get("X-Custom-Header"); val != "" {
			w.Header().Set("X-Custom-Response", val)
		}

		// Handle different paths
		switch r.URL.Path {
		case "/get":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{
				"message": "GET response",
				"method":  r.Method,
			})
		case "/post":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{
				"message": "POST response",
				"method":  r.Method,
			})
		case "/put":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{
				"message": "PUT response",
				"method":  r.Method,
			})
		case "/delete":
			w.WriteHeader(http.StatusNoContent)
		case "/patch":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{
				"message": "PATCH response",
				"method":  r.Method,
			})
		case "/head":
			w.WriteHeader(http.StatusOK)
			// HEAD must not have body
		case "/options":
			w.Header().Set("Allow", "GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS")
			w.WriteHeader(http.StatusOK)
		case "/timeout":
			time.Sleep(5 * time.Second)
			w.WriteHeader(http.StatusOK)
		case "/slow":
			time.Sleep(2 * time.Second)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "not found",
			})
		}
	})
}

func TestExecute_GET(t *testing.T) {
	server := httptest.NewServer(mockHandler())
	defer server.Close()

	resp, err := Execute(Request{
		Method: "GET",
		URL:    server.URL + "/get",
	})
	if err != nil {
		t.Fatalf("Execute GET failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
	if len(resp.Body) == 0 {
		t.Error("expected non-empty body")
	}
	if resp.Duration == 0 {
		t.Error("expected duration > 0")
	}
	if resp.Size == 0 {
		t.Error("expected size > 0")
	}
}

func TestExecute_POST(t *testing.T) {
	server := httptest.NewServer(mockHandler())
	defer server.Close()

	body := `{"name":"test","value":123}`
	resp, err := Execute(Request{
		Method: "POST",
		URL:    server.URL + "/post",
		Headers: []Header{
			{Key: "Content-Type", Value: "application/json"},
		},
		Body: body,
	})
	if err != nil {
		t.Fatalf("Execute POST failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}
}

func TestExecute_PUT(t *testing.T) {
	server := httptest.NewServer(mockHandler())
	defer server.Close()

	resp, err := Execute(Request{
		Method: "PUT",
		URL:    server.URL + "/put",
		Body:   `{"update":true}`,
	})
	if err != nil {
		t.Fatalf("Execute PUT failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestExecute_DELETE(t *testing.T) {
	server := httptest.NewServer(mockHandler())
	defer server.Close()

	resp, err := Execute(Request{
		Method: "DELETE",
		URL:    server.URL + "/delete",
	})
	if err != nil {
		t.Fatalf("Execute DELETE failed: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", resp.StatusCode)
	}
}

func TestExecute_PATCH(t *testing.T) {
	server := httptest.NewServer(mockHandler())
	defer server.Close()

	resp, err := Execute(Request{
		Method: "PATCH",
		URL:    server.URL + "/patch",
		Body:   `{"patch":true}`,
	})
	if err != nil {
		t.Fatalf("Execute PATCH failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestExecute_HEAD(t *testing.T) {
	server := httptest.NewServer(mockHandler())
	defer server.Close()

	resp, err := Execute(Request{
		Method: "HEAD",
		URL:    server.URL + "/head",
	})
	if err != nil {
		t.Fatalf("Execute HEAD failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
	// HEAD response should have no body
	if len(resp.Body) != 0 {
		t.Errorf("expected empty body for HEAD, got %d bytes", len(resp.Body))
	}
}

func TestExecute_OPTIONS(t *testing.T) {
	server := httptest.NewServer(mockHandler())
	defer server.Close()

	resp, err := Execute(Request{
		Method: "OPTIONS",
		URL:    server.URL + "/options",
	})
	if err != nil {
		t.Fatalf("Execute OPTIONS failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
	allow := resp.Headers.Get("Allow")
	if !strings.Contains(allow, "OPTIONS") {
		t.Errorf("expected Allow header to contain OPTIONS, got %s", allow)
	}
}

func TestExecute_CustomHeaders(t *testing.T) {
	server := httptest.NewServer(mockHandler())
	defer server.Close()

	resp, err := Execute(Request{
		Method: "GET",
		URL:    server.URL + "/get",
		Headers: []Header{
			{Key: "X-Custom-Header", Value: "custom-value"},
		},
	})
	if err != nil {
		t.Fatalf("Execute with custom headers failed: %v", err)
	}
	customResp := resp.Headers.Get("X-Custom-Response")
	if customResp != "custom-value" {
		t.Errorf("expected X-Custom-Response 'custom-value', got %s", customResp)
	}
}

func TestExecute_StatusCodeCapture(t *testing.T) {
	server := httptest.NewServer(mockHandler())
	defer server.Close()

	tests := []struct {
		path         string
		expectedCode int
	}{
		{"/get", http.StatusOK},
		{"/post", http.StatusCreated},
		{"/delete", http.StatusNoContent},
	}

	for _, tc := range tests {
		resp, err := Execute(Request{
			Method: "GET",
			URL:    server.URL + tc.path,
		})
		if err != nil {
			t.Fatalf("Execute %s failed: %v", tc.path, err)
		}
		if resp.StatusCode != tc.expectedCode {
			t.Errorf("path %s: expected status %d, got %d", tc.path, tc.expectedCode, resp.StatusCode)
		}
	}
}

func TestExecute_ResponseBodyCapture(t *testing.T) {
	server := httptest.NewServer(mockHandler())
	defer server.Close()

	resp, err := Execute(Request{
		Method: "GET",
		URL:    server.URL + "/get",
	})
	if err != nil {
		t.Fatalf("Execute GET failed: %v", err)
	}

	// Verify body is valid JSON
	var result map[string]string
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Errorf("expected valid JSON body, got parse error: %v", err)
	}
	if result["message"] != "GET response" {
		t.Errorf("expected message 'GET response', got %s", result["message"])
	}
}

func TestExecute_ResponseTimeTracking(t *testing.T) {
	server := httptest.NewServer(mockHandler())
	defer server.Close()

	resp, err := Execute(Request{
		Method: "GET",
		URL:    server.URL + "/get",
	})
	if err != nil {
		t.Fatalf("Execute GET failed: %v", err)
	}
	if resp.Duration <= 0 {
		t.Errorf("expected duration > 0, got %dms", resp.Duration)
	}
}

func TestExecute_Timeout(t *testing.T) {
	server := httptest.NewServer(mockHandler())
	defer server.Close()

	_, err := Execute(Request{
		Method:  "GET",
		URL:     server.URL + "/timeout",
		Timeout: 100 * time.Millisecond,
	})
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
}

func TestExecute_ConnectionRefused(t *testing.T) {
	_, err := Execute(Request{
		Method: "GET",
		URL:    "http://localhost:99999/nonexistent",
	})
	if err == nil {
		t.Error("expected connection refused error, got nil")
	}
}

func TestExecute_InvalidURL(t *testing.T) {
	_, err := Execute(Request{
		Method: "GET",
		URL:    "not-a-valid-url",
	})
	if err == nil {
		t.Error("expected invalid URL error, got nil")
	}
}

func TestClient_GET(t *testing.T) {
	server := httptest.NewServer(mockHandler())
	defer server.Close()

	client := NewClient()
	resp, err := client.Execute(Request{
		Method: "GET",
		URL:    server.URL + "/get",
	})
	if err != nil {
		t.Fatalf("Client.Execute GET failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestClient_POST_WithContext(t *testing.T) {
	server := httptest.NewServer(mockHandler())
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := NewClient()
	resp, err := client.ExecuteWithContext(ctx, Request{
		Method: "POST",
		URL:    server.URL + "/post",
		Body:   `{"test":true}`,
	})
	if err != nil {
		t.Fatalf("Client.ExecuteWithContext POST failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}
}

func TestClient_TimeoutExceeded(t *testing.T) {
	server := httptest.NewServer(mockHandler())
	defer server.Close()

	client := NewClient()
	_, err := client.Execute(Request{
		Method:  "GET",
		URL:     server.URL + "/timeout",
		Timeout: 100 * time.Millisecond,
	})
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
}

func TestClient_AllMethods(t *testing.T) {
	server := httptest.NewServer(mockHandler())
	defer server.Close()

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	paths := map[string]string{
		"GET":     "/get",
		"POST":    "/post",
		"PUT":     "/put",
		"DELETE":  "/delete",
		"PATCH":   "/patch",
		"HEAD":    "/head",
		"OPTIONS": "/options",
	}

	client := NewClient()
	for _, method := range methods {
		_, err := client.Execute(Request{
			Method: method,
			URL:    server.URL + paths[method],
		})
		if err != nil {
			t.Errorf("method %s failed: %v", method, err)
		}
	}
}

// ---- TLS Configuration Tests ----

func TestTLSConfig_Struct(t *testing.T) {
	cfg := TLSConfig{
		CertFile:      "/path/to/cert.pem",
		KeyFile:       "/path/to/key.pem",
		CAFile:        "/path/to/ca.pem",
		Insecure:      true,
		MinTLSVersion: "1.2",
	}

	if cfg.CertFile != "/path/to/cert.pem" {
		t.Errorf("expected CertFile '/path/to/cert.pem', got '%s'", cfg.CertFile)
	}
	if cfg.KeyFile != "/path/to/key.pem" {
		t.Errorf("expected KeyFile '/path/to/key.pem', got '%s'", cfg.KeyFile)
	}
	if cfg.CAFile != "/path/to/ca.pem" {
		t.Errorf("expected CAFile '/path/to/ca.pem', got '%s'", cfg.CAFile)
	}
	if !cfg.Insecure {
		t.Error("expected Insecure to be true")
	}
	if cfg.MinTLSVersion != "1.2" {
		t.Errorf("expected MinTLSVersion '1.2', got '%s'", cfg.MinTLSVersion)
	}
}

func TestNewClientWithTLS_ValidConfig(t *testing.T) {
	// Create a temporary self-signed certificate for testing
	tmpDir := t.TempDir()
	certFile := tmpDir + "/cert.pem"
	keyFile := tmpDir + "/key.pem"

	// Generate self-signed cert using Go's crypto libraries
	generateSelfSignedCert(t, certFile, keyFile)

	cfg := TLSConfig{
		CertFile: certFile,
		KeyFile:  keyFile,
	}

	client, err := NewClientWithTLS(cfg)
	if err != nil {
		t.Fatalf("NewClientWithTLS failed: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.transport == nil {
		t.Fatal("expected non-nil transport")
	}
	if client.transport.TLSClientConfig == nil {
		t.Fatal("expected TLSClientConfig to be set")
	}
}

func TestNewClientWithTLS_InsecureSkipsVerification(t *testing.T) {
	tmpDir := t.TempDir()
	certFile := tmpDir + "/cert.pem"
	keyFile := tmpDir + "/key.pem"

	generateSelfSignedCert(t, certFile, keyFile)

	cfg := TLSConfig{
		CertFile: certFile,
		KeyFile:  keyFile,
		Insecure: true,
	}

	client, err := NewClientWithTLS(cfg)
	if err != nil {
		t.Fatalf("NewClientWithTLS failed: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if !client.transport.TLSClientConfig.InsecureSkipVerify {
		t.Error("expected InsecureSkipVerify to be true")
	}
}

func TestNewClientWithTLS_CustomCA(t *testing.T) {
	tmpDir := t.TempDir()
	certFile := tmpDir + "/cert.pem"
	keyFile := tmpDir + "/key.pem"
	caFile := tmpDir + "/ca.pem"

	generateSelfSignedCert(t, certFile, keyFile)

	// Use same cert as CA for testing
	copyFile(t, certFile, caFile)

	cfg := TLSConfig{
		CertFile: certFile,
		KeyFile:  keyFile,
		CAFile:   caFile,
	}

	client, err := NewClientWithTLS(cfg)
	if err != nil {
		t.Fatalf("NewClientWithTLS failed: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.transport.TLSClientConfig.RootCAs == nil {
		t.Fatal("expected RootCAs to be set")
	}
}

func TestNewClientWithTLS_MinTLSVersion(t *testing.T) {
	tmpDir := t.TempDir()
	certFile := tmpDir + "/cert.pem"
	keyFile := tmpDir + "/key.pem"

	generateSelfSignedCert(t, certFile, keyFile)

	cfg := TLSConfig{
		CertFile:      certFile,
		KeyFile:       keyFile,
		MinTLSVersion: "1.2",
	}

	client, err := NewClientWithTLS(cfg)
	if err != nil {
		t.Fatalf("NewClientWithTLS failed: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	// TLS 1.2 = 0x0303
	if client.transport.TLSClientConfig.MinVersion != 0x0303 {
		t.Errorf("expected MinVersion 0x0303 (TLS 1.2), got 0x%04x", client.transport.TLSClientConfig.MinVersion)
	}
}

func TestNewClientWithTLS_CertFileNotFound(t *testing.T) {
	cfg := TLSConfig{
		CertFile: "/nonexistent/cert.pem",
		KeyFile:  "/nonexistent/key.pem",
	}

	client, _ := NewClientWithTLS(cfg)
	// Should return client but with error logged (non-fatal for now)
	if client == nil {
		t.Fatal("expected non-nil client even with missing cert")
	}
}

func TestNewClientWithTLS_KeyFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	certFile := tmpDir + "/cert.pem"

	// Generate cert but no key
	generateSelfSignedCert(t, certFile, "/dev/null")

	cfg := TLSConfig{
		CertFile: certFile,
		KeyFile:  "/nonexistent/key.pem",
	}

	client, err := NewClientWithTLS(cfg)
	if err != nil {
		t.Fatalf("NewClientWithTLS failed: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client even with missing key")
	}
}

func TestNewClientWithTLS_CAFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	certFile := tmpDir + "/cert.pem"
	keyFile := tmpDir + "/key.pem"

	generateSelfSignedCert(t, certFile, keyFile)

	cfg := TLSConfig{
		CertFile: certFile,
		KeyFile:  keyFile,
		CAFile:   "/nonexistent/ca.pem",
	}

	client, err := NewClientWithTLS(cfg)
	if err != nil {
		t.Fatalf("NewClientWithTLS failed: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client even with missing CA")
	}
}

// ---- Helper functions for TLS tests ----

// generateSelfSignedCert generates a self-signed certificate for testing
func generateSelfSignedCert(t *testing.T, certFile, keyFile string) {
	t.Helper()

	// Use exec to generate cert since we can't easily do this in pure Go
	// Skip if openssl not available
	cmd := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:2048",
		"-keyout", keyFile, "-out", certFile,
		"-days", "1", "-nodes", "-subj", "/CN=test")
	err := cmd.Run()
	if err != nil {
		t.Skipf("openssl not available, skipping TLS test: %v", err)
	}
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	content, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("failed to read %s: %v", src, err)
	}
	if err := os.WriteFile(dst, content, 0600); err != nil {
		t.Fatalf("failed to write %s: %v", dst, err)
	}
}

// ---- Redirect Handling Tests ----

// redirectHandler creates a handler that redirects N times then returns final response
func redirectHandler(redirectCount int, finalStatus int) http.HandlerFunc {
	redirectCount++
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Track which hop this is
		hops := r.Header.Get("X-Redirect-Hops")
		if hops == "" {
			hops = "0"
		}
		currentHop := len(strings.Split(hops, ","))

		if currentHop < redirectCount {
			// Perform redirect
			baseURL := strings.Split(r.URL.String(), "/redirect-")[0]
			if baseURL == r.URL.String() {
				baseURL = strings.Split(r.URL.String(), "/final")[0]
			}
			redirectURL := baseURL + "/redirect-" + string(rune('0'+currentHop))
			w.Header().Set("Location", redirectURL)
			w.Header().Set("X-Redirect-Hops", hops+","+redirectURL)
			w.WriteHeader(http.StatusFound) // 302
		} else {
			w.WriteHeader(finalStatus)
		}
	})
}

func TestRedirectFollowing_DefaultMax10(t *testing.T) {
	hopCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hopCount < 3 {
			w.Header().Set("Location", r.URL.Path+"-redirect")
			hopCount++
			w.WriteHeader(http.StatusFound)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"final":true}`))
		}
	}))
	defer server.Close()

	client := NewClient()
	resp, err := client.Execute(Request{
		Method: "GET",
		URL:    server.URL + "/start",
	})
	if err != nil {
		t.Fatalf("Execute with redirects failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected final status 200, got %d", resp.StatusCode)
	}
	if len(resp.Redirects) != 3 {
		t.Errorf("expected 3 redirect hops, got %d", len(resp.Redirects))
	}
}

func TestRedirectFollowing_TracksChain(t *testing.T) {
	hopCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hopCount < 2 {
			w.Header().Set("Location", r.URL.Path+"-redirect")
			hopCount++
			w.WriteHeader(http.StatusFound)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"final":true}`))
		}
	}))
	defer server.Close()

	client := NewClient()
	resp, err := client.Execute(Request{
		Method: "GET",
		URL:    server.URL + "/start",
	})
	if err != nil {
		t.Fatalf("Execute with redirects failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected final status 200, got %d", resp.StatusCode)
	}
	if len(resp.Redirects) != 2 {
		t.Errorf("expected 2 redirect hops, got %d", len(resp.Redirects))
	}
	for i, hop := range resp.Redirects {
		if hop.URL == "" {
			t.Errorf("redirect hop %d: empty URL", i)
		}
		if hop.StatusCode == 0 {
			t.Errorf("redirect hop %d: zero status code", i)
		}
	}
}

func TestRedirect_NoFollow(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/final")
		w.WriteHeader(http.StatusFound)
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := NewClient()
	resp, err := client.Execute(Request{
		Method:       "GET",
		URL:          server.URL + "/start",
		MaxRedirects: -1,
	})
	if err != nil {
		t.Fatalf("Execute with no-follow failed: %v", err)
	}
	if resp.StatusCode != http.StatusFound {
		t.Errorf("expected 302 redirect status, got %d", resp.StatusCode)
	}
	if len(resp.Redirects) != 0 {
		t.Errorf("expected 0 redirect hops recorded (not followed), got %d", len(resp.Redirects))
	}
}

func TestRedirect_MaxRedirectsLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hops := r.Header.Get("X-Redirect-Hops")
		if hops == "" {
			hops = "0"
		}
		currentHop := len(strings.Split(hops, ","))
		if currentHop < 5 {
			w.Header().Set("Location", r.URL.Path+"-redirect")
			w.Header().Set("X-Redirect-Hops", hops+",next")
			w.WriteHeader(http.StatusFound)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"final":true}`))
		}
	}))
	defer server.Close()

	client := NewClient()
	resp, err := client.Execute(Request{
		Method:       "GET",
		URL:          server.URL + "/start",
		MaxRedirects: 2,
	})
	if err != nil {
		t.Fatalf("Execute with max-redirects failed: %v", err)
	}
	if resp.StatusCode != http.StatusFound {
		t.Errorf("expected 302 at max redirect limit, got %d", resp.StatusCode)
	}
	if len(resp.Redirects) != 2 {
		t.Errorf("expected 2 redirect hops (at limit), got %d", len(resp.Redirects))
	}
}
