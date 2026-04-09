package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/urfave/cli/v3"
)

// ExportCommand creates the export command
func ExportCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "export",
		Aliases: []string{"exp"},
		Usage:   "Export requests to file",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "name",
				Aliases: []string{"n"},
				Usage:   "Export specific request by name",
			},
			&cli.StringFlag{
				Name:    "collection",
				Aliases: []string{"c"},
				Usage:   "Export entire collection",
			},
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
				Usage:   "Export all requests",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output file (default: stdout)",
			},
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"fmt"},
				Usage:   "Export format (gurl)",
				Value:   "gurl",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			var requests []*types.SavedRequest
			var err error

			if c.String("name") != "" {
				req, err := db.GetRequestByName(c.String("name"))
				if err != nil {
					return fmt.Errorf("request not found: %w", err)
				}
				requests = []*types.SavedRequest{req}
			} else if c.String("collection") != "" {
				requests, err = db.ListRequests(&storage.ListOptions{
					Collection: c.String("collection"),
				})
				if err != nil {
					return fmt.Errorf("failed to list collection: %w", err)
				}
			} else if c.Bool("all") {
				requests, err = db.ListRequests(&storage.ListOptions{})
				if err != nil {
					return fmt.Errorf("failed to list requests: %w", err)
				}
			} else {
				return fmt.Errorf("specify --name, --collection, or --all")
			}

			exportData := struct {
				Version    string                `json:"version"`
				ExportedAt string                `json:"exported_at"`
				Requests   []*types.SavedRequest `json:"requests"`
			}{
				Version:    "1.0",
				ExportedAt: time.Now().UTC().Format(time.RFC3339),
				Requests:   requests,
			}

			data, err := json.MarshalIndent(exportData, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal export: %w", err)
			}

			outputPath := c.String("output")
			if outputPath != "" {
				if err := validateOutputPath(outputPath); err != nil {
					return fmt.Errorf("invalid output path: %w", err)
				}
				if err := os.WriteFile(outputPath, data, 0644); err != nil {
					return fmt.Errorf("failed to write output file: %w", err)
				}
				fmt.Printf("✓ Exported %d request(s) to %s\n", len(requests), outputPath)
			} else {
				fmt.Println(string(data))
			}

			return nil
		},
	}
}

func validateOutputPath(outputPath string) error {
	cleaned := filepath.Clean(outputPath)
	resolvedPath, err := filepath.EvalSymlinks(cleaned)
	if err != nil {
		resolvedPath = cleaned
	}
	if strings.Contains(resolvedPath, "..") {
		return fmt.Errorf("output path must not contain '..': %s", outputPath)
	}
	return nil
}
