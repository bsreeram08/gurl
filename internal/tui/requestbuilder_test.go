package tui

import (
	"strings"
	"testing"

	"github.com/sreeram/gurl/pkg/types"
)

func TestRequestBuilderCollectVarAwareRequestAppliesAPIKeyQueryAuth(t *testing.T) {
	rb := NewRequestBuilder(nil)
	rb.LoadRequest(&types.SavedRequest{
		Name:   "query-auth",
		Method: "GET",
		URL:    "https://api.example.com/widgets",
		AuthConfig: &types.AuthConfig{
			Type: "apikey",
			Params: map[string]string{
				"in":         "query",
				"key":        "{{api_key}}",
				"param_name": "access_token",
			},
		},
	})

	clientReq, err := rb.collectVarAwareRequest(map[string]string{"api_key": "secret value"})
	if err != nil {
		t.Fatalf("collectVarAwareRequest returned error: %v", err)
	}

	if clientReq.URL != "https://api.example.com/widgets?access_token=secret+value" {
		t.Fatalf("unexpected URL %q", clientReq.URL)
	}
	for _, h := range clientReq.Headers {
		if h.Key == "X-API-Key" {
			t.Fatalf("did not expect legacy header auth for query API key, got %#v", clientReq.Headers)
		}
	}
}

func TestRequestBuilderCollectVarAwareRequestUnknownAuthReturnsError(t *testing.T) {
	rb := NewRequestBuilder(nil)
	rb.LoadRequest(&types.SavedRequest{
		Name:   "unknown-auth",
		Method: "GET",
		URL:    "https://api.example.com/widgets",
		AuthConfig: &types.AuthConfig{
			Type:   "made-up",
			Params: map[string]string{"token": "abc123"},
		},
	})

	_, err := rb.collectVarAwareRequest(nil)
	if err == nil {
		t.Fatal("expected unknown auth type error")
	}
	if !strings.Contains(err.Error(), `unknown auth type "made-up"`) {
		t.Fatalf("expected unknown auth type error, got %v", err)
	}
}
