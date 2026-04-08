package builtins

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/sreeram/gurl/internal/client"
	"github.com/sreeram/gurl/internal/plugins"
)

// newTestRequestContext creates a RequestContext with a GET request for testing.
func newTestRequestContext() *plugins.RequestContext {
	return &plugins.RequestContext{
		Request: &client.Request{
			Method:  "GET",
			URL:     "https://api.example.com/test",
			Headers: []client.Header{},
		},
		Env: map[string]string{},
	}
}

// newTestResponseContextForMiddleware creates a ResponseContext with a 200 OK response for testing.
func newTestResponseContextForMiddleware() *plugins.ResponseContext {
	return &plugins.ResponseContext{
		Request: &client.Request{
			Method: "GET",
			URL:    "https://api.example.com/test",
		},
		Response: &client.Response{
			StatusCode: 200,
			Body:       []byte(`{"status":"ok"}`),
			Headers:    http.Header{"Content-Type": {"application/json"}},
			Duration:   145 * time.Millisecond,
			Size:       17,
		},
		Env: map[string]string{},
	}
}

// TestMiddleware_Timing tests that TimingMiddleware records start time and elapsed time.
func TestMiddleware_Timing(t *testing.T) {
	m := &TimingMiddleware{}

	// Test BeforeRequest records start time
	ctx := newTestRequestContext()
	ctx.Env["_timing_start"] = ""
	result := m.BeforeRequest(ctx)
	if result == nil {
		t.Fatal("BeforeRequest returned nil")
	}
	if result.Env["_timing_start"] == "" {
		t.Error("BeforeRequest did not record _timing_start")
	}

	// Simulate some time passing
	time.Sleep(10 * time.Millisecond)

	// Test AfterResponse computes elapsed time
	respCtx := newTestResponseContextForMiddleware()
	respCtx.Env["_timing_start"] = result.Env["_timing_start"]
	respResult := m.AfterResponse(respCtx)
	if respResult == nil {
		t.Fatal("AfterResponse returned nil")
	}
	elapsed, ok := respResult.Env["_timing_elapsed_ms"]
	if !ok {
		t.Fatal("AfterResponse did not set _timing_elapsed_ms")
	}
	// Elapsed should be >= 0 (allowing for timing variance)
	if !strings.HasPrefix(elapsed, "0") && !strings.HasPrefix(elapsed, "1") {
		t.Errorf("AfterResponse returned unexpected elapsed: %s", elapsed)
	}
}

// TestMiddleware_UserAgent tests that UserAgentMiddleware sets User-Agent header if not present.
func TestMiddleware_UserAgent(t *testing.T) {
	m := &UserAgentMiddleware{Version: "1.0.0"}

	// Test BeforeRequest sets User-Agent when not present
	ctx := newTestRequestContext()
	result := m.BeforeRequest(ctx)
	if result == nil {
		t.Fatal("BeforeRequest returned nil")
	}

	found := false
	for _, h := range result.Request.Headers {
		if h.Key == "User-Agent" && h.Value == "gurl/1.0.0" {
			found = true
			break
		}
	}
	if !found {
		t.Error("BeforeRequest did not set User-Agent header")
	}

	// Test BeforeRequest does not override existing User-Agent
	ctx2 := newTestRequestContext()
	ctx2.Request.Headers = []client.Header{{Key: "User-Agent", Value: "custom-agent/2.0"}}
	result2 := m.BeforeRequest(ctx2)
	if result2 == nil {
		t.Fatal("BeforeRequest returned nil")
	}

	found = false
	for _, h := range result2.Request.Headers {
		if h.Key == "User-Agent" && h.Value == "custom-agent/2.0" {
			found = true
			break
		}
	}
	if !found {
		t.Error("BeforeRequest overrode existing User-Agent header")
	}

	// Test AfterResponse is pass-through
	respCtx := newTestResponseContextForMiddleware()
	respResult := m.AfterResponse(respCtx)
	if respResult != respCtx {
		t.Error("AfterResponse did not return same context (pass-through)")
	}
}

