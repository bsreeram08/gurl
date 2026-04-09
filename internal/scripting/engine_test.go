package scripting

import (
	"strings"
	"testing"
	"time"
)

func newTestEngine(t *testing.T, opts ...EngineOption) *Engine {
	eng := NewEngine(nil, opts...)
	eng.variables = map[string]string{
		"BASE_URL": "https://example.com",
		"TOKEN":    "test-token-123",
	}
	return eng
}

func TestJS_BasicExecution(t *testing.T) {
	eng := newTestEngine(t)
	result, err := eng.Execute("var x = 1 + 2; x")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Error != nil {
		t.Fatalf("Script error: %v", result.Error)
	}
	val, ok := result.Value.(int64)
	if !ok {
		t.Fatalf("Expected int64, got %T", result.Value)
	}
	if val != 3 {
		t.Errorf("Expected 3, got %d", val)
	}
}

func TestJS_ConsoleLog(t *testing.T) {
	eng := newTestEngine(t)
	_, err := eng.Execute(`console.log("hello")`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if len(eng.outputBuffer) == 0 {
		t.Fatal("Expected console output, got empty buffer")
	}
	if !strings.Contains(eng.outputBuffer, "hello") {
		t.Errorf("Expected 'hello' in output, got: %s", eng.outputBuffer)
	}
}

func TestJS_ConsoleWarn(t *testing.T) {
	eng := newTestEngine(t)
	_, err := eng.Execute(`console.warn("warning message")`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(eng.outputBuffer, "warning message") {
		t.Errorf("Expected 'warning message' in output, got: %s", eng.outputBuffer)
	}
}

func TestJS_ConsoleError(t *testing.T) {
	eng := newTestEngine(t)
	_, err := eng.Execute(`console.error("error message")`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(eng.outputBuffer, "error message") {
		t.Errorf("Expected 'error message' in output, got: %s", eng.outputBuffer)
	}
}

func TestJS_SetVariable(t *testing.T) {
	eng := newTestEngine(t)
	_, err := eng.Execute(`gurl.setVar("myToken", "secret123")`)
	if err != nil {
		t.Fatalf("setVar failed: %v", err)
	}
	result, err := eng.Execute(`gurl.getVar("myToken")`)
	if err != nil {
		t.Fatalf("getVar failed: %v", err)
	}
	val := result.Value.(string)
	if val != "secret123" {
		t.Errorf("Expected 'secret123', got '%s'", val)
	}
}

func TestJS_GetVariable(t *testing.T) {
	eng := newTestEngine(t)
	result, err := eng.Execute(`gurl.getVar("TOKEN")`)
	if err != nil {
		t.Fatalf("getVar failed: %v", err)
	}
	val := result.Value.(string)
	if val != "test-token-123" {
		t.Errorf("Expected 'test-token-123', got '%s'", val)
	}
}

func TestJS_GetVariable_NotFound(t *testing.T) {
	eng := newTestEngine(t)
	result, err := eng.Execute(`gurl.getVar("NONEXISTENT_VAR")`)
	if err != nil {
		t.Fatalf("getVar failed: %v", err)
	}
	if result.Value != nil {
		t.Errorf("Expected nil for nonexistent var, got %v", result.Value)
	}
}

func TestJS_SetHeader(t *testing.T) {
	eng := newTestEngine(t)
	req := &ScriptRequest{
		Method: "GET",
		URL:    "https://example.com",
		Headers: []Header{
			{Key: "Content-Type", Value: "application/json"},
		},
	}
	eng.PrepareRequest(req)

	_, err := eng.Execute(`gurl.request.headers.set("X-Custom", "my-value")`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	found := false
	for _, h := range req.Headers {
		if h.Key == "X-Custom" && h.Value == "my-value" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected X-Custom header to be set, got headers: %+v", req.Headers)
	}
}

func TestJS_ReadResponse(t *testing.T) {
	eng := newTestEngine(t)
	resp := &ScriptResponse{
		Status:     200,
		Body:       []byte(`{"message":"success"}`),
		StatusText: "OK",
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Time:       150 * time.Millisecond,
		Size:       24,
	}
	eng.PrepareResponse(resp)

	result, err := eng.Execute(`
		gurl.response.status;
	`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Value.(int64) != 200 {
		t.Errorf("Expected status 200, got %v", result.Value)
	}

	result, err = eng.Execute(`gurl.response.body`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(result.Value.(string), "success") {
		t.Errorf("Expected body to contain 'success', got %v", result.Value)
	}
}

func TestJS_Assert(t *testing.T) {
	eng := newTestEngine(t)
	resp := &ScriptResponse{
		Status:     200,
		Body:       []byte(`{"message":"success"}`),
		StatusText: "OK",
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Time:       150 * time.Millisecond,
		Size:       24,
	}
	eng.PrepareResponse(resp)

	_, err := eng.Execute(`
		gurl.test("status is 200", function() {
			gurl.expect(gurl.response.status).to.equal(200);
		});
	`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if len(eng.testResults) == 0 {
		t.Fatal("Expected test results, got none")
	}
	if eng.testResults[0].Name != "status is 200" {
		t.Errorf("Expected test name 'status is 200', got '%s'", eng.testResults[0].Name)
	}
	if !eng.testResults[0].Passed {
		t.Errorf("Expected test to pass, got failed: %s", eng.testResults[0].Error)
	}
}

func TestJS_Assert_Failed(t *testing.T) {
	eng := newTestEngine(t)
	resp := &ScriptResponse{
		Status:     404,
		Body:       []byte(`{"error":"not found"}`),
		StatusText: "Not Found",
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Time:       50 * time.Millisecond,
		Size:       20,
	}
	eng.PrepareResponse(resp)

	_, _ = eng.Execute(`
		gurl.test("status is 200", function() {
			gurl.expect(gurl.response.status).to.equal(200);
		});
	`)
	if len(eng.testResults) == 0 {
		t.Fatal("Expected test results, got none")
	}
	if eng.testResults[0].Passed {
		t.Errorf("Expected test to fail, got passed")
	}
}

func TestJS_Timeout(t *testing.T) {
	eng := newTestEngine(t, WithTimeout(100*time.Millisecond))
	_, err := eng.Execute(`
		var start = Date.now();
		while (Date.now() - start < 3000) {
		}
	`)
	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "deadline") && !strings.Contains(err.Error(), "canceled") {
		t.Errorf("Expected deadline/canceled error, got: %v", err)
	}
}

func TestJS_SandboxNoFS(t *testing.T) {
	eng := newTestEngine(t)
	_, err := eng.Execute(`require("fs")`)
	if err == nil {
		t.Fatal("Expected error when requiring fs module, got nil")
	}
}

func TestJS_SandboxNoNet(t *testing.T) {
	eng := newTestEngine(t)
	_, err := eng.Execute(`require("net")`)
	if err == nil {
		t.Fatal("Expected error when requiring net module, got nil")
	}
}

func TestJS_SandboxNoOS(t *testing.T) {
	eng := newTestEngine(t)
	_, err := eng.Execute(`require("os")`)
	if err == nil {
		t.Fatal("Expected error when requiring os module, got nil")
	}
}

func TestJS_SandboxNoChildProcess(t *testing.T) {
	eng := newTestEngine(t)
	_, err := eng.Execute(`require("child_process")`)
	if err == nil {
		t.Fatal("Expected error when requiring child_process module, got nil")
	}
}

func TestJS_SandboxNoHTTP(t *testing.T) {
	eng := newTestEngine(t)
	_, err := eng.Execute(`require("http")`)
	if err == nil {
		t.Fatal("Expected error when requiring http module, got nil")
	}
}

func TestJS_SandboxNoHTTPS(t *testing.T) {
	eng := newTestEngine(t)
	_, err := eng.Execute(`require("https")`)
	if err == nil {
		t.Fatal("Expected error when requiring https module, got nil")
	}
}

func TestJS_CryptoDigestThrows(t *testing.T) {
	eng := newTestEngine(t)
	_, err := eng.Execute(`
		var hash = crypto.createHash("sha256");
		hash.update("hello");
		hash.digest("hex");
	`)
	if err == nil {
		t.Fatal("Expected error when calling digest, got nil")
	}
}

func TestJS_BufferFromThrows(t *testing.T) {
	eng := newTestEngine(t)
	_, err := eng.Execute(`
		var buf = Buffer.from("hello");
	`)
	if err == nil {
		t.Fatal("Expected error when calling Buffer.from, got nil")
	}
}

func TestJS_AllowedJSON(t *testing.T) {
	eng := newTestEngine(t)
	_, err := eng.Execute(`
		var obj = JSON.parse('{"key":"value"}');
		JSON.stringify(obj);
	`)
	if err != nil {
		t.Fatalf("JSON should be allowed, got error: %v", err)
	}
}

func TestJS_AllowedMath(t *testing.T) {
	eng := newTestEngine(t)
	_, err := eng.Execute(`
		Math.random();
		Math.floor(1.5);
	`)
	if err != nil {
		t.Fatalf("Math should be allowed, got error: %v", err)
	}
}

func TestJS_AllowedDate(t *testing.T) {
	eng := newTestEngine(t)
	_, err := eng.Execute(`
		var now = Date.now();
		typeof now;
	`)
	if err != nil {
		t.Fatalf("Date should be allowed, got error: %v", err)
	}
}

func TestJS_skipRequest(t *testing.T) {
	eng := newTestEngine(t)
	eng.PrepareRequest(&ScriptRequest{
		Method: "GET",
		URL:    "https://example.com",
	})

	_, err := eng.Execute(`gurl.skipRequest()`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !eng.skipRequest {
		t.Errorf("Expected skipRequest to be true")
	}
}

func TestJS_setNextRequest(t *testing.T) {
	eng := newTestEngine(t)
	eng.PrepareRequest(&ScriptRequest{
		Method: "GET",
		URL:    "https://example.com",
	})

	_, err := eng.Execute(`gurl.setNextRequest("my-next-request")`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if eng.nextRequest != "my-next-request" {
		t.Errorf("Expected nextRequest to be 'my-next-request', got '%s'", eng.nextRequest)
	}
}

func TestJS_RequestURL(t *testing.T) {
	eng := newTestEngine(t)
	eng.PrepareRequest(&ScriptRequest{
		Method: "GET",
		URL:    "https://example.com/api/users",
	})

	result, err := eng.Execute(`gurl.request.url`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Value.(string) != "https://example.com/api/users" {
		t.Errorf("Expected URL 'https://example.com/api/users', got '%s'", result.Value)
	}
}

func TestJS_RequestMethod(t *testing.T) {
	eng := newTestEngine(t)
	eng.PrepareRequest(&ScriptRequest{
		Method: "POST",
		URL:    "https://example.com/api/users",
	})

	result, err := eng.Execute(`gurl.request.method`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Value.(string) != "POST" {
		t.Errorf("Expected method 'POST', got '%s'", result.Value)
	}
}

func TestJS_RequestBody(t *testing.T) {
	eng := newTestEngine(t)
	eng.PrepareRequest(&ScriptRequest{
		Method: "POST",
		URL:    "https://example.com/api/users",
		Body:   `{"name":"test"}`,
	})

	result, err := eng.Execute(`gurl.request.body`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(result.Value.(string), "test") {
		t.Errorf("Expected body to contain 'test', got '%s'", result.Value)
	}
}

func TestJS_BlockedEval(t *testing.T) {
	eng := newTestEngine(t)
	_, err := eng.Execute(`eval("1+1")`)
	if err == nil {
		t.Fatal("Expected error when calling eval, got nil")
	}
	if !strings.Contains(err.Error(), "eval is not allowed") {
		t.Errorf("Expected 'eval is not allowed' error, got: %v", err)
	}
}

func TestJS_BlockedFunction(t *testing.T) {
	eng := newTestEngine(t)
	_, err := eng.Execute(`new Function("code")()`)
	if err == nil {
		t.Fatal("Expected error when calling Function constructor, got nil")
	}
	if !strings.Contains(err.Error(), "Function is not allowed") {
		t.Errorf("Expected 'Function is not allowed' error, got: %v", err)
	}
}
