package builtins

import (
	"fmt"

	"github.com/sreeram/gurl/internal/plugins"
)

type MinimalOutput struct{}

func (m *MinimalOutput) Name() string   { return "minimal" }
func (m *MinimalOutput) Format() string { return "minimal" }

func (m *MinimalOutput) Render(ctx *plugins.ResponseContext) string {
	if ctx == nil || ctx.Response == nil {
		return ""
	}

	resp := ctx.Response
	statusText := getStatusText(resp.StatusCode)
	if statusText == "" {
		statusText = "Unknown"
	}

	return fmt.Sprintf("%d %s (%dms) %dB",
		resp.StatusCode,
		statusText,
		resp.Duration.Milliseconds(),
		resp.Size,
	)
}
