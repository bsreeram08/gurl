package plugins

import (
	"github.com/sreeram/gurl/internal/client"
)

// RequestContext holds the request and environment before a plugin's BeforeRequest runs.
type RequestContext struct {
	Request *client.Request
	Env     map[string]string
}

// ResponseContext holds request, response, and environment after a plugin's AfterResponse runs.
type ResponseContext struct {
	Request  *client.Request
	Response *client.Response
	Env      map[string]string
}

// MiddlewarePlugin is implemented by plugins that want to intercept and modify
// requests before they are sent and responses before they are returned.
type MiddlewarePlugin interface {
	Name() string
	BeforeRequest(ctx *RequestContext) *RequestContext
	AfterResponse(ctx *ResponseContext) *ResponseContext
}

// OutputPlugin is implemented by plugins that format and render responses.
type OutputPlugin interface {
	Name() string
	Format() string
	Render(ctx *ResponseContext) string
}

// CommandPlugin is implemented by plugins that add new commands to the CLI.
type CommandPlugin interface {
	Name() string
	Command() string
	Description() string
	Run(args []string) error
}
