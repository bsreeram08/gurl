package resolve

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/sreeram/gurl/internal/auth"
	"github.com/sreeram/gurl/internal/client"
	"github.com/sreeram/gurl/pkg/types"
)

func TestVarsPrecedenceAndOrigins(t *testing.T) {
	tests := []struct {
		name       string
		scopes     Scopes
		wantValue  string
		wantOrigin Origin
	}{
		{"environment only", Scopes{Env: map[string]string{"k": "env"}}, "env", OriginEnvironment},
		{"collection over environment", Scopes{Env: map[string]string{"k": "env"}, Collection: map[string]string{"k": "col"}}, "col", OriginCollection},
		{"data over collection", Scopes{Collection: map[string]string{"k": "col"}, Data: map[string]string{"k": "row"}}, "row", OriginData},
		{"data over environment", Scopes{Env: map[string]string{"k": "env"}, Data: map[string]string{"k": "row"}}, "row", OriginData},
		{"cli over data", Scopes{Data: map[string]string{"k": "row"}, CLI: map[string]string{"k": "cli"}}, "cli", OriginCLI},
		{"cli over collection", Scopes{Collection: map[string]string{"k": "col"}, CLI: map[string]string{"k": "cli"}}, "cli", OriginCLI},
		{"cli over environment", Scopes{Env: map[string]string{"k": "env"}, CLI: map[string]string{"k": "cli"}}, "cli", OriginCLI},
		{
			"runtime over cli keeps its own origin",
			Scopes{CLI: map[string]string{"k": "cli"}, Runtime: map[string]Var{"k": {Value: "rt", Origin: OriginCollection}}},
			"rt", OriginCollection,
		},
		{
			"all scopes",
			Scopes{
				Env:        map[string]string{"k": "env"},
				Collection: map[string]string{"k": "col"},
				Data:       map[string]string{"k": "row"},
				CLI:        map[string]string{"k": "cli"},
				Runtime:    map[string]Var{"k": {Value: "rt", Origin: OriginEnvironment}},
			},
			"rt", OriginEnvironment,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Vars(tt.scopes)["k"]
			if got.Value != tt.wantValue || got.Origin != tt.wantOrigin {
				t.Fatalf("got %+v, want value %q origin %q", got, tt.wantValue, tt.wantOrigin)
			}
		})
	}
}

func TestVarsSecretFlags(t *testing.T) {
	s := Scopes{
		Env:               map[string]string{"envSecret": "a", "shadowed": "b", "plain": "c"},
		EnvSecrets:        map[string]bool{"envSecret": true, "shadowed": true, "rtEnv": true},
		Collection:        map[string]string{"colSecret": "d", "shadowed": "e"},
		CollectionSecrets: map[string]bool{"colSecret": true},
		CLI:               map[string]string{"envSecretCLI": "f"},
		Runtime: map[string]Var{
			"rtEnv":      {Value: "g", Origin: OriginEnvironment},
			"rtExplicit": {Value: "h", Origin: OriginCLI, Secret: true},
		},
	}
	want := map[string]bool{
		"envSecret":    true,
		"shadowed":     false, // the winning collection value is not secret
		"plain":        false,
		"colSecret":    true,
		"envSecretCLI": false,
		"rtEnv":        true, // runtime value of an environment secret key
		"rtExplicit":   true,
	}
	vars := Vars(s)
	for key, secret := range want {
		if vars[key].Secret != secret {
			t.Errorf("%s: Secret = %v, want %v", key, vars[key].Secret, secret)
		}
	}
}

func TestVarsCollectionReferencesEnvironment(t *testing.T) {
	s := Scopes{
		Env: map[string]string{"host": "api.example.com", "cliOnly": "env"},
		Collection: map[string]string{
			"base":    "https://{{host}}/v1",
			"unknown": "{{nope}}",
			"typed":   "{{host:num}}",
			"cliRef":  "{{cliOnly}}",
		},
		CLI: map[string]string{"cliOnly": "cli"},
	}
	vars := Vars(s)
	want := map[string]string{
		"base":    "https://api.example.com/v1",
		"unknown": "{{nope}}",     // left for request resolution to report
		"typed":   "{{host:num}}", // only plain {{name}} references are resolved
		"cliRef":  "env",          // resolved against the environment, not higher scopes
	}
	for key, value := range want {
		if vars[key].Value != value {
			t.Errorf("%s = %q, want %q", key, vars[key].Value, value)
		}
	}
}

