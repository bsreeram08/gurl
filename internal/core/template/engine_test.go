package template

import (
	"strings"
	"testing"

	"github.com/sreeram/gurl/pkg/types"
)

// TestSubstitute tests the template variable substitution function
func TestSubstitute(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		vars    map[string]string
		want    string
		wantErr bool
	}{
		{
			name:  "simple variable substitution",
			input: "curl {{url}}",
			vars: map[string]string{
				"url": "https://example.com",
			},
			want:    "curl https://example.com",
			wantErr: false,
		},
		{
			name:  "multiple variables",
			input: "curl -H 'Authorization: {{token}}' {{url}}",
			vars: map[string]string{
				"url":   "https://example.com",
				"token": "Bearer abc123",
			},
			want:    "curl -H 'Authorization: Bearer abc123' https://example.com",
			wantErr: false,
		},
		{
			name:    "missing variable returns error",
			input:   "curl {{url}}",
			vars:    map[string]string{},
			wantErr: true,
		},
		{
			name:  "extra variables are ignored",
			input: "curl {{url}}",
			vars: map[string]string{
				"url":   "https://example.com",
				"extra": "ignored",
			},
			want:    "curl https://example.com",
			wantErr: false,
		},
		{
			name:  "no variables returns original",
			input: "curl https://example.com",
			vars: map[string]string{
				"url": "https://other.com",
			},
			want:    "curl https://example.com",
			wantErr: false,
		},
		{
			name:    "empty vars with template returns error",
			input:   "curl {{url}}",
			vars:    map[string]string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Substitute(tt.input, tt.vars)
			if (err != nil) != tt.wantErr {
				t.Errorf("Substitute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Substitute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSubstitute_DeterministicOrder(t *testing.T) {
	vars := map[string]string{"a": "1", "b": "2", "c": "3"}
	cmd := "={{a}}={{b}}={{c}}"
	// Run 10 times and verify same result
	for i := 0; i < 10; i++ {
		result, err := Substitute(cmd, vars)
		if err != nil {
			t.Fatalf("Substitute failed: %v", err)
		}
		if result != "=1=2=3" {
			t.Errorf("Iteration %d: got %q, want %q", i, result, "=1=2=3")
		}
	}
}

func TestSubstitute_InjectionPrevention(t *testing.T) {
	// A value containing {{...}} must NOT be expanded a second time (it could
	// pull a secret into a request); it is reported as an error instead.
	vars := map[string]string{"var": "{{inner}}", "inner": "UNSAFE"}
	result, err := Substitute("start {{var}} end", vars)
	if err == nil {
		t.Fatalf("Substitute() = %q, want error", result)
	}
	want := `variable "var" contains unresolved placeholder {{inner}}`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
	if strings.Contains(err.Error(), "UNSAFE") {
		t.Errorf("error leaked second-level value: %q", err.Error())
	}
}

func TestSubstitute_Typed(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		vars    map[string]string
		want    string
		wantErr string // exact expected error; empty means success
	}{
		{
			name:    "nested value is not expanded",
			input:   "/orders/{{ORDER}}",
			vars:    map[string]string{"ORDER": "{{orderId}}", "orderId": "42"},
			wantErr: `variable "ORDER" contains unresolved placeholder {{orderId}}`,
		},
		{
			name:    "all nested placeholders reported in order with missing first",
			input:   "{{missing}} {{A}} {{B}} {{A}}",
			vars:    map[string]string{"A": "{{x}}-{{y:num}}", "B": "{{z}}"},
			wantErr: `missing required variables: missing; variable "A" contains unresolved placeholder {{x}}; variable "A" contains unresolved placeholder {{y:num}}; variable "B" contains unresolved placeholder {{z}}`,
		},
		{
			name:  "quoted num drops quotes",
			input: `{"amount":"{{amount:num}}"}`,
			vars:  map[string]string{"amount": "5"},
			want:  `{"amount":5}`,
		},
		{
			name:  "unquoted num inserted",
			input: `{"amount":{{amount:num}}}`,
			vars:  map[string]string{"amount": "-1.5e3"},
			want:  `{"amount":-1.5e3}`,
		},
		{
			name:  "quoted bool drops quotes",
			input: `{"ok":"{{flag:bool}}"}`,
			vars:  map[string]string{"flag": "true"},
			want:  `{"ok":true}`,
		},
		{
			name:  "unquoted bool inserted",
			input: `{"ok":{{flag:bool}}}`,
			vars:  map[string]string{"flag": "false"},
			want:  `{"ok":false}`,
		},
		{
			name:  "quoted raw inserts object verbatim",
			input: `{"meta":"{{meta:raw}}"}`,
			vars:  map[string]string{"meta": `{"a":[1,2]}`},
			want:  `{"meta":{"a":[1,2]}}`,
		},
		{
			name:  "unquoted raw inserted",
			input: `{"meta":{{meta:raw}}}`,
			vars:  map[string]string{"meta": `null`},
			want:  `{"meta":null}`,
		},
		{
			name:  "plain quoted placeholder keeps quotes",
			input: `{"id":"{{id}}"}`,
			vars:  map[string]string{"id": "7"},
			want:  `{"id":"7"}`,
		},
		{
			name:    "invalid num errors naming variable",
			input:   `{"amount":"{{amount:num}}"}`,
			vars:    map[string]string{"amount": "five"},
			wantErr: `variable "amount" must be a number`,
		},
		{
			name:    "invalid bool errors naming variable",
			input:   `{{flag:bool}}`,
			vars:    map[string]string{"flag": "yes"},
			wantErr: `variable "flag" must be true or false`,
		},
		{
			name:    "missing typed reports base name",
			input:   `{"amount":"{{amount:num}}"}`,
			vars:    map[string]string{},
			wantErr: "missing required variables: amount",
		},
		{
			name:  "dynamic placeholder satisfied by vars (current behaviour)",
			input: "{{$uuid}}",
			vars:  map[string]string{"$uuid": "u-1"},
			want:  "u-1",
		},
		{
			name:    "dynamic placeholder missing is reported (current behaviour)",
			input:   "{{$uuid}}",
			vars:    map[string]string{},
			wantErr: "missing required variables: $uuid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Substitute(tt.input, tt.vars)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("Substitute() = %q, want error %q", got, tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("Substitute() error = %q, want %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Substitute() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Substitute() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveKnown(t *testing.T) {
	tests := []struct {
		name  string
		input string
		vars  map[string]string
		want  string
	}{
		{"leaves unknown intact", "{{a}}/{{b}}", map[string]string{"a": "1"}, "1/{{b}}"},
		{"typed", `{"n":"{{n:num}}","x":"{{x}}"}`, map[string]string{"n": "3", "x": "y"}, `{"n":3,"x":"y"}`},
		{"single pass", "{{a}}", map[string]string{"a": "{{b}}", "b": "B"}, "{{b}}"},
		{"invalid typed left intact", `"{{n:num}}"`, map[string]string{"n": "abc"}, `"{{n:num}}"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveKnown(tt.input, tt.vars); got != tt.want {
				t.Errorf("ResolveKnown() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTypedPlaceholderBaseNames(t *testing.T) {
	input := `{"amount":"{{amount:num}}","ok":{{flag:bool}},"m":{{meta:raw}},"id":"{{id}}","amt2":"{{amount}}"}`
	got := ExtractVarNames(input)
	want := []string{"amount", "flag", "meta", "id"}
	if len(got) != len(want) {
		t.Fatalf("ExtractVarNames() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ExtractVarNames() = %v, want %v", got, want)
		}
	}

	if err := Validate(input, map[string]string{"amount": "1", "flag": "true", "meta": "{}", "id": "x"}); err != nil {
		t.Errorf("Validate() unexpected error: %v", err)
	}
	if err := Validate(input, map[string]string{}); err == nil || err.Error() != "missing required variables: amount, flag, meta, id" {
		t.Errorf("Validate() error = %v", err)
	}

	req := &types.SavedRequest{URL: "https://x/{{id}}", Body: input}
	vars := GetVariablesFromRequest(req)
	names := make([]string, 0, len(vars))
	for _, v := range vars {
		names = append(names, v.Name)
	}
	if strings.Join(names, ",") != "id,amount,flag,meta" {
		t.Errorf("GetVariablesFromRequest() names = %v", names)
	}
}

// TestValidate tests the template validation function
func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		vars    map[string]string
		wantErr bool
	}{
		{
			name:    "valid template with all vars",
			input:   "curl {{url}}",
			vars:    map[string]string{"url": "https://example.com"},
			wantErr: false,
		},
		{
			name:    "missing variable",
			input:   "curl {{url}}",
			vars:    map[string]string{},
			wantErr: true,
		},
		{
			name:    "no template returns nil",
			input:   "curl https://example.com",
			vars:    map[string]string{},
			wantErr: false,
		},
		{
			name:    "partial variables",
			input:   "curl {{url}} -H 'Token: {{token}}'",
			vars:    map[string]string{"url": "https://example.com"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.input, tt.vars)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestExtractVarNames tests the variable name extraction function
func TestExtractVarNames(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single variable",
			input: "curl {{url}}",
			want:  []string{"url"},
		},
		{
			name:  "multiple variables",
			input: "curl -H 'Token: {{token}}' {{url}}",
			want:  []string{"token", "url"},
		},
		{
			name:  "no variables",
			input: "curl https://example.com",
			want:  []string{},
		},
		{
			name:  "duplicate variables",
			input: "{{url}} -H 'Authorization: {{token}}' {{url}}",
			want:  []string{"url", "token"},
		},
		{
			name:  "variable in body",
			input: "curl -d '{\"name\":\"{{name}}\"}' {{url}}",
			want:  []string{"name", "url"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractVarNames(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("ExtractVarNames() len = %v, want %v", len(got), len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("ExtractVarNames()[%d] = %v, want %v", i, v, tt.want[i])
				}
			}
		})
	}
}

// TestHasVariables tests the template detection function
func TestHasVariables(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "has variable",
			input: "curl {{url}}",
			want:  true,
		},
		{
			name:  "no variable",
			input: "curl https://example.com",
			want:  false,
		},
		{
			name:  "multiple variables",
			input: "{{url}} -H 'Token: {{token}}'",
			want:  true,
		},
		{
			name:  "empty string",
			input: "",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasVariables(tt.input)
			if got != tt.want {
				t.Errorf("HasVariables() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetVariablesFromRequest tests variable extraction from SavedRequest
func TestGetVariablesFromRequest(t *testing.T) {
	tests := []struct {
		name    string
		request *types.SavedRequest
		wantLen int
	}{
		{
			name: "URL with variable",
			request: &types.SavedRequest{
				URL:    "{{base_url}}/api",
				Method: "GET",
			},
			wantLen: 1,
		},
		{
			name: "URL and body with variables",
			request: &types.SavedRequest{
				URL:    "{{base_url}}/api",
				Method: "POST",
				Body:   "{\"name\":\"{{name}}\"}",
			},
			wantLen: 2,
		},
		{
			name: "no variables",
			request: &types.SavedRequest{
				URL:    "https://example.com/api",
				Method: "GET",
			},
			wantLen: 0,
		},
		{
			name: "same variable in URL and body",
			request: &types.SavedRequest{
				URL:    "{{base_url}}/api",
				Method: "POST",
				Body:   "{{base_url}}/other",
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetVariablesFromRequest(tt.request)
			if len(got) != tt.wantLen {
				t.Errorf("GetVariablesFromRequest() len = %v, want %v", len(got), tt.wantLen)
			}
		})
	}
}
