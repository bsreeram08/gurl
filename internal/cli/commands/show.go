package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sreeram/gurl/internal/storage"
	"github.com/urfave/cli/v3"
)

func shellEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// ShowCommand creates the show/inspect command
func ShowCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "show",
		Aliases: []string{"inspect", "view", "info"},
		Usage:   "Show details of a saved request",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Usage:   "Output format (pretty|json|curl)",
				Value:   "pretty",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			args := c.Args()
			if args.Len() < 1 {
				return fmt.Errorf("request name argument is required")
			}
			name := args.Get(0)

			req, err := db.GetRequestByName(name)
			if err != nil {
				return fmt.Errorf("request not found: %s", name)
			}

			format := c.String("format")
			switch format {
			case "json":
				data, err := json.MarshalIndent(req, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal request: %w", err)
				}
				fmt.Println(string(data))
			case "curl":
				curlParts := []string{"curl", "-X", req.Method}
				for _, h := range req.Headers {
					curlParts = append(curlParts, "-H", fmt.Sprintf("%s: %s", shellEscape(h.Key), shellEscape(h.Value)))
				}
				if req.Body != "" {
					curlParts = append(curlParts, "-d", shellEscape(req.Body))
				}
				curlParts = append(curlParts, req.URL)
				fmt.Println(strings.Join(curlParts, " \\\n  "))
			default:
				// Pretty print
				fmt.Printf("Name:       %s\n", req.Name)
				fmt.Printf("Method:     %s\n", req.Method)
				fmt.Printf("URL:        %s\n", req.URL)

				if req.Collection != "" {
					fmt.Printf("Collection: %s\n", req.Collection)
				}
				if len(req.Tags) > 0 {
					fmt.Printf("Tags:       %s\n", strings.Join(req.Tags, ", "))
				}
				if req.Folder != "" {
					fmt.Printf("Folder:     %s\n", req.Folder)
				}

				if len(req.Headers) > 0 {
					fmt.Println("\nHeaders:")
					for _, h := range req.Headers {
						fmt.Printf("  %s: %s\n", h.Key, h.Value)
					}
				}

				if req.Body != "" {
					fmt.Println("\nBody:")
					// Try to pretty-print JSON body
					var jsonBody interface{}
					if err := json.Unmarshal([]byte(req.Body), &jsonBody); err == nil {
						pretty, _ := json.MarshalIndent(jsonBody, "  ", "  ")
						fmt.Printf("  %s\n", string(pretty))
					} else {
						fmt.Printf("  %s\n", req.Body)
					}
				}

				if len(req.Variables) > 0 {
					fmt.Println("\nVariables:")
					for _, v := range req.Variables {
						fmt.Printf("  {{%s}}", v.Name)
						if v.Example != "" {
							fmt.Printf(" = %s", v.Example)
						}
						fmt.Println()
					}
				}

				if len(req.Assertions) > 0 {
					fmt.Println("\nAssertions:")
					for _, a := range req.Assertions {
						fmt.Printf("  %s %s %s\n", a.Field, a.Op, a.Value)
					}
				}

				if req.Timeout != "" {
					fmt.Printf("\nTimeout:    %s\n", req.Timeout)
				}
			}

			return nil
		},
	}
}
