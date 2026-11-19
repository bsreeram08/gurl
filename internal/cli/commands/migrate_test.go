package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bsreeram08/apigent/internal/brand"
)

func TestMigrationMovesLegacyLocationsAndKeepsData(t *testing.T) {
	home := t.TempDir()
	legacyDB := filepath.Join(brand.LegacyDataDir(home), "gurl.db")
	if err := os.MkdirAll(legacyDB, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(brand.LegacyDataDir(home), ".secret-key"), []byte("k"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(brand.LegacyConfigDir(home), 0o755); err != nil {
		t.Fatal(err)
	}
	// The new config dir already exists, so config must be left alone.
	if err := os.MkdirAll(brand.NewConfigDir(home), 0o755); err != nil {
		t.Fatal(err)
	}
	projRoot := t.TempDir()
	legacyProj := filepath.Join(projRoot, ".gurl")
	if err := os.Mkdir(legacyProj, 0o755); err != nil {
		t.Fatal(err)
	}

	steps := planMigration(home, legacyProj)
	got := map[string]string{}
	for _, s := range steps {
		got[s.What] = s.Status
	}
	want := map[string]string{"data": "move", "config": "destination-exists", "plugins": "nothing-to-move", "project": "move"}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("plan %s = %q, want %q", k, got[k], v)
		}
	}

	if err := applyMigration(steps); err != nil {
		t.Fatal(err)
	}
	if brand.DataDir(home) != brand.NewDataDir(home) {
		t.Fatalf("data dir should now be the new one")
	}
	// The moved gurl.db and secret key are found in the new location.
	if got := brand.DBPath(brand.DataDir(home)); got != filepath.Join(brand.NewDataDir(home), "gurl.db") {
		t.Fatalf("db path after migrate = %s", got)
	}
	if _, err := os.Stat(filepath.Join(brand.NewDataDir(home), ".secret-key")); err != nil {
		t.Fatalf("secret key should move with the data: %v", err)
	}
	if !pathExists(brand.LegacyConfigDir(home)) {
		t.Fatal("config with an existing destination must not move")
	}
	if !pathExists(filepath.Join(projRoot, ".apigent")) || pathExists(legacyProj) {
		t.Fatal("project .gurl should be renamed to .apigent")
	}
}

func TestMigrationIgnoresProjectAlreadyOnNewName(t *testing.T) {
	for _, s := range planMigration(t.TempDir(), filepath.Join(t.TempDir(), ".apigent")) {
		if s.What == "project" {
			t.Fatalf("a .apigent project needs no step, got %+v", s)
		}
	}
}
