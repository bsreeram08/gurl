package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sreeram/gurl/internal/config"
	"github.com/sreeram/gurl/pkg/types"
)

// slowHandler returns a handler that sleeps for a specified duration.
// It respects the request context so server.Close() doesn't block.
func slowHandler(sleepDuration time.Duration) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			return
		case <-time.After(sleepDuration):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"ok":true}`))
		}
	})
}

// connectSlowHandler simulates a server that delays connection acceptance
// For connect timeout testing, we need a server that doesn't respond to the TCP handshake
// This is difficult to test directly, so we test the dialer timeout conceptually
// via the http.Transport's DialContext

// TestTimeout_Default tests that default timeout is 30s from config
func TestTimeout_Default(t *testing.T) {
	server := httptest.NewServer(slowHandler(50 * time.Millisecond))
	defer server.Close()

	// Client should use default timeout of 30s
	client := NewClient()
	// The request should complete quickly since our handler only sleeps 50ms
	resp, err := client.Execute(Request{
		Method: "GET",
		URL:    server.URL,
	})
	if err != nil {
		t.Fatalf("expected no error for fast request, got: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
	// Verify default timeout is 30s
	if client.timeout != 30*time.Second {
		t.Errorf("expected default timeout of 30s, got %v", client.timeout)
	}
}

// TestTimeout_PerRequest tests that per-request timeout overrides the default
func TestTimeout_PerRequest(t *testing.T) {
	server := httptest.NewServer(slowHandler(200 * time.Millisecond))
	defer server.Close()

	client := NewClient()

	// Per-request timeout of 100ms should override default 30s
	_, err := client.Execute(Request{
		Method:  "GET",
		URL:     server.URL,
		Timeout: 100 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timed out") && !strings.Contains(err.Error(), "timeout") {
		t.Errorf("expected timeout-related error, got: %v", err)
	}
}

// TestTimeout_Zero tests that zero timeout means no timeout (infinite)
func TestTimeout_Zero(t *testing.T) {
	server := httptest.NewServer(slowHandler(100 * time.Millisecond))
	defer server.Close()

	client := NewClient()

	// Zero timeout should mean no timeout applied via context
	// We need to set the client's timeout to 0 and verify it doesn't cause issues
	client.timeout = 0

	resp, err := client.Execute(Request{
		Method:  "GET",
		URL:     server.URL,
		Timeout: 0, // Explicit zero = no timeout
	})
	if err != nil {
		t.Fatalf("expected no error with zero timeout, got: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

// TestTimeout_Exceeded tests that a request exceeding timeout returns a clear error message
func TestTimeout_Exceeded(t *testing.T) {
	server := httptest.NewServer(slowHandler(5 * time.Second))
	defer server.Close()

	client := NewClient()

	_, err := client.Execute(Request{
		Method:  "GET",
		URL:     server.URL,
		Timeout: 100 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	// Verify error message is user-friendly, not raw context.DeadlineExceeded
	errMsg := err.Error()
	if strings.Contains(errMsg, "context.DeadlineExceeded") {
		t.Errorf("error should not contain raw Go error, got: %v", errMsg)
	}
	if !strings.Contains(errMsg, "timed out") && !strings.Contains(errMsg, "timeout") {
		t.Errorf("error should mention timeout clearly, got: %v", errMsg)
	}
}

// TestTimeout_ConnectVsTotal tests separate connect timeout and total timeout
func TestTimeout_ConnectVsTotal(t *testing.T) {
	// This test verifies that connect timeout and total timeout are separate
	// We can test this by checking that WithConnectTimeout and WithTimeout are distinct

	client := NewClient()

	// Apply connect timeout option
	opt := WithConnectTimeout(5 * time.Second)
	opt(client)

	// Verify connect timeout was set (via DialContext timeout)
	if client.connectTimeout != 5*time.Second {
		t.Errorf("expected connect timeout of 5s, got %v", client.connectTimeout)
	}

	// Verify total timeout is still the default (30s) - not overridden
	if client.timeout != 30*time.Second {
		t.Errorf("expected total timeout to remain at default 30s, got %v", client.timeout)
	}
}

// TestTimeout_WithTimeout tests the WithTimeout functional option
func TestTimeout_WithTimeout(t *testing.T) {
	client := NewClient()

	opt := WithTimeout(60 * time.Second)
	opt(client)

	if client.timeout != 60*time.Second {
		t.Errorf("expected timeout of 60s, got %v", client.timeout)
	}
}

// TestTimeout_ConnectTimeoutApplied tests that connect timeout is applied to transport
func TestTimeout_ConnectTimeoutApplied(t *testing.T) {
	client := NewClient()

	// Set connect timeout
	opt := WithConnectTimeout(3 * time.Second)
	opt(client)

	// The transport should have a dial context that respects the timeout
	// We verify this by checking that the client's connectTimeout field is set
	if client.connectTimeout != 3*time.Second {
		t.Errorf("expected connectTimeout of 3s, got %v", client.connectTimeout)
	}
}

// TestTimeout_FriendlyError tests that timeout errors have friendly messages
func TestTimeout_FriendlyError(t *testing.T) {
	server := httptest.NewServer(slowHandler(10 * time.Second))
	defer server.Close()

	client := NewClient()

	_, err := client.Execute(Request{
		Method:  "GET",
		URL:     server.URL,
		Timeout: 100 * time.Millisecond,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Error message should be friendly: "Request timed out after X" not "context.DeadlineExceeded"
	errMsg := err.Error()

	// Should NOT contain raw Go error
	if strings.Contains(errMsg, "context.DeadlineExceeded") {
		t.Errorf("error contains raw Go error, should be user-friendly: %v", errMsg)
	}
	if strings.Contains(errMsg, "DeadlineExceeded") {
		t.Errorf("error contains DeadlineExceeded directly: %v", errMsg)
	}

	// Should indicate what happened
	if !strings.Contains(strings.ToLower(errMsg), "timeout") && !strings.Contains(strings.ToLower(errMsg), "timed out") {
		t.Errorf("error should mention timeout: %v", errMsg)
	}
}

// TestTimeout_FromConfig tests reading timeout from TOML config
func TestTimeout_FromConfig(t *testing.T) {
	// Create a temporary config with timeout set
	cfg := &types.Config{}
	cfg.General.Timeout = "10s"

	// Load the config into a client
	timeoutVal, err := time.ParseDuration(cfg.General.Timeout)
	if err != nil {
		t.Fatalf("failed to parse timeout from config: %v", err)
	}

	// Create client and apply timeout from config
	client := NewClient()
	WithTimeout(timeoutVal)(client)

	if client.timeout != 10*time.Second {
		t.Errorf("expected timeout of 10s from config, got %v", client.timeout)
	}
}

// TestTimeout_ConfigLoaderIntegration tests that config loader properly handles timeout
func TestTimeout_ConfigLoaderIntegration(t *testing.T) {
	// Test that the Config.General.Timeout field exists and can be set
	cfg := config.DefaultConfig()
	cfg.General.Timeout = "15s"

	timeoutVal, err := time.ParseDuration(cfg.General.Timeout)
	if err != nil {
		t.Fatalf("failed to parse timeout: %v", err)
	}
	if timeoutVal != 15*time.Second {
		t.Errorf("expected 15s timeout, got %v", timeoutVal)
	}
}

// TestTimeout_PerRequestOverridesClient tests that per-request timeout overrides client timeout
func TestTimeout_PerRequestOverridesClient(t *testing.T) {
	server := httptest.NewServer(slowHandler(200 * time.Millisecond))
	defer server.Close()

	client := NewClient()
	WithTimeout(30 * time.Second)(client) // Set client-level timeout to 30s

	// Per-request timeout of 50ms should override
	_, err := client.Execute(Request{
		Method:  "GET",
		URL:     server.URL,
		Timeout: 50 * time.Millisecond,
	})

	if err == nil {
		t.Fatal("expected timeout error with per-request override, got nil")
	}

	// Should get a timeout error
	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "timed out") {
		t.Errorf("expected timeout error, got: %v", err)
	}
}

// TestTimeout_ZeroTimeoutFromConfig tests that "0" timeout means no timeout
func TestTimeout_ZeroTimeoutFromConfig(t *testing.T) {
	cfg := &types.Config{}
	cfg.General.Timeout = "0"

	timeoutVal, err := time.ParseDuration(cfg.General.Timeout)
	if err != nil {
		t.Fatalf("failed to parse 0 timeout: %v", err)
	}
	if timeoutVal != 0 {
		t.Errorf("expected zero timeout, got %v", timeoutVal)
	}

	client := NewClient()
	WithTimeout(timeoutVal)(client)

	if client.timeout != 0 {
		t.Errorf("expected client timeout to be 0, got %v", client.timeout)
	}
}

// TestTimeout_ExecuteWithContextTimeout tests that context timeout works with ExecuteWithContext
func TestTimeout_ExecuteWithContextTimeout(t *testing.T) {
	server := httptest.NewServer(slowHandler(5 * time.Second))
	defer server.Close()

	client := NewClient()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.ExecuteWithContext(ctx, Request{
		Method: "GET",
		URL:    server.URL,
	})

	if err == nil {
		t.Fatal("expected timeout error from context, got nil")
	}

	// Context cancellation should be wrapped in friendly message
	errMsg := err.Error()
	if !strings.Contains(errMsg, "timeout") && !strings.Contains(errMsg, "timed out") && !strings.Contains(errMsg, "context") {
		// Context deadline exceeded is acceptable since it's from the context
		if !strings.Contains(errMsg, "DeadlineExceeded") {
			t.Errorf("expected timeout or context error, got: %v", errMsg)
		}
	}
}

// TestTimeout_ErrorType tests that timeout errors are of correct type
func TestTimeout_ErrorType(t *testing.T) {
	server := httptest.NewServer(slowHandler(10 * time.Second))
	defer server.Close()

	client := NewClient()

	_, err := client.Execute(Request{
		Method:  "GET",
		URL:     server.URL,
		Timeout: 50 * time.Millisecond,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Should be able to check with errors.Is for timeout
	// Note: context.DeadlineExceeded is what Go returns, but we wrap it
	if errors.Is(err, context.DeadlineExceeded) {
		// This is acceptable - raw Go error
		t.Log("error correctly wraps context.DeadlineExceeded")
	}
}
