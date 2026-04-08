package codegen

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/sreeram/gurl/pkg/types"
)

// JavaScriptGenerator generates JavaScript fetch code
type JavaScriptGenerator struct{}

// Language returns the language name
func (g *JavaScriptGenerator) Language() string {
	return "javascript"
}

const javascriptTemplate = `async function makeRequest() {
    const url = "{{ .URL }}";
{{- if .HasHeaders }}
    const headers = {
{{- range .Headers }}
        "{{ .Key }}": "{{ .Value }}",
{{- end }}
    };
{{- end }}
{{- if .HasBody }}
{{- if .IsJSONBody }}
    const body = JSON.stringify({{ .Body }});
{{- else }}
    const body = ` + "`" + `{{ .Body }}` + "`" + `;
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
        method: "{{ .Method }}",
{{- if .HasHeaders }}
        headers,
{{- end }}
{{- if .HasBody }}
        body,
{{- end }}
    };

    try {
        const response = await fetch(url, options);
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
	URL        string
	Method     string
	Headers    []types.Header
	Body       string
	HasHeaders bool
	HasBody    bool
	HasAuth    bool
	AuthType   string
	IsJSONBody bool
}

// Generate creates JavaScript fetch code from a SavedRequest
func (g *JavaScriptGenerator) Generate(req *types.SavedRequest, opts *GenOptions) (string, error) {
	data := JavaScriptCodeGenData{
		URL:        req.URL,
		Method:     req.Method,
		Headers:    req.Headers,
		HasHeaders: len(req.Headers) > 0,
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
	if req.Body != "" {
		data.HasBody = true
		// Check content-type header for JSON
		for _, h := range req.Headers {
			if (h.Key == "Content-Type" || h.Key == "content-type") && containsJSON(h.Value) {
				data.IsJSONBody = true
			}
		}
		data.Body = req.Body
	}

	tmpl, err := template.New("javascript").Parse(javascriptTemplate)
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
