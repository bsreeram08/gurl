package codegen

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/sreeram/gurl/pkg/types"
)

// PythonGenerator generates Python requests code
type PythonGenerator struct{}

// Language returns the language name
func (g *PythonGenerator) Language() string {
	return "python"
}

const pythonTemplate = `import requests
{{- if .HasHeaders }}
import json
{{- end }}

def make_request():
    url = "{{ .URL }}"
{{- if .HasHeaders }}
    headers = {
{{- range .Headers }}
        "{{ .Key }}": "{{ .Value }}",
{{- end }}
    }
{{- end }}
{{- if .HasBody }}
    data = {{ .Body }}
{{- end }}
{{- if .HasAuth }}
{{- if eq .AuthType "bearer" }}
    headers["Authorization"] = "Bearer <your-token-here>"
{{- else if eq .AuthType "basic" }}
    auth = ("<your-username>", "<your-password>")
{{- end }}
{{- end }}

{{- if .HasBody }}
{{- if eq .Method "GET" }}
    response = requests.get(url, headers=headers{{ if .HasAuth }}{{ if eq .AuthType "basic" }}, auth=auth{{ end }}{{ end }})
{{- else }}
    response = requests.{{ .MethodCall }}(url, headers=headers{{ if .HasAuth }}{{ if eq .AuthType "basic" }}, auth=auth{{ end }}{{ end }}{{ if .HasBody }}, json=data{{ end }})
{{- end }}
{{- else }}
    response = requests.{{ .MethodCall }}(url, headers=headers{{ if .HasAuth }}{{ if eq .AuthType "basic" }}, auth=auth{{ end }}{{ end }})
{{- end }}

    print(response.status_code)
    print(response.text)

if __name__ == "__main__":
    make_request()
`

// PythonCodeGenData holds template data for Python generation
type PythonCodeGenData struct {
	URL        string
	Method     string
	MethodCall string
	Headers    []types.Header
	Body       string
	HasHeaders bool
	HasBody    bool
	HasAuth    bool
	AuthType   string
	Comment    string
}

// Generate creates Python requests code from a SavedRequest
func (g *PythonGenerator) Generate(req *types.SavedRequest, opts *GenOptions) (string, error) {
	data := PythonCodeGenData{
		URL:        req.URL,
		Method:     req.Method,
		Headers:    req.Headers,
		HasHeaders: len(req.Headers) > 0,
	}

	// Map method to requests method call
	methodCall := strings.ToLower(req.Method)
	if methodCall == "" {
		methodCall = "get"
	}
	data.MethodCall = methodCall

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

	// Handle body
	if req.Body != "" {
		data.HasBody = true
		data.Body = req.Body
	}

	tmpl, err := template.New("python").Parse(pythonTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
