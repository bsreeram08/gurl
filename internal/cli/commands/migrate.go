package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bsreeram08/apigent/internal/brand"
	"github.com/bsreeram08/apigent/internal/project"
	"github.com/urfave/cli/v3"
)

// migrationStep moves one pre-rename location to its apigent name.
type migrationStep struct {
	What   string `json:"what"`
	From   string `json:"from"`
	To     string `json:"to"`
	Status string `json:"status"` // move, moved, nothing-to-move, destination-exists
}

// planMigration lists the moves for home and, if projectDir is a legacy
// .gurl directory, for that project. It touches nothing.
func planMigration(home, projectDir string) []migrationStep {
	steps := []migrationStep{
		{What: "data", From: brand.LegacyDataDir(home), To: brand.NewDataDir(home)},
		{What: "config", From: brand.LegacyConfigDir(home), To: brand.NewConfigDir(home)},
		{What: "plugins", From: brand.LegacyPluginDir(home), To: brand.NewPluginDir(home)},
	}
	if projectDir != "" && filepath.Base(projectDir) == brand.LegacyProjectDirName {
		steps = append(steps, migrationStep{
			What: "project",
			From: projectDir,
			To:   filepath.Join(filepath.Dir(projectDir), brand.ProjectDirName),
		})
	}
	for i := range steps {
		switch {
		case !pathExists(steps[i].From):
			steps[i].Status = "nothing-to-move"
		case pathExists(steps[i].To):
			steps[i].Status = "destination-exists"
		default:
			steps[i].Status = "move"
		}
	}
	return steps
}

// applyMigration renames every step marked "move". A rename is atomic, so
// each location is either fully old or fully new; the lookups in package
// brand read either, so stopping partway loses nothing.
func applyMigration(steps []migrationStep) error {
	for i := range steps {
		if steps[i].Status != "move" {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(steps[i].To), 0o755); err != nil {
			return fmt.Errorf("migrating %s: %w", steps[i].What, err)
		}
		if err := os.Rename(steps[i].From, steps[i].To); err != nil {
			return fmt.Errorf("migrating %s from %s to %s: %w", steps[i].What, steps[i].From, steps[i].To, err)
		}
		steps[i].Status = "moved"
	}
	return nil
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// MigrateCommand moves data, config, plugins and the current project from
// their pre-rename gurl locations to the apigent ones.
func MigrateCommand() *cli.Command {
	return &cli.Command{
		Name:  "migrate",
		Usage: "Move data, config, plugins and the current project from gurl locations to apigent ones",
		Description: `Moves ~/.local/share/gurl, ~/.config/gurl, ~/.gurl/plugins and the current
project's .gurl directory to their apigent names. Nothing is copied or
deleted, and a location whose new name already exists is left alone.
apigent reads the old locations until you migrate, so this is optional.
Releases of gurl before the rename do not read the new locations.`,
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "dry-run", Usage: "Show what would move without moving it"},
			&cli.BoolFlag{Name: "json", Aliases: []string{"j"}, Usage: "JSON output"},
		},
		Action: jsonAction(func(ctx context.Context, c *cli.Command) error {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}
			var projectDir string
			if proj, err := project.Discover("", c.String("project-dir")); err == nil && proj != nil {
				projectDir = proj.Dir
			}

			steps := planMigration(home, projectDir)
			dryRun := c.Bool("dry-run")
			applyErr := error(nil)
			if !dryRun {
				applyErr = applyMigration(steps)
			}

			if jsonOutput(c) {
				if err := writeJSON(os.Stdout, struct {
					OK     bool            `json:"ok"`
					DryRun bool            `json:"dry_run"`
					Steps  []migrationStep `json:"steps"`
					Error  string          `json:"error,omitempty"`
				}{OK: applyErr == nil, DryRun: dryRun, Steps: steps, Error: errString(applyErr)}); err != nil {
					return err
				}
				return markReported(applyErr)
			}

			for _, s := range steps {
				switch s.Status {
				case "move":
					fmt.Printf("would move %-8s %s -> %s\n", s.What, s.From, s.To)
				case "moved":
					fmt.Printf("✓ moved %-8s %s -> %s\n", s.What, s.From, s.To)
				case "destination-exists":
					fmt.Printf("- kept   %-8s %s (%s already exists)\n", s.What, s.From, s.To)
				}
			}
			return applyErr
		}),
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
