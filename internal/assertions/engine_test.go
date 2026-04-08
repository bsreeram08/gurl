package assertions

import (
	"net/http"
	"testing"
	"time"

	"github.com/sreeram/gurl/internal/client"
	"github.com/sreeram/gurl/pkg/types"
)

func newTestResponse(statusCode int, body string, headers http.Header, duration time.Duration, size int64) *client.Response {
	return &client.Response{
		StatusCode: statusCode,
		Body:       []byte(body),
		Headers:    headers,
		Duration:   duration,
		Size:       size,
	}
}

func TestAssert_StatusCode(t *testing.T) {
	resp := newTestResponse(200, "OK", nil, 0, 2)
	evaluator := NewEvaluator()

	assertions := []Assertion{
		{Field: "status", Op: "=", Value: "200"},
	}
	results := evaluator.Evaluate(resp, assertions)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Passed {
		t.Errorf("expected status=200 to pass, got: %s", results[0].Message)
	}
}

func TestAssert_StatusRange(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		op       string
		value    string
		expected bool
	}{
		{name: "2xx success", status: 201, op: ">=", value: "200", expected: true},
		{name: "2xx less than 300", status: 299, op: "<", value: "300", expected: true},
		{name: "not 404", status: 200, op: "!=", value: "404", expected: true},
	}

	evaluator := NewEvaluator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := newTestResponse(tt.status, "OK", nil, 0, 2)
			assertions := []Assertion{{Field: "status", Op: tt.op, Value: tt.value}}
			results := evaluator.Evaluate(resp, assertions)

			if results[0].Passed != tt.expected {
				t.Errorf("expected pass=%v, got %v - %s", tt.expected, results[0].Passed, results[0].Message)
			}
		})
	}
}

func TestAssert_HeaderExists(t *testing.T) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	headers.Set("X-Request-Id", "abc123")
	resp := newTestResponse(200, "{}", headers, 0, 2)
	evaluator := NewEvaluator()

	tests := []struct {
		name     string
		field    string
		expected bool
	}{
		{"Content-Type exists", "headers.Content-Type", true},
		{"X-Request-Id exists", "headers.X-Request-Id", true},
		{"Cache-Control exists", "headers.Cache-Control", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertions := []Assertion{{Field: tt.field, Op: "exists", Value: ""}}
			results := evaluator.Evaluate(resp, assertions)

			if results[0].Passed != tt.expected {
				t.Errorf("expected pass=%v, got %v", tt.expected, results[0].Passed)
			}
		})
	}
}

func TestAssert_HeaderValue(t *testing.T) {
	headers := map[string][]string{
		"Content-Type": {"application/json"},
	}
	resp := newTestResponse(200, "{}", headers, 0, 2)
	evaluator := NewEvaluator()

	assertions := []Assertion{
		{Field: "headers.Content-Type", Op: "=", Value: "application/json"},
	}
	results := evaluator.Evaluate(resp, assertions)

	if !results[0].Passed {
		t.Errorf("expected Content-Type=application/json to pass, got: %s", results[0].Message)
	}
}

func TestAssert_HeaderContains(t *testing.T) {
	headers := map[string][]string{
		"Cache-Control": {"no-cache, no-store, must-revalidate"},
	}
	resp := newTestResponse(200, "{}", headers, 0, 2)
	evaluator := NewEvaluator()

	assertions := []Assertion{
		{Field: "headers.Cache-Control", Op: "contains", Value: "no-store"},
	}
	results := evaluator.Evaluate(resp, assertions)

	if !results[0].Passed {
		t.Errorf("expected header contains 'no-store' to pass, got: %s", results[0].Message)
	}
}

func TestAssert_BodyContains(t *testing.T) {
	resp := newTestResponse(200, `{"message":"success","code":200}`, nil, 0, 40)
	evaluator := NewEvaluator()

	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"contains success", "success", true},
		{"contains 200", "200", true},
		{"contains error", "error", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertions := []Assertion{{Field: "body", Op: "contains", Value: tt.value}}
			results := evaluator.Evaluate(resp, assertions)

			if results[0].Passed != tt.expected {
				t.Errorf("expected pass=%v, got %v", tt.expected, results[0].Passed)
			}
		})
	}
}

