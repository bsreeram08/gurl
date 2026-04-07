package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
