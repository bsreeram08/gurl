package auth

import (
	"github.com/sreeram/gurl/pkg/types"
)

// ResolveAuthConfig resolves the effective auth config for a request using inheritance.
// Resolution order: request.AuthConfig > collection.AuthConfig > nil
// Uses slice iteration for clean nil-safe resolution (no if-else chains).
func ResolveAuthConfig(request *types.SavedRequest, collection *types.Collection) *types.AuthConfig {
	// Build precedence list: request first, then collection
	precedence := make([]*types.AuthConfig, 0, 2)
	if request != nil {
		precedence = append(precedence, request.AuthConfig)
	}
	if collection != nil {
		precedence = append(precedence, collection.AuthConfig)
	}

	// Return first non-nil auth config
	for _, cfg := range precedence {
		if cfg != nil {
			return cfg
		}
	}

	return nil
}
