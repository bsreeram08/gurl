package auth

import (
	"fmt"

	"github.com/sreeram/gurl/internal/client"
)

type Registry struct {
	handlers map[string]Handler
}

func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[string]Handler),
	}
}

func (r *Registry) Register(h Handler) {
	if h == nil {
		return
	}
	r.handlers[h.Name()] = h
}

func (r *Registry) Get(name string) Handler {
	return r.handlers[name]
}

func (r *Registry) Apply(authType string, req *client.Request, params map[string]string) error {
	if r == nil {
		return fmt.Errorf("auth registry is nil")
	}
	handler, ok := r.handlers[authType]
	if !ok {
		return fmt.Errorf("unknown auth type %q", authType)
	}
	if err := handler.Apply(req, params); err != nil {
		return fmt.Errorf("%s auth: %w", authType, err)
	}
	return nil
}

func BuiltinRegistry() *Registry {
	registry := NewRegistry()
	registry.Register(&BasicHandler{})
	registry.Register(&BearerHandler{})
	registry.Register(&APIKeyHandler{})
	registry.Register(&OAuth1Handler{})
	registry.Register(&OAuth2Handler{})
	registry.Register(&AWSv4Handler{})
	registry.Register(&DigestHandler{})
	registry.Register(&NTLMHandler{})
	return registry
}
