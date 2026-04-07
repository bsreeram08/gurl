package formatter

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// TestFormatJSON_PrettyPrint tests that minified JSON gets indented
func TestFormatJSON_PrettyPrint(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		indent string
		want   string
	}{
		{
			name:   "minified JSON gets indented",
			input:  `{"name":"test","count":42}`,
			indent: "  ",
			want:   "{\n  \"name\": \"test\",\n  \"count\": 42\n}",
		},
		{
			name:   "nested JSON indented",
			input:  `{"user":{"name":"alice","age":30},"active":true}`,
			indent: "  ",
			want:   "{\n  \"user\": {\n    \"name\": \"alice\",\n    \"age\": 30\n  },\n  \"active\": true\n}",
		},
		{
			name:   "custom indent tab",
			input:  `{"key":"value"}`,
			indent: "\t",
			want:   "{\n\t\"key\": \"value\"\n}",
		},
		{
			name:   "empty object",
			input:  `{}`,
			indent: "  ",
			want:   "{}",
		},
		{
			name:   "empty array",
			input:  `[]`,
			indent: "  ",
			want:   "[]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := FormatOptions{Indent: tt.indent, Color: false, MaxWidth: 0}
			got := FormatJSON([]byte(tt.input), opts)
			if got != tt.want {
				t.Errorf("FormatJSON() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestFormatJSON_SyntaxHighlight tests ANSI color codes in JSON output
func TestFormatJSON_SyntaxHighlight(t *testing.T) {
	input := `{"name":"test","count":42,"active":true,"data":null}`
	opts := FormatOptions{Indent: "  ", Color: true, MaxWidth: 0}
	got := FormatJSON([]byte(input), opts)

	// Keys should be cyan
	if !strings.Contains(got, Cyan) {
		t.Errorf("FormatJSON() missing cyan for keys, got %q", got)
	}
	// Strings should be green
	if !strings.Contains(got, Green) {
		t.Errorf("FormatJSON() missing green for strings, got %q", got)
	}
	// Numbers should be yellow
	if !strings.Contains(got, Yellow) {
		t.Errorf("FormatJSON() missing yellow for numbers, got %q", got)
	}
	// Booleans should be magenta
	if !strings.Contains(got, Magenta) {
		t.Errorf("FormatJSON() missing magenta for booleans, got %q", got)
	}
	// Null should be red
	if !strings.Contains(got, Red) {
		t.Errorf("FormatJSON() missing red for null, got %q", got)
	}
	// Reset should appear after colored spans
	if !strings.Contains(got, Reset) {
		t.Errorf("FormatJSON() missing reset codes, got %q", got)
	}
}

// TestFormatXML_PrettyPrint tests that compact XML gets indented
func TestFormatXML_PrettyPrint(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		indent string
	}{
		{
			name:   "compact XML indented",
			input:  `<root><child>text</child></root>`,
			indent: "  ",
		},
		{
			name:   "XML with attributes",
			input:  `<root attr="val"><child id="1">text</child></root>`,
			indent: "  ",
		},
		{
			name:   "nested XML",
			input:  `<root><parent><child>value</child></parent></root>`,
			indent: "  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := FormatOptions{Indent: tt.indent, Color: false, MaxWidth: 0}
			got := FormatXML([]byte(tt.input), opts)
			// Output should have newlines and indentation
			if !strings.Contains(got, "\n") {
				t.Errorf("FormatXML() no newlines in output %q", got)
			}
			// Check that indent appears in output
			if !strings.Contains(got, tt.indent) {
				t.Errorf("FormatXML() indent %q not found in output %q", tt.indent, got)
			}
		})
	}
}

// TestFormatXML_SyntaxHighlight tests ANSI colors for XML
func TestFormatXML_SyntaxHighlight(t *testing.T) {
	input := `<root attr="val">text</root>`
	opts := FormatOptions{Indent: "  ", Color: true, MaxWidth: 0}
	got := FormatXML([]byte(input), opts)

	// Tags should be cyan
	if !strings.Contains(got, Cyan) {
		t.Errorf("FormatXML() missing cyan for tags, got %q", got)
	}
	// Attributes should be yellow
	if !strings.Contains(got, Yellow) {
		t.Errorf("FormatXML() missing yellow for attributes, got %q", got)
	}
	// Reset should appear
	if !strings.Contains(got, Reset) {
		t.Errorf("FormatXML() missing reset codes, got %q", got)
	}
}

// TestFormatHTML_PrettyPrint tests that minified HTML gets indented
func TestFormatHTML_PrettyPrint(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		indent string
	}{
		{
			name:   "inline HTML indented",
			input:  `<html><body><p>Hello</p></body></html>`,
			indent: "  ",
		},
		{
			name:   "HTML with attributes",
			input:  `<div class="container"><span style="color:red">text</span></div>`,
			indent: "  ",
		},
		{
			name:   "nested HTML tags",
			input:  `<html><head><title>Page</title></head><body><div><p>Content</p></div></body></html>`,
			indent: "  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := FormatOptions{Indent: tt.indent, Color: false, MaxWidth: 0}
			got := FormatHTML([]byte(tt.input), opts)
			// Output should have newlines
			if !strings.Contains(got, "\n") {
				t.Errorf("FormatHTML() no newlines in output %q", got)
			}
			// Check indent appears
			if !strings.Contains(got, tt.indent) {
				t.Errorf("FormatHTML() indent %q not found in output %q", tt.indent, got)
			}
		})
	}
}

