package commands

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
	"github.com/sreeram/gurl/internal/storage"
)

// EditCommand creates the edit command
func EditCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "edit",
		Aliases: []string{"e"},
		Usage:   "Edit a saved request",
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

			fmt.Printf("┌─ Edit: %s ──────────────────────────────────────────────┐\n", name)
			fmt.Println("│                                                           │")
			fmt.Printf("│  URL:      %s\n", req.URL)
			fmt.Printf("│  Method:   %s\n", req.Method)
			fmt.Printf("│  Collection: %s\n", req.Collection)
			fmt.Println("│                                                           │")
			fmt.Println("│  Edit feature is under development (TUI mode)            │")
			fmt.Println("│                                                           │")
			fmt.Println("└───────────────────────────────────────────────────────────┘")

			return nil
		},
	}
}
