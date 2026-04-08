package builtins

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/sreeram/gurl/internal/client"
	"github.com/sreeram/gurl/internal/plugins"
)

func newTestResponseContext() *plugins.ResponseContext {
	return &plugins.ResponseContext{
		Request: &client.Request{
			Method: "GET",
			URL:    "https://example.com/api/test",
		},
		Response: &client.Response{
			StatusCode: 200,
			Headers:    http.Header{"Content-Type": []string{"application/json"}},
			Body:       []byte(`{"message": "hello", "count": 42}`),
			Duration:   145 * time.Millisecond,
			Size:       1024,
		},
	}
}

func TestOutputPlugin_Slack(t *testing.T) {
	ctx := newTestResponseContext()
	plugin := &SlackOutput{}

	result := plugin.Render(ctx)

	// Check emoji is present (✅ for 2xx)
	if !strings.Contains(result, "✅") {
		t.Errorf("expected ✅ emoji in slack output for 2xx status, got: %s", result)
	}

	// Check method and URL are present
	if !strings.Contains(result, "GET") {
		t.Errorf("expected GET method in slack output, got: %s", result)
	}
	if !strings.Contains(result, "https://example.com/api/test") {
		t.Errorf("expected URL in slack output, got: %s", result)
	}

	// Check status code
	if !strings.Contains(result, "200") {
		t.Errorf("expected 200 status code in slack output, got: %s", result)
	}

	// Check fenced code block exists
	if !strings.Contains(result, "```json") {
		t.Errorf("expected fenced code block in slack output, got: %s", result)
	}
}

func TestOutputPlugin_Slack_4xx(t *testing.T) {
	ctx := &plugins.ResponseContext{
		Request: &client.Request{
			Method: "GET",
			URL:    "https://example.com/api/test",
		},
		Response: &client.Response{
			StatusCode: 404,
			Headers:    http.Header{},
			Body:       []byte(`{"error": "not found"}`),
			Duration:   50 * time.Millisecond,
			Size:       256,
		},
	}
	plugin := &SlackOutput{}

	result := plugin.Render(ctx)

	// Check emoji is ❌ for 4xx
	if !strings.Contains(result, "❌") {
		t.Errorf("expected ❌ emoji in slack output for 4xx status, got: %s", result)
	}
}

func TestOutputPlugin_Markdown(t *testing.T) {
	ctx := newTestResponseContext()
	plugin := &MarkdownOutput{}

	result := plugin.Render(ctx)

	// Check H1 heading with method and URL
	if !strings.Contains(result, "# GET https://example.com/api/test (200)") {
		t.Errorf("expected H1 heading in markdown output, got: %s", result)
	}

	// Check headers table
	if !strings.Contains(result, "| Header | Value |") {
		t.Errorf("expected headers table in markdown output, got: %s", result)
	}

	// Check Content-Type is in the headers
	if !strings.Contains(result, "Content-Type") {
		t.Errorf("expected Content-Type header in markdown output, got: %s", result)
	}

	// Check fenced code block
	if !strings.Contains(result, "```json") {
		t.Errorf("expected fenced code block in markdown output, got: %s", result)
	}

	// Check duration footer
	if !strings.Contains(result, "Duration: 145ms") {
		t.Errorf("expected duration footer in markdown output, got: %s", result)
	}
}

func TestOutputPlugin_CSV(t *testing.T) {
	ctx := newTestResponseContext()
	plugin := &CSVOutput{}

	result := plugin.Render(ctx)

	// Should contain status code
	if !strings.Contains(result, "200") {
		t.Errorf("expected 200 status code in csv output, got: %s", result)
	}

	// Should contain URL
	if !strings.Contains(result, "https://example.com/api/test") {
		t.Errorf("expected URL in csv output, got: %s", result)
	}

	// Should contain duration
	if !strings.Contains(result, "145") {
		t.Errorf("expected duration in csv output, got: %s", result)
	}

	// Should contain content type
	if !strings.Contains(result, "application/json") {
		t.Errorf("expected content type in csv output, got: %s", result)
	}
}

func TestOutputPlugin_Minimal(t *testing.T) {
	ctx := newTestResponseContext()
	plugin := &MinimalOutput{}

	result := plugin.Render(ctx)

	// Should contain status code and text
	if !strings.Contains(result, "200 OK") {
		t.Errorf("expected '200 OK' in minimal output, got: %s", result)
	}

	// Should contain duration
	if !strings.Contains(result, "145ms") {
		t.Errorf("expected '145ms' in minimal output, got: %s", result)
	}

	// Should contain size
	if !strings.Contains(result, "1024B") {
		t.Errorf("expected '1024B' in minimal output, got: %s", result)
	}

	// Should be single line (no newlines)
	lines := strings.Split(result, "\n")
	if len(lines) > 1 {
		t.Errorf("expected single line minimal output, got: %s", result)
	}
}

