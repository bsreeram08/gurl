package env

import (
	"testing"
)

// TestVariableScoping tests the variable resolution order:
// global → environment → CLI --var (later overrides earlier)
func TestVariableScoping(t *testing.T) {
	tests := []struct {
		name       string
		globalVars map[string]string
		envVars    map[string]string
		cliVars    map[string]string
		want       map[string]string
	}{
		{
			name:       "all empty",
			globalVars: nil,
			envVars:    nil,
			cliVars:    nil,
			want:       map[string]string{},
		},
		{
			name:       "global only",
			globalVars: map[string]string{"BASE_URL": "https://global.com"},
			envVars:    nil,
			cliVars:    nil,
			want:       map[string]string{"BASE_URL": "https://global.com"},
		},
		{
			name:       "env overrides global",
			globalVars: map[string]string{"BASE_URL": "https://global.com", "SHARED": "global"},
			envVars:    map[string]string{"BASE_URL": "https://env.com"},
			cliVars:    nil,
			want:       map[string]string{"BASE_URL": "https://env.com", "SHARED": "global"},
		},
		{
			name:       "cli overrides env and global",
			globalVars: map[string]string{"BASE_URL": "https://global.com"},
			envVars:    map[string]string{"BASE_URL": "https://env.com"},
			cliVars:    map[string]string{"BASE_URL": "https://cli.com"},
			want:       map[string]string{"BASE_URL": "https://cli.com"},
		},
		{
			name:       "cli adds new vars not in global or env",
			globalVars: map[string]string{"GLOBAL_VAR": "gv"},
			envVars:    map[string]string{"ENV_VAR": "ev"},
			cliVars:    map[string]string{"CLI_VAR": "cv"},
			want: map[string]string{
				"GLOBAL_VAR": "gv",
				"ENV_VAR":    "ev",
				"CLI_VAR":    "cv",
			},
		},
		{
			name:       "complex precedence - all three layers",
			globalVars: map[string]string{"A": "global-a", "B": "global-b", "C": "global-c"},
			envVars:    map[string]string{"B": "env-b", "C": "env-c"},
			cliVars:    map[string]string{"C": "cli-c"},
			want:       map[string]string{"A": "global-a", "B": "env-b", "C": "cli-c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveVariables(tt.cliVars, tt.envVars, tt.globalVars)

			// Check all expected keys exist with correct values
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("ResolveVariables()[%q] = %q, want %q", k, got[k], v)
				}
			}

			// Check no extra keys
			for k, v := range got {
				if _, ok := tt.want[k]; !ok {
					t.Errorf("ResolveVariables() has unexpected key %q = %q", k, v)
				}
			}
		})
	}
}

func TestVariableScopingEmptyInputs(t *testing.T) {
	// nil maps should be handled gracefully
	result := ResolveVariables(nil, nil, nil)
	if len(result) != 0 {
		t.Errorf("expected empty result for all nil inputs, got %v", result)
	}
}

func TestVariableScopingPartialNil(t *testing.T) {
	// Test with some nil inputs
	result := ResolveVariables(map[string]string{"CLI": "cv"}, nil, nil)
	if result["CLI"] != "cv" {
		t.Errorf("expected CLI var, got %v", result)
	}

	result = ResolveVariables(nil, map[string]string{"ENV": "ev"}, nil)
	if result["ENV"] != "ev" {
		t.Errorf("expected ENV var, got %v", result)
	}

	result = ResolveVariables(nil, nil, map[string]string{"GLOBAL": "gv"})
	if result["GLOBAL"] != "gv" {
		t.Errorf("expected GLOBAL var, got %v", result)
	}
}