// TestAutoDetect_ContentType tests automatic formatter selection
func TestAutoDetect_ContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		input       string
		expectRaw   bool // if true, expect raw passthrough
	}{
		{
			name:        "application/json uses JSON formatter",
			contentType: "application/json",
			input:       `{"key":"value"}`,
			expectRaw:   false,
		},
		{
			name:        "text/json uses JSON formatter",
			contentType: "text/json",
			input:       `{"key":"value"}`,
			expectRaw:   false,
		},
		{
			name:        "text/xml uses XML formatter",
			contentType: "text/xml",
			input:       `<root><child>text</child></root>`,
			expectRaw:   false,
		},
		{
			name:        "application/xml uses XML formatter",
			contentType: "application/xml",
			input:       `<root><child>text</child></root>`,
			expectRaw:   false,
		},
		{
			name:        "text/html uses HTML formatter",
			contentType: "text/html",
			input:       `<html><body>text</body></html>`,
			expectRaw:   false,
		},
		{
			name:        "application/xhtml+xml uses HTML formatter",
			contentType: "application/xhtml+xml",
			input:       `<html xmlns="http://www.w3.org/1999/xhtml"><body>text</body></html>`,
			expectRaw:   false,
		},
		{
			name:        "text/plain returns raw",
			contentType: "text/plain",
			input:       "plain text",
			expectRaw:   true,
		},
		{
			name:        "unknown type returns raw",
			contentType: "application/octet-stream",
			input:       "some binary data",
			expectRaw:   true,
		},
		{
			name:        "empty content type returns raw",
			contentType: "",
			input:       "no content type",
			expectRaw:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := FormatOptions{Indent: "  ", Color: false, MaxWidth: 0}
			got := Format([]byte(tt.input), tt.contentType, opts)
			if tt.expectRaw {
				// Raw passthrough should return input as-is
				if got != tt.input {
					t.Errorf("Format() for %q expected raw passthrough %q, got %q", tt.contentType, tt.input, got)
				}
			} else {
				// Formatted output should be different from input
				if got == tt.input {
					t.Errorf("Format() for %q expected formatted output different from input", tt.contentType)
				}
			}
		})
	}
}

