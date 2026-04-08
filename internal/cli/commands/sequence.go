package commands

import (
	"context"
	"fmt"
	"strconv"

	"github.com/sreeram/gurl/internal/runner"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/urfave/cli/v3"
)

func SequenceCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "sequence",
		Aliases: []string{"seq"},
		Usage:   "Manage request execution order",
		Commands: []*cli.Command{
			{
				Name:    "set",
				Aliases: []string{"s"},
				Usage:   "Set execution order for a request",
				Action: func(ctx context.Context, c *cli.Command) error {
					args := c.Args()
					if args.Len() < 2 {
						return fmt.Errorf("request name and order number required")
					}
					requestName := args.Get(0)
					orderStr := args.Get(1)
					order, err := strconv.Atoi(orderStr)
					if err != nil {
						return fmt.Errorf("invalid order number: %s", orderStr)
					}
					if order < 0 {
						return fmt.Errorf("order must be >= 0")
					}
					if err := runner.SetSortOrder(db, requestName, order); err != nil {
						return fmt.Errorf("failed to set sort order: %w", err)
					}
					fmt.Printf("✓ Set %s order to %d\n", requestName, order)
					return nil
				},
			},
			{
				Name:    "list",
				Aliases: []string{"ls", "l"},
				Usage:   "List requests in execution order",
				Action: func(ctx context.Context, c *cli.Command) error {
					args := c.Args()
					if args.Len() < 1 {
						return fmt.Errorf("collection name required")
					}
					collection := args.Get(0)
					reqs, err := runner.GetSequence(db, collection)
					if err != nil {
						return fmt.Errorf("failed to get sequence: %w", err)
					}
					if len(reqs) == 0 {
						fmt.Printf("Collection '%s' is empty or does not exist\n", collection)
						return nil
					}
					fmt.Printf("┌─ %s Sequence ──────────────────────────────────────┐\n", collection)
					fmt.Printf("│  #   REQUEST                    ORDER                │\n")
					fmt.Printf("├─────────────────────────────────────────────────────┤\n")
					for i, req := range reqs {
						name := req.Name
						if len(name) > 22 {
							name = name[:19] + "..."
						}
						orderStr := strconv.Itoa(req.SortOrder)
						if req.SortOrder == 0 {
							orderStr = "(auto)"
						}
						fmt.Printf("│  %-3d %-24s %-12s\n", i+1, name, orderStr)
					}
					fmt.Printf("└─────────────────────────────────────────────────────┘\n")
					return nil
				},
			},
		},
	}
}
