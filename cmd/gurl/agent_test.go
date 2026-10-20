package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/internal/storage/memstore"
	"github.com/urfave/cli/v3"
)

func TestInvokedAsAgent(t *testing.T) {
	tests := []struct {
		arg0 string
		want bool
	}{
		{"agentgurl", true},
		{"/usr/local/bin/agentgurl", true},
		{"agentgurl.exe", true},
		{"AgentGurl.EXE", true},
		{"gurl", false},
		{"/usr/local/bin/gurl", false},
		{"agentgurl2", false},
		{"my-agentgurl", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.arg0, func(t *testing.T) {
			if got := invokedAsAgent(tt.arg0); got != tt.want {
				t.Fatalf("invokedAsAgent(%q) = %v, want %v", tt.arg0, got, tt.want)
			}
		})
	}
}

func setupMainTest(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GURL_DB_PATH", filepath.Join(t.TempDir(), "gurl.db"))
	t.Setenv("GURL_COOKIE_DB_PATH", filepath.Join(t.TempDir(), "gurl-cookies.db"))
	t.Setenv("GURL_PLUGIN_DIR", t.TempDir())
	t.Setenv("GURL_AGENT", "")
	t.Setenv("GURL_NO_INPUT", "")
}

type docsJSON struct {
	Commands []struct {
		Path  string `json:"path"`
		Flags []struct {
			Name string `json:"name"`
		} `json:"flags"`
	} `json:"commands"`
	Sections map[string]string `json:"sections"`
}

// visiblePaths lists "gurl ..." paths of every visible registered command.
func visiblePaths(cmd *cli.Command, path string) []string {
	paths := []string{path}
	for _, sub := range cmd.Commands {
		if !sub.Hidden {
			paths = append(paths, visiblePaths(sub, path+" "+sub.Name)...)
		}
	}
	return paths
}

func TestDocsJSONListsEveryCommandCompactly(t *testing.T) {
	setupMainTest(t)

	stdout, stderr, err := runWithArgs(t, []string{"gurl", "docs", "--json"})
	if err != nil {
		t.Fatalf("docs --json: %v\nstderr: %s", err, stderr)
	}
	const maxSize = 32 * 1024 // ~8k tokens: one cheap read for an agent
	if len(stdout) > maxSize {
		t.Fatalf("docs --json is %d bytes, want <= %d", len(stdout), maxSize)
	}
	var docs docsJSON
	if err := json.Unmarshal([]byte(stdout), &docs); err != nil {
		t.Fatalf("docs --json is not JSON: %v", err)
	}
	documented := make(map[string]bool, len(docs.Commands))
	for _, c := range docs.Commands {
		documented[c.Path] = true
	}
	app := newApp(memstore.New(), env.NewMemoryStore(), nil)
	for _, path := range visiblePaths(app, "gurl") {
		if !documented[path] {
			t.Errorf("command %q missing from gurl docs --json", path)
		}
	}
	for _, key := range []string{"agent_mode", "placeholders", "scopes", "extract", "persist", "run_json", "exit_codes", "exec", "diag"} {
		if docs.Sections[key] == "" {
			t.Errorf("section %q missing", key)
		}
	}
}

func TestDocsNarrowsToOneCommand(t *testing.T) {
	setupMainTest(t)

	stdout, stderr, err := runWithArgs(t, []string{"gurl", "--agent", "docs", "collection", "run"})
	if err != nil {
		t.Fatalf("docs: %v\nstderr: %s", err, stderr)
	}
	var docs docsJSON
	if err := json.Unmarshal([]byte(stdout), &docs); err != nil {
		t.Fatalf("agent mode docs is not JSON: %v\n%s", err, stdout)
	}
	if len(docs.Commands) != 1 || docs.Commands[0].Path != "gurl collection run" || len(docs.Sections) != 0 {
		t.Fatalf("expected only gurl collection run, got %+v", docs)
	}

	text, _, err := runWithArgs(t, []string{"gurl", "docs", "history"})
	if err != nil {
		t.Fatalf("docs history: %v", err)
	}
	if !strings.HasPrefix(text, "gurl history") || !strings.Contains(text, "--limit") {
		t.Fatalf("unexpected text reference:\n%s", text)
	}
}

func TestAgentModeUsageErrorIsJSON(t *testing.T) {
	setupMainTest(t)

	stdout, _, err := runWithArgs(t, []string{"gurl", "--agent", "list", "--no-such-flag"})
	if err == nil {
		t.Fatal("expected an error for an unknown flag")
	}
	var got struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if jerr := json.Unmarshal([]byte(stdout), &got); jerr != nil {
		t.Fatalf("stdout is not a JSON error object: %v\n%s", jerr, stdout)
	}
	if got.OK || !strings.Contains(got.Error, "no-such-flag") {
		t.Fatalf("unexpected error object: %+v", got)
	}
}

func TestAgentModeFromEnvAndFlagPrintsJSON(t *testing.T) {
	for name, args := range map[string][]string{
		"flag before command": {"gurl", "--agent", "list"},
		"flag after command":  {"gurl", "list", "--agent"},
		"env var":             {"gurl", "list"},
	} {
		t.Run(name, func(t *testing.T) {
			setupMainTest(t)
			if name == "env var" {
				t.Setenv("GURL_AGENT", "1")
			}
			stdout, stderr, err := runWithArgs(t, args)
			if err != nil {
				t.Fatalf("list: %v\nstderr: %s", err, stderr)
			}
			var list []any
			if err := json.Unmarshal([]byte(stdout), &list); err != nil {
				t.Fatalf("list output is not JSON: %v\n%s", err, stdout)
			}
		})
	}
}
