package codegen

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/sreeram/gurl/pkg/types"
)

// CurlGenerator generates curl commands
type CurlGenerator struct{}

// Language returns the language name
func (g *CurlGenerator) Language() string {
	return "curl"
}

const curlTemplate = `curl -X {{ .Method }}{{ range .Headers }} -H '{{ .Key }}: {{ .Value }}'{{ end }}{{ if .HasBody }} -d '{{ .Body }}'{{ end }} '{{ .URL }}'`

// CurlCodeGenData holds template data for curl generation
type CurlCodeGenData struct {
	URL       string
	Method    string
	Headers   []types.Header
	Body      string
	HasBody   bool
	HasAuth   bool
	AuthType  string
	AuthValue string
}

// Generate creates curl command from a SavedRequest
func (g *CurlGenerator) Generate(req *types.SavedRequest, opts *GenOptions) (string, error) {
	data := CurlCodeGenData{
		URL:     req.URL,
		Method:  req.Method,
		Headers: req.Headers,
		HasBody: req.Body != "",
		Body:    req.Body,
	}

	// Check for auth in headers
	for _, h := range req.Headers {
		if h.Key == "Authorization" || h.Key == "authorization" {
			data.HasAuth = true
			if len(h.Value) > 6 && h.Value[:6] == "Bearer" {
				data.AuthType = "bearer"
				data.AuthValue = "<your-token-here>"
			} else if len(h.Value) > 5 && h.Value[:5] == "Basic" {
				data.AuthType = "basic"
				data.AuthValue = "<your-credentials-here>"
			}
			break
		}
	}

	tmpl, err := template.New("curl").Parse(curlTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

func escapeCurlString(s string) string {
	// Escape single quotes for shell
	return string(bytes.Replace([]byte(s), []byte(`'`), []byte(`\'`), -1))
}

// BuildCurlCommand builds a curl command array for safe execution
func BuildCurlCommand(req *types.SavedRequest) ([]string, error) {
	cmd := []string{"curl"}

	if req.Method != "" && req.Method != "GET" {
		cmd = append(cmd, "-X", req.Method)
	}

	for _, h := range req.Headers {
		cmd = append(cmd, "-H", fmt.Sprintf("%s: %s", h.Key, h.Value))
	}

	if req.Body != "" {
		cmd = append(cmd, "-d", req.Body)
	}

	cmd = append(cmd, req.URL)

	return cmd, nil
}
