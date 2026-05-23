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
		Name: "gurl",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "project-dir"},
		},
		Commands: []*cli.Command{InitCommand()},
	}

	if err := app.Run(context.Background(), []string{"gurl", "--project-dir", root, "init"}); err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	for _, path := range []string{
		filepath.Join(root, ".gurl", "collections"),
		filepath.Join(root, ".gurl", "environments"),
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
	if _, err := os.Stat(filepath.Join(root, ".gurl", "collections")); err != nil {
		t.Fatalf("expected collections directory: %v", err)
	}
}
