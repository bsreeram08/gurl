package auth

import (
	"github.com/sreeram/gurl/internal/client"
)

// Handler is the interface all auth handlers must implement.
// Each handler knows how to apply its specific auth type to a request.
type Handler interface {
	// Name returns the unique name of the auth handler (e.g., "basic", "bearer", "apikey").
	Name() string
	// Apply applies the auth handler to the given request using params.
	// Params is a map of handler-specific parameters (e.g., {"username": "user", "password": "pass"}).
	Apply(req *client.Request, params map[string]string)
}
