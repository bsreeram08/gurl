package builtins

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sreeram/gurl/internal/plugins"
)

type MarkdownOutput struct{}

func (m *MarkdownOutput) Name() string   { return "markdown" }
func (m *MarkdownOutput) Format() string { return "markdown" }

func (m *MarkdownOutput) Render(ctx *plugins.ResponseContext) string {
	if ctx == nil || ctx.Response == nil {
		return "# No Response"
	}

	resp := ctx.Response
	req := ctx.Request

	var sb strings.Builder

	// H1: # METHOD URL (STATUS_CODE)
	sb.WriteString(fmt.Sprintf("# %s %s (%d)\n\n", req.Method, req.URL, resp.StatusCode))

	// Table: | Header | Value | with all response headers
	sb.WriteString("## Headers\n\n")
	sb.WriteString("| Header | Value |\n")
	sb.WriteString("|--------|-------|\n")

	// Sort headers for consistent output
	keys := make([]string, 0, len(resp.Headers))
	for k := range resp.Headers {
		keys = append(keys, k)
	}
	for _, k := range keys {
		values := resp.Headers[k]
		for _, v := range values {
			sb.WriteString(fmt.Sprintf("| %s | %s |\n", k, v))
		}
	}

	sb.WriteString("\n## Body\n\n")

	// Fenced code block with pretty-printed body
	bodyStr := string(resp.Body)
	if len(bodyStr) > 0 {
		var bodyObj interface{}
		if json.Unmarshal(resp.Body, &bodyObj) == nil {
			var buf strings.Builder
			enc := json.NewEncoder(&buf)
			enc.SetIndent("", "  ")
			if err := enc.Encode(bodyObj); err != nil {
				return ""
			}
			bodyStr = buf.String()
		}
		sb.WriteString("```json\n")
		sb.WriteString(bodyStr)
		sb.WriteString("\n```\n")
	} else {
		sb.WriteString("*Empty response body*\n")
	}

	// Footer: Duration
	sb.WriteString(fmt.Sprintf("\n---\n*Duration: %dms*\n", resp.Duration.Milliseconds()))

	return sb.String()
}
