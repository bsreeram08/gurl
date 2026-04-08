package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/urfave/cli/v3"
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

			folders, err := db.GetAllFolders()
			if err != nil {
				return fmt.Errorf("failed to get folders: %w", err)
			}
			sort.Strings(folders)

			folderTree := buildFolderTree(folders)
			displayTree(folderTree, requests)

			fmt.Printf("\n  %d requests\n", len(requests))

			return nil
		},
	}
}

type folderNode struct {
	name     string
	requests []*types.SavedRequest
	children map[string]*folderNode
}

func buildFolderTree(folders []string) map[string]*folderNode {
	tree := make(map[string]*folderNode)
	for _, path := range folders {
		parts := strings.Split(path, "/")
		current := tree
		for i, part := range parts {
			if current[part] == nil {
				current[part] = &folderNode{
					name:     part,
					children: make(map[string]*folderNode),
				}
			}
			if i == len(parts)-1 {
				break
			}
			current = current[part].children
		}
	}
	return tree
}

func displayTree(tree map[string]*folderNode, requests []*types.SavedRequest) {
	fmt.Println("┌─ Saved Requests ─────────────────────────────────────────┐")

	rootRequests := []*types.SavedRequest{}
	byFolder := make(map[string][]*types.SavedRequest)

	for _, req := range requests {
		if req.Folder == "" {
			rootRequests = append(rootRequests, req)
		} else {
			byFolder[req.Folder] = append(byFolder[req.Folder], req)
		}
	}

	sort.Slice(rootRequests, func(i, j int) bool {
		return rootRequests[i].Name < rootRequests[j].Name
	})

	for _, req := range rootRequests {
		printRequestRow(req, "")
	}

	displayFolderTree(tree, byFolder, "")

	fmt.Println("└─────────────────────────────────────────────────────────┘")
}

func displayFolderTree(tree map[string]*folderNode, byFolder map[string][]*types.SavedRequest, prefix string) {
	sortedKeys := make([]string, 0, len(tree))
	for k := range tree {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	for _, key := range sortedKeys {
		node := tree[key]
		fullPath := prefix + key

		fmt.Printf("│  %s%s/\n", prefix, key)

		if reqs, ok := byFolder[fullPath]; ok {
			sort.Slice(reqs, func(i, j int) bool {
				return reqs[i].Name < reqs[j].Name
			})
			for _, req := range reqs {
				printRequestRow(req, prefix+"  ")
			}
		}

		if len(node.children) > 0 {
			displayFolderTree(node.children, byFolder, prefix+"  ")
		}
	}
}

func printRequestRow(req *types.SavedRequest, indent string) {
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

	fmt.Printf("│  %s%-20s %-12s %-10s %s\n", indent, name, collection, tags, updated)
}
