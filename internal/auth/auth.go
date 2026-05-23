package auth

import (
	"fmt"

	"github.com/sreeram/gurl/internal/client"
)

// ParamDef describes a handler-specific auth parameter.
type ParamDef struct {
	Name        string
	Required    bool
	Secret      bool
	Description string
	Default     string
}

// Handler is the interface all auth handlers must implement.
// Each handler knows how to apply its specific auth type to a request.
type Handler interface {
	// Name returns the unique name of the auth handler (e.g., "basic", "bearer", "apikey").
	Name() string
	// Description returns a one-line summary of the auth type.
	Description() string
	// Params returns metadata for the handler's supported parameters.
	Params() []ParamDef
	// Apply applies the auth handler to the given request using params.
	// Params is a map of handler-specific parameters (e.g., {"username": "user", "password": "pass"}).
	Apply(req *client.Request, params map[string]string) error
}

func requireParam(authType string, params map[string]string, name string) (string, error) {
	value := params[name]
	if value == "" {
		return "", fmt.Errorf("%s: missing required param %q", authType, name)
	}
	return value, nil
}

func requireRequest(authType string, req *client.Request) error {
	if req == nil {
		return fmt.Errorf("%s: request is nil", authType)
	}
	return nil
}
