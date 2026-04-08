package builtins

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/sreeram/gurl/internal/plugins"
)

// LoggingMiddleware logs HTTP requests and responses.
type LoggingMiddleware struct {
	output *bytes.Buffer // for testing injection
}

// sensitiveHeaders is a map of header names that should be redacted in logs.
var sensitiveHeaders = map[string]bool{
	"authorization":       true,
	"cookie":              true,
	"set-cookie":          true,
	"proxy-authorization": true,
}

// Name returns the plugin name.
func (l *LoggingMiddleware) Name() string { return "logging" }

// BeforeRequest logs the outgoing request with method, URL, and headers (with sensitive values redacted).
func (l *LoggingMiddleware) BeforeRequest(ctx *plugins.RequestContext) *plugins.RequestContext {
	if ctx == nil || ctx.Request == nil {
		return ctx
	}

	var buf bytes.Buffer
	buf.WriteString("→ ")
	buf.WriteString(ctx.Request.Method)
	buf.WriteString(" ")
	buf.WriteString(ctx.Request.URL)
	buf.WriteString("\n")

	for _, h := range ctx.Request.Headers {
		buf.WriteString("  ")
		buf.WriteString(h.Key)
		buf.WriteString(": ")
		if sensitiveHeaders[strings.ToLower(h.Key)] {
			buf.WriteString("[REDACTED]")
		} else {
			buf.WriteString(h.Value)
		}
		buf.WriteString("\n")
	}

	l.write(buf.String())
	return ctx
}

// AfterResponse logs the incoming response with status code, duration, and size.
func (l *LoggingMiddleware) AfterResponse(ctx *plugins.ResponseContext) *plugins.ResponseContext {
	if ctx == nil || ctx.Response == nil {
		return ctx
	}

	var buf bytes.Buffer
	buf.WriteString("← ")
	buf.WriteString(fmt.Sprintf("%d", ctx.Response.StatusCode))
	buf.WriteString(" (")
	buf.WriteString(fmt.Sprintf("%d", ctx.Response.Duration.Milliseconds()))
	buf.WriteString("ms) ")
	buf.WriteString(fmt.Sprintf("%d", ctx.Response.Size))
	buf.WriteString("B\n")

	l.write(buf.String())
	return ctx
}

// write outputs the string to the configured buffer or discards if no buffer is set.
func (l *LoggingMiddleware) write(s string) {
	if l.output != nil {
		l.output.WriteString(s)
	}
}
