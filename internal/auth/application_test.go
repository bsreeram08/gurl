package auth

import (
	"strings"
	"testing"

	"github.com/sreeram/gurl/internal/client"
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

func headerValue(headers []client.Header, key string) string {
	for _, header := range headers {
		if header.Key == key {
			return header.Value
		}
	}
	return ""
}
