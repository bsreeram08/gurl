package commands

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sreeram/gurl/internal/core/template"
	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/urfave/cli/v3"
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

func TestRunCommandPersistWritesSingleRequestDirtyVars(t *testing.T) {
	db := newMockDB()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"orderId":"ord_123"}}`))
	}))
	defer server.Close()

	db.requests["req-persist"] = &types.SavedRequest{
		ID:         "req-persist",
		Name:       "create order",
		URL:        "{{baseUrl}}/orders/{{cliOnly}}",
		Method:     "POST",
		PostScript: `gurl.setVar("flowNote", "scripted")`,
		Extracts: []types.Extract{
			{Name: "orderId", Source: "jsonpath:$.data.orderId"},
		},
	}
	db.names["create order"] = "req-persist"

	envStorage := newRunTestEnvStorage(t)
	beta := env.NewEnvironment("beta", "")
	beta.SetVariable("baseUrl", server.URL)
	beta.SetVariable("unchanged", "keep")
	if err := envStorage.SaveEnv(beta); err != nil {
		t.Fatalf("failed to save env: %v", err)
	}

	cmd := RunCommand(db, envStorage)
	output := captureStdout(t, func() {
		if err := cmd.Run(context.Background(), []string{"run", "create order", "--env", "beta", "--var", "cliOnly=123", "--persist"}); err != nil {
			t.Fatalf("run command failed: %v", err)
		}
	})

	reloaded, err := envStorage.GetEnvByName("beta")
	if err != nil {
		t.Fatalf("failed to reload env: %v", err)
	}
	if reloaded.Variables["orderId"] != "ord_123" || reloaded.Variables["flowNote"] != "scripted" {
		t.Fatalf("expected dirty vars to persist, got %+v", reloaded.Variables)
	}
	if _, ok := reloaded.Variables["cliOnly"]; ok {
		t.Fatalf("expected CLI var not to persist, got %+v", reloaded.Variables)
	}
	if reloaded.Variables["unchanged"] != "keep" {
		t.Fatalf("expected existing env var to remain, got %+v", reloaded.Variables)
	}
	if !strings.Contains(output, "Persisted 2 variables to environment \"beta\"") || !strings.Contains(output, "flowNote = scripted") || !strings.Contains(output, "orderId = ord_123") {
		t.Fatalf("expected persist summary with exact keys and values, got output:\n%s", output)
	}
}

func TestRunCommandPersistWithoutEnvFailsBeforeRequestOutsideCollection(t *testing.T) {
	db := newMockDB()
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	db.requests["req-persist-no-env"] = &types.SavedRequest{
		ID:         "req-persist-no-env",
		Name:       "persist no env",
		URL:        server.URL,
		Method:     "GET",
		PostScript: `gurl.setVar("flowNote", "scripted")`,
	}
	db.names["persist no env"] = "req-persist-no-env"

	cmd := RunCommand(db, newRunTestEnvStorage(t))
	err := cmd.Run(context.Background(), []string{"run", "persist no env", "--persist"})
	if err == nil {
		t.Fatal("expected --persist without env to fail")
	}
	if !strings.Contains(err.Error(), "--persist requires --env or an active environment") {
		t.Fatalf("expected clear persist target error, got %v", err)
	}
	if requestCount != 0 {
		t.Fatalf("expected fail-fast before HTTP, got %d requests", requestCount)
	}
}

func TestRunCommandChainRefreshesCollectionContextPerStep(t *testing.T) {
	db := storage.NewLMDBWithPath(filepath.Join(t.TempDir(), "chain-collections.db"))
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.URL.Path != "/first" && r.URL.Path != "/second/beta-token" {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"wrong path"}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	alpha := types.NewCollection("alpha")
	alpha.SetVariable("BASE_URL", server.URL)
	alpha.SetVariable("SECOND_TOKEN", "alpha-token")
	if err := db.SaveCollection(alpha); err != nil {
		t.Fatalf("failed to save alpha collection: %v", err)
	}
	beta := types.NewCollection("beta")
	beta.SetVariable("BASE_URL", server.URL)
	beta.SetVariable("SECOND_TOKEN", "beta-token")
	if err := db.SaveCollection(beta); err != nil {
		t.Fatalf("failed to save beta collection: %v", err)
	}

	if err := db.SaveRequest(&types.SavedRequest{
		Name:       "first",
		URL:        "{{BASE_URL}}/first",
		Method:     "GET",
		Collection: "alpha",
		PostScript: `gurl.setNextRequest("second")`,
	}); err != nil {
		t.Fatalf("failed to save first request: %v", err)
	}
	if err := db.SaveRequest(&types.SavedRequest{
		Name:       "second",
		URL:        "{{BASE_URL}}/second/{{SECOND_TOKEN}}",
		Method:     "GET",
		Collection: "beta",
		PostScript: `gurl.setVar("SECOND_TOKEN", "changed-token")`,
	}); err != nil {
		t.Fatalf("failed to save second request: %v", err)
	}

	cmd := RunCommand(db, newRunTestEnvStorage(t))
	output := captureStdout(t, func() {
		if err := cmd.Run(context.Background(), []string{"run", "first", "--chain", "--persist"}); err != nil {
			t.Fatalf("run command failed: %v", err)
		}
	})

	if strings.Join(paths, ",") != "/first,/second/beta-token" {
		t.Fatalf("expected second request to use beta collection vars, got paths %+v", paths)
	}
	reloadedAlpha, err := db.GetCollectionByName("alpha")
	if err != nil {
		t.Fatalf("failed to reload alpha collection: %v", err)
	}
	if reloadedAlpha.Variables["SECOND_TOKEN"] != "alpha-token" {
		t.Fatalf("expected alpha collection to remain unchanged, got %+v", reloadedAlpha.Variables)
	}
	reloadedBeta, err := db.GetCollectionByName("beta")
	if err != nil {
		t.Fatalf("failed to reload beta collection: %v", err)
	}
	if reloadedBeta.Variables["SECOND_TOKEN"] != "changed-token" {
		t.Fatalf("expected dirty beta var to persist to beta, got %+v", reloadedBeta.Variables)
	}
	if !strings.Contains(output, "Persisted 1 variable to collection \"beta\"") || strings.Contains(output, "collection \"alpha\"") {
		t.Fatalf("expected persist summary for beta only, got output:\n%s", output)
	}
}

func TestRunCommandWithoutPersistLeavesEnvUnchanged(t *testing.T) {
	db := newMockDB()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"orderId":"ord_123"}}`))
	}))
	defer server.Close()

	db.requests["req-no-persist"] = &types.SavedRequest{
		ID:     "req-no-persist",
		Name:   "create order no persist",
		URL:    "{{baseUrl}}/orders",
		Method: "POST",
		Extracts: []types.Extract{
			{Name: "orderId", Source: "jsonpath:$.data.orderId"},
		},
	}
	db.names["create order no persist"] = "req-no-persist"

	envStorage := newRunTestEnvStorage(t)
	beta := env.NewEnvironment("beta", "")
	beta.SetVariable("baseUrl", server.URL)
	if err := envStorage.SaveEnv(beta); err != nil {
		t.Fatalf("failed to save env: %v", err)
	}

	cmd := RunCommand(db, envStorage)
	output := captureStdout(t, func() {
		if err := cmd.Run(context.Background(), []string{"run", "create order no persist", "--env", "beta"}); err != nil {
			t.Fatalf("run command failed: %v", err)
		}
	})

	reloaded, err := envStorage.GetEnvByName("beta")
	if err != nil {
		t.Fatalf("failed to reload env: %v", err)
	}
	if _, ok := reloaded.Variables["orderId"]; ok {
		t.Fatalf("expected no dirty var persistence without --persist, got %+v", reloaded.Variables)
	}
	if strings.Contains(output, "Persisted") {
		t.Fatalf("expected no persist summary without --persist, got output:\n%s", output)
	}
}

