package commands

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/internal/project"
	"github.com/sreeram/gurl/internal/redact"
	"github.com/sreeram/gurl/internal/runner"
	"github.com/sreeram/gurl/internal/secrets"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)

var collectionDeleteIsInteractive = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

var collectionPassphraseIsInteractive = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

var readCollectionPassphrase = func(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	value, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("failed to read passphrase: %w", err)
	}
	return string(value), nil
}

// CollectionCommand creates the collection command.
func CollectionCommand(db storage.DB, envStorage env.Store, runnerOpts ...runner.Option) *cli.Command {
	return &cli.Command{
		Name:    "collection",
		Aliases: []string{"collections", "col", "c"},
		Usage:   "Manage collections",
		Commands: []*cli.Command{
			runner.CollectionRunCommand(db, envStorage, runnerOpts...),
			collectionShowCommand(db),
			collectionListCommand(db),
			collectionCreateCommand(db),
			collectionSetVarCommand(db),
			collectionUnsetVarCommand(db),
			collectionMigrateCommand(db),
			collectionUnlockCommand(db),
			collectionExportCommand(db),
			collectionImportCommand(db),
			collectionRemoveCommand(db),
			collectionDeleteCommand(db),
			collectionRenameCommand(db),
		},
	}
}

func collectionShowCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "show",
		Aliases: []string{"view", "info"},
		Usage:   "Show requests and variables in a collection",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "json",
				Aliases: []string{"j"},
				Usage:   "JSON output (secrets masked)",
			},
		},
		Action: jsonAction(func(ctx context.Context, c *cli.Command) error {
			args := c.Args()
			if args.Len() < 1 {
				return fmt.Errorf("collection name argument is required")
			}
			name := args.Get(0)

			collection, _ := db.GetCollectionByName(name)
			requests, err := db.ListRequests(&storage.ListOptions{Collection: name})
			if err != nil {
				return fmt.Errorf("failed to list collection: %w", err)
			}
			if collection == nil && len(requests) == 0 {
				return fmt.Errorf("collection '%s' not found or empty", name)
			}

			sortRequests(requests)

			if jsonOutput(c) {
				return writeJSON(os.Stdout, collectionShowOutputOf(name, collection, requests))
			}

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
		}),
	}
}

// collectionShowOutput is `collection show --json` output.
type collectionShowOutput struct {
	Name      string            `json:"name"`
	ID        string            `json:"id,omitempty"`
	Variables map[string]string `json:"variables"`
	Secrets   []string          `json:"secrets"`
	Requests  []collectionReq   `json:"requests"`
}

type collectionReq struct {
	Name   string `json:"name"`
	Method string `json:"method"`
	URL    string `json:"url"`
	Folder string `json:"folder,omitempty"`
}

// collectionShowOutputOf masks the values of secret and sensitively named
// variables and literal secrets in request URLs.
func collectionShowOutputOf(name string, collection *types.Collection, requests []*types.SavedRequest) collectionShowOutput {
	out := collectionShowOutput{Name: name, Variables: map[string]string{}, Secrets: []string{}, Requests: []collectionReq{}}
	if collection != nil {
		out.ID = collection.ID
		for key, value := range collection.Variables {
			if collection.IsSecret(key) {
				out.Secrets = append(out.Secrets, key)
			}
			if (collection.IsSecret(key) || redact.SensitiveName(key)) && value != "" {
				value = redact.Mask
			}
			out.Variables[key] = value
		}
		sort.Strings(out.Secrets)
	}
	for _, req := range requests {
		masked := redact.SavedRequest(req)
		method := masked.Method
		if method == "" {
			method = "GET"
		}
		out.Requests = append(out.Requests, collectionReq{Name: masked.Name, Method: method, URL: masked.URL, Folder: masked.Folder})
	}
	return out
}

func collectionListCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls", "l"},
		Usage:   "List all collections",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "json",
				Aliases: []string{"j"},
				Usage:   "JSON output",
			},
		},
		Action: jsonAction(func(ctx context.Context, c *cli.Command) error {
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

			collections, err := db.ListCollections()
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

			rows := make([]collectionListRow, 0, len(stats))
			for _, row := range stats {
				rows = append(rows, row)
			}
			sort.SliceStable(rows, func(i, j int) bool {
				return rows[i].Name < rows[j].Name
			})

			if jsonOutput(c) {
				type collectionSummary struct {
					Name      string `json:"name"`
					Requests  int    `json:"requests"`
					Variables int    `json:"variables"`
					Updated   string `json:"updated,omitempty"`
				}
				out := make([]collectionSummary, 0, len(rows))
				for _, row := range rows {
					summary := collectionSummary{Name: row.Name, Requests: row.Requests, Variables: row.Variables}
					if row.UpdatedAt > 0 {
						summary.Updated = time.Unix(row.UpdatedAt, 0).UTC().Format(time.RFC3339)
					}
					out = append(out, summary)
				}
				return writeJSON(os.Stdout, out)
			}

			if len(rows) == 0 {
				fmt.Println("No collections found.")
				return nil
			}

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
		}),
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

			collection := types.NewCollection(name)
			if err := applyCollectionVarFlags(collection, c.StringSlice("var"), false); err != nil {
				return err
			}
			if err := applyCollectionVarFlags(collection, c.StringSlice("secret"), true); err != nil {
				return err
			}
			if err := db.SaveCollection(collection); err != nil {
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
			collection, err := db.GetCollectionByName(args.Get(0))
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
			if err := db.SaveCollection(collection); err != nil {
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
			collection, err := db.GetCollectionByName(args.Get(0))
			if err != nil {
				return err
			}
			collection.DeleteVariable(args.Get(1))
			if err := db.SaveCollection(collection); err != nil {
				return fmt.Errorf("failed to update collection: %w", err)
			}
			fmt.Printf("✓ Updated collection '%s'\n", collection.Name)
			return nil
		},
	}
}

func collectionMigrateCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:  "migrate",
		Usage: "Migrate a DB-backed collection into project file storage",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "project-dir",
				Usage: "Project root for .gurl file storage",
			},
			&cli.StringFlag{
				Name:  "passphrase",
				Usage: "Write collection secrets with passphrase-derived encryption",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			args := c.Args()
			if args.Len() < 1 {
				return fmt.Errorf("collection name argument is required")
			}
			name := args.Get(0)

			passphrase := collectionPassphraseValue(c)
			if passphrase == "" && projectDirFlag(c) == "" {
				count, path, err := db.MigrateCollectionToFiles(name)
				if err == nil {
					fmt.Printf("✓ Migrated collection '%s' to %s (%d requests)\n", name, path, count)
					return nil
				}
				if !errors.Is(err, storage.ErrNotSupported) {
					return err
				}
			}

			count, path, err := migrateCollectionToProjectFiles(c, db, name)
			if err != nil {
				return err
			}
			fmt.Printf("✓ Migrated collection '%s' to %s (%d requests)\n", name, path, count)
			return nil
		},
	}
}

func collectionUnlockCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:  "unlock",
		Usage: "Unlock a passphrase-protected file collection",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "passphrase",
				Usage: "Collection import passphrase",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			args := c.Args()
			if args.Len() < 1 {
				return fmt.Errorf("collection name argument is required")
			}
			passphrase, err := collectionPassphrase(c, fmt.Sprintf("Passphrase for collection %q: ", args.Get(0)))
			if err != nil {
				return err
			}
			if err := db.UnlockCollection(args.Get(0), passphrase); err != nil {
				if errors.Is(err, storage.ErrNotSupported) {
					return fmt.Errorf("collection unlock requires project file storage")
				}
				return err
			}
			fmt.Printf("✓ Unlocked collection '%s'\n", args.Get(0))
			return nil
		},
	}
}

func collectionExportCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:  "export",
		Usage: "Export a collection with passphrase-encrypted secrets",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output file (default: stdout)",
			},
			&cli.StringFlag{
				Name:  "passphrase",
				Usage: "Passphrase for encrypting collection secrets",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			args := c.Args()
			if args.Len() < 1 {
				return fmt.Errorf("collection name argument is required")
			}
			name := args.Get(0)
			collection, _ := db.GetCollectionByName(name)
			requests, err := db.ListRequests(&storage.ListOptions{Collection: name})
			if err != nil {
				return fmt.Errorf("failed to list collection: %w", err)
			}
			if collection == nil && len(requests) == 0 {
				return fmt.Errorf("collection %q not found or empty", name)
			}
			if collection == nil {
				collection = types.NewCollection(name)
			}

			passphrase := collectionPassphraseValue(c)
			if passphrase == "" && collectionHasSecretVariables(collection) {
				passphrase, err = collectionPassphrase(c, fmt.Sprintf("Passphrase for exporting collection %q: ", name))
				if err != nil {
					return err
				}
			}
			exportData, err := storage.BuildCollectionExport(collection, requests, passphrase)
			if err != nil {
				return err
			}
			data, err := storage.MarshalCollectionExport(exportData)
			if err != nil {
				return fmt.Errorf("failed to marshal collection export: %w", err)
			}

			if outputPath := c.String("output"); outputPath != "" {
				if err := validateOutputPath(outputPath); err != nil {
					return fmt.Errorf("invalid output path: %w", err)
				}
				if err := os.WriteFile(outputPath, data, 0644); err != nil {
					return fmt.Errorf("failed to write output file: %w", err)
				}
				fmt.Printf("✓ Exported collection '%s' to %s (%d requests)\n", name, outputPath, len(requests))
				return nil
			}
			fmt.Println(string(data))
			return nil
		},
	}
}

func collectionImportCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:  "import",
		Usage: "Import a collection export, collection directory, or .env variables",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Overwrite existing collection/request records",
			},
			&cli.StringFlag{
				Name:  "file",
				Usage: "Path to .env file for collection variable import",
			},
			&cli.StringFlag{
				Name:  "passphrase",
				Usage: "Passphrase for decrypting collection secrets",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			if filePath := c.String("file"); filePath != "" {
				return importCollectionVariablesFromFile(db, c.Args().First(), filePath)
			}

			path := c.Args().First()
			if path == "" {
				return fmt.Errorf("file path is required")
			}
			passphrase := collectionPassphraseValue(c)
			collection, requests, err := parseCollectionImportSource(path, passphrase)
			if err != nil && passphrase == "" && isCollectionImportPassphraseRequired(err) && noInput(c) {
				return errPassphraseRequired
			}
			if err != nil && passphrase == "" && isCollectionImportPassphraseRequired(err) && collectionPassphraseIsInteractive() {
				passphrase, err = collectionPassphrase(c, "Passphrase for collection import: ")
				if err != nil {
					return err
				}
				collection, requests, err = parseCollectionImportSource(path, passphrase)
			}
			if err != nil {
				return err
			}
			if collection == nil {
				return fmt.Errorf("collection export is missing collection metadata")
			}

			allowLockedSave := false
			existing, err := db.GetCollectionByName(collection.Name)
			if storage.IsCollectionLocked(err) {
				if !c.Bool("force") {
					return fmt.Errorf("collection %q already exists (use --force to overwrite)", collection.Name)
				}
				raw, rawErr := db.GetRawCollectionByName(collection.Name)
				if errors.Is(rawErr, storage.ErrNotSupported) {
					return fmt.Errorf("failed to inspect locked collection %q: %w", collection.Name, err)
				}
				if rawErr != nil {
					return fmt.Errorf("failed to inspect locked collection %q: %w", collection.Name, rawErr)
				}
				existing = raw
				allowLockedSave = true
			} else if err != nil && !storage.IsCollectionNotFound(err) {
				return fmt.Errorf("failed to inspect collection %q: %w", collection.Name, err)
			}
			if existing != nil {
				if !c.Bool("force") {
					return fmt.Errorf("collection %q already exists (use --force to overwrite)", collection.Name)
				}
				collection.ID = existing.ID
				collection.CreatedAt = existing.CreatedAt
			}
			if err := saveCollectionRecord(db, collection, allowLockedSave); err != nil {
				return err
			}

			imported := 0
			skipped := 0
			for _, req := range requests {
				req.Collection = collection.Name
				if existing, err := db.GetRequestByName(req.Name); err == nil && existing != nil {
					if !c.Bool("force") {
						skipped++
						continue
					}
					req.ID = existing.ID
					if req.CreatedAt == 0 {
						req.CreatedAt = existing.CreatedAt
					}
					if err := db.DeleteRequest(existing.ID); err != nil {
						return fmt.Errorf("failed to overwrite request %q: %w", req.Name, err)
					}
				}
				if err := db.SaveRequest(req); err != nil {
					return fmt.Errorf("failed to import request %q: %w", req.Name, err)
				}
				imported++
			}

			fmt.Printf("✓ Imported collection '%s' (%d requests, %d skipped)\n", collection.Name, imported, skipped)
			return nil
		},
	}
}

