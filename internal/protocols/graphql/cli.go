package graphql

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sreeram/gurl/internal/formatter"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/urfave/cli/v3"
)

func GraphQLCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "graphql",
		Aliases: []string{"gql"},
		Usage:   "Execute a GraphQL query",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "query",
				Aliases: []string{"q"},
				Usage:   "GraphQL query string",
			},
			&cli.StringFlag{
				Name:    "query-file",
				Aliases: []string{"f"},
				Usage:   "Path to a .graphql file containing the query",
			},
			&cli.StringFlag{
				Name:    "vars",
				Aliases: []string{"v"},
				Usage:   "JSON object with query variables (e.g., '{\"limit\": 10}')",
			},
			&cli.StringFlag{
				Name:    "operation-name",
				Aliases: []string{"op"},
				Usage:   "GraphQL operation name",
			},
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"fmt"},
				Usage:   "Output format (auto|json|table)",
				Value:   "auto",
			},
			&cli.BoolFlag{
				Name:    "color",
				Aliases: []string{"c"},
				Usage:   "Enable syntax highlighting",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			endpoint := c.Args().Get(0)
			if endpoint == "" {
				return fmt.Errorf("endpoint is required")
			}

			query := c.String("query")
			queryFile := c.String("query-file")

			if query == "" && queryFile == "" {
				return fmt.Errorf("either --query or --query-file is required")
			}

			if queryFile != "" {
				data, err := os.ReadFile(queryFile)
				if err != nil {
					return fmt.Errorf("failed to read query file: %w", err)
				}
				query = string(data)
			}

			vars := make(map[string]interface{})
			if varsStr := c.String("vars"); varsStr != "" {
				if err := json.Unmarshal([]byte(varsStr), &vars); err != nil {
					return fmt.Errorf("failed to parse variables JSON: %w", err)
				}
			}

			operationName := c.String("operation-name")

			client := NewClient()
			resp, err := client.Execute(ctx, endpoint, Request{
				Query:         query,
				Variables:     vars,
				OperationName: operationName,
			})
			if err != nil {
				return fmt.Errorf("GraphQL request failed: %w", err)
			}

			color := c.Bool("color")

			var output []byte
			if resp.Data != nil {
				opts := formatter.FormatOptions{
					Indent: "  ",
					Color:  color,
				}
				formatted := formatter.Format(resp.Data, "application/json", opts)
				output = []byte(formatted)
			}

			if resp.Errors != nil {
				fmt.Fprintf(os.Stderr, "GraphQL Errors:\n")
				for _, e := range resp.Errors {
					locStr := ""
					if len(e.Locations) > 0 {
						locs := make([]string, len(e.Locations))
						for i, loc := range e.Locations {
							locs[i] = fmt.Sprintf("line %d, column %d", loc.Line, loc.Column)
						}
						locStr = " (" + strings.Join(locs, "; ") + ")"
					}
					fmt.Fprintf(os.Stderr, "  - %s%s\n", e.Message, locStr)
				}
				if resp.Data == nil {
					return nil
				}
				fmt.Fprintf(os.Stderr, "\n")
			}

			if output != nil {
				fmt.Println(string(output))
			}

			return nil
		},
	}
}
