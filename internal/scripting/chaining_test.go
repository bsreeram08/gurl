package scripting

import (
	"testing"

	"github.com/sreeram/gurl/internal/client"
)

func TestChain_SetNextRequest(t *testing.T) {
	eng := NewEngine(nil)
	ce := NewChainExecutor(eng)

	script := `gurl.setNextRequest("login")`

	resp := &client.Response{
		StatusCode: 200,
		Body:       []byte(`{"token":"abc123"}`),
	}

	_, err := RunPostResponse(eng, script, resp)
	if err != nil {
		t.Fatalf("RunPostResponse failed: %v", err)
	}

	if eng.nextRequest != "login" {
		t.Errorf("Expected nextRequest to be 'login', got '%s'", eng.nextRequest)
	}

	ce.MarkIteration(eng.nextRequest)
	nextReq := ce.GetNextRequest()
	if nextReq != "login" {
		t.Errorf("Expected GetNextRequest to return 'login', got '%s'", nextReq)
	}
}

func TestChain_PassVariable(t *testing.T) {
	eng := NewEngine(nil)

	script1 := `
		gurl.setVar("auth_token", "Bearer xyz789");
		gurl.setNextRequest("profile");
	`
	resp1 := &client.Response{
		StatusCode: 200,
		Body:       []byte(`{"logged_in":true}`),
	}

	_, err := RunPostResponse(eng, script1, resp1)
	if err != nil {
		t.Fatalf("RunPostResponse failed: %v", err)
	}

	if eng.variables["auth_token"] != "Bearer xyz789" {
		t.Errorf("Expected auth_token to be 'Bearer xyz789', got '%s'", eng.variables["auth_token"])
	}

	script2 := `
		var token = gurl.getVar("auth_token");
		gurl.request.headers.set("Authorization", token);
	`
	req := &client.Request{
		Method:  "GET",
		URL:     "https://example.com/api/profile",
		Headers: []client.Header{{Key: "Existing-Header", Value: "existing-value"}},
	}

	modifiedReq, err := RunPreRequest(eng, script2, req)
	if err != nil {
		t.Fatalf("RunPreRequest failed: %v", err)
	}

	found := false
	for _, h := range modifiedReq.Headers {
		if h.Key == "Authorization" && h.Value == "Bearer xyz789" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected Authorization header to be set with persisted variable, got headers: %+v", modifiedReq.Headers)
	}
}

func TestChain_StopChain(t *testing.T) {
	eng := NewEngine(nil)
	ce := NewChainExecutor(eng)

	script := `gurl.setNextRequest(null)`
	resp := &client.Response{
		StatusCode: 200,
		Body:       []byte(`{}`),
	}

	_, err := RunPostResponse(eng, script, resp)
	if err != nil {
		t.Fatalf("RunPostResponse failed: %v", err)
	}

	ce.MarkIteration(eng.nextRequest)
	nextReq := ce.GetNextRequest()
	if nextReq != "" {
		t.Errorf("Expected GetNextRequest to return empty string (stop chain), got '%s'", nextReq)
	}
}

func TestChain_CircularDetection(t *testing.T) {
	eng := NewEngine(nil)
	ce := NewChainExecutor(eng)

	requests := []string{"request1", "request2", "request1", "request2", "request1"}

	for _, reqName := range requests {
		eng.nextRequest = reqName
		ce.MarkIteration(eng.nextRequest)
	}

	if !ce.IsCircular() {
		t.Errorf("Expected circular detection to be true after 3 repetitions")
	}

	ce2 := NewChainExecutor(eng)
	requests2 := []string{"request2", "request2", "request2"}
	for _, reqName := range requests2 {
		eng.nextRequest = reqName
		ce2.MarkIteration(eng.nextRequest)
	}

	if !ce2.IsCircular() {
		t.Errorf("Expected circular detection to be true after 3 repetitions of request2")
	}

	ce3 := NewChainExecutor(eng)
	requests3 := []string{"request1", "request2", "request1"}
	for _, reqName := range requests3 {
		eng.nextRequest = reqName
		ce3.MarkIteration(eng.nextRequest)
	}

	if ce3.IsCircular() {
		t.Errorf("Expected circular detection to be false with only 2 repetitions")
	}
}

func TestChain_MaxIterations(t *testing.T) {
	eng := NewEngine(nil)

	ce := NewChainExecutor(eng)
	if ce.maxIterations != 100 {
		t.Errorf("Expected default maxIterations to be 100, got %d", ce.maxIterations)
	}

	ceCustom := NewChainExecutor(eng, WithMaxIterations(50))
	if ceCustom.maxIterations != 50 {
		t.Errorf("Expected custom maxIterations to be 50, got %d", ceCustom.maxIterations)
	}

	for i := 0; i < 100; i++ {
		eng.nextRequest = "request"
		ce.MarkIteration(eng.nextRequest)
	}

	if !ce.MaxIterationsReached() {
		t.Errorf("Expected MaxIterationsReached to be true after 100 iterations")
	}
}

func TestChain_ConditionalBranch(t *testing.T) {
	eng := NewEngine(nil)

	script := `
		if (gurl.response.status === 200) {
			gurl.setNextRequest("success-path");
		} else {
			gurl.setNextRequest("error-handler");
		}
	`
	resp := &client.Response{
		StatusCode: 200,
		Body:       []byte(`{"data":"ok"}`),
	}

	_, err := RunPostResponse(eng, script, resp)
	if err != nil {
		t.Fatalf("RunPostResponse failed: %v", err)
	}

	if eng.nextRequest != "success-path" {
		t.Errorf("Expected nextRequest to be 'success-path' for status 200, got '%s'", eng.nextRequest)
	}

	eng2 := NewEngine(nil)
	resp404 := &client.Response{
		StatusCode: 404,
		Body:       []byte(`{"error":"not found"}`),
	}

	_, err = RunPostResponse(eng2, script, resp404)
	if err != nil {
		t.Fatalf("RunPostResponse failed: %v", err)
	}

	if eng2.nextRequest != "error-handler" {
		t.Errorf("Expected nextRequest to be 'error-handler' for status 404, got '%s'", eng2.nextRequest)
	}
}

func TestChain_ExecutionOrder(t *testing.T) {
	eng := NewEngine(nil)
	ce := NewChainExecutor(eng)

	flow := []string{"login", "get-token", "profile"}
	executed := make([]string, 0)

	for i, reqName := range flow {
		eng.nextRequest = reqName
		ce.MarkIteration(eng.nextRequest)

		if i == 0 && ce.GetNextRequest() != "login" {
			t.Errorf("Expected first request to be 'login'")
		}

		switch reqName {
		case "login":
			eng.nextRequest = "get-token"
		case "get-token":
			eng.nextRequest = "profile"
		case "profile":
			eng.nextRequest = ""
		}

		ce.MarkIteration(eng.nextRequest)
		executed = append(executed, reqName)
	}

	if len(executed) != 3 {
		t.Errorf("Expected 3 requests to be executed, got %d", len(executed))
	}

	expected := []string{"login", "get-token", "profile"}
	for i, exp := range expected {
		if executed[i] != exp {
			t.Errorf("Expected execution order [%d] to be '%s', got '%s'", i, exp, executed[i])
		}
	}
}
