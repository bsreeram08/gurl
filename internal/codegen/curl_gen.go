package codegen

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"text/template"

	"github.com/sreeram/gurl/pkg/types"
)

// CurlGenerator generates curl commands
type CurlGenerator struct{}

// Language returns the language name
func (g *CurlGenerator) Language() string {
	return "curl"
}

const curlTemplate = `curl -X {{ .Method }}{{ range .Headers }} -H {{ .Key | escapeCurl }}: {{ .Value | escapeCurl }}{{ end }}{{ if .HasBody }}{{ if .IsBinaryBody }}# Binary body omitted{{ else if .IsMultipartBody }} -F 'data=@file'{{ else }} -d {{ .Body | escapeCurl }}{{ end }}{{ end }}{{ if .HasCert }} --cert {{ .Cert }}{{ end }}{{ if .HasKey }} --key {{ .Key }}{{ end }}{{ if .HasCacert }} --cacert {{ .Cacert }}{{ end }}{{ if .Insecure }} -k{{ end }}{{ if .HasCookie }} --cookie {{ .Cookie }}{{ end }}{{ if .HasCookieJar }} --cookie-jar {{ .CookieJar }}{{ end }} {{ .URL | escapeCurl }}{{ if .HasAssertions }}{{ if .HasStatusAssertion }}\n# Expected status: {{ .ExpectedStatus }}{{ end }}{{ end }}`

// CurlCodeGenData holds template data for curl generation
type CurlCodeGenData struct {
	URL               string
	Method            string
	Headers           []types.Header
	Body              string
	HasBody           bool
	HasAuth           bool
	AuthType          string
	AuthValue         string
	HasCert           bool
	Cert              string
	HasKey            bool
	Key               string
	HasCacert         bool
	Cacert            string
	Insecure          bool
	HasCookie         bool
	Cookie            string
	HasCookieJar      bool
	CookieJar         string
	IsBinaryBody      bool
	IsMultipartBody   bool
	HasAssertions     bool
	HasStatusAssertion bool
	ExpectedStatus    string
}

// Generate creates curl command from a SavedRequest
func (g *CurlGenerator) Generate(req *types.SavedRequest, opts *GenOptions) (string, error) {
	opts = sanitizeOpts(opts)

	// Apply variable substitution
	url := substituteVariables(req.URL, req.Variables)
	body := substituteVariables(req.Body, req.Variables)

	// Check content type for binary/multipart handling
	contentType := getContentTypeFromHeaders(req.Headers)
	isBinary := isBinaryContentType(contentType)
	isMultipart := isMultipartContentType(contentType)

	data := CurlCodeGenData{
		URL:             url,
		Method:          req.Method,
		Headers:         filterAuthHeaders(req.Headers),
		HasBody:         body != "" && !isBinary,
		IsBinaryBody:    isBinary,
		IsMultipartBody: isMultipart,
	}

	// Only set body if not binary
	if body != "" && !isBinary {
		data.Body = body
	}

	// Check for auth in headers
	for _, h := range req.Headers {
		if h.Key == "Authorization" || h.Key == "authorization" {
			data.HasAuth = true
			if len(h.Value) > 6 && h.Value[:6] == "Bearer" {
				data.AuthType = "bearer"
				data.AuthValue = "***"
			} else if len(h.Value) > 5 && h.Value[:5] == "Basic" {
				data.AuthType = "basic"
				data.AuthValue = "***"
			} else {
				data.AuthType = "other"
				data.AuthValue = "***"
			}
			break
		}
	}

	// Handle TLS config from AuthConfig
	if req.AuthConfig != nil {
		if cert, ok := req.AuthConfig.Params["cert"]; ok && cert != "" {
			data.HasCert = true
			data.Cert = cert
		}
		if key, ok := req.AuthConfig.Params["key"]; ok && key != "" {
			data.HasKey = true
			data.Key = key
		}
		if cacert, ok := req.AuthConfig.Params["cacert"]; ok && cacert != "" {
			data.HasCacert = true
			data.Cacert = cacert
		}
		if insecure, ok := req.AuthConfig.Params["insecure"]; ok && insecure == "true" {
			data.Insecure = true
		}
	}

	// Handle cookies from AuthConfig
	if req.AuthConfig != nil {
		if cookie, ok := req.AuthConfig.Params["cookie"]; ok && cookie != "" {
			data.HasCookie = true
			data.Cookie = cookie
		}
		if cookieJar, ok := req.AuthConfig.Params["cookie_jar"]; ok && cookieJar != "" {
			data.HasCookieJar = true
			data.CookieJar = cookieJar
		}
	}

	// Handle assertions
	if len(req.Assertions) > 0 {
		data.HasAssertions = true
		for _, assertion := range req.Assertions {
			if assertion.Field == "status" && assertion.Op == "eq" {
				data.HasStatusAssertion = true
				data.ExpectedStatus = assertion.Value
			}
		}
	}

	tmpl, err := template.New("curl").Funcs(template.FuncMap{
		"escapeCurl": escapeCurlString,
	}).Parse(curlTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// filterAuthHeaders filters out Authorization headers and replaces them with a placeholder
func filterAuthHeaders(headers []types.Header) []types.Header {
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

func escapeCurlString(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\'`) + "'"
}

// BuildCurlCommand builds a curl command array for safe execution using exec.Command
// This avoids shell injection vulnerabilities
func BuildCurlCommand(req *types.SavedRequest) ([]string, error) {
	cmd := []string{"curl"}

	if req.Method != "" && req.Method != "GET" {
		cmd = append(cmd, "-X", req.Method)
	}

	for _, h := range req.Headers {
		// Filter auth headers for safe execution
		if h.Key == "Authorization" || h.Key == "authorization" {
			cmd = append(cmd, "-H", fmt.Sprintf("%s: ***", h.Key))
		} else {
			cmd = append(cmd, "-H", fmt.Sprintf("%s: %s", h.Key, h.Value))
		}
	}

	// Skip binary body
	contentType := getContentTypeFromHeaders(req.Headers)
	if req.Body != "" && !isBinaryContentType(contentType) {
		cmd = append(cmd, "-d", req.Body)
	}

	// Handle TLS config from AuthConfig
	if req.AuthConfig != nil {
		if cert, ok := req.AuthConfig.Params["cert"]; ok && cert != "" {
			cmd = append(cmd, "--cert", cert)
		}
		if key, ok := req.AuthConfig.Params["key"]; ok && key != "" {
			cmd = append(cmd, "--key", key)
		}
		if cacert, ok := req.AuthConfig.Params["cacert"]; ok && cacert != "" {
			cmd = append(cmd, "--cacert", cacert)
		}
		if insecure, ok := req.AuthConfig.Params["insecure"]; ok && insecure == "true" {
			cmd = append(cmd, "-k")
		}
	}

	// Handle cookies
	if req.AuthConfig != nil {
		if cookie, ok := req.AuthConfig.Params["cookie"]; ok && cookie != "" {
			cmd = append(cmd, "--cookie", cookie)
		}
		if cookieJar, ok := req.AuthConfig.Params["cookie_jar"]; ok && cookieJar != "" {
			cmd = append(cmd, "--cookie-jar", cookieJar)
		}
	}

	cmd = append(cmd, req.URL)

	return cmd, nil
}

// BuildCurlCommandWithExec runs the curl command using exec.Command for safe execution
func BuildCurlCommandWithExec(req *types.SavedRequest) *exec.Cmd {
	args, _ := BuildCurlCommand(req)
	return exec.Command(args[0], args[1:]...)
}

func shellEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\'`) + "'"
}
