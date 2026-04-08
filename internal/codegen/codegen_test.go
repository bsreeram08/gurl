package codegen

import (
	"strings"
	"testing"

	"github.com/sreeram/gurl/pkg/types"
)

func TestCodeGen_Go(t *testing.T) {
	req := &types.SavedRequest{
		URL:     "https://api.example.com/test",
		Method:  "GET",
		Headers: []types.Header{},
		Body:    "",
	}

	code, err := Generate("go", req, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !strings.Contains(code, "https://api.example.com/test") {
		t.Errorf("generated code should contain URL")
	}
	if !strings.Contains(code, "http.NewRequest") {
		t.Errorf("generated code should use http.NewRequest")
	}
	if !strings.Contains(code, "client.Do(req)") {
		t.Errorf("generated code should use client.Do(req)")
	}
	if !strings.Contains(code, "io.ReadAll") {
		t.Errorf("generated code should use io.ReadAll")
	}
	if !strings.Contains(code, "defer resp.Body.Close()") {
		t.Errorf("generated code should defer resp.Body.Close()")
	}
}

func TestCodeGen_Python(t *testing.T) {
	req := &types.SavedRequest{
		URL:     "https://api.example.com/test",
		Method:  "GET",
		Headers: []types.Header{},
		Body:    "",
	}

	code, err := Generate("python", req, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !strings.Contains(code, "https://api.example.com/test") {
		t.Errorf("generated code should contain URL")
	}
	if !strings.Contains(code, "requests.get") {
		t.Errorf("generated code should use requests.get")
	}
	if !strings.Contains(code, "print(response.status_code)") {
		t.Errorf("generated code should print status code")
	}
}

func TestCodeGen_JavaScript(t *testing.T) {
	req := &types.SavedRequest{
		URL:     "https://api.example.com/test",
		Method:  "GET",
		Headers: []types.Header{},
		Body:    "",
	}

	code, err := Generate("javascript", req, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !strings.Contains(code, "https://api.example.com/test") {
		t.Errorf("generated code should contain URL")
	}
	if !strings.Contains(code, "async function") {
		t.Errorf("generated code should be async function")
	}
	if !strings.Contains(code, "fetch(url, options)") {
		t.Errorf("generated code should use fetch")
	}
	if !strings.Contains(code, "await response.text()") {
		t.Errorf("generated code should await response text")
	}
}

func TestCodeGen_Curl(t *testing.T) {
	req := &types.SavedRequest{
		URL:     "https://api.example.com/test",
		Method:  "GET",
		Headers: []types.Header{},
		Body:    "",
	}

	code, err := Generate("curl", req, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !strings.Contains(code, "curl") {
		t.Errorf("generated code should start with curl")
	}
	if !strings.Contains(code, "https://api.example.com/test") {
		t.Errorf("generated code should contain URL")
	}
}

func TestCodeGen_Go_WithAuth(t *testing.T) {
	req := &types.SavedRequest{
		URL:    "https://api.example.com/test",
		Method: "GET",
		Headers: []types.Header{
			{Key: "Authorization", Value: "Bearer my-secret-token"},
		},
		Body: "",
	}

	code, err := Generate("go", req, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !strings.Contains(code, "<your-token-here>") {
		t.Errorf("generated code should use placeholder token, not actual token")
	}
	if !strings.Contains(code, "Bearer") {
		t.Errorf("generated code should indicate bearer auth")
	}
}

func TestCodeGen_Go_WithBody(t *testing.T) {
	req := &types.SavedRequest{
		URL:    "https://api.example.com/test",
		Method: "POST",
		Headers: []types.Header{
			{Key: "Content-Type", Value: "application/json"},
		},
		Body: `{"name":"test"}`,
	}

	code, err := Generate("go", req, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !strings.Contains(code, "bytes.NewBufferString") {
		t.Errorf("generated code should use bytes.NewBufferString for body")
	}
	if !strings.Contains(code, "POST") {
		t.Errorf("generated code should use POST method")
	}
}

func TestCodeGen_Python_WithHeaders(t *testing.T) {
	req := &types.SavedRequest{
		URL:    "https://api.example.com/test",
		Method: "GET",
		Headers: []types.Header{
			{Key: "X-Custom-Header", Value: "custom-value"},
			{Key: "Authorization", Value: "Bearer my-secret-token"},
		},
		Body: "",
	}

	code, err := Generate("python", req, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !strings.Contains(code, "headers = {") {
		t.Errorf("generated code should have headers dict")
	}
	if !strings.Contains(code, "X-Custom-Header") {
		t.Errorf("generated code should include X-Custom-Header")
	}
	if !strings.Contains(code, "<your-token-here>") {
		t.Errorf("generated code should use placeholder token")
	}
}

func TestCodeGen_JavaScript_WithFormData(t *testing.T) {
	req := &types.SavedRequest{
		URL:    "https://api.example.com/test",
		Method: "POST",
		Headers: []types.Header{
			{Key: "Content-Type", Value: "multipart/form-data"},
		},
		Body: `{"name":"test"}`,
	}

	code, err := Generate("javascript", req, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !strings.Contains(code, "body") {
		t.Errorf("generated code should include body")
	}
	if !strings.Contains(code, "POST") {
		t.Errorf("generated code should use POST method")
	}
}

func TestCodeGen_UnknownLanguage(t *testing.T) {
	req := &types.SavedRequest{
		URL:     "https://api.example.com/test",
		Method:  "GET",
		Headers: []types.Header{},
		Body:    "",
	}

	_, err := Generate("ruby", req, nil)
	if err == nil {
		t.Fatalf("expected error for unknown language")
	}

	expectedMsg := "unsupported language 'ruby', available: go, python, javascript, curl"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestCodeGen_AllLanguages(t *testing.T) {
	languages := ListLanguages()

	expected := []string{"go", "python", "javascript", "curl"}
	if len(languages) != len(expected) {
		t.Fatalf("expected %d languages, got %d", len(expected), len(languages))
	}

	for i, lang := range languages {
		if lang != expected[i] {
			t.Errorf("expected language %s at index %d, got %s", expected[i], i, lang)
		}
	}
}

func TestCodeGen_Python_WithBody(t *testing.T) {
	req := &types.SavedRequest{
		URL:    "https://api.example.com/test",
		Method: "POST",
		Headers: []types.Header{
			{Key: "Content-Type", Value: "application/json"},
		},
		Body: `{"name":"test"}`,
	}

	code, err := Generate("python", req, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !strings.Contains(code, "requests.post") {
		t.Errorf("generated code should use requests.post for POST with body")
	}
	if !strings.Contains(code, "json=data") {
		t.Errorf("generated code should use json=data parameter")
	}
}

func TestCodeGen_Curl_WithHeaders(t *testing.T) {
	req := &types.SavedRequest{
		URL:    "https://api.example.com/test",
		Method: "GET",
		Headers: []types.Header{
			{Key: "X-Custom-Header", Value: "custom-value"},
		},
		Body: "",
	}

	code, err := Generate("curl", req, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !strings.Contains(code, "-H") {
		t.Errorf("generated code should include -H flag for headers")
	}
	if !strings.Contains(code, "X-Custom-Header") {
		t.Errorf("generated code should include header key")
	}
}