// TestMiddleware_Logging tests that LoggingMiddleware logs requests and responses with redaction.
func TestMiddleware_Logging(t *testing.T) {
	var buf bytes.Buffer
	m := &LoggingMiddleware{output: &buf}

	// Test BeforeRequest logs method and URL
	ctx := newTestRequestContext()
	ctx.Request.Headers = []client.Header{
		{Key: "Authorization", Value: "Bearer secret-token"},
		{Key: "Content-Type", Value: "application/json"},
	}
	result := m.BeforeRequest(ctx)
	if result == nil {
		t.Fatal("BeforeRequest returned nil")
	}

	output := buf.String()
	if !strings.Contains(output, "→ GET https://api.example.com/test") {
		t.Errorf("BeforeRequest log missing request line: %s", output)
	}
	if !strings.Contains(output, "Authorization: [REDACTED]") {
		t.Errorf("BeforeRequest log did not redact Authorization: %s", output)
	}
	if strings.Contains(output, "secret-token") {
		t.Errorf("BeforeRequest log leaked secret: %s", output)
	}
	if !strings.Contains(output, "Content-Type: application/json") {
		t.Errorf("BeforeRequest log missing Content-Type: %s", output)
	}

	// Test AfterResponse logs status code, duration, and size
	buf.Reset()
	respCtx := newTestResponseContextForMiddleware()
	respResult := m.AfterResponse(respCtx)
	if respResult == nil {
		t.Fatal("AfterResponse returned nil")
	}

	output = buf.String()
	if !strings.Contains(output, "← 200") {
		t.Errorf("AfterResponse log missing status code: %s", output)
	}
	if !strings.Contains(output, "ms)") {
		t.Errorf("AfterResponse log missing duration: %s", output)
	}
	if !strings.Contains(output, "B") {
		t.Errorf("AfterResponse log missing size: %s", output)
	}
}

// TestMiddleware_Retry401 tests that Retry401Middleware sets _retry_401 flag on 401 responses.
func TestMiddleware_Retry401(t *testing.T) {
	m := &Retry401Middleware{}

	// Test BeforeRequest is pass-through
	ctx := newTestRequestContext()
	result := m.BeforeRequest(ctx)
	if result != ctx {
		t.Error("BeforeRequest did not return same context (pass-through)")
	}

	// Test AfterResponse sets flag on 401
	respCtx401 := newTestResponseContext()
	respCtx401.Response.StatusCode = 401
	respResult := m.AfterResponse(respCtx401)
	if respResult == nil {
		t.Fatal("AfterResponse returned nil for 401")
	}
	if respResult.Env["_retry_401"] != "true" {
		t.Errorf("AfterResponse did not set _retry_401 flag for 401: %v", respResult.Env)
	}

	// Test AfterResponse does not set flag on 200
	respCtx200 := newTestResponseContext()
	respCtx200.Response.StatusCode = 200
	respResult200 := m.AfterResponse(respCtx200)
	if respResult200 == nil {
		t.Fatal("AfterResponse returned nil for 200")
	}
	if respResult200.Env["_retry_401"] == "true" {
		t.Error("AfterResponse set _retry_401 flag for 200")
	}
}

