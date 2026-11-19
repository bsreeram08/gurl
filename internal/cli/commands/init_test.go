package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/urfave/cli/v3"
)

func TestInitCommandCreatesProjectAtGlobalProjectDir(t *testing.T) {
	root := t.TempDir()
	app := &cli.Command{
		Name: "valetkey",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "project-dir"},
		},
		Commands: []*cli.Command{InitCommand()},
	}

	if err := app.Run(context.Background(), []string{"valetkey", "--project-dir", root, "init"}); err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	for _, path := range []string{
		filepath.Join(root, ".valetkey", "collections"),
		filepath.Join(root, ".valetkey", "environments"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}
}

func TestInitCommandAcceptsLocalProjectDirFlag(t *testing.T) {
	root := t.TempDir()
	cmd := InitCommand()

	if err := cmd.Run(context.Background(), []string{"init", "--project-dir", root}); err != nil {
		t.Fatalf("init command failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".valetkey", "collections")); err != nil {
		t.Fatalf("expected collections directory: %v", err)
	}
}

func TestInitKeepsExistingLegacyGurlDir(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".gurl", "collections"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := InitCommand().Run(context.Background(), []string{"init", "--project-dir", root}); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".valetkey")); !os.IsNotExist(err) {
		t.Fatalf("init should reuse .gurl, not create .valetkey next to it (stat err: %v)", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".gurl", "environments")); err != nil {
		t.Fatalf("expected init to fill in the legacy .gurl dir: %v", err)
	}
}