func parseCollectionImportSource(path string, passphrase string) (*types.Collection, []*types.SavedRequest, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to inspect collection import source: %w", err)
	}
	if info.IsDir() {
		return storage.ParseCollectionDirectory(path, passphrase)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read collection export: %w", err)
	}
	return storage.ParseCollectionExport(data, passphrase)
}

func importCollectionVariablesFromFile(db storage.DB, name, filePath string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("collection name argument is required when using --file")
	}

	vars, err := env.ParseDotenvFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse .env file: %w", err)
	}

	collection, err := loadOrCreateCollectionForVariableImport(db, name)
	if err != nil {
		return err
	}
	for key, value := range vars {
		collection.SetVariable(key, value)
	}
	if err := saveCollectionRecord(db, collection, false); err != nil {
		return fmt.Errorf("failed to save collection: %w", err)
	}

	fmt.Printf("✓ Imported %d variable(s) from '%s' into collection '%s'\n", len(vars), filePath, name)
	return nil
}

func loadOrCreateCollectionForVariableImport(db storage.DB, name string) (*types.Collection, error) {
	collection, err := db.GetCollectionByName(name)
	if err == nil {
		if collection != nil {
			return collection, nil
		}
		return types.NewCollection(name), nil
	}
	if storage.IsCollectionLocked(err) {
		return nil, err
	}
	if !storage.IsCollectionNotFound(err) {
		return nil, fmt.Errorf("failed to load collection %q: %w", name, err)
	}
	return types.NewCollection(name), nil
}

func saveCollectionRecord(db storage.DB, collection *types.Collection, allowLocked bool) error {
	if allowLocked {
		return db.SaveCollectionAllowLocked(collection)
	}
	if collection.ID != "" {
		if existing, err := db.GetCollection(collection.ID); err == nil && existing != nil {
			return db.UpdateCollection(collection)
		}
	}
	return db.SaveCollection(collection)
}

func collectionPassphraseValue(c *cli.Command) string {
	if passphrase := c.String("passphrase"); passphrase != "" {
		return passphrase
	}
	return os.Getenv("GURL_IMPORT_PASSPHRASE")
}

func collectionPassphrase(c *cli.Command, prompt string) (string, error) {
	if passphrase := collectionPassphraseValue(c); passphrase != "" {
		return passphrase, nil
	}
	if noInput(c) {
		return "", errPassphraseRequired
	}
	if !collectionPassphraseIsInteractive() {
		return "", nil
	}
	passphrase, err := readCollectionPassphrase(prompt)
	if err != nil {
		return "", err
	}
	if passphrase == "" {
		return "", fmt.Errorf("passphrase cannot be empty")
	}
	return passphrase, nil
}

// errPassphraseRequired is returned instead of prompting for a passphrase
// in non-interactive mode.
var errPassphraseRequired = errors.New("passphrase required: pass --passphrase or set GURL_IMPORT_PASSPHRASE")

func isCollectionImportPassphraseRequired(err error) bool {
	return err != nil && strings.Contains(err.Error(), "passphrase is required to import collection secrets")
}

func collectionHasSecretVariables(collection *types.Collection) bool {
	if collection == nil {
		return false
	}
	for _, isSecret := range collection.SecretKeys {
		if isSecret {
			return true
		}
	}
	return false
}

