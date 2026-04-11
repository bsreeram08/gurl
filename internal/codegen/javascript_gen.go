package codegen

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/sreeram/gurl/pkg/types"
)

// JavaScriptGenerator generates JavaScript fetch code
type JavaScriptGenerator struct{}

// Language returns the language name
func (g *JavaScriptGenerator) Language() string {
	return "javascript"
}

// Note: fetch() is available in modern browsers and Node.js 18+.
// For older Node.js versions, use node-fetch or cross-fetch polyfills.

const javascriptTemplate = `// Note: fetch is available in browsers and Node.js 18+
// For older Node.js, use: npm install node-fetch
async function makeRequest() {
    const url = {{ .URL | escapeJS }};
{{- if .HasHeaders }}
    const headers = {
{{- range .Headers }}
        {{ .Key | escapeJS }}: {{ .Value | escapeJS }},
{{- end }}
    };
{{- end }}
{{- if .HasBody }}
{{- if .IsBinaryBody }}
    // Binary body not supported in generated JavaScript code
{{- else if .IsMultipartBody }}
    // Multipart body not fully supported - consider using FormData
    const body = {{ .Body | escapeJS }};
{{- else if .IsJSONBody }}
    const body = JSON.stringify({{ .Body }});
{{- else }}
    const body = {{ .Body | escapeJS }};
{{- end }}
{{- end }}
{{- if .HasAuth }}
{{- if eq .AuthType "bearer" }}
    headers["Authorization"] = "Bearer <your-token-here>";
{{- else if eq .AuthType "basic" }}
    headers["Authorization"] = "Basic <your-credentials-here>";
{{- end }}
{{- end }}

    const options = {
        method: {{ .Method | escapeJS }},
{{- if .HasHeaders }}
        headers,
{{- end }}
{{- if .HasBody }}
{{- if not .IsBinaryBody }}
        body,
{{- end }}
{{- end }}
    };

    try {
        const response = await fetch(url, options);
{{- if .HasAssertions }}
        // Assertions
{{- if .HasStatusAssertion }}
        if (response.status !== {{ .ExpectedStatus }}) {
            console.error("Assertion failed: expected status {{ .ExpectedStatus }}, got " + response.status);
            return;
        }
{{- end }}
{{- end }}
        const text = await response.text();
        console.log(response.status);
        console.log(text);
    } catch (error) {
        console.error('Error:', error);
    }
}

makeRequest();
`

// JavaScriptCodeGenData holds template data for JavaScript generation
type JavaScriptCodeGenData struct {
	URL                 string
	Method              string
	Headers             []types.Header
	Body                string
	HasHeaders          bool
	HasBody             bool
	HasAuth             bool
	AuthType            string
	IsJSONBody          bool
	IsBinaryBody        bool
	IsMultipartBody     bool
	HasAssertions       bool
	HasStatusAssertion  bool
	ExpectedStatus      int
}

// Generate creates JavaScript fetch code from a SavedRequest
func (g *JavaScriptGenerator) Generate(req *types.SavedRequest, opts *GenOptions) (string, error) {
	opts = sanitizeOpts(opts)

	// Apply variable substitution
	url := substituteVariables(req.URL, req.Variables)
	body := substituteVariables(req.Body, req.Variables)

	// Check content type for binary/multipart handling
	contentType := getContentTypeFromHeaders(req.Headers)
	isBinary := isBinaryContentType(contentType)
	isMultipart := isMultipartContentType(contentType)

	data := JavaScriptCodeGenData{
		URL:             url,
		Method:          req.Method,
		Headers:         req.Headers,
		HasHeaders:      len(req.Headers) > 0,
		IsBinaryBody:    isBinary,
		IsMultipartBody: isMultipart,
	}

	// Check for auth in headers
	for _, h := range req.Headers {
		if h.Key == "Authorization" || h.Key == "authorization" {
			data.HasAuth = true
			if len(h.Value) > 6 && h.Value[:6] == "Bearer" {
				data.AuthType = "bearer"
			} else if len(h.Value) > 5 && h.Value[:5] == "Basic" {
				data.AuthType = "basic"
			}
			break
		}
	}

	// Check if body is JSON
	if body != "" && !isBinary {
		data.HasBody = true
		// Check content-type header for JSON
		for _, h := range req.Headers {
			if (h.Key == "Content-Type" || h.Key == "content-type") && containsJSON(h.Value) {
				data.IsJSONBody = true
			}
		}
		data.Body = body
	}

	// Handle assertions
	if len(req.Assertions) > 0 {
		data.HasAssertions = true
		for _, assertion := range req.Assertions {
			if assertion.Field == "status" && assertion.Op == "eq" {
				if status, err := strconv.Atoi(assertion.Value); err == nil {
					data.HasStatusAssertion = true
					data.ExpectedStatus = status
				}
			}
		}
	}

	tmpl, err := template.New("javascript").Funcs(javascriptFuncMap()).Parse(javascriptTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

func containsJSON(contentType string) bool {
	return len(contentType) >= 16 && (contentType[:16] == "application/json" || contentType[:16] == "application/json;")
}

func escapeJSString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "'", `\'`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "<", `\x3c`)
	s = strings.ReplaceAll(s, ">", `\x3e`)
	return `"` + s + `"`
}

func javascriptFuncMap() template.FuncMap {
	return template.FuncMap{
		"escapeJS": escapeJSString,
	}
}

// jsVariablePattern matches {{var}} patterns for JavaScript
var jsVariablePattern = regexp.MustCompile(`\{\{([^}]+)\}\}`)
