package codegen

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/sreeram/gurl/pkg/types"
)

// GoGenerator generates Go net/http code
type GoGenerator struct{}

// Language returns the language name
func (g *GoGenerator) Language() string {
	return "go"
}

const goTemplate = `package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
{{- if .IncludeImports }}
	"net/http"
{{- end }}
)

func main() {
{{- if .IncludeComments }}
	// {{ .Comment }}
{{- end }}
	url := "{{ .URL }}"
	method := "{{ .Method }}"
{{- if .HasHeaders }}
	headers := map[string]string{
{{- range .Headers }}
		"{{ .Key }}": "{{ .Value }}",
{{- end }}
	}
{{- end }}
{{- if .HasBody }}
	body := {{ .Body }}
{{- end }}
{{- if .HasAuth }}
	// Set authorization header
{{- if eq .AuthType "bearer" }}
	req.Header.Set("Authorization", "Bearer <your-token-here>")
{{- else if eq .AuthType "basic" }}
	req.Header.Set("Authorization", "Basic <your-credentials-here>")
{{- end }}
{{- end }}

	req, err := http.NewRequest(method, url{{ if .HasBody }}, bytes.NewBufferString(body){{ else }}, nil{{ end }})
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

{{- if .HasHeaders }}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
{{- end }}
{{- if .HasAuth }}
{{- if eq .AuthType "bearer" }}
	req.Header.Set("Authorization", "Bearer <your-token-here>")
{{- else if eq .AuthType "basic" }}
	req.Header.Set("Authorization", "Basic <your-credentials-here>")
{{- end }}
{{- end }}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return
	}
	fmt.Println(string(responseBody))
}
`

// GoCodeGenData holds template data for Go generation
type GoCodeGenData struct {
	URL             string
	Method          string
	Headers         []types.Header
	Body            string
	HasHeaders      bool
	HasBody         bool
	HasAuth         bool
	AuthType        string
	IncludeImports  bool
	IncludeComments bool
	Comment         string
}

// Generate creates Go net/http code from a SavedRequest
func (g *GoGenerator) Generate(req *types.SavedRequest, opts *GenOptions) (string, error) {
	data := GoCodeGenData{
		URL:             req.URL,
		Method:          req.Method,
		Headers:         req.Headers,
		HasHeaders:      len(req.Headers) > 0,
		IncludeImports:  opts != nil && opts.IncludeImports,
		IncludeComments: opts != nil && opts.IncludeComments,
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

	// Handle body
	if req.Body != "" {
		data.HasBody = true
		data.Body = escapeGoString(req.Body)
	}

	// Handle comment
	if data.IncludeComments {
		data.Comment = fmt.Sprintf("Request: %s %s", req.Method, req.URL)
	}

	tmpl, err := template.New("go").Parse(goTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

func escapeGoString(s string) string {
	// Escape backticks and dollar signs for Go template
	s = string(bytes.Replace([]byte(s), []byte("`"), []byte("`+\"`\"+`"), -1))
	s = string(bytes.Replace([]byte(s), []byte("$"), []byte("\\$"), -1))
	return s
}