func TestAssert_BodyJSONPath(t *testing.T) {
	body := `{"data":{"id":123,"name":"test","active":true},"meta":{"page":1}}`
	resp := newTestResponse(200, body, nil, 0, int64(len(body)))
	evaluator := NewEvaluator()

	tests := []struct {
		name     string
		path     string
		op       string
		value    string
		expected bool
	}{
		{"data.id equals 123", "body.$.data.id", "=", "123", true},
		{"data.name equals test", "body.$.data.name", "=", "test", true},
		{"data.active equals true", "body.$.data.active", "=", "true", true},
		{"data.id not 999", "body.$.data.id", "!=", "999", true},
		{"meta.page greater 0", "body.$.meta.page", ">", "0", true},
		{"data.id less 200", "body.$.data.id", "<", "200", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertions := []Assertion{{Field: tt.path, Op: tt.op, Value: tt.value}}
			results := evaluator.Evaluate(resp, assertions)

			if results[0].Passed != tt.expected {
				t.Errorf("expected pass=%v, got %v - %s (actual: %s)", tt.expected, results[0].Passed, results[0].Message, results[0].Actual)
			}
		})
	}
}

func TestAssert_ResponseTime(t *testing.T) {
	resp := newTestResponse(200, "OK", nil, 450*time.Millisecond, 2)
	evaluator := NewEvaluator()

	tests := []struct {
		name     string
		op       string
		value    string
		expected bool
	}{
		{"time less than 500ms", "<", "500", true},
		{"time less than 400ms", "<", "400", false},
		{"time greater than 400ms", ">", "400", true},
		{"time greater than 500ms", ">", "500", false},
		{"time <= 450ms", "<=", "450", true},
		{"time >= 450ms", ">=", "450", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertions := []Assertion{{Field: "time", Op: tt.op, Value: tt.value}}
			results := evaluator.Evaluate(resp, assertions)

			if results[0].Passed != tt.expected {
				t.Errorf("expected pass=%v, got %v - %s", tt.expected, results[0].Passed, results[0].Message)
			}
		})
	}
}

func TestAssert_BodySize(t *testing.T) {
	body := `{"data":"test"}`
	size := int64(len(body))
	resp := newTestResponse(200, body, nil, 0, size)
	evaluator := NewEvaluator()

	tests := []struct {
		name     string
		op       string
		value    string
		expected bool
	}{
		{"size less than 20", "<", "20", true},
		{"size equals 15", "=", "15", true},
		{"size greater than 10", ">", "10", true},
		{"size not 100", "!=", "100", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertions := []Assertion{{Field: "size", Op: tt.op, Value: tt.value}}
			results := evaluator.Evaluate(resp, assertions)

			if results[0].Passed != tt.expected {
				t.Errorf("expected pass=%v, got %v - %s", tt.expected, results[0].Passed, results[0].Message)
			}
		})
	}
}

func TestAssert_MultipleAssertions(t *testing.T) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	resp := newTestResponse(200, `{"status":"ok"}`, headers, 100*time.Millisecond, 17)
	evaluator := NewEvaluator()

	assertions := []Assertion{
		{Field: "status", Op: "=", Value: "200"},
		{Field: "headers.Content-Type", Op: "contains", Value: "json"},
		{Field: "time", Op: "<", Value: "200"},
		{Field: "body", Op: "contains", Value: "ok"},
	}
	results := evaluator.Evaluate(resp, assertions)

	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}

	for i, r := range results {
		if !r.Passed {
			t.Errorf("assertion[%d] failed: %s", i, r.Message)
		}
	}
}

func TestAssert_FromSavedRequest(t *testing.T) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	headers.Set("X-Custom", "header")

	body := `{"success":true,"data":{"id":42}}`
	resp := newTestResponse(201, body, headers, 300*time.Millisecond, int64(len(body)))
	evaluator := NewEvaluator()

	savedReq := types.SavedRequest{
		Assertions: []types.Assertion{
			{Field: "status", Op: "=", Value: "201"},
			{Field: "headers.Content-Type", Op: "contains", Value: "json"},
			{Field: "body.$.data.id", Op: "=", Value: "42"},
			{Field: "time", Op: "<", Value: "500"},
		},
	}

	// Convert types.Assertion to assertions.Assertion for evaluation
	assertions := make([]Assertion, len(savedReq.Assertions))
	for i, a := range savedReq.Assertions {
		assertions[i] = Assertion{Field: a.Field, Op: a.Op, Value: a.Value}
	}

	results := evaluator.Evaluate(resp, assertions)

	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}

	passed := 0
	for _, r := range results {
		if r.Passed {
			passed++
		}
	}

	if passed != 4 {
		t.Errorf("expected all 4 assertions to pass, got %d/%d", passed, len(results))
		for _, r := range results {
			if !r.Passed {
				t.Logf("FAILED: %s", r.Message)
			}
		}
	}
}

