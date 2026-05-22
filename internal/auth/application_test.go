package auth

import (
	"strings"
	"testing"

	"github.com/sreeram/gurl/internal/client"
	"github.com/sreeram/gurl/pkg/types"
)

func TestBuiltinRegistryRegistersAllHandlersWithParams(t *testing.T) {
	registry := BuiltinRegistry()

	expected := map[string][]string{
		"basic":  {"username", "password", "charset"},
		"bearer": {"token"},
		"apikey": {"header", "value", "in", "key", "header_name", "param_name"},
		"oauth1": {"consumer_key", "consumer_secret", "token", "token_secret"},
		"oauth2": {"client_id", "client_secret", "token_url", "flow", "auth_code", "redirect_uri", "registered_redirect_uri", "scope"},
		"awsv4":  {"access_key", "secret_key", "region", "service", "session_token"},
		"digest": {"username", "password", "realm", "nonce", "qop", "opaque", "algorithm", "client_qop"},
		"ntlm":   {"username", "password", "domain", "workstation", "challenge"},
	}

	for authType, expectedParamNames := range expected {
		t.Run(authType, func(t *testing.T) {
			handler := registry.Get(authType)
			if handler == nil {
				t.Fatalf("expected %q handler to be registered", authType)
			}

			params := handler.Params()
			if len(params) == 0 {
				t.Fatalf("expected %q handler to expose parameter metadata", authType)
			}

			byName := make(map[string]ParamDef, len(params))
			for _, param := range params {
				byName[param.Name] = param
				if param.Description == "" {
					t.Errorf("param %q should have a description", param.Name)
				}
			}

			for _, name := range expectedParamNames {
				if _, ok := byName[name]; !ok {
					t.Errorf("expected %q metadata to include param %q", authType, name)
				}
			}
		})
	}
}

func TestRegistryApplyErrorsForUnknownTypeAndPropagatesHandlerErrors(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&BearerHandler{})

	req := &client.Request{Method: "GET", URL: "https://example.com"}
	if err := registry.Apply("unknown", req, map[string]string{}); err == nil {
		t.Fatal("expected unknown auth type to return an error")
	}

	err := registry.Apply("bearer", req, map[string]string{})
	if err == nil {
		t.Fatal("expected bearer handler error for missing token")
	}
	if !strings.Contains(err.Error(), "bearer") || !strings.Contains(err.Error(), "token") {
		t.Fatalf("expected error to include auth type and missing param, got %q", err.Error())
	}
}

func TestApplyAuthNilSafeAndTemplateSubstitutesParams(t *testing.T) {
	if err := ApplyAuth(nil, nil, &client.Request{}, nil); err != nil {
		t.Fatalf("nil auth config should be a no-op, got %v", err)
	}

	config := &types.AuthConfig{Type: "basic", Params: map[string]string{
		"username": "api-{{user}}",
		"password": "{{password}}",
	}}

	if err := ApplyAuth(nil, config, &client.Request{}, nil); err == nil {
		t.Fatal("nil registry should error when auth config requires auth")
	}

	req := &client.Request{Method: "GET", URL: "https://example.com"}
	err := ApplyAuth(BuiltinRegistry(), config, req, map[string]string{
		"user":     "admin",
		"password": "secret",
	})
	if err != nil {
		t.Fatalf("ApplyAuth returned error: %v", err)
	}

	got := headerValue(req.Headers, "Authorization")
	if got == "" {
		t.Fatal("expected Authorization header to be set")
	}
	if !strings.HasPrefix(got, "Basic ") {
		t.Fatalf("expected Basic authorization header, got %q", got)
	}

	missingReq := &client.Request{Method: "GET", URL: "https://example.com"}
	missingErr := ApplyAuth(BuiltinRegistry(), config, missingReq, map[string]string{"user": "admin"})
	if missingErr == nil {
		t.Fatal("expected missing template variable to return an error")
	}
	if !strings.Contains(missingErr.Error(), "password") {
		t.Fatalf("expected missing template variable in error, got %q", missingErr.Error())
	}
}

func TestApplyAuthSupportsSavedAPIKeyHeaderValueParams(t *testing.T) {
	req := &client.Request{Method: "GET", URL: "https://example.com"}
	config := &types.AuthConfig{Type: "apikey", Params: map[string]string{
		"header": "X-API-Key",
		"value":  "key-{{env}}",
	}}

	err := ApplyAuth(BuiltinRegistry(), config, req, map[string]string{"env": "dev"})
	if err != nil {
		t.Fatalf("ApplyAuth returned error: %v", err)
	}

	if got := headerValue(req.Headers, "X-API-Key"); got != "key-dev" {
		t.Fatalf("expected saved API key header params to set X-API-Key=key-dev, got %q", got)
	}
}

func headerValue(headers []client.Header, key string) string {
	for _, header := range headers {
		if header.Key == key {
			return header.Value
		}
	}
	return ""
}
