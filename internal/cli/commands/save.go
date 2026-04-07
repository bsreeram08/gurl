package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/urfave/cli/v3"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
)

// SaveCommand creates the save command
func SaveCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "save",
		Aliases: []string{"s"},
		Usage:   "Save a curl request with a name",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "collection",
				Aliases: []string{"c"},
				Usage:   "Assign to collection",
			},
			&cli.StringFlag{
				Name:    "tag",
				Aliases: []string{"t"},
				Usage:   "Add tag (can repeat)",
			},
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Usage:   "Output format preference (auto|json|table)",
				Value:   "auto",
			},
			&cli.StringFlag{
				Name:    "description",
				Aliases: []string{"d"},
				Usage:   "Human-readable description",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			args := c.Args()
			if args.Len() < 2 {
				return fmt.Errorf("name and URL arguments are required")
			}
			name := args.Get(0)
			url := args.Get(1)

			req := &types.SavedRequest{
				Name:         name,
				URL:          url,
				Method:       "GET",
				OutputFormat: c.String("format"),
				CreatedAt:    time.Now().Unix(),
				UpdatedAt:    time.Now().Unix(),
			}

			if err := db.SaveRequest(req); err != nil {
				return fmt.Errorf("failed to save request: %w", err)
			}

			fmt.Printf("✓ Saved request '%s'\n", name)
			return nil
		},
	}
}