// TestFormatJSON_InvalidInput tests that invalid JSON returns raw input without panic
func TestFormatJSON_InvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "invalid JSON - missing quote",
			input: `{name:"test"}`,
		},
		{
			name:  "invalid JSON - trailing comma",
			input: `{"key": "value",}`,
		},
		{
			name:  "invalid JSON - single quotes",
			input: `{'name': 'test'}`,
		},
		{
			name:  "invalid JSON - unclosed object",
			input: `{"key": "value"`,
		},
		{
			name:  "invalid JSON - plain text",
			input: `this is not json`,
		},
		{
			name:  "invalid JSON - empty string",
			input: ``,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("FormatJSON() panicked on invalid input: %v", r)
				}
			}()
			opts := FormatOptions{Indent: "  ", Color: false, MaxWidth: 0}
			got := FormatJSON([]byte(tt.input), opts)
			// Should return raw input on error
			if got != tt.input {
				t.Errorf("FormatJSON() = %q, want raw input %q", got, tt.input)
			}
		})
	}
}

// TestFormatJSON_LargePayload tests that large JSON can be formatted without OOM
func TestFormatJSON_LargePayload(t *testing.T) {
	// Create a moderately large JSON (not 10MB to keep test fast, but large enough to verify no OOM)
	// Build 100KB JSON array with many objects
	var buf bytes.Buffer
	buf.WriteString(`{"items":[`)
	for i := 0; i < 1000; i++ {
		if i > 0 {
			buf.WriteString(",")
		}
		buf.WriteString(`{"id":`)
		buf.WriteString(fmt.Sprintf("%d", i%10))
		buf.WriteString(`,"name":"item`)
		buf.WriteString(fmt.Sprintf("%d", i%10))
		buf.WriteString(`","data":"`)
		// Add some repetitive string data
		for j := 0; j < 50; j++ {
			buf.WriteString("x")
		}
		buf.WriteString(`"}`)
	}
	buf.WriteString(`]}`)

	input := buf.String()
	if len(input) < 50000 {
		t.Skip("test buffer too small")
	}

	opts := FormatOptions{Indent: "  ", Color: false, MaxWidth: 0}
	got := FormatJSON([]byte(input), opts)

	// Should produce valid output (not empty, not panicked)
	if len(got) == 0 {
		t.Errorf("FormatJSON() returned empty for large payload")
	}
	// Should be longer than input due to indentation
	if len(got) <= len(input) {
		t.Errorf("FormatJSON() output not longer than input: in=%d, out=%d", len(input), len(got))
	}
}

// TestFormatOptions tests the FormatOptions struct
func TestFormatOptions(t *testing.T) {
	opts := FormatOptions{
		Indent:   "  ",
		Color:    true,
		MaxWidth: 80,
	}

	if opts.Indent != "  " {
		t.Errorf("FormatOptions.Indent = %q, want %q", opts.Indent, "  ")
	}
	if opts.Color != true {
		t.Errorf("FormatOptions.Color = %v, want true", opts.Color)
	}
	if opts.MaxWidth != 80 {
		t.Errorf("FormatOptions.MaxWidth = %d, want 80", opts.MaxWidth)
	}
}

// TestFormatJSONOptions tests FormatOptions with JSON
func TestFormatJSONOptions(t *testing.T) {
	input := `{"name":"test"}`

	// Test with Color: false (no ANSI codes)
	t.Run("color_false_no_ansi", func(t *testing.T) {
		opts := FormatOptions{Indent: "  ", Color: false, MaxWidth: 0}
		got := FormatJSON([]byte(input), opts)
		// Should not contain ANSI codes
		if strings.Contains(got, "\033[") {
			t.Errorf("FormatJSON(Color=false) contains ANSI codes: %q", got)
		}
	})

	// Test with Color: true (has ANSI codes)
	t.Run("color_true_has_ansi", func(t *testing.T) {
		opts := FormatOptions{Indent: "  ", Color: true, MaxWidth: 0}
		got := FormatJSON([]byte(input), opts)
		// Should contain ANSI codes
		if !strings.Contains(got, "\033[") {
			t.Errorf("FormatJSON(Color=true) missing ANSI codes: %q", got)
		}
	})
}
