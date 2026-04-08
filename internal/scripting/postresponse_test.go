package scripting

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/sreeram/gurl/internal/client"
)

func TestPostResponse_ReadStatus(t *testing.T) {
	eng := NewEngine(nil)
	resp := &client.Response{
		StatusCode: 200,
		Headers:    http.Header{},
		Body:       []byte(`{"message":"success"}`),
		Duration:   100 * time.Millisecond,
		Size:       24,
	}

	result, err := RunPostResponse(eng, `gurl.response.status`, resp)
	if err != nil {
		t.Fatalf("RunPostResponse failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if len(result.Assertions) != 0 {
		t.Errorf("Expected 0 assertions, got %d", len(result.Assertions))
	}

	evalResult, evalErr := eng.Execute(`gurl.response.status`)
	if evalErr != nil {
		t.Fatalf("Execute failed: %v", evalErr)
	}
	val, ok := evalResult.Value.(int64)
	if !ok {
		t.Fatalf("Expected int64, got %T", evalResult.Value)
	}
	if val != 200 {
		t.Errorf("Expected status 200, got %d", val)
	}
}

func TestPostResponse_ReadBody(t *testing.T) {
	eng := NewEngine(nil)
	resp := &client.Response{
		StatusCode: 200,
		Headers:    http.Header{},
		Body:       []byte(`{"message":"success"}`),
		Duration:   100 * time.Millisecond,
		Size:       24,
	}

	result, err := RunPostResponse(eng, `gurl.response.body`, resp)
	if err != nil {
		t.Fatalf("RunPostResponse failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	evalResult, evalErr := eng.Execute(`gurl.response.body`)
	if evalErr != nil {
		t.Fatalf("Execute failed: %v", evalErr)
	}
	body, ok := evalResult.Value.(string)
	if !ok {
		t.Fatalf("Expected string, got %T", evalResult.Value)
	}
	if !strings.Contains(body, "success") {
		t.Errorf("Expected body to contain 'success', got '%s'", body)
	}
}

func TestPostResponse_ReadHeaders(t *testing.T) {
	eng := NewEngine(nil)
	resp := &client.Response{
		StatusCode: 200,
		Headers:    http.Header{"Content-Type": {"application/json"}},
		Body:       []byte(`{"message":"success"}`),
		Duration:   100 * time.Millisecond,
		Size:       24,
	}

	result, err := RunPostResponse(eng, `gurl.response.headers.get("Content-Type")`, resp)
	if err != nil {
		t.Fatalf("RunPostResponse failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	evalResult, evalErr := eng.Execute(`gurl.response.headers.get("Content-Type")`)
	if evalErr != nil {
		t.Fatalf("Execute failed: %v", evalErr)
	}
	ct, ok := evalResult.Value.(string)
	if !ok {
		t.Fatalf("Expected string, got %T", evalResult.Value)
	}
	if ct != "application/json" {
		t.Errorf("Expected 'application/json', got '%s'", ct)
	}
}

func TestPostResponse_ReadTime(t *testing.T) {
	eng := NewEngine(nil)
	resp := &client.Response{
		StatusCode: 200,
		Headers:    http.Header{},
		Body:       []byte(`{"message":"success"}`),
		Duration:   150 * time.Millisecond,
		Size:       24,
	}

	result, err := RunPostResponse(eng, `gurl.response.time`, resp)
	if err != nil {
		t.Fatalf("RunPostResponse failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	evalResult, evalErr := eng.Execute(`gurl.response.time`)
	if evalErr != nil {
		t.Fatalf("Execute failed: %v", evalErr)
	}
	timeMs, ok := evalResult.Value.(int64)
	if !ok {
		t.Fatalf("Expected int64, got %T", evalResult.Value)
	}
	if timeMs != 150000000 {
		t.Errorf("Expected 150000000 (150ms in ns), got %d", timeMs)
	}
}

func TestPostResponse_SetEnvVar(t *testing.T) {
	eng := NewEngine(nil)
	resp := &client.Response{
		StatusCode: 200,
		Headers:    http.Header{},
		Body:       []byte(`{"token":"abc123"}`),
		Duration:   100 * time.Millisecond,
		Size:       20,
	}

	result, err := RunPostResponse(eng, `gurl.setVar("authToken", "Bearer abc123")`, resp)
	if err != nil {
		t.Fatalf("RunPostResponse failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if result.Variables == nil {
		t.Fatal("Expected variables map, got nil")
	}
	if val, ok := result.Variables["authToken"]; !ok || val != "Bearer abc123" {
		t.Errorf("Expected authToken='Bearer abc123', got '%s'", val)
	}
}

func TestPostResponse_TestAssertion(t *testing.T) {
	eng := NewEngine(nil)
	resp := &client.Response{
		StatusCode: 200,
		Headers:    http.Header{},
		Body:       []byte(`{"message":"success"}`),
		Duration:   100 * time.Millisecond,
		Size:       24,
	}

	result, err := RunPostResponse(eng, `
		gurl.test("status ok", function() {
			gurl.expect(gurl.response.status).to.equal(200);
		});
	`, resp)
	if err != nil {
		t.Fatalf("RunPostResponse failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if len(result.Assertions) != 1 {
		t.Fatalf("Expected 1 assertion, got %d", len(result.Assertions))
	}
	if !result.Assertions[0].Passed {
		t.Errorf("Expected assertion to pass, got failed: %s", result.Assertions[0].Error)
	}
	if result.Assertions[0].Name != "status ok" {
		t.Errorf("Expected name 'status ok', got '%s'", result.Assertions[0].Name)
	}
}

func TestPostResponse_ExtractJSONPath(t *testing.T) {
	eng := NewEngine(nil)
	resp := &client.Response{
		StatusCode: 200,
		Headers:    http.Header{},
		Body:       []byte(`{"data":{"users":[{"id":"user123","name":"Alice"}]}}`),
		Duration:   100 * time.Millisecond,
		Size:       60,
	}

	result, err := RunPostResponse(eng, `
		var jsonData = gurl.response.json();
		jsonData.data.users[0].id;
	`, resp)
	if err != nil {
		t.Fatalf("RunPostResponse failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	evalResult, evalErr := eng.Execute(`
		var jsonData = gurl.response.json();
		jsonData.data.users[0].id;
	`)
	if evalErr != nil {
		t.Fatalf("Execute failed: %v", evalErr)
	}
	userID, ok := evalResult.Value.(string)
	if !ok {
		t.Fatalf("Expected string, got %T", evalResult.Value)
	}
	if userID != "user123" {
		t.Errorf("Expected 'user123', got '%s'", userID)
	}
}

func TestPostResponse_ErrorDoesNotLoseResponse(t *testing.T) {
	eng := NewEngine(nil)
	resp := &client.Response{
		StatusCode: 200,
		Headers:    http.Header{},
		Body:       []byte(`{"message":"success"}`),
		Duration:   100 * time.Millisecond,
		Size:       24,
	}

	_, err := RunPostResponse(eng, `
		throw new Error("intentional error");
	`, resp)

	if err == nil {
		t.Fatal("Expected error from script, got nil")
	}
	if !strings.Contains(err.Error(), "intentional error") {
		t.Errorf("Expected error to contain 'intentional error', got: %v", err)
	}

	evalResult, evalErr := eng.Execute(`gurl.response.status`)
	if evalErr != nil {
		t.Fatalf("Response should still be accessible after error, got: %v", evalErr)
	}
	if evalResult.Value.(int64) != 200 {
		t.Errorf("Expected status 200 after error, got %v", evalResult.Value)
	}
}

func TestPostResponse_SetNextRequest(t *testing.T) {
	eng := NewEngine(nil)
	resp := &client.Response{
		StatusCode: 200,
		Headers:    http.Header{},
		Body:       []byte(`{"message":"success"}`),
		Duration:   100 * time.Millisecond,
		Size:       24,
	}

	result, err := RunPostResponse(eng, `gurl.setNextRequest("next-request")`, resp)
	if err != nil {
		t.Fatalf("RunPostResponse failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if eng.nextRequest != "next-request" {
		t.Errorf("Expected nextRequest='next-request', got '%s'", eng.nextRequest)
	}
}
