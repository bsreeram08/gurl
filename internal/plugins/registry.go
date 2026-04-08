package plugins

import (
	"fmt"
	"os"
)

// Registry manages plugins and provides dispatch for applying them to requests/responses.
type Registry struct {
	middleware []MiddlewarePlugin
	outputs    []OutputPlugin
	commands   []CommandPlugin
}

// NewRegistry creates a new Registry with no plugins registered.
func NewRegistry() *Registry {
	return &Registry{
		middleware: []MiddlewarePlugin{},
		outputs:    []OutputPlugin{},
		commands:   []CommandPlugin{},
	}
}

// Register categorizes a plugin into the appropriate slice based on its concrete type.
// Uses type switch (NOT reflect) for plugin categorization.
func (r *Registry) Register(plugin interface{}) {
	switch p := plugin.(type) {
	case MiddlewarePlugin:
		r.middleware = append(r.middleware, p)
	case OutputPlugin:
		r.outputs = append(r.outputs, p)
	case CommandPlugin:
		r.commands = append(r.commands, p)
	default:
		// Unknown plugin type - ignore silently
		fmt.Fprintf(os.Stderr, "Warning: unknown plugin type %T, ignoring\n", plugin)
	}
}

// Middleware returns all registered middleware plugins in registration order.
func (r *Registry) Middleware() []MiddlewarePlugin {
	return r.middleware
}

// Outputs returns all registered output plugins.
func (r *Registry) Outputs() []OutputPlugin {
	return r.outputs
}

// Commands returns all registered command plugins.
func (r *Registry) Commands() []CommandPlugin {
	return r.commands
}

// ApplyBeforeRequest chains all middleware plugins' BeforeRequest in registration order.
// Each plugin's call is wrapped in recover() to prevent a panicking plugin from
// breaking the entire chain.
func (r *Registry) ApplyBeforeRequest(ctx *RequestContext) *RequestContext {
	if ctx == nil {
		return nil
	}
	result := ctx
	for _, m := range r.middleware {
		result = r.safeBeforeRequest(m, result)
		if result == nil {
			break
		}
	}
	return result
}

// safeBeforeRequest calls plugin.BeforeRequest with panic recovery.
func (r *Registry) safeBeforeRequest(plugin MiddlewarePlugin, ctx *RequestContext) (result *RequestContext) {
	defer func() {
		if rec := recover(); rec != nil {
			fmt.Fprintf(os.Stderr, "Warning: middleware %s panicked in BeforeRequest: %v\n", plugin.Name(), rec)
			result = ctx // Continue with original context on panic
		}
	}()
	return plugin.BeforeRequest(ctx)
}

// ApplyAfterResponse chains all middleware plugins' AfterResponse in reverse order.
// Each plugin's call is wrapped in recover() to prevent a panicking plugin from
// breaking the entire chain.
func (r *Registry) ApplyAfterResponse(ctx *ResponseContext) *ResponseContext {
	if ctx == nil {
		return nil
	}
	result := ctx
	// Reverse order for AfterResponse
	for i := len(r.middleware) - 1; i >= 0; i-- {
		m := r.middleware[i]
		result = r.safeAfterResponse(m, result)
		if result == nil {
			break
		}
	}
	return result
}

// safeAfterResponse calls plugin.AfterResponse with panic recovery.
func (r *Registry) safeAfterResponse(plugin MiddlewarePlugin, ctx *ResponseContext) (result *ResponseContext) {
	defer func() {
		if rec := recover(); rec != nil {
			fmt.Fprintf(os.Stderr, "Warning: middleware %s panicked in AfterResponse: %v\n", plugin.Name(), rec)
			result = ctx // Continue with original context on panic
		}
	}()
	return plugin.AfterResponse(ctx)
}

// GetOutputByFormat looks up an output plugin by its format name.
// Returns the plugin and true if found, or nil and false if not found.
func (r *Registry) GetOutputByFormat(format string) (OutputPlugin, bool) {
	for _, o := range r.outputs {
		if o.Format() == format {
			return o, true
		}
	}
	return nil, false
}