func TestResolveFillsEveryField(t *testing.T) {
	req := &types.SavedRequest{
		Method:  "POST",
		URL:     "{{base}}/orders/{{id}}",
		Headers: []types.Header{{Key: "X-{{region}}", Value: "{{tenant}}"}},
		Body:    `{"amount":"{{amount:num}}","live":"{{live:bool}}","meta":{{meta:raw}},"ref":"{{ref}}"}`,
	}
	s := Scopes{
		Env: map[string]string{"base": "https://api.example.com", "region": "eu"},
		CLI: map[string]string{"id": "7", "tenant": "acme", "amount": " 12.5 ", "live": "true", "meta": `{"a":1}`, "ref": "r1"},
	}
	got, problems := Resolve(req, s, Options{})
	if problems != nil {
		t.Fatalf("unexpected problems: %v", problems)
	}
	want := client.Request{
		Method:  "POST",
		URL:     "https://api.example.com/orders/7",
		Headers: []client.Header{{Key: "X-eu", Value: "acme"}},
		Body:    `{"amount":12.5,"live":true,"meta":{"a":1},"ref":"r1"}`,
	}
	if !reflect.DeepEqual(got.Request, want) {
		t.Fatalf("got %+v\nwant %+v", got.Request, want)
	}
	if got.Vars["tenant"].Origin != OriginCLI || got.Vars["base"].Origin != OriginEnvironment {
		t.Fatalf("unexpected Vars: %+v", got.Vars)
	}
}

func TestResolveDefaultsMethodToGET(t *testing.T) {
	got, problems := Resolve(&types.SavedRequest{URL: "https://x"}, Scopes{}, Options{})
	if problems != nil || got.Request.Method != "GET" {
		t.Fatalf("got method %q problems %v", got.Request.Method, problems)
	}
}

func TestResolveAliasedEnvironment(t *testing.T) {
	// Callers pass the environment alias-resolved; the aliased value is used
	// as is, and an unresolved alias is a leftover placeholder.
	req := &types.SavedRequest{URL: "https://x", Headers: []types.Header{{Key: "X-Key", Value: "{{API_KEY}}"}}}
	got, problems := Resolve(req, Scopes{Env: map[string]string{"API_KEY": "sa-123", "SA_KEY": "sa-123"}}, Options{})
	if problems != nil || got.Request.Headers[0].Value != "sa-123" {
		t.Fatalf("got %+v problems %v", got.Request.Headers, problems)
	}

	_, problems = Resolve(req, Scopes{Env: map[string]string{"API_KEY": "{{MISSING_KEY}}"}}, Options{})
	if len(problems) != 1 || problems[0].Kind != ProblemLeftoverPlaceholder || problems[0].Location != "header X-Key" {
		t.Fatalf("expected one leftover problem in header X-Key, got %#v", problems)
	}
}

func TestResolveDynamicFunctions(t *testing.T) {
	req := &types.SavedRequest{
		URL:     "https://x/orders/{{$uuid}}",
		Headers: []types.Header{{Key: "Idempotency-Key", Value: "{{$randomString(12)}}"}},
		Body:    `{"ts":"{{$timestamp}}","ref":"{{ref}}"}`,
	}
	s := Scopes{CLI: map[string]string{"ref": "r1"}}

	got, problems := Resolve(req, s, Options{})
	if problems != nil {
		t.Fatalf("unexpected problems: %v", problems)
	}
	if !regexp.MustCompile(`^https://x/orders/[0-9a-f-]{36}$`).MatchString(got.Request.URL) {
		t.Fatalf("uuid not resolved: %q", got.Request.URL)
	}
	if len(got.Request.Headers[0].Value) != 12 || strings.Contains(got.Request.Body, "{{") || !strings.Contains(got.Request.Body, `"ref":"r1"`) {
		t.Fatalf("dynamic functions not resolved: %+v", got.Request)
	}

	preview, problems := Resolve(req, s, Options{Preview: true})
	if problems != nil {
		t.Fatalf("preview should not report dynamic functions: %v", problems)
	}
	if preview.Request.URL != req.URL || preview.Request.Headers[0].Value != "{{$randomString(12)}}" || preview.Request.Body != `{"ts":"{{$timestamp}}","ref":"r1"}` {
		t.Fatalf("preview should leave dynamic functions in place: %+v", preview.Request)
	}
}

