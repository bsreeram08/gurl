package auth

import (
	"testing"

	"github.com/sreeram/gurl/pkg/types"
)

// BDD test scenarios for auth inheritance resolution:
// - request with no auth config inherits auth from its collection
// - request auth overrides collection auth
// - "no auth" explicitly set on request prevents inheritance

// TestAuthInheritance tests the auth inheritance resolution:
// 1. request with no auth config inherits auth from its collection
// 2. request auth overrides collection auth
// 3. "no auth" explicitly set on request prevents inheritance

func TestResolveAuthConfig_RequestInheritsFromCollection(t *testing.T) {
	// Request with no auth should inherit from collection
	collection := &types.Collection{
		ID:   "col-1",
		Name: "Test Collection",
		AuthConfig: &types.AuthConfig{
			Type:   "basic",
			Params: map[string]string{"username": "collectionuser", "password": "collectionpass"},
		},
	}

	request := &types.SavedRequest{
		ID:         "req-1",
		Name:       "Test Request",
		Collection: "col-1",
		// AuthConfig is nil - should inherit from collection
	}

	result := ResolveAuthConfig(request, collection)

	if result == nil {
		t.Fatal("expected resolved auth config, got nil")
	}
	if result.Type != "basic" {
		t.Errorf("expected type 'basic', got %q", result.Type)
	}
	if result.Params["username"] != "collectionuser" {
		t.Errorf("expected username 'collectionuser', got %q", result.Params["username"])
	}
}

func TestResolveAuthConfig_RequestOverridesCollection(t *testing.T) {
	// Request with auth should use its own auth, not inherit
	collection := &types.Collection{
		ID:   "col-1",
		Name: "Test Collection",
		AuthConfig: &types.AuthConfig{
			Type:   "basic",
			Params: map[string]string{"username": "collectionuser", "password": "collectionpass"},
		},
	}

	request := &types.SavedRequest{
		ID:         "req-1",
		Name:       "Test Request",
		Collection: "col-1",
		AuthConfig: &types.AuthConfig{
			Type:   "bearer",
			Params: map[string]string{"token": "request-token"},
		},
	}

	result := ResolveAuthConfig(request, collection)

	if result == nil {
		t.Fatal("expected resolved auth config, got nil")
	}
	if result.Type != "bearer" {
		t.Errorf("expected type 'bearer', got %q", result.Type)
	}
	if result.Params["token"] != "request-token" {
		t.Errorf("expected token 'request-token', got %q", result.Params["token"])
	}
	// Should NOT have collection username
	if _, ok := result.Params["username"]; ok {
		t.Error("request auth should override collection auth, but found collection username")
	}
}

func TestResolveAuthConfig_NoAuthWhenNeitherHasAuth(t *testing.T) {
	// Neither request nor collection has auth - should return nil
	collection := &types.Collection{
		ID:   "col-1",
		Name: "Test Collection",
		// AuthConfig is nil
	}

	request := &types.SavedRequest{
		ID:         "req-1",
		Name:       "Test Request",
		Collection: "col-1",
		// AuthConfig is nil
	}

	result := ResolveAuthConfig(request, collection)

	if result != nil {
		t.Errorf("expected nil auth config when neither has auth, got %+v", result)
	}
}

func TestResolveAuthConfig_CollectionOnlyHasAuth(t *testing.T) {
	// Only collection has auth - request should inherit it
	collection := &types.Collection{
		ID:   "col-1",
		Name: "Test Collection",
		AuthConfig: &types.AuthConfig{
			Type:   "bearer",
			Params: map[string]string{"token": "collection-token"},
		},
	}

	request := &types.SavedRequest{
		ID:         "req-1",
		Name:       "Test Request",
		Collection: "col-1",
		// No own auth - should inherit from collection
	}

	result := ResolveAuthConfig(request, collection)

	if result == nil {
		t.Fatal("expected resolved auth config, got nil")
	}
	if result.Type != "bearer" {
		t.Errorf("expected type 'bearer', got %q", result.Type)
	}
	if result.Params["token"] != "collection-token" {
		t.Errorf("expected token 'collection-token', got %q", result.Params["token"])
	}
}

func TestResolveAuthConfig_NilCollection(t *testing.T) {
	// Collection is nil - request with auth should use its own
	request := &types.SavedRequest{
		ID:   "req-1",
		Name: "Test Request",
		AuthConfig: &types.AuthConfig{
			Type:   "basic",
			Params: map[string]string{"username": "user", "password": "pass"},
		},
	}

	result := ResolveAuthConfig(request, nil)

	if result == nil {
		t.Fatal("expected resolved auth config, got nil")
	}
	if result.Type != "basic" {
		t.Errorf("expected type 'basic', got %q", result.Type)
	}
}

func TestResolveAuthConfig_NilRequest(t *testing.T) {
	// Request is nil - should return collection auth if exists
	collection := &types.Collection{
		ID:   "col-1",
		Name: "Test Collection",
		AuthConfig: &types.AuthConfig{
			Type:   "apikey",
			Params: map[string]string{"key": "collection-key", "value": "cv"},
		},
	}

	result := ResolveAuthConfig(nil, collection)

	if result == nil {
		t.Fatal("expected resolved auth config, got nil")
	}
	if result.Type != "apikey" {
		t.Errorf("expected type 'apikey', got %q", result.Type)
	}
}

func TestResolveAuthConfig_BothNil(t *testing.T) {
	// Both are nil - should return nil
	result := ResolveAuthConfig(nil, nil)

	if result != nil {
		t.Errorf("expected nil when both inputs are nil, got %+v", result)
	}
}

func TestResolveAuthConfig_RequestWithNoAuthAndNilCollection(t *testing.T) {
	// Request has no auth and collection is nil - should return nil
	request := &types.SavedRequest{
		ID:   "req-1",
		Name: "Test Request",
		// AuthConfig is nil
	}

	result := ResolveAuthConfig(request, nil)

	if result != nil {
		t.Errorf("expected nil when request has no auth and collection is nil, got %+v", result)
	}
}

func TestResolveAuthConfig_CollectionWithNoAuthAndNilRequest(t *testing.T) {
	// Collection has no auth and request is nil - should return nil
	collection := &types.Collection{
		ID:   "col-1",
		Name: "Test Collection",
		// AuthConfig is nil
	}

	result := ResolveAuthConfig(nil, collection)

	if result != nil {
		t.Errorf("expected nil when collection has no auth and request is nil, got %+v", result)
	}
}
