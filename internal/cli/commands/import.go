package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
	"github.com/sreeram/gurl/internal/importers"
	"github.com/sreeram/gurl/internal/storage"
)

// ImportCommand creates the import command
func ImportCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "import",
		Aliases: []string{"imp"},
		Usage:   "Import requests from external formats (OpenAPI, Insomnia, Bruno, Postman, HAR)",
		ArgsUsage: "<path>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Overwrite existing requests with the same name",
			},
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"fmt"},
				Usage:   "Force specific format (auto-detected if not specified)",
			},
			&cli.BoolFlag{
				Name:    "list",
				Aliases: []string{"l"},
				Usage:   "List all supported formats",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			// List supported formats
			if c.Bool("list") {
				fmt.Println("Supported formats:")
				fmt.Println("  .yaml, .yml  - OpenAPI/Swagger 3.x")
				fmt.Println("  .json        - Insomnia, Postman, OpenAPI, HAR")
				fmt.Println("  .bru         - Bruno")
				fmt.Println("  .har         - HAR (HTTP Archive)")
				return nil
			}

			// Get file path
			path := c.Args().First()
			if path == "" {
				return fmt.Errorf("file path is required")
			}

			// Check if file exists
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return fmt.Errorf("file not found: %s", path)
			}

			// Import requests
			requests, err := importers.Import(path)
			if err != nil {
				return fmt.Errorf("import failed: %w", err)
			}

			if len(requests) == 0 {
				fmt.Println("No requests found to import")
				return nil
			}

			// Save imported requests
			imported := 0
			skipped := 0

			for _, req := range requests {
				// Check if request already exists
				if !c.Bool("force") {
					if existing, err := db.GetRequestByName(req.Name); err == nil && existing != nil {
						fmt.Printf("⚠ Skipped '%s' (already exists, use --force to overwrite)\n", req.Name)
						skipped++
						continue
					}
				}

				if err := db.SaveRequest(req); err != nil {
					fmt.Printf("⚠ Failed to import '%s': %v\n", req.Name, err)
					continue
				}

				imported++
			}

			fmt.Printf("✓ Imported %d request(s) (%d skipped)\n", imported, skipped)
			return nil
		},
	}
}
