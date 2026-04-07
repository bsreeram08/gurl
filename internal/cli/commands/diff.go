package commands

import (
	"context"
	"fmt"

	"github.com/sreeram/gurl/internal/formatter"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/urfave/cli/v3"
)

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

			histA := history[0]
			histB := history[1]

			diff, err := formatter.DiffResponses(*histA, *histB)
			if err != nil {
				return fmt.Errorf("failed to diff responses: %w", err)
			}

			fmt.Println(diff)
			return nil
		},
	}
}
