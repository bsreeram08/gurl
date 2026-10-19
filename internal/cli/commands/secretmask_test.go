package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/sreeram/gurl/pkg/types"
)

func TestMaskRequestSecrets(t *testing.T) {
	tests := []struct {
		name   string
		header types.Header
		want   string
	}{
		{name: "template value untouched", header: types.Header{Key: "Authorization", Value: "Bearer {{TOKEN}}"}, want: "Bearer {{TOKEN}}"},
		{name: "bare template untouched", header: types.Header{Key: "X-Api-Key", Value: "{{API_KEY}}"}, want: "{{API_KEY}}"},
		{name: "literal authorization masked", header: types.Header{Key: "authorization", Value: "Bearer sk_live_123"}, want: secretMask},
		{name: "literal key header masked", header: types.Header{Key: "X-API-KEY", Value: "abc"}, want: secretMask},
		{name: "cookie masked", header: types.Header{Key: "Cookie", Value: "sid=1"}, want: secretMask},
		{name: "signature masked", header: types.Header{Key: "X-Webhook-Signature", Value: "deadbeef"}, want: secretMask},
		{name: "non-sensitive header untouched", header: types.Header{Key: "Content-Type", Value: "application/json"}, want: "application/json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &types.SavedRequest{Headers: []types.Header{tt.header}}
			got := maskRequestSecrets(req)
			if got.Headers[0].Value != tt.want {
				t.Errorf("got %q, want %q", got.Headers[0].Value, tt.want)
			}
			if req.Headers[0].Value != tt.header.Value {
				t.Errorf("original request mutated: %q", req.Headers[0].Value)
			}
		})
	}
}

func TestMaskRequestSecretsAuthAndCurl(t *testing.T) {
	req := &types.SavedRequest{
		CurlCmd: "curl -H 'Authorization: Bearer sk_live_123' https://api.example.com",
		Headers: []types.Header{{Key: "Authorization", Value: "Bearer sk_live_123"}},
		AuthConfig: &types.AuthConfig{Type: "basic", Params: map[string]string{
			"username": "alice",
			"password": "hunter2",
			"token":    "{{TOKEN}}",
		}},
	}
	got := maskRequestSecrets(req)

	if strings.Contains(got.CurlCmd, "sk_live_123") {
		t.Errorf("curl command still contains secret: %s", got.CurlCmd)
	}
	if got.AuthConfig.Params["password"] != secretMask {
		t.Errorf("password not masked: %q", got.AuthConfig.Params["password"])
	}
	if got.AuthConfig.Params["username"] != "alice" {
		t.Errorf("username should be untouched: %q", got.AuthConfig.Params["username"])
	}
	if got.AuthConfig.Params["token"] != "{{TOKEN}}" {
		t.Errorf("templated token should be untouched: %q", got.AuthConfig.Params["token"])
	}
	if req.AuthConfig.Params["password"] != "hunter2" {
		t.Error("original auth config mutated")
	}
	if maskRequestSecrets(nil) != nil {
		t.Error("nil request should return nil")
	}
}

func TestShowCommandMasksSecrets(t *testing.T) {
	db := newMockDB()
	db.requests["req-1"] = &types.SavedRequest{
		ID:      "req-1",
		Name:    "charge",
		URL:     "https://api.example.com/charges",
		Method:  "POST",
		Headers: []types.Header{{Key: "Authorization", Value: "Bearer sk_live_123"}},
	}
	db.names["charge"] = "req-1"

	tests := []struct {
		name       string
		args       []string
		wantSecret bool
	}{
		{name: "pretty masks", args: []string{"show", "charge"}},
		{name: "json masks", args: []string{"show", "--json", "charge"}},
		{name: "curl masks", args: []string{"show", "--format", "curl", "charge"}},
		{name: "reveal shows", args: []string{"show", "--json", "--reveal", "charge"}, wantSecret: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var runErr error
			out := captureStdout(t, func() {
				runErr = ShowCommand(db).Run(context.Background(), tt.args)
			})
			if runErr != nil {
				t.Fatalf("unexpected error: %v", runErr)
			}
			if got := strings.Contains(out, "sk_live_123"); got != tt.wantSecret {
				t.Errorf("secret present = %v, want %v; output:\n%s", got, tt.wantSecret, out)
			}
		})
	}
}
