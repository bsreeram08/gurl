package commands

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
	"github.com/sreeram/gurl/internal/storage"
)

// DetectCommand creates the detect command
func DetectCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "detect",
		Aliases: []string{"parse", "d"},
		Usage:   "Parse curl from stdin or file",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "file",
				Aliases: []string{"f"},
				Usage:   "Read from file instead of stdin",
			},
			&cli.StringFlag{
				Name:    "name",
				Aliases: []string{"n"},
				Usage:   "Name for the detected request",
			},
			&cli.StringFlag{
				Name:    "collection",
				Aliases: []string{"c"},
				Usage:   "Assign to collection",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			// TODO: Implement curl parsing from stdin/file
			// For now, show a message that this feature is being developed
			fmt.Println("┌─ Detect ────────────────────────────────────────────────┐")
			fmt.Println("│                                                         │")
			fmt.Println("│  Detect feature is under development                     │")
			fmt.Println("│                                                         │")
			fmt.Println("│  Currently supported:                                   │")
			fmt.Println("│    - Save requests with: scurl save <name> <url>        │")
			fmt.Println("│    - Parse curl: Coming in Phase 2                     │")
			fmt.Println("│                                                         │")
			fmt.Println("└─────────────────────────────────────────────────────────┘")

			return nil
		},
	}
}
