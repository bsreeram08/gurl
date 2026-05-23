package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitCreatesProjectLayout(t *testing.T) {
	root := t.TempDir()

	proj, err := Init(root)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	for _, path := range []string{
		proj.CollectionsDir(),
		proj.EnvironmentsDir(),
		filepath.Join(proj.GurlDir, ".gitignore"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}
}

func TestDiscoverWalksUpFromNestedDirectory(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	nested := filepath.Join(root, "api", "v1")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}

	proj, err := Discover(nested, "")
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	if proj == nil || proj.Root != root {
		t.Fatalf("expected project root %s, got %+v", root, proj)
	}
}

func TestDiscoverUsesEnvOverride(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	t.Setenv(EnvProjectDir, root)

	proj, err := Discover(t.TempDir(), "")
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	if proj == nil || proj.Root != root {
		t.Fatalf("expected env project root %s, got %+v", root, proj)
	}
}