// TestMiddleware_Chain tests that middleware chain applies in correct order.
func TestMiddleware_Chain(t *testing.T) {
	registry := plugins.NewRegistry()
	RegisterBuiltins(registry)

	middleware := registry.Middleware()
	if len(middleware) != 4 {
		t.Fatalf("Expected 4 middleware, got %d", len(middleware))
	}

	// Verify order: timing, user-agent, retry, logging
	if middleware[0].Name() != "timing" {
		t.Errorf("Expected timing first, got %s", middleware[0].Name())
	}
	if middleware[1].Name() != "user-agent" {
		t.Errorf("Expected user-agent second, got %s", middleware[1].Name())
	}
	if middleware[2].Name() != "retry-401" {
		t.Errorf("Expected retry-401 third, got %s", middleware[2].Name())
	}
	if middleware[3].Name() != "logging" {
		t.Errorf("Expected logging last, got %s", middleware[3].Name())
	}

	// Test full chain - BeforeRequest
	ctx := newTestRequestContext()
	result := registry.ApplyBeforeRequest(ctx)
	if result == nil {
		t.Fatal("ApplyBeforeRequest returned nil")
	}

	// Timing should have set start time
	if result.Env["_timing_start"] == "" {
		t.Error("Timing middleware did not set _timing_start")
	}

	// UserAgent should have set User-Agent
	found := false
	for _, h := range result.Request.Headers {
		if h.Key == "User-Agent" {
			found = true
			break
		}
	}
	if !found {
		t.Error("UserAgent middleware did not set User-Agent")
	}

	// Test full chain - AfterResponse
	respCtx := newTestResponseContextForMiddleware()
	respCtx.Request = ctx.Request
	respCtx.Env = result.Env
	respResult := registry.ApplyAfterResponse(respCtx)
	if respResult == nil {
		t.Fatal("ApplyAfterResponse returned nil")
	}

	// Retry401 should have set flag for 200 response (no, it shouldn't for 200)
	// Timing should have computed elapsed
	if respResult.Env["_timing_elapsed_ms"] == "" {
		t.Error("Timing middleware did not compute _timing_elapsed_ms")
	}
}

// testCustomMiddleware is a test middleware that injects X-Request-ID from env.
type testCustomMiddleware struct{}

func (c *testCustomMiddleware) Name() string { return "custom-header" }
func (c *testCustomMiddleware) BeforeRequest(ctx *plugins.RequestContext) *plugins.RequestContext {
	if ctx == nil || ctx.Env == nil {
		return ctx
	}
	if reqID, ok := ctx.Env["X-Request-ID"]; ok {
		ctx.Request.Headers = append(ctx.Request.Headers, client.Header{
			Key:   "X-Request-ID",
			Value: reqID,
		})
	}
	return ctx
}
func (c *testCustomMiddleware) AfterResponse(ctx *plugins.ResponseContext) *plugins.ResponseContext {
	return ctx
}

// TestMiddleware_CustomHeader tests that middleware can inject custom headers from env.
func TestMiddleware_CustomHeader(t *testing.T) {
	m := &testCustomMiddleware{}
	ctx := newTestRequestContext()
	ctx.Env["X-Request-ID"] = "req-12345"

	result := m.BeforeRequest(ctx)
	if result == nil {
		t.Fatal("BeforeRequest returned nil")
	}

	found := false
	for _, h := range result.Request.Headers {
		if h.Key == "X-Request-ID" && h.Value == "req-12345" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Custom middleware did not inject X-Request-ID header")
	}
}

// TestMiddleware_NilContext tests that middleware handles nil context gracefully.
func TestMiddleware_NilContext(t *testing.T) {
	m := &TimingMiddleware{}
	if m.BeforeRequest(nil) != nil {
		t.Error("TimingMiddleware.BeforeRequest did not return nil for nil input")
	}
	if m.AfterResponse(nil) != nil {
		t.Error("TimingMiddleware.AfterResponse did not return nil for nil input")
	}

	um := &UserAgentMiddleware{}
	if um.BeforeRequest(nil) != nil {
		t.Error("UserAgentMiddleware.BeforeRequest did not return nil for nil input")
	}
	if um.AfterResponse(nil) != nil {
		t.Error("UserAgentMiddleware.AfterResponse did not return nil for nil input")
	}

	rm := &Retry401Middleware{}
	if rm.BeforeRequest(nil) != nil {
		t.Error("Retry401Middleware.BeforeRequest did not return nil for nil input")
	}
	if rm.AfterResponse(nil) != nil {
		t.Error("Retry401Middleware.AfterResponse did not return nil for nil input")
	}
}
