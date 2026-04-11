package builtins

import (
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/sreeram/gurl/internal/plugins"
)

type CSVOutput struct{}

func (c *CSVOutput) Name() string   { return "csv" }
func (c *CSVOutput) Format() string { return "csv" }

func (c *CSVOutput) Render(ctx *plugins.ResponseContext) string {
	if ctx == nil || ctx.Response == nil {
		return ""
	}

	resp := ctx.Response
	req := ctx.Request

	var sb strings.Builder
	writer := csv.NewWriter(&sb)

	// Get content type
	contentType := ""
	if resp.Headers != nil {
		if v := resp.Headers.Get("Content-Type"); v != "" {
			contentType = v
		}
	}

	// Escape fields properly
	record := []string{
		fmt.Sprintf("%d", resp.StatusCode),
		req.URL,
		fmt.Sprintf("%d", resp.Duration.Milliseconds()),
		contentType,
	}

	_ = writer.Write(record)
	if err := writer.Error(); err != nil {
		return ""
	}
	writer.Flush()

	return sb.String()
}