func TestOutputPlugin_Registration(t *testing.T) {
	registry := plugins.NewRegistry()
	RegisterBuiltins(registry)

	// Test Slack
	slack, found := registry.GetOutputByFormat("slack")
	if !found {
		t.Error("expected to find slack output plugin")
	}
	if slack.Name() != "slack" {
		t.Errorf("expected slack plugin name 'slack', got: %s", slack.Name())
	}

	// Test Markdown
	markdown, found := registry.GetOutputByFormat("markdown")
	if !found {
		t.Error("expected to find markdown output plugin")
	}
	if markdown.Name() != "markdown" {
		t.Errorf("expected markdown plugin name 'markdown', got: %s", markdown.Name())
	}

	// Test CSV
	csv, found := registry.GetOutputByFormat("csv")
	if !found {
		t.Error("expected to find csv output plugin")
	}
	if csv.Name() != "csv" {
		t.Errorf("expected csv plugin name 'csv', got: %s", csv.Name())
	}

	// Test Minimal
	minimal, found := registry.GetOutputByFormat("minimal")
	if !found {
		t.Error("expected to find minimal output plugin")
	}
	if minimal.Name() != "minimal" {
		t.Errorf("expected minimal plugin name 'minimal', got: %s", minimal.Name())
	}

	// Test non-existent format
	_, found = registry.GetOutputByFormat("nonexistent")
	if found {
		t.Error("expected not to find nonexistent output plugin")
	}
}

func TestOutputPlugin_EmptyBody(t *testing.T) {
	testCases := []struct {
		name   string
		plugin plugins.OutputPlugin
	}{
		{"SlackOutput", &SlackOutput{}},
		{"MarkdownOutput", &MarkdownOutput{}},
		{"CSVOutput", &CSVOutput{}},
		{"MinimalOutput", &MinimalOutput{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test with nil response
			ctx := &plugins.ResponseContext{
				Request: &client.Request{
					Method: "GET",
					URL:    "https://example.com",
				},
				Response: nil,
			}
			result := tc.plugin.Render(ctx)
			// Slack and Markdown should return non-empty for nil response
			// CSV and Minimal may return empty
			if result == "" && tc.name != "CSVOutput" && tc.name != "MinimalOutput" {
				t.Errorf("%s: expected non-empty result for nil response", tc.name)
			}
		})
	}
}

func TestOutputPlugin_EmptyBodyContent(t *testing.T) {
	testCases := []struct {
		name   string
		plugin plugins.OutputPlugin
	}{
		{"SlackOutput", &SlackOutput{}},
		{"MarkdownOutput", &MarkdownOutput{}},
		{"CSVOutput", &CSVOutput{}},
		{"MinimalOutput", &MinimalOutput{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test with empty body
			ctx := &plugins.ResponseContext{
				Request: &client.Request{
					Method: "GET",
					URL:    "https://example.com",
				},
				Response: &client.Response{
					StatusCode: 204,
					Headers:    http.Header{},
					Body:       []byte{},
					Duration:   10 * time.Millisecond,
					Size:       0,
				},
			}
			// Should not panic
			result := tc.plugin.Render(ctx)
			if result == "" && tc.name != "CSVOutput" && tc.name != "MinimalOutput" {
				// CSV and Minimal can return empty for empty body
				t.Errorf("%s: expected non-empty result for empty body", tc.name)
			}
		})
	}
}

func TestOutputPlugin_Markdown_HandlesNonJSONBody(t *testing.T) {
	ctx := &plugins.ResponseContext{
		Request: &client.Request{
			Method: "GET",
			URL:    "https://example.com/api/html",
		},
		Response: &client.Response{
			StatusCode: 200,
			Headers:    http.Header{"Content-Type": []string{"text/html"}},
			Body:       []byte("<html><body>Hello</body></html>"),
			Duration:   100 * time.Millisecond,
			Size:       512,
		},
	}
	plugin := &MarkdownOutput{}

	result := plugin.Render(ctx)

	// Should still render even if body is not JSON
	if !strings.Contains(result, "# GET") {
		t.Errorf("expected heading in markdown output, got: %s", result)
	}
	if !strings.Contains(result, "Hello") {
		t.Errorf("expected body content in markdown output, got: %s", result)
	}
}