func TestResolveProblems(t *testing.T) {
	tests := []struct {
		name   string
		req    *types.SavedRequest
		scopes Scopes
		opts   Options
		want   Problems
	}{
		{
			name: "missing in every location",
			req: &types.SavedRequest{
				URL:        "https://x/{{a}}",
				Headers:    []types.Header{{Key: "X-{{b}}", Value: "{{c}}"}},
				Body:       "{{d}}",
				AuthConfig: &types.AuthConfig{Type: "bearer", Params: map[string]string{"token": "{{e}}"}},
			},
			want: Problems{
				{Kind: ProblemMissing, Variable: "a", Location: "url", Message: `missing required variable "a"`},
				{Kind: ProblemMissing, Variable: "b", Location: "header X-{{b}}", Message: `missing required variable "b"`},
				{Kind: ProblemMissing, Variable: "c", Location: "header X-{{b}}", Message: `missing required variable "c"`},
				{Kind: ProblemMissing, Variable: "d", Location: "body", Message: `missing required variable "d"`},
				{Kind: ProblemMissing, Variable: "e", Location: "auth", Message: `missing required variable "e"`},
			},
		},
		{
			name:   "leftover placeholder",
			req:    &types.SavedRequest{URL: "https://x/{{order}}"},
			scopes: Scopes{Runtime: map[string]Var{"order": {Value: "{{API_KEY}}", Origin: OriginEnvironment}}},
			want: Problems{
				{Kind: ProblemLeftoverPlaceholder, Variable: "order", Location: "url", Message: `variable "order" contains unresolved placeholder {{API_KEY}}`},
			},
		},
		{
			name:   "bad types",
			req:    &types.SavedRequest{URL: "https://x", Body: `{"n":"{{n:num}}","b":{{b:bool}}}`},
			scopes: Scopes{CLI: map[string]string{"n": "ten", "b": "yes"}},
			want: Problems{
				{Kind: ProblemBadType, Variable: "n", Location: "body", Message: `variable "n" must be a number`},
				{Kind: ProblemBadType, Variable: "b", Location: "body", Message: `variable "b" must be true or false`},
			},
		},
		{
			name: "unknown dynamic function",
			req:  &types.SavedRequest{URL: "https://x/{{$nope}}"},
			want: Problems{
				{Kind: ProblemUnknownDynamic, Variable: "$nope", Location: "url", Message: "unknown dynamic function: $nope"},
			},
		},
		{
			name: "unknown auth type is reported alongside other problems",
			req: &types.SavedRequest{
				URL:        "https://x/{{a}}",
				AuthConfig: &types.AuthConfig{Type: "made-up"},
			},
			want: Problems{
				{Kind: ProblemMissing, Variable: "a", Location: "url", Message: `missing required variable "a"`},
				{Kind: ProblemAuth, Location: "auth", Message: `unknown auth type "made-up"`},
			},
		},
		{
			name: "auth handler error",
			req:  &types.SavedRequest{URL: "https://x", AuthConfig: &types.AuthConfig{Type: "bearer", Params: map[string]string{}}},
			want: Problems{
				{Kind: ProblemAuth, Location: "auth", Message: `bearer auth: bearer: missing required param "token"`},
			},
		},
		{
			name: "preview checks the auth type",
			req:  &types.SavedRequest{URL: "https://x", AuthConfig: &types.AuthConfig{Type: "made-up"}},
			opts: Options{Preview: true},
			want: Problems{
				{Kind: ProblemAuth, Location: "auth", Message: `unknown auth type "made-up"`},
			},
		},
		{
			name: "preview does not apply auth",
			req:  &types.SavedRequest{URL: "https://x", AuthConfig: &types.AuthConfig{Type: "bearer", Params: map[string]string{}}},
			opts: Options{Preview: true},
		},
		{
			name: "duplicates in one location are reported once",
			req:  &types.SavedRequest{URL: "https://x/{{a}}/{{a}}", Body: "{{a}}"},
			want: Problems{
				{Kind: ProblemMissing, Variable: "a", Location: "url", Message: `missing required variable "a"`},
				{Kind: ProblemMissing, Variable: "a", Location: "body", Message: `missing required variable "a"`},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, got := Resolve(tt.req, tt.scopes, tt.opts)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got  %#v\nwant %#v", got, tt.want)
			}
		})
	}
}

