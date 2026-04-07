package auth

import (
	"github.com/sreeram/gurl/internal/client"
)

// Registry manages auth handlers and provides dispatch for applying auth to requests.
type Registry struct {
	handlers map[string]Handler
}

// NewRegistry creates a new Registry with no handlers registered.
func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[string]Handler),
	}
}

// Register registers an auth handler with the registry.
// The handler's Name() is used as the key.
func (r *Registry) Register(h Handler) {
	r.handlers[h.Name()] = h
}

// Get returns the handler with the given name, or nil if not found.
func (r *Registry) Get(name string) Handler {
	return r.handlers[name]
}

// Apply looks up the handler for authType and applies it to the request.
// If no handler is registered for authType, this is a no-op.
func (r *Registry) Apply(authType string, req *client.Request, params map[string]string) {
	handler, ok := r.handlers[authType]
	if !ok {
		return
	}
	handler.Apply(req, params)
}
