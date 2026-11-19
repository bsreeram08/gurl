package brand

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDataDirPrefersNewThenLegacy(t *testing.T) {
	home := t.TempDir()

	// Fresh install: neither exists, so the new directory is used.
	if got, want := DataDir(home), NewDataDir(home); got != want {
		t.Fatalf("fresh install: got %s, want %s", got, want)
	}

	// Pre-rename install: only the gurl directory exists and is kept.
	if err := os.MkdirAll(LegacyDataDir(home), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, want := DataDir(home), LegacyDataDir(home); got != want {
		t.Fatalf("legacy install: got %s, want %s", got, want)
	}
	if err := os.MkdirAll(filepath.Join(LegacyDataDir(home), "gurl.db"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, want := DBPath(DataDir(home)), filepath.Join(LegacyDataDir(home), "gurl.db"); got != want {
		t.Fatalf("legacy db path: got %s, want %s", got, want)
	}

	// Migrated by moving the directory: the new directory wins, and its
	// gurl.db is still found.
	if err := os.Rename(LegacyDataDir(home), NewDataDir(home)); err != nil {
		t.Fatal(err)
	}
	if got, want := DataDir(home), NewDataDir(home); got != want {
		t.Fatalf("migrated install: got %s, want %s", got, want)
	}
	if got, want := DBPath(DataDir(home)), filepath.Join(NewDataDir(home), "gurl.db"); got != want {
		t.Fatalf("moved db path: got %s, want %s", got, want)
	}
	if err := os.RemoveAll(filepath.Join(NewDataDir(home), "gurl.db")); err != nil {
		t.Fatal(err)
	}
	if got, want := DBPath(DataDir(home)), filepath.Join(NewDataDir(home), "valetkey.db"); got != want {
		t.Fatalf("new db path: got %s, want %s", got, want)
	}
	if got, want := CookieDBPath(DataDir(home)), filepath.Join(NewDataDir(home), "valetkey-cookies.db"); got != want {
		t.Fatalf("new cookie db path: got %s, want %s", got, want)
	}
}

func TestGetenvFallsBackToLegacy(t *testing.T) {
	t.Setenv("VALETKEY_DB_PATH", "")
	t.Setenv("GURL_DB_PATH", "/old")
	if got := Getenv("DB_PATH"); got != "/old" {
		t.Fatalf("got %q, want legacy value", got)
	}
	t.Setenv("VALETKEY_DB_PATH", "/new")
	if got := Getenv("DB_PATH"); got != "/new" {
		t.Fatalf("got %q, want new value to win", got)
	}
}

func TestIsAgentBinary(t *testing.T) {
	for name, want := range map[string]bool{
		"/usr/local/bin/agentvaletkey": true,
		"agentgurl":                    true,
		"AgentValetkey.exe":            true,
		"valetkey":                     false,
		"gurl":                         false,
	} {
		if got := IsAgentBinary(name); got != want {
			t.Errorf("IsAgentBinary(%q) = %v, want %v", name, got, want)
		}
	}
}
