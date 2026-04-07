package commands

import (
	"testing"

	"github.com/sreeram/gurl/internal/core/template"
	"github.com/sreeram/gurl/pkg/types"
)

func TestRunCommandMultipleVars(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		requestURL string
		requestBod string
		wantURL    string
		wantBody   string
		wantErr    bool
	}{
		{
			name:       "multiple var flags substitute correctly",
			args:       []string{"test", "--var", "KEY1=val1", "--var", "KEY2=val2"},
			requestURL: "https://{{KEY1}}.com/{{KEY2}}",
			requestBod: "",
			wantURL:    "https://val1.com/val2",
			wantBody:   "",
			wantErr:    false,
		},
		{
			name:       "single var with multiple flags still works",
			args:       []string{"test", "--var", "BASE=https://api.example.com", "--var", "PATH=users"},
			requestURL: "{{BASE}}/{{PATH}}",
			requestBod: "",
			wantURL:    "https://api.example.com/users",
			wantErr:    false,
		},
		{
			name:       "var in body gets substituted",
			args:       []string{"test", "--var", "NAME=testuser", "--var", "TOKEN=abc123"},
			requestURL: "https://example.com/api",
			requestBod: "{\"name\":\"{{NAME}}\",\"token\":\"{{TOKEN}}\"}",
			wantURL:    "https://example.com/api",
			wantBody:   "{\"name\":\"testuser\",\"token\":\"abc123\"}",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newMockDB()

			db.names["test"] = "test-id"
			db.requests["test-id"] = &types.SavedRequest{
				ID:     "test-id",
				Name:   "test",
				URL:    tt.requestURL,
				Method: "GET",
				Body:   tt.requestBod,
			}

			fullArgs := append([]string{"run"}, tt.args...)

			gotURL, err := template.Substitute(tt.requestURL, varsFromArgs(fullArgs))
			if (err != nil) != tt.wantErr {
				t.Errorf("template.Substitute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotURL != tt.wantURL {
				t.Errorf("URL = %v, want %v", gotURL, tt.wantURL)
			}

			if tt.requestBod != "" {
				gotBody, _ := template.Substitute(tt.requestBod, varsFromArgs(fullArgs))
				if gotBody != tt.wantBody {
					t.Errorf("Body = %v, want %v", gotBody, tt.wantBody)
				}
			}
		})
	}
}

func varsFromArgs(args []string) map[string]string {
	vars := make(map[string]string)
	for i, arg := range args {
		if (arg == "--var" || arg == "-v") && i+1 < len(args) {
			pair := args[i+1]
			for j := 0; j < len(pair); j++ {
				if pair[j] == '=' {
					vars[pair[:j]] = pair[j+1:]
					break
				}
			}
		}
	}
	return vars
}
