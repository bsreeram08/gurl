package builtins

import (
	"github.com/sreeram/gurl/internal/plugins"
)

// Retry401Middleware flags 401 responses for retry by the execution engine.
type Retry401Middleware struct{}

// Name returns the plugin name.
func (r *Retry401Middleware) Name() string { return "retry-401" }

// BeforeRequest is a pass-through that returns ctx unchanged.
func (r *Retry401Middleware) BeforeRequest(ctx *plugins.RequestContext) *plugins.RequestContext {
	return ctx
}

// AfterResponse sets ctx.Env["_retry_401"] = "true" if the response status code is 401.
func (r *Retry401Middleware) AfterResponse(ctx *plugins.ResponseContext) *plugins.ResponseContext {
	if ctx == nil {
		return nil
	}
	if ctx.Env == nil {
		ctx.Env = make(map[string]string)
	}

	if ctx.Response != nil && ctx.Response.StatusCode == 401 {
		ctx.Env["_retry_401"] = "true"
	}

	return ctx
}
