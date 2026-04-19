package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/sreeram/gurl/internal/storage"
	"github.com/urfave/cli/v3"
)

const removedTUIMessage = "gurl tui has been removed. Use `gurl shell` for the new typed interactive flow."

// TUICommand is kept only as a hidden compatibility stub.
func TUICommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "tui",
		Aliases: []string{"ui"},
		Usage:   "Removed; use `gurl shell` instead",
		Hidden:  true,
		Action: func(ctx context.Context, c *cli.Command) error {
			return fmt.Errorf(removedTUIMessage)
		},
	}
}

// RunTUI is a compatibility stub.
func RunTUI(db storage.DB) error {
	return fmt.Errorf(removedTUIMessage)
}

// ExitWithTUI exits with the compatibility message.
func ExitWithTUI(db storage.DB) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", removedTUIMessage)
	os.Exit(1)
}
