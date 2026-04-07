package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/urfave/cli/v3"
	"github.com/sreeram/gurl/internal/storage"
)

// RenameCommand creates the rename command
func RenameCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "rename",
		Aliases: []string{"mv", "ren"},
		Usage:   "Rename a saved request",
		Action: func(ctx context.Context, c *cli.Command) error {
			args := c.Args()
			if args.Len() < 2 {
				return fmt.Errorf("both old and new name arguments are required")
			}
			oldName := args.Get(0)
			newName := args.Get(1)

			req, err := db.GetRequestByName(oldName)
			if err != nil {
				return fmt.Errorf("request not found: %s", oldName)
			}

			req.Name = newName
			req.UpdatedAt = time.Now().Unix()

			if err := db.UpdateRequest(req); err != nil {
				return fmt.Errorf("failed to rename request: %w", err)
			}

			fmt.Printf("✓ Renamed request '%s' to '%s'\n", oldName, newName)
			return nil
		},
	}
}
