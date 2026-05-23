package plugins

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/sreeram/gurl/internal/client"
)

// Registry manages plugins and provides dispatch for applying them to requests/responses.
type Registry struct {
	middleware []MiddlewarePlugin
	outputs    []OutputPlugin
	commands   []CommandPlugin
	auths      []AuthPlugin
}

// NewRegistry creates a new Registry with no plugins registered.
func NewRegistry() *Registry {
	return &Registry{
		middleware: []MiddlewarePlugin{},
		outputs:    []OutputPlugin{},
		commands:   []CommandPlugin{},
		auths:      []AuthPlugin{},
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
	case AuthPlugin:
		r.RegisterAuth(p)
	default:
		// Unknown plugin type - ignore silently
		fmt.Fprintf(os.Stderr, "Warning: unknown plugin type %T, ignoring\n", plugin)
	}
}

// RegisterAuth registers an auth plugin.
func (r *Registry) RegisterAuth(plugin AuthPlugin) {
	r.auths = append(r.auths, plugin)
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

// Auths returns all registered auth plugins.
func (r *Registry) Auths() []AuthPlugin {
	return r.auths
}

// ListAuth returns all registered auth plugins.
func (r *Registry) ListAuth() []AuthPlugin {
	return r.auths
}

// GetAuth looks up an auth plugin by its name.
// Returns the plugin and true if found, or nil and false if not found.
func (r *Registry) GetAuth(name string) (AuthPlugin, bool) {
	for _, a := range r.auths {
		if a.Name() == name {
			return a, true
		}
	}
	return nil, false
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
			fmt.Fprintf(os.Stderr, "Warning: middleware %s panicked in BeforeRequest: %v\n%s\n", plugin.Name(), rec, debug.Stack())
			result = ctx
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
			fmt.Fprintf(os.Stderr, "Warning: middleware %s panicked in AfterResponse: %v\n%s\n", plugin.Name(), rec, debug.Stack())
			result = ctx
		}
	}()
	return plugin.AfterResponse(ctx)
}

// ApplyAuth dispatches to a named auth plugin with panic recovery.
func (r *Registry) ApplyAuth(name string, req *client.Request, params map[string]string) error {
	plugin, ok := r.GetAuth(name)
	if !ok {
		return fmt.Errorf("unknown auth plugin %q", name)
	}
	return r.safeApplyAuth(plugin, req, params)
}

func (r *Registry) safeApplyAuth(plugin AuthPlugin, req *client.Request, params map[string]string) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("auth plugin %s panicked: %v", plugin.Name(), rec)
			fmt.Fprintf(os.Stderr, "Warning: auth plugin %s panicked in Apply: %v\n%s\n", plugin.Name(), rec, debug.Stack())
		}
	}()
	return plugin.Apply(req, params)
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
