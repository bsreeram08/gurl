package commands

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sreeram/gurl/internal/core/template"
	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/pkg/types"
)

func varsFromEnvAndArgs(envVars map[string]string, args []string) map[string]string {
	vars := make(map[string]string)
	for k, v := range envVars {
		vars[k] = v
	}

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

func TestRunWithEnvLoadsVariables(t *testing.T) {
	envVars := map[string]string{
		"BASE_URL": "api.dev.com",
		"VERSION":  "v1",
	}

	vars := varsFromEnvAndArgs(envVars, []string{"run", "test", "--env", "dev"})

	substitutedURL, err := template.Substitute("https://{{BASE_URL}}/{{VERSION}}/users", vars)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if substitutedURL != "https://api.dev.com/v1/users" {
		t.Errorf("expected URL with env vars substituted, got: %s", substitutedURL)
	}
}

func TestRunVarOverridesEnv(t *testing.T) {
	envVars := map[string]string{
		"BASE_URL": "api.dev.com",
		"VERSION":  "v1",
	}

	vars := varsFromEnvAndArgs(envVars, []string{"run", "test", "--var", "BASE_URL=api.override.com"})

	substitutedURL, err := template.Substitute("https://{{BASE_URL}}/{{VERSION}}/users", vars)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if substitutedURL != "https://api.override.com/v1/users" {
		t.Errorf("expected CLI var to override env var, got: %s", substitutedURL)
	}
}

func TestRunWithEnvDevNoVarOverride(t *testing.T) {
	devEnvVars := map[string]string{
		"API_KEY":  "dev-secret-key",
		"ENDPOINT": "/api/dev",
	}

	vars := varsFromEnvAndArgs(devEnvVars, []string{"run", "test", "--env", "dev"})

	substitutedURL, err := template.Substitute("https://example.com{{ENDPOINT}}?key={{API_KEY}}", vars)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	expected := "https://example.com/api/dev?key=dev-secret-key"
	if substitutedURL != expected {
		t.Errorf("expected %s, got: %s", expected, substitutedURL)
	}
}

func TestRunBackwardCompatNoEnv(t *testing.T) {
	cliVars := map[string]string{
		"NAME": "testuser",
	}

	substitutedURL, err := template.Substitute("https://example.com/{{NAME}}", cliVars)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if substitutedURL != "https://example.com/testuser" {
		t.Errorf("expected URL with CLI vars only, got: %s", substitutedURL)
	}
}

func TestRunCommandSavedAssertionFailureReturnsError(t *testing.T) {
	db := newMockDB()
	envStorage := &env.EnvStorage{}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	db.requests["req-assert"] = &types.SavedRequest{
		ID:     "req-assert",
		Name:   "assert-fail",
		URL:    ts.URL,
		Method: "GET",
		Assertions: []types.Assertion{
			{Field: "status", Op: "equals", Value: "201"},
		},
	}
	db.names["assert-fail"] = "req-assert"

	cmd := RunCommand(db, envStorage)
	err := cmd.Run(context.Background(), []string{"run", "assert-fail"})
	if err == nil {
		t.Fatal("expected saved assertion failure to return an error")
	}
	if !strings.Contains(err.Error(), "assertion failed") {
		t.Fatalf("expected assertion failure error, got %v", err)
	}
	if len(db.history) != 1 {
		t.Fatalf("expected saved assertion failure to preserve history save, got %d entries", len(db.history))
	}
}

func TestRunCommandCLIAssertionFailureReturnsError(t *testing.T) {
	db := newMockDB()
	envStorage := &env.EnvStorage{}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	db.requests["req-cli-assert"] = &types.SavedRequest{
		ID:     "req-cli-assert",
		Name:   "cli-assert",
		URL:    ts.URL,
		Method: "GET",
	}
	db.names["cli-assert"] = "req-cli-assert"

	cmd := RunCommand(db, envStorage)
	err := cmd.Run(context.Background(), []string{"run", "cli-assert", "--assert", "status=201"})
	if err == nil {
		t.Fatal("expected CLI assertion failure to return an error")
	}
	if !strings.Contains(err.Error(), "assertion failed") || !strings.Contains(err.Error(), "status") || !strings.Contains(err.Error(), "201") || !strings.Contains(err.Error(), "200") {
		t.Fatalf("expected useful assertion failure details, got %v", err)
	}
	if len(db.history) != 1 {
		t.Fatalf("expected CLI assertion failure to preserve history save, got %d entries", len(db.history))
	}
}

func TestRunCommandCLIAssertionPassReturnsNilAndSavesHistory(t *testing.T) {
	db := newMockDB()
	envStorage := &env.EnvStorage{}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	db.requests["req-cli-assert-pass"] = &types.SavedRequest{
		ID:     "req-cli-assert-pass",
		Name:   "cli-assert-pass",
		URL:    ts.URL,
		Method: "GET",
	}
	db.names["cli-assert-pass"] = "req-cli-assert-pass"

	cmd := RunCommand(db, envStorage)
	err := cmd.Run(context.Background(), []string{"run", "cli-assert-pass", "--assert", "status=200"})
	if err != nil {
		t.Fatalf("expected passing CLI assertion to return nil, got %v", err)
	}
	if len(db.history) != 1 {
		t.Fatalf("expected history to be saved after passing CLI assertion, got %d entries", len(db.history))
	}
}

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
