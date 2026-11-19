package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/bsreeram08/valetkey/internal/storage"
	"github.com/urfave/cli/v3"
)

const removedTUIMessage = "valetkey tui has been removed. Use `valetkey shell` for the new typed interactive flow."

// TUICommand is kept only as a hidden compatibility stub.
func TUICommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "tui",
		Aliases: []string{"ui"},
		Usage:   "Removed; use `valetkey shell` instead",
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
