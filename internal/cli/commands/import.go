package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sreeram/gurl/internal/importers"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/urfave/cli/v3"
)

// ImportCommand creates the import command
func ImportCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:      "import",
		Aliases:   []string{"imp"},
		Usage:     "Import requests from external formats (OpenAPI, Insomnia, Bruno, Postman, HAR)",
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
			&cli.StringFlag{
				Name:  "passphrase",
				Usage: "Passphrase for decrypting native collection exports",
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
				fmt.Println("  .gurl        - Native gurl request or collection export format")
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

			cleanPath := filepath.Clean(path)
			resolvedPath, err := filepath.EvalSymlinks(cleanPath)
			if err != nil {
				resolvedPath = cleanPath
			}
			if strings.Contains(resolvedPath, "..") {
				return fmt.Errorf("import path must not contain '..': %s", path)
			}

			if imported, err := importNativeCollectionExport(db, c, path); err != nil {
				return err
			} else if imported {
				return nil
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
					fmt.Printf("✗ ERROR: Failed to import '%s': %v\n", req.Name, err)
					continue
				}

				imported++
			}

			fmt.Printf("✓ Imported %d request(s) (%d skipped)\n", imported, skipped)
			return nil
		},
	}
}

type collectionExportProbe struct {
	Collection *types.Collection `json:"collection"`
}

func importNativeCollectionExport(db storage.DB, c *cli.Command, path string) (bool, error) {
	if filepath.Ext(path) != ".gurl" {
		return false, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("failed to read native export: %w", err)
	}
	var probe collectionExportProbe
	if err := json.Unmarshal(data, &probe); err != nil || probe.Collection == nil {
		return false, nil
	}

	collection, requests, err := storage.ParseCollectionExport(data, importCommandPassphrase(c))
	if err != nil {
		return true, err
	}
	imported, skipped, err := importCollectionRecords(db, collection, requests, c.Bool("force"))
	if err != nil {
		return true, err
	}
	fmt.Printf("✓ Imported collection '%s' (%d requests, %d skipped)\n", collection.Name, imported, skipped)
	return true, nil
}

func importCommandPassphrase(c *cli.Command) string {
	if passphrase := c.String("passphrase"); passphrase != "" {
		return passphrase
	}
	return os.Getenv("GURL_IMPORT_PASSPHRASE")
}

func importCollectionRecords(db storage.DB, collection *types.Collection, requests []*types.SavedRequest, force bool) (int, int, error) {
	if collection == nil {
		return 0, 0, fmt.Errorf("collection export is missing collection metadata")
	}

	allowLockedSave := false
	if store, ok := db.(storage.CollectionStore); ok {
		existing, err := store.GetCollectionByName(collection.Name)
		if storage.IsCollectionLocked(err) {
			if !force {
				return 0, 0, fmt.Errorf("collection %q already exists (use --force to overwrite)", collection.Name)
			}
			rawStore, ok := db.(rawCollectionByNameStore)
			if !ok {
				return 0, 0, fmt.Errorf("failed to inspect locked collection %q: %w", collection.Name, err)
			}
			existing, err = rawStore.GetRawCollectionByName(collection.Name)
			if err != nil {
				return 0, 0, fmt.Errorf("failed to inspect locked collection %q: %w", collection.Name, err)
			}
			allowLockedSave = true
		}
		if err == nil && existing != nil {
			if !force {
				return 0, 0, fmt.Errorf("collection %q already exists (use --force to overwrite)", collection.Name)
			}
			collection.ID = existing.ID
			collection.CreatedAt = existing.CreatedAt
		}
	}
	if err := saveCollectionRecord(db, collection, allowLockedSave); err != nil {
		return 0, 0, err
	}

	imported := 0
	skipped := 0
	for _, req := range requests {
		req.Collection = collection.Name
		if existing, err := db.GetRequestByName(req.Name); err == nil && existing != nil {
			if !force {
				skipped++
				continue
			}
			req.ID = existing.ID
			if req.CreatedAt == 0 {
				req.CreatedAt = existing.CreatedAt
			}
			if err := db.DeleteRequest(existing.ID); err != nil {
				return 0, 0, fmt.Errorf("failed to overwrite request %q: %w", req.Name, err)
			}
		}
		if err := db.SaveRequest(req); err != nil {
			return 0, 0, fmt.Errorf("failed to import request %q: %w", req.Name, err)
		}
		imported++
	}
	return imported, skipped, nil
}