func TestCLIParser(t *testing.T) {
	parser := NewCLIParser()

	tests := []struct {
		name        string
		input       string
		expectedOp  string
		expectedVal string
		expectError bool
	}{
		{"status equals", "status=200", "=", "200", false},
		{"status not equals", "status!=404", "!=", "404", false},
		{"status less than", "status<300", "<", "300", false},
		{"status greater than", "status>100", ">", "100", false},
		{"body contains", "body contains success", "contains", "success", false},
		{"body not_contains", "body not_contains error", "not_contains", "error", false},
		{"body matches regex", "body matches \\d+", "matches", "\\d+", false},
		{"header exists", "headers.X-Custom exists", "exists", "", false},
		{"complex jsonpath", "body.$.data.id=123", "=", "123", false},
		{"empty string", "", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := parser.Parse(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for input %q", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if a.Op != tt.expectedOp {
				t.Errorf("expected op %q, got %q", tt.expectedOp, a.Op)
			}
			if a.Value != tt.expectedVal {
				t.Errorf("expected value %q, got %q", tt.expectedVal, a.Value)
			}
		})
	}
}

func TestTOMLParser(t *testing.T) {
	parser := NewTOMLParser()

	tomlContent := `
[[assertions]]
field = "status"
op = "="
value = "200"

[[assertions]]
field = "body"
op = "contains"
value = "success"
`

	assertions, err := parser.ParseTOML(tomlContent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(assertions) != 2 {
		t.Fatalf("expected 2 assertions, got %d", len(assertions))
	}

	if assertions[0].Field != "status" || assertions[0].Op != "=" || assertions[0].Value != "200" {
		t.Errorf("first assertion mismatch: %+v", assertions[0])
	}

	if assertions[1].Field != "body" || assertions[1].Op != "contains" || assertions[1].Value != "success" {
		t.Errorf("second assertion mismatch: %+v", assertions[1])
	}
}

func TestSummary(t *testing.T) {
	results := []Result{
		{Passed: true},
		{Passed: true},
		{Passed: false},
		{Passed: false},
		{Passed: true},
	}

	summary := Summarize(results)

	if summary.Total != 5 {
		t.Errorf("expected Total=5, got %d", summary.Total)
	}
	if summary.Passed != 3 {
		t.Errorf("expected Passed=3, got %d", summary.Passed)
	}
	if summary.Failed != 2 {
		t.Errorf("expected Failed=2, got %d", summary.Failed)
	}
}

func TestAssert_NotContains(t *testing.T) {
	resp := newTestResponse(200, `{"secret":"value123"}`, nil, 0, 20)
	evaluator := NewEvaluator()

	assertions := []Assertion{
		{Field: "body", Op: "not_contains", Value: "password"},
	}
	results := evaluator.Evaluate(resp, assertions)

	if !results[0].Passed {
		t.Errorf("expected body not_contains 'password' to pass, got: %s", results[0].Message)
	}
}

func TestAssert_Matches(t *testing.T) {
	resp := newTestResponse(200, "abc123def", nil, 0, 9)
	evaluator := NewEvaluator()

	tests := []struct {
		name     string
		pattern  string
		expected bool
	}{
		{"matches digits", `^\d+$`, false},
		{"matches alphanum", `^[a-z0-9]+$`, true},
		{"matches with digits", `\d+`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertions := []Assertion{{Field: "body", Op: "matches", Value: tt.pattern}}
			results := evaluator.Evaluate(resp, assertions)

			if results[0].Passed != tt.expected {
				t.Errorf("expected pass=%v, got %v", tt.expected, results[0].Passed)
			}
		})
	}
}