func TestRunCommandDataDrivenPersistOnlyWritesDirtyVars(t *testing.T) {
	db := newMockDB()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"orderId":"ord_from_response"}}`))
	}))
	defer server.Close()

	db.requests["req-data-persist"] = &types.SavedRequest{
		ID:         "req-data-persist",
		Name:       "data persist",
		URL:        "{{baseUrl}}/orders/{{rowOrder}}",
		Method:     "POST",
		PostScript: `gurl.setVar("scriptOutput", gurl.getVar("rowOrder") + "-script")`,
		Extracts: []types.Extract{
			{Name: "orderId", Source: "jsonpath:$.data.orderId"},
		},
	}
	db.names["data persist"] = "req-data-persist"

	dataPath := filepath.Join(t.TempDir(), "rows.csv")
	if err := os.WriteFile(dataPath, []byte("rowOrder\nrow_123\n"), 0644); err != nil {
		t.Fatalf("failed to write data file: %v", err)
	}

	envStorage := newRunTestEnvStorage(t)
	beta := env.NewEnvironment("beta", "")
	beta.SetVariable("baseUrl", server.URL)
	if err := envStorage.SaveEnv(beta); err != nil {
		t.Fatalf("failed to save env: %v", err)
	}

	cmd := RunCommand(db, envStorage)
	output := captureStdout(t, func() {
		if err := cmd.Run(context.Background(), []string{"run", "data persist", "--env", "beta", "--data", dataPath, "--persist"}); err != nil {
			t.Fatalf("run command failed: %v", err)
		}
	})

	reloaded, err := envStorage.GetEnvByName("beta")
	if err != nil {
		t.Fatalf("failed to reload env: %v", err)
	}
	if reloaded.Variables["orderId"] != "ord_from_response" || reloaded.Variables["scriptOutput"] != "row_123-script" {
		t.Fatalf("expected dirty extraction/script vars to persist, got %+v", reloaded.Variables)
	}
	if _, ok := reloaded.Variables["rowOrder"]; ok {
		t.Fatalf("expected data row var not to persist, got %+v", reloaded.Variables)
	}
	if !strings.Contains(output, "Persisted 2 variables to environment \"beta\"") {
		t.Fatalf("expected persist summary, got output:\n%s", output)
	}
}

func TestRunCommandPersistDryRunFailsFastWithoutMutation(t *testing.T) {
	db := newMockDB()
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"orderId":"ord_123"}}`))
	}))
	defer server.Close()

	db.requests["req-dry-run-persist"] = &types.SavedRequest{
		ID:     "req-dry-run-persist",
		Name:   "dry run persist",
		URL:    "{{baseUrl}}/orders",
		Method: "POST",
		Extracts: []types.Extract{
			{Name: "orderId", Source: "jsonpath:$.data.orderId"},
		},
	}
	db.names["dry run persist"] = "req-dry-run-persist"

	envStorage := newRunTestEnvStorage(t)
	beta := env.NewEnvironment("beta", "")
	beta.SetVariable("baseUrl", server.URL)
	if err := envStorage.SaveEnv(beta); err != nil {
		t.Fatalf("failed to save env: %v", err)
	}

	cmd := RunCommand(db, envStorage)
	cmd.ExitErrHandler = func(context.Context, *cli.Command, error) {}
	err := cmd.Run(context.Background(), []string{"run", "dry run persist", "--env", "beta", "--persist", "--dry-run"})
	if err == nil {
		t.Fatal("expected persist/dry-run incompatibility error")
	}
	if !strings.Contains(err.Error(), "--persist and --dry-run cannot be used together") {
		t.Fatalf("expected clear incompatible flags error, got %v", err)
	}
	if requestCount != 0 {
		t.Fatalf("expected fail-fast before HTTP, got %d requests", requestCount)
	}
	reloaded, err := envStorage.GetEnvByName("beta")
	if err != nil {
		t.Fatalf("failed to reload env: %v", err)
	}
	if _, ok := reloaded.Variables["orderId"]; ok {
		t.Fatalf("expected no env mutation on incompatible flags, got %+v", reloaded.Variables)
	}
}

func newRunTestEnvStorage(t *testing.T) *env.EnvStorage {
	t.Helper()
	db := storage.NewLMDBWithPath(filepath.Join(t.TempDir(), "env.db"))
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open env db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return env.NewEnvStorage(db)
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
