package commands

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/internal/runner"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/urfave/cli/v3"
)

// CollectionCommand creates the collection command.
func CollectionCommand(db storage.DB, envStorage *env.EnvStorage) *cli.Command {
	return &cli.Command{
		Name:    "collection",
		Aliases: []string{"collections", "col", "c"},
		Usage:   "Manage collections",
		Commands: []*cli.Command{
			runner.CollectionRunCommand(db, envStorage),
			collectionShowCommand(db),
			collectionListCommand(db),
			collectionCreateCommand(db),
			collectionSetVarCommand(db),
			collectionUnsetVarCommand(db),
			collectionRemoveCommand(db),
			collectionRenameCommand(db),
		},
	}
}

func collectionShowCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "show",
		Aliases: []string{"view", "info"},
		Usage:   "Show requests and variables in a collection",
		Action: func(ctx context.Context, c *cli.Command) error {
			args := c.Args()
			if args.Len() < 1 {
				return fmt.Errorf("collection name argument is required")
			}
			name := args.Get(0)

			collection, _ := loadCollectionByName(db, name)
			requests, err := db.ListRequests(&storage.ListOptions{Collection: name})
			if err != nil {
				return fmt.Errorf("failed to list collection: %w", err)
			}
			if collection == nil && len(requests) == 0 {
				return fmt.Errorf("collection '%s' not found or empty", name)
			}

			sortRequests(requests)

			fmt.Printf("Collection: %s\n", name)
			if collection != nil {
				fmt.Printf("ID:         %s\n", collection.ID)
			}
			fmt.Printf("Requests:   %d\n", len(requests))
			printCollectionVariables(collection)
			fmt.Println()
			for _, req := range requests {
				method := req.Method
				if method == "" {
					method = "GET"
				}
				fmt.Printf("- %s\n", req.Name)
				fmt.Printf("  Method: %s\n", method)
				fmt.Printf("  URL:    %s\n", req.URL)
				if req.Folder != "" {
					fmt.Printf("  Folder: %s\n", req.Folder)
				}
			}
			return nil
		},
	}
}

func collectionListCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls", "l"},
		Usage:   "List all collections",
		Action: func(ctx context.Context, c *cli.Command) error {
			requests, err := db.ListRequests(&storage.ListOptions{})
			if err != nil {
				return fmt.Errorf("failed to list requests: %w", err)
			}

			stats := make(map[string]collectionListRow)
			for _, req := range requests {
				if req.Collection == "" {
					continue
				}
				row := stats[req.Collection]
				row.Name = req.Collection
				row.Requests++
				if req.UpdatedAt > row.UpdatedAt {
					row.UpdatedAt = req.UpdatedAt
				}
				stats[req.Collection] = row
			}

			if store, ok := db.(storage.CollectionStore); ok {
				collections, err := store.ListCollections()
				if err != nil {
					return fmt.Errorf("failed to list collections: %w", err)
				}
				for _, collection := range collections {
					row := stats[collection.Name]
					row.Name = collection.Name
					row.Variables = len(collection.Variables)
					if row.UpdatedAt == 0 || collection.UpdatedAt > row.UpdatedAt {
						row.UpdatedAt = collection.UpdatedAt
					}
					stats[collection.Name] = row
				}
			}

			if len(stats) == 0 {
				fmt.Println("No collections found.")
				return nil
			}

			rows := make([]collectionListRow, 0, len(stats))
			for _, row := range stats {
				rows = append(rows, row)
			}
			sort.SliceStable(rows, func(i, j int) bool {
				return rows[i].Name < rows[j].Name
			})

			fmt.Println("┌─ Collections ────────────────────────────────────────────┐")
			fmt.Println("│  NAME          REQUESTS   VARIABLES   UPDATED           │")
			fmt.Println("├──────────────────────────────────────────────────────────┤")
			for _, row := range rows {
				updated := "-"
				if row.UpdatedAt > 0 {
					updated = time.Unix(row.UpdatedAt, 0).Format("2006-01-02 15:04")
				}
				fmt.Printf("│  %-13s %-9d %-11d %s\n", row.Name, row.Requests, row.Variables, updated)
			}
			fmt.Println("└──────────────────────────────────────────────────────────┘")

			return nil
		},
	}
}

func collectionCreateCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "create",
		Aliases: []string{"add", "new"},
		Usage:   "Create a new collection",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:    "var",
				Aliases: []string{"v"},
				Usage:   "Variable in KEY=VALUE format (can repeat)",
			},
			&cli.StringSliceFlag{
				Name:    "secret",
				Aliases: []string{"s"},
				Usage:   "Secret variable in KEY=VALUE format (can repeat)",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			args := c.Args()
			if args.Len() < 1 {
				return fmt.Errorf("collection name argument is required")
			}
			name := args.Get(0)

			store, ok := db.(storage.CollectionStore)
			if !ok {
				fmt.Printf("✓ Collection '%s' created (collections are created when saving requests)\n", name)
				return nil
			}

			collection := types.NewCollection(name)
			if err := applyCollectionVarFlags(collection, c.StringSlice("var"), false); err != nil {
				return err
			}
			if err := applyCollectionVarFlags(collection, c.StringSlice("secret"), true); err != nil {
				return err
			}
			if err := store.SaveCollection(collection); err != nil {
				return fmt.Errorf("failed to create collection: %w", err)
			}

			fmt.Printf("✓ Collection '%s' created\n", name)
			return nil
		},
	}
}

func collectionSetVarCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "set-var",
		Aliases: []string{"set"},
		Usage:   "Set a variable in a collection",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "secret",
				Usage: "Store the variable as a secret",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			args := c.Args()
			if args.Len() < 2 {
				return fmt.Errorf("usage: collection set-var <collection> KEY=VALUE")
			}
			store, ok := db.(storage.CollectionStore)
			if !ok {
				return fmt.Errorf("collection variables are not supported by this storage backend")
			}
			collection, err := store.GetCollectionByName(args.Get(0))
			if err != nil {
				return err
			}
			key, value, err := parseKeyValue(args.Get(1))
			if err != nil {
				return err
			}
			if c.Bool("secret") {
				collection.SetSecretVariable(key, value)
			} else {
				collection.SetVariable(key, value)
			}
			if err := store.SaveCollection(collection); err != nil {
				return fmt.Errorf("failed to update collection: %w", err)
			}
			fmt.Printf("✓ Updated collection '%s'\n", collection.Name)
			return nil
		},
	}
}

func collectionUnsetVarCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "unset-var",
		Aliases: []string{"unset"},
		Usage:   "Unset a variable in a collection",
		Action: func(ctx context.Context, c *cli.Command) error {
			args := c.Args()
			if args.Len() < 2 {
				return fmt.Errorf("usage: collection unset-var <collection> KEY")
			}
			store, ok := db.(storage.CollectionStore)
			if !ok {
				return fmt.Errorf("collection variables are not supported by this storage backend")
			}
			collection, err := store.GetCollectionByName(args.Get(0))
			if err != nil {
				return err
			}
			collection.DeleteVariable(args.Get(1))
			if err := store.SaveCollection(collection); err != nil {
				return fmt.Errorf("failed to update collection: %w", err)
			}
			fmt.Printf("✓ Updated collection '%s'\n", collection.Name)
			return nil
		},
	}
}

func collectionRemoveCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "remove",
		Aliases: []string{"rm", "delete", "del"},
		Usage:   "Remove a collection",
		Action: func(ctx context.Context, c *cli.Command) error {
			args := c.Args()
			if args.Len() < 1 {
				return fmt.Errorf("collection name argument is required")
			}
			name := args.Get(0)

			requests, err := db.ListRequests(&storage.ListOptions{Collection: name})
			if err != nil {
				return fmt.Errorf("failed to list collection: %w", err)
			}
			for _, req := range requests {
				req.Collection = ""
				if err := db.UpdateRequest(req); err != nil {
					fmt.Printf("⚠ Failed to update request '%s': %v\n", req.Name, err)
				}
			}

			if store, ok := db.(storage.CollectionStore); ok {
				if collection, err := store.GetCollectionByName(name); err == nil && collection != nil {
					if err := store.DeleteCollection(collection.ID); err != nil {
						return fmt.Errorf("failed to delete collection: %w", err)
					}
				}
			}

			fmt.Printf("✓ Removed collection '%s' (%d requests unassigned)\n", name, len(requests))
			return nil
		},
	}
}

func collectionRenameCommand(db storage.DB) *cli.Command {
	return &cli.Command{
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

			if store, ok := db.(storage.CollectionStore); ok {
				if collection, err := store.GetCollectionByName(oldName); err == nil && collection != nil {
					collection.Name = newName
					if err := store.UpdateCollection(collection); err != nil {
						return fmt.Errorf("failed to rename collection: %w", err)
					}
				}
			}

			requests, err := db.ListRequests(&storage.ListOptions{Collection: oldName})
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
	}
}

type collectionListRow struct {
	Name      string
	Requests  int
	Variables int
	UpdatedAt int64
}

func loadCollectionByName(db storage.DB, name string) (*types.Collection, error) {
	store, ok := db.(storage.CollectionStore)
	if !ok {
		return nil, nil
	}
	return store.GetCollectionByName(name)
}

func sortRequests(requests []*types.SavedRequest) {
	sort.SliceStable(requests, func(i, j int) bool {
		if requests[i].SortOrder != requests[j].SortOrder {
			return requests[i].SortOrder < requests[j].SortOrder
		}
		return requests[i].Name < requests[j].Name
	})
}

func printCollectionVariables(collection *types.Collection) {
	if collection == nil || len(collection.Variables) == 0 {
		fmt.Println("Variables:  (none)")
		return
	}
	fmt.Println("Variables:")
	keys := make([]string, 0, len(collection.Variables))
	for key := range collection.Variables {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := collection.Variables[key]
		if collection.IsSecret(key) {
			value = env.MaskSecret(value)
		}
		fmt.Printf("  %s = %s\n", key, value)
	}
}

func applyCollectionVarFlags(collection *types.Collection, pairs []string, secret bool) error {
	for _, pair := range pairs {
		key, value, err := parseKeyValue(pair)
		if err != nil {
			return err
		}
		if secret {
			collection.SetSecretVariable(key, value)
		} else {
			collection.SetVariable(key, value)
		}
	}
	return nil
}

func parseKeyValue(value string) (string, string, error) {
	key, val, ok := strings.Cut(value, "=")
	if !ok {
		return "", "", fmt.Errorf("invalid variable format '%s': must be KEY=VALUE", value)
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return "", "", fmt.Errorf("invalid variable format '%s': KEY cannot be empty", value)
	}
	return key, val, nil
}
