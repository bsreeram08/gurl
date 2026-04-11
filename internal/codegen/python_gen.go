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

const pythonTemplateSync = `import requests
{{- if .HasHeaders }}
import json
{{- end }}

def make_request():
    url = {{ .URL | escapePython }}
{{- if .HasHeaders }}
    headers = {
{{- range .Headers }}
        {{ .Key | escapePython }}: {{ .Value | escapePython }},
{{- end }}
    }
{{- end }}
{{- if .HasBody }}
{{- if .IsBinaryBody }}
    # Binary body not supported in generated Python code
    data = None
{{- else if .IsMultipartBody }}
    # Multipart body requires manual file handling
    data = None
{{- else }}
    data = {{ .Body }}
{{- end }}
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
    response = requests.{{ .MethodCall }}(url, headers=headers{{ if .HasAuth }}{{ if eq .AuthType "basic" }}, auth=auth{{ end }}{{ end }}{{ if .HasBody }}{{ if not .IsBinaryBody }}{{ if not .IsMultipartBody }}, json=data{{ end }}{{ end }}{{ end }})
{{- end }}
{{- else }}
    response = requests.{{ .MethodCall }}(url, headers=headers{{ if .HasAuth }}{{ if eq .AuthType "basic" }}, auth=auth{{ end }}{{ end }})
{{- end }}

    print(response.status_code)
    print(response.text)

{{- if .HasAssertions }}
    # Assertions
{{- if .HasStatusAssertion }}
    assert response.status_code == {{ .ExpectedStatus }}, f"Expected status {{ .ExpectedStatus }}, got {response.status_code}"
{{- end }}
{{- end }}

if __name__ == "__main__":
    make_request()
`

const pythonTemplateAsync = `import httpx
{{- if .HasHeaders }}
import json
{{- end }}

async def make_request():
    url = {{ .URL | escapePython }}
{{- if .HasHeaders }}
    headers = {
{{- range .Headers }}
        {{ .Key | escapePython }}: {{ .Value | escapePython }},
{{- end }}
    }
{{- end }}
{{- if .HasBody }}
{{- if .IsBinaryBody }}
    # Binary body not supported in generated Python code
    data = None
{{- else if .IsMultipartBody }}
    # Multipart body requires manual file handling
    data = None
{{- else }}
    data = {{ .Body }}
{{- end }}
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
    async with httpx.AsyncClient() as client:
        response = await client.get(url, headers=headers{{ if .HasAuth }}{{ if eq .AuthType "basic" }}, auth=auth{{ end }}{{ end }})
{{- else }}
    async with httpx.AsyncClient() as client:
        response = await client.{{ .MethodCall }}(url, headers=headers{{ if .HasAuth }}{{ if eq .AuthType "basic" }}, auth=auth{{ end }}{{ end }}{{ if .HasBody }}{{ if not .IsBinaryBody }}{{ if not .IsMultipartBody }}, json=data{{ end }}{{ end }}{{ end }})
{{- end }}
{{- else }}
    async with httpx.AsyncClient() as client:
        response = await client.{{ .MethodCall }}(url, headers=headers{{ if .HasAuth }}{{ if eq .AuthType "basic" }}, auth=auth{{ end }}{{ end }})
{{- end }}

    print(response.status_code)
    print(response.text)

{{- if .HasAssertions }}
    # Assertions
{{- if .HasStatusAssertion }}
    assert response.status_code == {{ .ExpectedStatus }}, f"Expected status {{ .ExpectedStatus }}, got {response.status_code}"
{{- end }}
{{- end }}

if __name__ == "__main__":
    import asyncio
    asyncio.run(make_request())
`

// PythonCodeGenData holds template data for Python generation
type PythonCodeGenData struct {
	URL                 string
	Method              string
	MethodCall          string
	Headers             []types.Header
	Body                string
	HasHeaders          bool
	HasBody             bool
	HasAuth             bool
	AuthType            string
	Comment             string
	IsBinaryBody        bool
	IsMultipartBody     bool
	HasAssertions       bool
	HasStatusAssertion  bool
	ExpectedStatus      int
}

// Generate creates Python requests code from a SavedRequest
func (g *PythonGenerator) Generate(req *types.SavedRequest, opts *GenOptions) (string, error) {
	opts = sanitizeOpts(opts)

	// Apply variable substitution
	url := substituteVariables(req.URL, req.Variables)
	body := substituteVariables(req.Body, req.Variables)

	// Check content type for binary/multipart handling
	contentType := getContentTypeFromHeaders(req.Headers)
	isBinary := isBinaryContentType(contentType)
	isMultipart := isMultipartContentType(contentType)

	data := PythonCodeGenData{
		URL:            url,
		Method:         req.Method,
		Headers:        req.Headers,
		HasHeaders:     len(req.Headers) > 0,
		IsBinaryBody:   isBinary,
		IsMultipartBody: isMultipart,
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

	// Handle body - skip binary content
	if body != "" && !isBinary {
		data.HasBody = true
		data.Body = escapePythonString(body)
	}

	// Handle assertions
	if len(req.Assertions) > 0 {
		data.HasAssertions = true
		for _, assertion := range req.Assertions {
			if assertion.Field == "status" && assertion.Op == "eq" {
				var status int
				if _, err := fmt.Sscanf(assertion.Value, "%d", &status); err == nil {
					data.HasStatusAssertion = true
					data.ExpectedStatus = status
				}
			}
		}
	}

	// Select template based on opts
	var tmplStr string
	if opts.AsyncPython {
		tmplStr = pythonTemplateAsync
	} else {
		tmplStr = pythonTemplateSync
	}

	tmpl, err := template.New("python").Funcs(pythonFuncMap()).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

func escapePythonString(s string) string {
	// Use single quotes and escape only what's necessary
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return `'` + s + `'`
}

func pythonFuncMap() template.FuncMap {
	return template.FuncMap{
		"escapePython": escapePythonString,
	}
}
