package builtins

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sreeram/gurl/internal/plugins"
)

type SlackOutput struct{}

func (s *SlackOutput) Name() string   { return "slack" }
func (s *SlackOutput) Format() string { return "slack" }

func (s *SlackOutput) Render(ctx *plugins.ResponseContext) string {
	if ctx == nil || ctx.Response == nil {
		return "❌ No response"
	}

	resp := ctx.Response
	req := ctx.Request

	// Status emoji based on status code
	emoji := "⚠️"
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		emoji = "✅"
	} else if resp.StatusCode >= 400 {
		emoji = "❌"
	}

	// Get status text from common HTTP status codes
	statusText := getStatusText(resp.StatusCode)

	// Build the output
	var sb strings.Builder

	// Line 1: emoji + method + URL + status
	sb.WriteString(fmt.Sprintf("%s %s %s (%d %s)\n", emoji, req.Method, req.URL, resp.StatusCode, statusText))

	// Line 2+: fenced code block with body
	bodyStr := string(resp.Body)
	if len(bodyStr) > 1000 {
		bodyStr = bodyStr[:1000] + "..."
	}

	// Try to pretty-print JSON
	var bodyObj interface{}
	if json.Unmarshal(resp.Body, &bodyObj) == nil {
		var buf strings.Builder
		enc := json.NewEncoder(&buf)
		enc.SetIndent("", "  ")
		_ = enc.Encode(bodyObj)
		bodyStr = buf.String()
		if len(bodyStr) > 1000 {
			bodyStr = bodyStr[:1000] + "..."
		}
	}

	sb.WriteString("```json\n")
	sb.WriteString(bodyStr)
	sb.WriteString("\n```")

	return sb.String()
}

func getStatusText(code int) string {
	switch code {
	case 200:
		return "OK"
	case 201:
		return "Created"
	case 204:
		return "No Content"
	case 301:
		return "Moved Permanently"
	case 302:
		return "Found"
	case 304:
		return "Not Modified"
	case 400:
		return "Bad Request"
	case 401:
		return "Unauthorized"
	case 403:
		return "Forbidden"
	case 404:
		return "Not Found"
	case 405:
		return "Method Not Allowed"
	case 408:
		return "Request Timeout"
	case 409:
		return "Conflict"
	case 422:
		return "Unprocessable Entity"
	case 429:
		return "Too Many Requests"
	case 500:
		return "Internal Server Error"
	case 501:
		return "Not Implemented"
	case 502:
		return "Bad Gateway"
	case 503:
		return "Service Unavailable"
	case 504:
		return "Gateway Timeout"
	default:
		return ""
	}
}
