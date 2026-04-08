package scripting

import (
	"strings"
	"testing"
	"time"

	"github.com/sreeram/gurl/internal/client"
)

func TestPreRequest_ModifyHeader(t *testing.T) {
	eng := NewEngine(nil)
	req := &client.Request{
		Method: "GET",
		URL:    "https://example.com/api",
		Headers: []client.Header{
			{Key: "Content-Type", Value: "application/json"},
		},
	}

	script := `gurl.request.headers.set("X-Custom-Header", "custom-value")`

	modifiedReq, err := RunPreRequest(eng, script, req)
	if err != nil {
		t.Fatalf("RunPreRequest failed: %v", err)
	}

	found := false
	for _, h := range modifiedReq.Headers {
		if h.Key == "X-Custom-Header" && h.Value == "custom-value" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected X-Custom-Header to be set, got headers: %+v", modifiedReq.Headers)
	}
}

func TestPreRequest_ModifyURL(t *testing.T) {
	eng := NewEngine(nil)
	req := &client.Request{
		Method:  "GET",
		URL:     "https://original.com/api",
		Headers: []client.Header{},
	}

	script := `gurl.request.url = "https://modified.com/api/v2"`

	modifiedReq, err := RunPreRequest(eng, script, req)
	if err != nil {
		t.Fatalf("RunPreRequest failed: %v", err)
	}

	if modifiedReq.URL != "https://modified.com/api/v2" {
		t.Errorf("Expected URL 'https://modified.com/api/v2', got '%s'", modifiedReq.URL)
	}
}

func TestPreRequest_SetAuthToken(t *testing.T) {
	eng := NewEngine(nil)
	req := &client.Request{
		Method:  "GET",
		URL:     "https://example.com/api",
		Headers: []client.Header{},
	}

	script := `gurl.request.headers.set("Authorization", "Bearer test-token-123")`

	modifiedReq, err := RunPreRequest(eng, script, req)
	if err != nil {
		t.Fatalf("RunPreRequest failed: %v", err)
	}

	found := false
	for _, h := range modifiedReq.Headers {
		if h.Key == "Authorization" && h.Value == "Bearer test-token-123" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected Authorization header to be set, got headers: %+v", modifiedReq.Headers)
	}
}

func TestPreRequest_ModifyBody(t *testing.T) {
	eng := NewEngine(nil)
	req := &client.Request{
		Method: "POST",
		URL:    "https://example.com/api/users",
		Headers: []client.Header{
			{Key: "Content-Type", Value: "application/json"},
		},
		Body: `{"name":"old-name"}`,
	}

	script := `gurl.request.body = JSON.stringify({"name":"new-name","added":true})`

	modifiedReq, err := RunPreRequest(eng, script, req)
	if err != nil {
		t.Fatalf("RunPreRequest failed: %v", err)
	}

	if !strings.Contains(modifiedReq.Body, "new-name") {
		t.Errorf("Expected body to contain 'new-name', got '%s'", modifiedReq.Body)
	}
	if !strings.Contains(modifiedReq.Body, "added") {
		t.Errorf("Expected body to contain 'added', got '%s'", modifiedReq.Body)
	}
}

func TestPreRequest_SkipRequest(t *testing.T) {
	eng := NewEngine(nil)
	req := &client.Request{
		Method:  "GET",
		URL:     "https://example.com/api",
		Headers: []client.Header{},
	}

	script := `gurl.skipRequest()`

	modifiedReq, err := RunPreRequest(eng, script, req)
	if err != nil {
		t.Fatalf("RunPreRequest failed: %v", err)
	}

	if !eng.skipRequest {
		t.Errorf("Expected skipRequest to be true after gurl.skipRequest()")
	}

	_ = modifiedReq
}

func TestPreRequest_GenerateTimestamp(t *testing.T) {
	eng := NewEngine(nil)
	req := &client.Request{
		Method:  "GET",
		URL:     "https://example.com/api",
		Headers: []client.Header{},
	}

	script := `gurl.request.headers.set("X-Timestamp", String(Date.now()))`

	modifiedReq, err := RunPreRequest(eng, script, req)
	if err != nil {
		t.Fatalf("RunPreRequest failed: %v", err)
	}

	found := false
	for _, h := range modifiedReq.Headers {
		if h.Key == "X-Timestamp" {
			found = true
			if h.Value == "" {
				t.Errorf("Expected X-Timestamp to have a value, got empty string")
			}
			break
		}
	}
	if !found {
		t.Errorf("Expected X-Timestamp header to be set, got headers: %+v", modifiedReq.Headers)
	}
}

func TestPreRequest_ErrorHaltsExecution(t *testing.T) {
	eng := NewEngine(nil)
	req := &client.Request{
		Method:  "GET",
		URL:     "https://example.com/api",
		Headers: []client.Header{},
	}

	script := `throw new Error("Script intentional error")`

	modifiedReq, err := RunPreRequest(eng, script, req)
	if err == nil {
		t.Fatalf("Expected error from RunPreRequest, got nil")
	}

	if !strings.Contains(err.Error(), "Script intentional error") {
		t.Errorf("Expected error message to contain 'Script intentional error', got: %v", err)
	}

	_ = modifiedReq
}