func migrateCollectionToProjectFiles(c *cli.Command, db storage.DB, name string) (int, string, error) {
	proj, err := project.Require("", projectDirFlag(c))
	if err != nil {
		return 0, "", err
	}
	fileStore := storage.NewFileStore(proj)

	collection, _ := db.GetCollectionByName(name)
	requests, err := db.ListRequests(&storage.ListOptions{Collection: name})
	if err != nil {
		return 0, "", fmt.Errorf("failed to list collection: %w", err)
	}
	if collection == nil && len(requests) == 0 {
		return 0, "", fmt.Errorf("collection %q not found or empty", name)
	}
	if collection == nil {
		collection = types.NewCollection(name)
	}
	if err := fileStore.SaveCollection(collection); err != nil {
		return 0, "", err
	}
	for _, req := range requests {
		copy := *req
		copy.Collection = name
		if err := fileStore.SaveRequest(&copy); err != nil {
			return 0, "", err
		}
	}
	if passphrase := collectionPassphraseValue(c); passphrase != "" {
		if err := fileStore.SaveCollectionPassphrase(collection, passphrase); err != nil {
			return 0, "", err
		}
	}
	path, err := fileStore.CollectionPath(name)
	if err != nil {
		return 0, "", err
	}
	return len(requests), path, nil
}

func collectionRemoveCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "remove",
		Aliases: []string{"rm"},
		Usage:   "Remove a collection label and keep its requests",
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

			if collection, err := db.GetCollectionByName(name); err == nil && collection != nil {
				if err := db.DeleteCollection(collection.ID); err != nil {
					return fmt.Errorf("failed to delete collection: %w", err)
				}
			}

			fmt.Printf("✓ Removed collection '%s' (%d requests unassigned)\n", name, len(requests))
			return nil
		},
	}
}

func collectionDeleteCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "delete",
		Aliases: []string{"del"},
		Usage:   "Delete a collection and all of its requests",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "force",
				Usage: "Delete without interactive confirmation",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			args := c.Args()
			if args.Len() < 1 {
				return fmt.Errorf("collection name argument is required")
			}
			name := args.Get(0)

			collection, _ := db.GetCollectionByName(name)
			requests, err := db.ListRequests(&storage.ListOptions{Collection: name})
			if err != nil {
				return fmt.Errorf("failed to list collection: %w", err)
			}
			if collection == nil && len(requests) == 0 {
				return fmt.Errorf("collection %q not found or empty", name)
			}

			secretCount := 0
			if collection != nil {
				for _, isSecret := range collection.SecretKeys {
					if isSecret {
						secretCount++
					}
				}
			}

			if !c.Bool("force") {
				confirmed, err := confirmCollectionDelete(c, name, len(requests), secretCount)
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			for _, req := range requests {
				if err := db.DeleteRequest(req.ID); err != nil {
					return fmt.Errorf("failed to delete request %q: %w", req.Name, err)
				}
			}

			if collection != nil {
				if err := db.DeleteCollection(collection.ID); err != nil {
					return fmt.Errorf("failed to delete collection: %w", err)
				}
			}

			fmt.Printf("✓ Deleted collection '%s' (%d requests deleted)\n", name, len(requests))
			return nil
		},
	}
}

func confirmCollectionDelete(c *cli.Command, name string, requestCount int, secretCount int) (bool, error) {
	requestLabel := pluralizeCount(requestCount, "request", "requests")
	secretLabel := pluralizeCount(secretCount, "secret", "secrets")
	if !canPrompt(c, collectionDeleteIsInteractive) {
		return false, fmt.Errorf("collection %q has %s and %s; use --force to delete non-interactively", name, requestLabel, secretLabel)
	}

	fmt.Printf("Collection %q has %s and %s. Delete all? [y/N] ", name, requestLabel, secretLabel)
	answer, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && strings.TrimSpace(answer) == "" {
		return false, fmt.Errorf("failed to read confirmation: %w", err)
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes", nil
}

func pluralizeCount(count int, singular string, plural string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, singular)
	}
	return fmt.Sprintf("%d %s", count, plural)
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

			if collection, err := db.GetCollectionByName(oldName); err == nil && collection != nil {
				collection.Name = newName
				if err := db.UpdateCollection(collection); err != nil {
					return fmt.Errorf("failed to rename collection: %w", err)
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
			value = secrets.Mask(value)
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