func TestProblemsError(t *testing.T) {
	req := &types.SavedRequest{
		URL:  "https://x/{{a}}/{{order}}",
		Body: `{{b}} {{a}} {{n:num}}`,
	}
	s := Scopes{CLI: map[string]string{"order": "{{inner}}", "n": "ten"}}
	_, problems := Resolve(req, s, Options{})
	want := `missing required variables: a, b; url: variable "order" contains unresolved placeholder {{inner}}; body: variable "n" must be a number`
	if problems.Error() != want {
		t.Fatalf("got  %q\nwant %q", problems.Error(), want)
	}
	if !reflect.DeepEqual(problems.Missing(), []string{"a", "b"}) {
		t.Fatalf("Missing() = %v", problems.Missing())
	}
	if len(problems.Others()) != 2 {
		t.Fatalf("Others() = %v", problems.Others())
	}
	if Problems(nil).Err() != nil {
		t.Fatal("Err() of no problems must be nil")
	}
}

func TestProblemsNeverContainValues(t *testing.T) {
	const secret = "sk_live_TOPSECRET"
	req := &types.SavedRequest{
		URL:     "https://x/{{n:num}}",
		Headers: []types.Header{{Key: "Authorization", Value: "{{flag:bool}}"}},
		Body:    "{{missing}}",
	}
	s := Scopes{
		Env:        map[string]string{"n": secret, "flag": secret},
		EnvSecrets: map[string]bool{"n": true, "flag": true},
	}
	_, problems := Resolve(req, s, Options{})
	if len(problems) != 3 {
		t.Fatalf("expected 3 problems, got %v", problems)
	}
	if strings.Contains(problems.Error(), secret) {
		t.Fatalf("problem message leaked a value: %s", problems.Error())
	}
	for _, p := range problems {
		if strings.Contains(p.Message, secret) {
			t.Fatalf("problem leaked a value: %+v", p)
		}
	}
}

func TestResolveRequestIsBestEffortWithProblems(t *testing.T) {
	req := &types.SavedRequest{URL: "https://{{host}}/{{missing}}"}
	got, problems := Resolve(req, Scopes{Env: map[string]string{"host": "h"}}, Options{})
	if len(problems) != 1 || got.Request.URL != "https://h/{{missing}}" {
		t.Fatalf("got URL %q problems %v", got.Request.URL, problems)
	}
}

func TestResolveAppliesAuthFromInjectedRegistry(t *testing.T) {
	registry := auth.NewRegistry()
	registry.Register(stubHandler{})
	req := &types.SavedRequest{
		URL:        "https://x",
		AuthConfig: &types.AuthConfig{Type: "stub", Params: map[string]string{"token": "tok-{{env}}"}},
	}
	got, problems := Resolve(req, Scopes{Env: map[string]string{"env": "dev"}}, Options{Auth: registry})
	if problems != nil {
		t.Fatalf("unexpected problems: %v", problems)
	}
	if len(got.Request.Headers) != 1 || got.Request.Headers[0] != (client.Header{Key: "X-Stub", Value: "tok-dev"}) {
		t.Fatalf("stub auth not applied: %+v", got.Request.Headers)
	}

	// The builtin registry does not know "stub".
	if _, problems := Resolve(req, Scopes{Env: map[string]string{"env": "dev"}}, Options{}); len(problems) != 1 || problems[0].Kind != ProblemAuth {
		t.Fatalf("expected unknown auth type with the builtin registry, got %v", problems)
	}
}

func TestResolveBuiltinAuth(t *testing.T) {
	tests := []struct {
		name       string
		cfg        *types.AuthConfig
		vars       map[string]string
		wantHeader client.Header
		wantURL    string
	}{
		{
			name:       "bearer after templates",
			cfg:        &types.AuthConfig{Type: "bearer", Params: map[string]string{"token": "{{token}}"}},
			vars:       map[string]string{"token": "abc"},
			wantHeader: client.Header{Key: "Authorization", Value: "Bearer abc"},
			wantURL:    "https://x",
		},
		{
			name:       "api key header",
			cfg:        &types.AuthConfig{Type: "apikey", Params: map[string]string{"header": "X-API-Key", "value": "key-{{env}}"}},
			vars:       map[string]string{"env": "dev"},
			wantHeader: client.Header{Key: "X-API-Key", Value: "key-dev"},
			wantURL:    "https://x",
		},
		{
			name:    "api key query",
			cfg:     &types.AuthConfig{Type: "apikey", Params: map[string]string{"in": "query", "key": "{{api_key}}", "param_name": "access_token"}},
			vars:    map[string]string{"api_key": "secret value"},
			wantURL: "https://x?access_token=secret+value",
		},
		{
			name:    "none is a no-op",
			cfg:     &types.AuthConfig{Type: "none"},
			wantURL: "https://x",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &types.SavedRequest{URL: "https://x", AuthConfig: tt.cfg}
			got, problems := Resolve(req, Scopes{CLI: tt.vars}, Options{})
			if problems != nil {
				t.Fatalf("unexpected problems: %v", problems)
			}
			if got.Request.URL != tt.wantURL {
				t.Fatalf("URL = %q, want %q", got.Request.URL, tt.wantURL)
			}
			if tt.wantHeader.Key == "" {
				if len(got.Request.Headers) != 0 {
					t.Fatalf("unexpected headers %+v", got.Request.Headers)
				}
				return
			}
			for _, h := range got.Request.Headers {
				if h == tt.wantHeader {
					return
				}
			}
			t.Fatalf("missing header %+v in %+v", tt.wantHeader, got.Request.Headers)
		})
	}
}

