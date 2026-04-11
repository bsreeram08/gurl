package codegen

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

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
{{- if or .IncludeImports .HasTimeout }}
	"time"
{{- end }}
)

func main() {
{{- if .IncludeComments }}
	// {{ .Comment }}
{{- end }}
	url := {{ .URL | escapeGo }}
	method := {{ .Method | escapeGo }}
{{- if .HasHeaders }}
	headers := map[string]string{
{{- range .Headers }}
		{{ .Key | escapeGo }}: {{ .Value | escapeGo }},
{{- end }}
	}
{{- end }}
{{- if .HasBody }}
{{- if .IsBinaryBody }}
	// Binary body not supported in generated Go code
{{- else if .IsMultipartBody }}
	// Multipart body not fully supported in generated Go code
{{- else }}
{{- end }}
{{- end }}
{{- if .HasAuth }}
	// Authorization header redacted for security
{{- end }}

	req, err := http.NewRequest(method, url{{ if and .HasBody (not .IsBinaryBody) }}, bytes.NewBufferString({{ .Body }}){{ else }}, nil{{ end }})
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
	req.Header.Set("Authorization", "Bearer <your-token-here>")
{{- end }}

{{- if .HasTimeout }}
	client := &http.Client{
		Timeout: {{ .Timeout }},
	}
{{- else }}
	client := &http.Client{}
{{- end }}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()

{{- if .HasAssertions }}
	// Assertions
{{- if .HasStatusAssertion }}
	if resp.StatusCode != {{ .ExpectedStatus }} {
		fmt.Printf("Assertion failed: expected status %d, got %d\n", {{ .ExpectedStatus }}, resp.StatusCode)
		return
	}
{{- end }}
{{- end }}

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
	URL              string
	Method           string
	Headers          []types.Header
	Body             string
	HasHeaders       bool
	HasBody          bool
	HasAuth          bool
	AuthType         string
	IncludeImports   bool
	IncludeComments  bool
	Comment          string
	HasTimeout       bool
	Timeout          string
	HasAssertions    bool
	HasStatusAssertion bool
	ExpectedStatus   int
	IsBinaryBody     bool
	IsMultipartBody  bool
}

// Generate creates Go net/http code from a SavedRequest
func (g *GoGenerator) Generate(req *types.SavedRequest, opts *GenOptions) (string, error) {
	opts = sanitizeOpts(opts)

	// Apply variable substitution
	url := substituteVariables(req.URL, req.Variables)
	body := substituteVariables(req.Body, req.Variables)

	// Check content type for binary/multipart handling
	contentType := getContentTypeFromHeaders(req.Headers)
	isBinary := isBinaryContentType(contentType)
	isMultipart := isMultipartContentType(contentType)

	data := GoCodeGenData{
		URL:             url,
		Method:          req.Method,
		Headers:         filterAuthHeadersGo(req.Headers),
		HasHeaders:      len(req.Headers) > 0,
		IncludeImports:  opts.IncludeImports,
		IncludeComments: opts.IncludeComments,
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

	// Handle body - skip binary content
	if body != "" && !isBinary {
		data.HasBody = true
		data.Body = escapeGoString(body)
	}

	// Handle timeout
	if req.Timeout != "" {
		if duration, err := parseTimeout(req.Timeout); err == nil {
			data.HasTimeout = true
			data.Timeout = fmt.Sprintf("%d * time.Second", int(duration.Seconds()))
		}
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

	// Handle comment
	if data.IncludeComments {
		data.Comment = fmt.Sprintf("Request: %s %s", req.Method, req.URL)
	}

	tmpl, err := template.New("go").Funcs(goFuncMap()).Parse(goTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// filterAuthHeadersGo filters out Authorization headers and replaces them with a placeholder
func filterAuthHeadersGo(headers []types.Header) []types.Header {
	filtered := make([]types.Header, 0, len(headers))
	for _, h := range headers {
		if h.Key == "Authorization" || h.Key == "authorization" {
			filtered = append(filtered, types.Header{
				Key:   h.Key,
				Value: "<your-token-here>",
			})
		} else {
			filtered = append(filtered, h)
		}
	}
	return filtered
}

func escapeGoString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return `"` + s + `"`
}

func goFuncMap() template.FuncMap {
	return template.FuncMap{
		"escapeGo": escapeGoString,
	}
}

// parseTimeout parses a timeout string into time.Duration
func parseTimeout(timeout string) (duration time.Duration, err error) {
	// Try to parse as plain number (seconds)
	if seconds, err := strconv.Atoi(timeout); err == nil {
		return time.Duration(seconds) * time.Second, nil
	}
	// Try to parse as duration string
	return time.ParseDuration(timeout)
}

// variablePattern matches {{var}} patterns
var variablePattern = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// substituteURLVariables replaces {{var}} patterns with values from vars map
func substituteURLVariables(url string, vars []types.Var) string {
	result := url
	for _, v := range vars {
		pattern := regexp.MustCompile(`\{\{`+regexp.QuoteMeta(v.Name)+`\}\}`)
		result = pattern.ReplaceAllString(result, v.Example)
	}
	return result
}

// sanitizeOpts ensures opts is never nil
func sanitizeOpts(opts *GenOptions) *GenOptions {
	if opts == nil {
		return &GenOptions{}
	}
	return opts
}
