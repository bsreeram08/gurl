package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
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
			sortVal := c.String("sort")
			if sortVal != "name" && sortVal != "updated" && sortVal != "collection" {
				return fmt.Errorf("invalid sort '%s': must be one of name, updated, collection", sortVal)
			}
			opts := &storage.ListOptions{
				Collection: c.String("collection"),
				Tag:        c.String("tag"),
				Pattern:    c.String("pattern"),
				Limit:      c.Int("limit"),
				Sort:       sortVal,
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

func termWidth() int {
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		return w
	}
	return 80
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func displayTree(tree map[string]*folderNode, requests []*types.SavedRequest) {
	width := termWidth()
	// Row layout: "│  " (3) + name + "  " (2) + collection (12) + "  " (2) + tags (10) + "  " (2) + date (16) = 47 fixed
	fixedCols := 47
	nameWidth := width - fixedCols
	if nameWidth < 20 {
		nameWidth = 20
	}

	border := strings.Repeat("─", width-2)
	fmt.Printf("┌%s┐\n", border)

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
		printRequestRow(req, "", nameWidth)
	}

	displayFolderTree(tree, byFolder, "", nameWidth)

	fmt.Printf("└%s┘\n", border)
}

func displayFolderTree(tree map[string]*folderNode, byFolder map[string][]*types.SavedRequest, prefix string, nameWidth int) {
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
				printRequestRow(req, prefix+"  ", nameWidth)
			}
		}

		if len(node.children) > 0 {
			displayFolderTree(node.children, byFolder, prefix+"  ", nameWidth)
		}
	}
}

func printRequestRow(req *types.SavedRequest, indent string, nameWidth int) {
	availName := nameWidth - len(indent)
	if availName < 10 {
		availName = 10
	}
	name := truncate(req.Name, availName)

	collection := req.Collection
	if collection == "" {
		collection = "-"
	}
	collection = truncate(collection, 12)

	tags := strings.Join(req.Tags, ",")
	if tags == "" {
		tags = "-"
	}
	tags = truncate(tags, 10)

	updated := time.Unix(req.UpdatedAt, 0).Format("2006-01-02 15:04")

	fmt.Printf("│  %s%-*s  %-12s  %-10s  %s\n", indent, availName, name, collection, tags, updated)
}
