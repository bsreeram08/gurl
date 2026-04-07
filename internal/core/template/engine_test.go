package template

import (
	"testing"

	"github.com/sreeram/gurl/pkg/types"
)

// TestSubstitute tests the template variable substitution function
func TestSubstitute(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		vars      map[string]string
		want      string
		wantErr   bool
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
			name:  "missing variable returns error",
			input: "curl {{url}}",
			vars: map[string]string{
				"other": "value",
			},
			want:    "",
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
			name:  "empty vars with template returns error",
			input: "curl {{url}}",
			vars:  map[string]string{},
			want:    "",
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
