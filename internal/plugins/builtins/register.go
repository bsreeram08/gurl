package builtins

import "github.com/sreeram/gurl/internal/plugins"

// RegisterBuiltins registers all built-in plugins with the given registry.
// Order matters for middleware: timing first, user-agent second, retry third, logging last.
func RegisterBuiltins(registry *plugins.Registry) {
	registry.Register(&TimingMiddleware{})
	registry.Register(&UserAgentMiddleware{Version: "0.1.0"})
	registry.Register(&Retry401Middleware{})
	registry.Register(&LoggingMiddleware{})

	registry.Register(&SlackOutput{})
	registry.Register(&MarkdownOutput{})
	registry.Register(&CSVOutput{})
	registry.Register(&MinimalOutput{})
}