func TestResolveInteractive(t *testing.T) {
	tests := []struct {
		name        string
		req         *types.SavedRequest
		scopes      Scopes
		wantURL     string
		wantMissing []string
	}{
		{
			name:    "path params from scopes, escaped",
			req:     &types.SavedRequest{URL: "https://x/users/:id/files/{file}", PathParams: []types.Var{{Name: "id"}}},
			scopes:  Scopes{CLI: map[string]string{"id": "42", "file": "a b"}},
			wantURL: "https://x/users/42/files/a%20b",
		},
		{
			name:    "placeholders are not path params",
			req:     &types.SavedRequest{URL: "{{base}}/users/:id?amount={{amount:num}}"},
			scopes:  Scopes{CLI: map[string]string{"base": "https://x", "id": "1", "amount": "5"}},
			wantURL: "https://x/users/1?amount=5",
		},
		{
			name:        "request path params not in the URL are still required",
			req:         &types.SavedRequest{URL: "https://x/:id", PathParams: []types.Var{{Name: "path_id"}}},
			scopes:      Scopes{CLI: map[string]string{"id": "1"}},
			wantURL:     "https://x/1",
			wantMissing: []string{"path_id"},
		},
		{
			name:        "blank values count as missing",
			req:         &types.SavedRequest{URL: "https://x/:id/{{tenant}}"},
			scopes:      Scopes{Env: map[string]string{"tenant": "acme", "id": "1"}, CLI: map[string]string{"tenant": "  ", "id": ""}},
			wantURL:     "https://x/:id/{{tenant}}",
			wantMissing: []string{"id", "tenant"},
		},
		{
			name: "all request fields",
			req: &types.SavedRequest{
				URL:        "https://{{host}}/{{id}}",
				Body:       `{"tenant":"{{tenant}}"}`,
				Headers:    []types.Header{{Key: "X-{{region}}", Value: "{{token}}"}},
				AuthConfig: &types.AuthConfig{Type: "oauth1", Params: map[string]string{"consumer_key": "{{consumer_key}}", "consumer_secret": "{{consumer_secret}}", "token": "{{oauth_token}}", "token_secret": "static"}},
				PathParams: []types.Var{{Name: "path_id"}},
			},
			scopes:      Scopes{CLI: map[string]string{"host": "api", "id": "1", "tenant": "acme", "consumer_key": "ck"}},
			wantURL:     "https://api/1",
			wantMissing: []string{"path_id", "region", "token", "consumer_secret", "oauth_token"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, problems := Resolve(tt.req, tt.scopes, Options{Interactive: true})
			if got.Request.URL != tt.wantURL {
				t.Fatalf("URL = %q, want %q", got.Request.URL, tt.wantURL)
			}
			if !reflect.DeepEqual(problems.Missing(), tt.wantMissing) || len(problems.Others()) != 0 {
				t.Fatalf("problems = %v, want missing %v", problems, tt.wantMissing)
			}
		})
	}
}

func TestResolveNonInteractiveLeavesPathParamsAndBlankValues(t *testing.T) {
	req := &types.SavedRequest{URL: "https://x/:id/{{tenant}}"}
	got, problems := Resolve(req, Scopes{CLI: map[string]string{"tenant": ""}}, Options{})
	if problems != nil || got.Request.URL != "https://x/:id/" {
		t.Fatalf("got URL %q problems %v", got.Request.URL, problems)
	}
}

type stubHandler struct{}

func (stubHandler) Name() string            { return "stub" }
func (stubHandler) Description() string     { return "test stub" }
func (stubHandler) Params() []auth.ParamDef { return nil }

func (stubHandler) Apply(req *client.Request, params map[string]string) error {
	req.Headers = append(req.Headers, client.Header{Key: "X-Stub", Value: params["token"]})
	return nil
}
