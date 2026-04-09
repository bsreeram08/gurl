package auth

import (
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
	r.handlers[h.Name()] = h
}

func (r *Registry) Get(name string) Handler {
	return r.handlers[name]
}

func (r *Registry) Apply(authType string, req *client.Request, params map[string]string) {
	handler, ok := r.handlers[authType]
	if !ok {
		return
	}
	handler.Apply(req, params)
}
