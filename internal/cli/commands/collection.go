package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/internal/runner"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/urfave/cli/v3"
)

// CollectionCommand creates the collection command
func CollectionCommand(db storage.DB, envStorage *env.EnvStorage) *cli.Command {
	return &cli.Command{
		Name:    "collection",
		Aliases: []string{"collections", "col", "c"},
		Usage:   "Manage collections",
		Commands: []*cli.Command{
			runner.CollectionRunCommand(db, envStorage),
			{
				Name:    "list",
				Aliases: []string{"ls", "l"},
				Usage:   "List all collections",
				Action: func(ctx context.Context, c *cli.Command) error {
					requests, err := db.ListRequests(&storage.ListOptions{})
					if err != nil {
						return fmt.Errorf("failed to list requests: %w", err)
					}

					// Group by collection
					collections := make(map[string][]*types.SavedRequest)
					for _, req := range requests {
						if req.Collection != "" {
							collections[req.Collection] = append(collections[req.Collection], req)
						}
					}

					if len(collections) == 0 {
						fmt.Println("No collections found.")
						return nil
					}

					fmt.Println("┌─ Collections ────────────────────────────────────────────┐")
					fmt.Println("│  NAME          REQUESTS   UPDATED                        │")
					fmt.Println("├──────────────────────────────────────────────────────────┤")

					for name, reqs := range collections {
						count := len(reqs)
						latest := time.Unix(0, 0)
						for _, req := range reqs {
							if req.UpdatedAt > latest.Unix() {
								latest = time.Unix(req.UpdatedAt, 0)
							}
						}
						updated := latest.Format("2006-01-02 15:04")

						fmt.Printf("│  %-13s %-9d %s\n", name, count, updated)
					}

					fmt.Println("└──────────────────────────────────────────────────────────┘")

					return nil
				},
			},
			{
				Name:    "add",
				Aliases: []string{"create", "new"},
				Usage:   "Create a new collection",
				Action: func(ctx context.Context, c *cli.Command) error {
					args := c.Args()
					if args.Len() < 1 {
						return fmt.Errorf("collection name argument is required")
					}
					// Collections are created implicitly when saving requests
					// This command is for consistency
					name := args.Get(0)
					fmt.Printf("✓ Collection '%s' created (collections are created when saving requests)\n", name)
					return nil
				},
			},
			{
				Name:    "remove",
				Aliases: []string{"rm", "delete", "del"},
				Usage:   "Remove a collection",
				Action: func(ctx context.Context, c *cli.Command) error {
					args := c.Args()
					if args.Len() < 1 {
						return fmt.Errorf("collection name argument is required")
					}
					name := args.Get(0)

					requests, err := db.ListRequests(&storage.ListOptions{
						Collection: name,
					})
					if err != nil {
						return fmt.Errorf("failed to list collection: %w", err)
					}

					for _, req := range requests {
						req.Collection = ""
						if err := db.UpdateRequest(req); err != nil {
							fmt.Printf("⚠ Failed to update request '%s': %v\n", req.Name, err)
						}
					}

					fmt.Printf("✓ Removed collection '%s' (%d requests unassigned)\n", name, len(requests))
					return nil
				},
			},
			{
				Name:    "rename",
				Aliases: []string{"mv", "ren"},
				Usage:   "Rename a collection",
				Action: func(ctx context.Context, c *cli.Command) error {
					args := c.Args()
					if args.Len() < 2 {
						return fmt.Errorf("both old and new name arguments are required")
					}
					oldName := args.Get(0)
					newName := args.Get(1)

					requests, err := db.ListRequests(&storage.ListOptions{
						Collection: oldName,
					})
					if err != nil {
						return fmt.Errorf("failed to list collection: %w", err)
					}

					for _, req := range requests {
						req.Collection = newName
						if err := db.UpdateRequest(req); err != nil {
							fmt.Printf("⚠ Failed to update request '%s': %v\n", req.Name, err)
						}
					}

					fmt.Printf("✓ Renamed collection '%s' to '%s' (%d requests updated)\n", oldName, newName, len(requests))
					return nil
				},
			},
		},
	}
}
