package redact

import (
	"strings"
	"testing"

	"github.com/sreeram/gurl/internal/resolve"
	"github.com/sreeram/gurl/pkg/types"
)

func TestSavedRequest(t *testing.T) {
	tests := []struct {
		name   string
		header types.Header
		want   string
	}{
		{name: "template value untouched", header: types.Header{Key: "Authorization", Value: "Bearer {{TOKEN}}"}, want: "Bearer {{TOKEN}}"},
		{name: "bare template untouched", header: types.Header{Key: "X-Api-Key", Value: "{{API_KEY}}"}, want: "{{API_KEY}}"},
		{name: "literal authorization masked", header: types.Header{Key: "authorization", Value: "Bearer sk_live_123"}, want: Mask},
		{name: "literal key header masked", header: types.Header{Key: "X-API-KEY", Value: "abc"}, want: Mask},
		{name: "cookie masked", header: types.Header{Key: "Cookie", Value: "sid=1"}, want: Mask},
		{name: "signature masked", header: types.Header{Key: "X-Webhook-Signature", Value: "deadbeef"}, want: Mask},
		{name: "non-sensitive header untouched", header: types.Header{Key: "Content-Type", Value: "application/json"}, want: "application/json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &types.SavedRequest{Headers: []types.Header{tt.header}}
			got := SavedRequest(req)
			if got.Headers[0].Value != tt.want {
				t.Errorf("got %q, want %q", got.Headers[0].Value, tt.want)
			}
			if req.Headers[0].Value != tt.header.Value {
				t.Errorf("original request mutated: %q", req.Headers[0].Value)
			}
		})
	}
}