func TestPreRequest_RemoveHeader(t *testing.T) {
	eng := NewEngine(nil)
	req := &client.Request{
		Method: "GET",
		URL:    "https://example.com/api",
		Headers: []client.Header{
			{Key: "X-To-Remove", Value: "remove-me"},
			{Key: "X-To-Keep", Value: "keep-me"},
		},
	}

	script := `gurl.request.headers.remove("X-To-Remove")`

	modifiedReq, err := RunPreRequest(eng, script, req)
	if err != nil {
		t.Fatalf("RunPreRequest failed: %v", err)
	}

	foundRemove := false
	foundKeep := false
	for _, h := range modifiedReq.Headers {
		if h.Key == "X-To-Remove" {
			foundRemove = true
		}
		if h.Key == "X-To-Keep" {
			foundKeep = true
		}
	}
	if foundRemove {
		t.Errorf("Expected X-To-Remove to be removed, but it was found in headers: %+v", modifiedReq.Headers)
	}
	if !foundKeep {
		t.Errorf("Expected X-To-Keep to remain, but it was not found in headers: %+v", modifiedReq.Headers)
	}
}

func TestPreRequest_PreserveExistingHeaders(t *testing.T) {
	eng := NewEngine(nil)
	req := &client.Request{
		Method: "GET",
		URL:    "https://example.com/api",
		Headers: []client.Header{
			{Key: "Existing-Header", Value: "existing-value"},
		},
	}

	script := `gurl.request.headers.set("New-Header", "new-value")`

	modifiedReq, err := RunPreRequest(eng, script, req)
	if err != nil {
		t.Fatalf("RunPreRequest failed: %v", err)
	}

	if len(modifiedReq.Headers) != 2 {
		t.Errorf("Expected 2 headers, got %d: %+v", len(modifiedReq.Headers), modifiedReq.Headers)
	}

	foundExisting := false
	foundNew := false
	for _, h := range modifiedReq.Headers {
		if h.Key == "Existing-Header" && h.Value == "existing-value" {
			foundExisting = true
		}
		if h.Key == "New-Header" && h.Value == "new-value" {
			foundNew = true
		}
	}
	if !foundExisting {
		t.Errorf("Expected Existing-Header to be preserved, got headers: %+v", modifiedReq.Headers)
	}
	if !foundNew {
		t.Errorf("Expected New-Header to be added, got headers: %+v", modifiedReq.Headers)
	}
}

func TestPreRequest_SetMultipleHeaders(t *testing.T) {
	eng := NewEngine(nil)
	req := &client.Request{
		Method:  "GET",
		URL:     "https://example.com/api",
		Headers: []client.Header{},
	}

	script := `
		gurl.request.headers.set("Header1", "value1");
		gurl.request.headers.set("Header2", "value2");
		gurl.request.headers.set("Header3", "value3");
	`

	modifiedReq, err := RunPreRequest(eng, script, req)
	if err != nil {
		t.Fatalf("RunPreRequest failed: %v", err)
	}

	if len(modifiedReq.Headers) != 3 {
		t.Errorf("Expected 3 headers, got %d: %+v", len(modifiedReq.Headers), modifiedReq.Headers)
	}
}

func TestPreRequest_BodyFromEmpty(t *testing.T) {
	eng := NewEngine(nil)
	req := &client.Request{
		Method:  "POST",
		URL:     "https://example.com/api",
		Headers: []client.Header{},
		Body:    "",
	}

	script := `gurl.request.body = '{"message":"hello"}'`

	modifiedReq, err := RunPreRequest(eng, script, req)
	if err != nil {
		t.Fatalf("RunPreRequest failed: %v", err)
	}

	if modifiedReq.Body != `{"message":"hello"}` {
		t.Errorf("Expected body to be set, got '%s'", modifiedReq.Body)
	}
}

func TestPreRequest_ConsoleLogInScript(t *testing.T) {
	eng := NewEngine(nil)
	req := &client.Request{
		Method:  "GET",
		URL:     "https://example.com/api",
		Headers: []client.Header{},
	}

	script := `
		console.log("Pre-request script running");
		gurl.request.headers.set("X-Log", "success");
	`

	modifiedReq, err := RunPreRequest(eng, script, req)
	if err != nil {
		t.Fatalf("RunPreRequest failed: %v", err)
	}

	if !strings.Contains(eng.outputBuffer, "Pre-request script running") {
		t.Errorf("Expected console log output, got: '%s'", eng.outputBuffer)
	}

	found := false
	for _, h := range modifiedReq.Headers {
		if h.Key == "X-Log" && h.Value == "success" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected X-Log header to be set, got headers: %+v", modifiedReq.Headers)
	}
}

func TestPreRequest_DoesNotAffectOriginalRequest(t *testing.T) {
	eng := NewEngine(nil)
	originalReq := &client.Request{
		Method: "GET",
		URL:    "https://example.com/api",
		Headers: []client.Header{
			{Key: "Original", Value: "header"},
		},
	}

	script := `
		gurl.request.headers.set("Modified", "header");
		gurl.request.url = "https://modified.com/api";
	`

	_, err := RunPreRequest(eng, script, originalReq)
	if err != nil {
		t.Fatalf("RunPreRequest failed: %v", err)
	}

	if originalReq.URL != "https://example.com/api" {
		t.Errorf("Expected original URL to be unchanged, got '%s'", originalReq.URL)
	}
	if len(originalReq.Headers) != 1 || originalReq.Headers[0].Key != "Original" {
		t.Errorf("Expected original headers to be unchanged, got %+v", originalReq.Headers)
	}
}

func TestPreRequest_WithTimeout(t *testing.T) {
	eng := NewEngine(nil, WithTimeout(100*time.Millisecond))
	req := &client.Request{
		Method:  "GET",
		URL:     "https://example.com/api",
		Headers: []client.Header{},
	}

	script := `
		var start = Date.now();
		while (Date.now() - start < 5000) {}
		gurl.request.headers.set("X-Done", "true");
	`

	_, err := RunPreRequest(eng, script, req)
	if err == nil {
		t.Fatalf("Expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "deadline") && !strings.Contains(err.Error(), "canceled") {
		t.Errorf("Expected deadline/canceled error, got: %v", err)
	}
}
