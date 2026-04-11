package commands

import (
	"context"
	"fmt"
	"os"

	"charm.land/bubbletea/v2"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/internal/tui"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/urfave/cli/v3"
)

// TUICommand creates the TUI command
func TUICommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "tui",
		Aliases: []string{"ui"},
		Usage:   "Launch the interactive TUI",
		Description: `gurl tui launches the interactive terminal user interface for managing
and executing your saved API requests in a 3-panel layout.`,
		Action: func(ctx context.Context, c *cli.Command) error {
			// Load config (using defaults for now since config loading is TBD)
			config := &types.Config{}

			// Create the TUI app
			app := tui.NewApp(db, config)

			// Run the bubbletea program
			p := tea.NewProgram(app)

			if _, err := p.Run(); err != nil {
				return fmt.Errorf("failed to run TUI: %w", err)
			}

			return nil
		},
	}
}

// RunTUI is a helper to launch the TUI programmatically
func RunTUI(db storage.DB) error {
	config := &types.Config{}
	app := tui.NewApp(db, config)

	p := tea.NewProgram(app)

	_, err := p.Run()
	return err
}

// ExitWithTUI runs the TUI and exits with appropriate code
func ExitWithTUI(db storage.DB) {
	config := &types.Config{}
	app := tui.NewApp(db, config)

	p := tea.NewProgram(app)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
