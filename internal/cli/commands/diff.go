package commands

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
	"github.com/sreeram/gurl/internal/storage"
)

// DiffCommand creates the diff command
func DiffCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "diff",
		Aliases: []string{"d"},
		Usage:   "Compare responses for a request",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "limit",
				Aliases: []string{"n", "l"},
				Usage:   "Number of responses to compare",
				Value:   2,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			args := c.Args()
			if args.Len() < 1 {
				return fmt.Errorf("request name argument is required")
			}
			name := args.Get(0)
			limit := c.Int("limit")

			req, err := db.GetRequestByName(name)
			if err != nil {
				return fmt.Errorf("request not found: %s", name)
			}

			history, err := db.GetHistory(req.ID, limit)
			if err != nil {
				return fmt.Errorf("failed to get history: %w", err)
			}

			if len(history) < 2 {
				return fmt.Errorf("not enough execution history to diff (need at least 2)")
			}

			// For now, just show a simple diff view
			// Full diff implementation would require response storage
			fmt.Printf("┌─ Diff: %s (Last %d executions) ──────────────────────┐\n", name, len(history))
			fmt.Println("│                                                         │")
			fmt.Println("│  Note: Full response diff requires response storage    │")
			fmt.Println("│  Currently showing execution metadata:                  │")
			fmt.Println("│                                                         │")

			for i, h := range history {
				timestamp := fmt.Sprintf("%d", h.Timestamp)
				fmt.Printf("│  Response #%d: status=%d, duration=%dms, time=%s\n",
					i+1, h.StatusCode, h.DurationMs, timestamp)
			}

			fmt.Println("│                                                         │")
			fmt.Println("└─────────────────────────────────────────────────────────┘")

			return nil
		},
	}
}