func TestSavedRequestAuthAndCurl(t *testing.T) {
	req := &types.SavedRequest{
		CurlCmd: "curl -H 'Authorization: Bearer sk_live_123' https://api.example.com",
		Headers: []types.Header{{Key: "Authorization", Value: "Bearer sk_live_123"}},
		AuthConfig: &types.AuthConfig{Type: "basic", Params: map[string]string{
			"username": "alice",
			"password": "hunter2",
			"token":    "{{TOKEN}}",
		}},
	}
	got := SavedRequest(req)

	if strings.Contains(got.CurlCmd, "sk_live_123") {
		t.Errorf("curl command still contains secret: %s", got.CurlCmd)
	}
	if got.AuthConfig.Params["password"] != Mask {
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
	if SavedRequest(nil) != nil {
		t.Error("nil request should return nil")
	}
}

func TestSavedRequestCoversURLBodyAndVars(t *testing.T) {
	req := &types.SavedRequest{
		URL:        "https://api.example.com/x?api_key=sk_live_q&page=2&token={{TOKEN}}",
		Headers:    []types.Header{{Key: "X-Api-Key", Value: "sk_live_h"}},
		Body:       `{"key":"sk_live_h"}`,
		Variables:  []types.Var{{Name: "secret", Example: "sk_live_v"}, {Name: "id", Example: "42"}},
		PathParams: []types.Var{{Name: "token", Example: "sk_live_p"}},
	}
	got := SavedRequest(req)
	for _, s := range []string{got.URL, got.Body, got.Variables[0].Example, got.PathParams[0].Example} {
		if strings.Contains(s, "sk_live") {
			t.Fatalf("secret leaked: %q", s)
		}
	}
	if !strings.Contains(got.URL, "page=2") || !strings.Contains(got.URL, "token={{TOKEN}}") || got.Variables[1].Example != "42" {
		t.Fatalf("non-secret data changed: %+v", got)
	}
	if !strings.Contains(req.URL, "sk_live_q") {
		t.Fatal("original request was modified")
	}
}

func TestSensitiveNameAndLiteral(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"Authorization", true},
		{"Proxy-Authorization", true},
		{"Set-Cookie", true},
		{"X-Api-Key", true},
		{"client_secret", true},
		{"refresh_token", true},
		{"PASSWORD", true},
		{"X-Webhook-Signature", true},
		{"Content-Type", false},
		{"page", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SensitiveName(tt.name); got != tt.want {
				t.Errorf("SensitiveName(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
	if Literal("") || Literal("Bearer {{TOKEN}}") || !Literal("sk_live") {
		t.Error("Literal: empty and templated values are not literal secrets")
	}
}

func TestReplaceMasksLongestFirst(t *testing.T) {
	got := Replace("a=abcd1234 b=abcd", []string{"abcd", "abcd1234", ""})
	if got != "a=***** b=*****" {
		t.Fatalf("got %q", got)
	}
}

func TestReplaceSkipsVeryShortSecrets(t *testing.T) {
	if got := Replace("status ok", []string{"t"}); got != "status ok" {
		t.Fatalf("short secret garbled text: %q", got)
	}
}

func TestScopeSecrets(t *testing.T) {
	s := resolve.Scopes{
		Env:               map[string]string{"TOKEN": "env-tok", "HOST": "h", "SA_KEY": "sa"},
		EnvSecrets:        map[string]bool{"TOKEN": true},
		Collection:        map[string]string{"PASS": "coll-pass"},
		CollectionSecrets: map[string]bool{"PASS": true},
		CLI:               map[string]string{"TOKEN": "cli-tok"},
	}
	got := strings.Join(ScopeSecrets(s, map[string]string{"session_token": "upd", "id": "42"}), ",")
	for _, want := range []string{"env-tok", "cli-tok", "sa", "coll-pass", "upd"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
	for _, unwanted := range []string{",h,", "42"} {
		if strings.Contains(","+got+",", unwanted) {
			t.Errorf("unexpected %q in %q", unwanted, got)
		}
	}
}

func TestExecutionMasksRequestVarsAndResponseHeaders(t *testing.T) {
	rec := &types.ExecutionHistory{
		URL:            "https://api.example.com/x?api_key=q-secret&k=env-secret&page=2",
		RequestHeaders: []types.Header{{Key: "Authorization", Value: "Bearer literal"}, {Key: "X-Trace", Value: "t-env-secret"}},
		RequestBody:    `{"pw":"env-secret"}`,
		Vars: []types.VarRecord{
			{Name: "SECRET_ONE", Origin: "environment", Value: "env-secret", Secret: true},
			{Name: "alias", Origin: "environment", Value: "env-secret"},
			{Name: "page", Origin: "cli", Value: "2"},
		},
		Response:        `{"echo":"env-secret"}`,
		ResponseHeaders: map[string][]string{"Set-Cookie": {"sid=1", "x=2"}, "Content-Type": {"application/json"}},
	}
	Execution(rec, []string{"cli-only"})

	all := rec.URL + rec.RequestBody + rec.RequestHeaders[0].Value + rec.RequestHeaders[1].Value + rec.Vars[0].Value + rec.Vars[1].Value
	for _, leak := range []string{"q-secret", "env-secret", "literal"} {
		if strings.Contains(all, leak) {
			t.Errorf("%q leaked: %+v", leak, rec)
		}
	}
	if !strings.Contains(rec.URL, "page=2") || rec.Vars[2].Value != "2" {
		t.Errorf("non-secret data changed: %+v", rec)
	}
	if got := rec.ResponseHeaders["Set-Cookie"]; len(got) != 2 || got[0] != Mask {
		t.Errorf("Set-Cookie not masked: %v", got)
	}
	if rec.ResponseHeaders["Content-Type"][0] != "application/json" {
		t.Errorf("Content-Type changed: %v", rec.ResponseHeaders)
	}
	if rec.Response != `{"echo":"env-secret"}` {
		t.Errorf("response body must be stored as received, got %q", rec.Response)
	}
}

func TestReplaceMasksEncodedSecrets(t *testing.T) {
	secret := `sk+live/a b"c`
	in := "q=sk%2Blive%2Fa+b%22c path=sk+live%2Fa%20b%22c json=sk+live/a b\"c esc=sk+live/a b\\\"c"
	got := Replace(in, []string{secret})
	for _, leak := range []string{"sk%2Blive", "sk+live", "live/a"} {
		if strings.Contains(got, leak) {
			t.Fatalf("encoded secret leaked (%q): %s", leak, got)
		}
	}
}
