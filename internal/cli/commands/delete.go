package commands

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
	"github.com/sreeram/gurl/internal/storage"
)

// DeleteCommand creates the delete command
func DeleteCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "delete",
		Aliases: []string{"rm", "del", "d"},
		Usage:   "Remove a saved request",
		Action: func(ctx context.Context, c *cli.Command) error {
			args := c.Args()
			if args.Len() < 1 {
				return fmt.Errorf("request name argument is required")
			}
			name := args.Get(0)

			req, err := db.GetRequestByName(name)
			if err != nil {
				return fmt.Errorf("request not found: %s", name)
			}

			if err := db.DeleteRequest(req.ID); err != nil {
				return fmt.Errorf("failed to delete request: %w", err)
			}

			fmt.Printf("✓ Deleted request '%s'\n", name)
			return nil
		},
	}
}
