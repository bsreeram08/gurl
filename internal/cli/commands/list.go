package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/urfave/cli/v3"
	"github.com/sreeram/gurl/internal/storage"
)

// ListCommand creates the list command
func ListCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls", "l"},
		Usage:   "List saved requests",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "pattern",
				Aliases: []string{"p"},
				Usage:   "Filter by name pattern",
			},
			&cli.StringFlag{
				Name:    "collection",
				Aliases: []string{"c"},
				Usage:   "Filter by collection",
			},
			&cli.StringFlag{
				Name:    "tag",
				Aliases: []string{"t"},
				Usage:   "Filter by tag",
			},
			&cli.BoolFlag{
				Name:    "json",
				Aliases: []string{"j"},
				Usage:   "JSON output",
			},
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Usage:   "Output format (table|list)",
				Value:   "table",
			},
			&cli.IntFlag{
				Name:    "limit",
				Aliases: []string{"n"},
				Usage:   "Limit number of results",
				Value:   0,
			},
			&cli.StringFlag{
				Name:    "sort",
				Aliases: []string{"s"},
				Usage:   "Sort by (name|updated|collection)",
				Value:   "updated",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			opts := &storage.ListOptions{
				Collection: c.String("collection"),
				Tag:        c.String("tag"),
				Pattern:    c.String("pattern"),
				Limit:      c.Int("limit"),
				Sort:       c.String("sort"),
			}

			requests, err := db.ListRequests(opts)
			if err != nil {
				return fmt.Errorf("failed to list requests: %w", err)
			}

			if c.Bool("json") {
				data, err := json.MarshalIndent(requests, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal JSON: %w", err)
				}
				fmt.Println(string(data))
				return nil
			}

			if len(requests) == 0 {
				fmt.Println("No saved requests found.")
				return nil
			}

			// Print table header
			fmt.Println("┌─ Saved Requests ─────────────────────────────────────────┐")
			fmt.Println("│  NAME                  COLLECTION    TAGS      UPDATED  │")
			fmt.Println("├─────────────────────────────────────────────────────────┤")

			for _, req := range requests {
				name := req.Name
				if len(name) > 20 {
					name = name[:17] + "..."
				}
				collection := req.Collection
				if collection == "" {
					collection = "-"
				}
				if len(collection) > 10 {
					collection = collection[:7] + "..."
				}
				tags := strings.Join(req.Tags, ",")
				if tags == "" {
					tags = "-"
				}
				if len(tags) > 10 {
					tags = tags[:7] + "..."
				}

				updated := time.Unix(req.UpdatedAt, 0).Format("2006-01-02 15:04")

				fmt.Printf("│  %-20s %-12s %-10s %s\n", name, collection, tags, updated)
			}

			fmt.Println("└─────────────────────────────────────────────────────────┘")
			fmt.Printf("  %d requests\n", len(requests))

			return nil
		},
	}
}
