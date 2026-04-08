package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/sreeram/gurl/internal/core/curl"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/urfave/cli/v3"
)

// SaveCommand creates the save command
func SaveCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "save",
		Aliases: []string{"s"},
		Usage:   "Save a curl request with a name",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "collection",
				Aliases: []string{"c"},
				Usage:   "Assign to collection",
			},
			&cli.StringFlag{
				Name:    "folder",
				Aliases: []string{"F"},
				Usage:   "Assign to folder (e.g., api/v2/users)",
			},
			&cli.StringSliceFlag{
				Name:    "tag",
				Aliases: []string{"t"},
				Usage:   "Add tag (can repeat)",
			},
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Usage:   "Output format preference (auto|json|table)",
				Value:   "auto",
			},
			&cli.StringFlag{
				Name:  "description",
				Usage: "Human-readable description",
			},
			&cli.StringFlag{
				Name:  "curl",
				Usage: "Full curl command as a string",
			},
			&cli.StringFlag{
				Name:    "X",
				Aliases: []string{"request"},
				Usage:   "HTTP method",
			},
			&cli.StringSliceFlag{
				Name:    "H",
				Aliases: []string{"header"},
				Usage:   "HTTP header (can repeat)",
			},
			&cli.StringFlag{
				Name:    "d",
				Aliases: []string{"data", "body"},
				Usage:   "Request body",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			args := c.Args()

			// Mode 1: --curl flag provided
			if curlFlag := c.String("curl"); curlFlag != "" {
				name := c.String("description") // reuse description flag for name when using --curl
				if name == "" && args.Len() >= 1 {
					name = args.Get(0)
				} else if name == "" {
					return fmt.Errorf("name is required")
				}

				parsed, err := curl.ParseCurl(curlFlag)
				if err != nil {
					return fmt.Errorf("failed to parse curl: %w", err)
				}

				req := types.ParsedCurlToSavedRequest(*parsed)
				req.Name = name
				req.ID = fmt.Sprintf("saved-%d", time.Now().UnixNano())
				req.OutputFormat = c.String("format")
				req.Tags = c.StringSlice("tag")
				req.Collection = c.String("collection")
				req.Folder = c.String("folder")
				req.CreatedAt = time.Now().Unix()
				req.UpdatedAt = time.Now().Unix()

				if err := db.SaveRequest(&req); err != nil {
					return fmt.Errorf("failed to save request: %w", err)
				}

				fmt.Printf("✓ Saved request '%s'\n", name)
				return nil
			}

			// Mode 2: Individual flags (-X, -H, -d) provided
			if c.String("X") != "" || len(c.StringSlice("H")) > 0 || c.String("d") != "" {
				if args.Len() < 1 {
					return fmt.Errorf("name and URL arguments are required")
				}
				name := args.Get(0)
				url := ""
				if args.Len() >= 2 {
					url = args.Get(1)
				}

				method := c.String("X")
				if method == "" {
					method = "GET"
				}

				headers := c.StringSlice("H")
				var headerList []types.Header
				for _, h := range headers {
					if idx := strings.Index(h, ":"); idx != -1 {
						headerList = append(headerList, types.Header{
							Key:   strings.TrimSpace(h[:idx]),
							Value: strings.TrimSpace(h[idx+1:]),
						})
					}
				}

				req := &types.SavedRequest{
					Name:         name,
					URL:          url,
					Method:       method,
					Headers:      headerList,
					Body:         c.String("d"),
					OutputFormat: c.String("format"),
					Tags:         c.StringSlice("tag"),
					Collection:   c.String("collection"),
					Folder:       c.String("folder"),
					CreatedAt:    time.Now().Unix(),
					UpdatedAt:    time.Now().Unix(),
				}

				if err := db.SaveRequest(req); err != nil {
					return fmt.Errorf("failed to save request: %w", err)
				}

				fmt.Printf("✓ Saved request '%s'\n", name)
				return nil
			}

			// Mode 3: Read from stdin (piping curl command)
			if args.Len() == 0 {
				data, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("failed to read stdin: %w", err)
				}
				input := strings.TrimSpace(string(data))
				if input == "" {
					return fmt.Errorf("empty stdin input")
				}

				parsed, err := curl.ParseCurl(input)
				if err != nil {
					return fmt.Errorf("failed to parse curl from stdin: %w", err)
				}

				name := generateNameFromURL(parsed.URL)
				req := types.ParsedCurlToSavedRequest(*parsed)
				req.Name = name
				req.ID = fmt.Sprintf("saved-%d", time.Now().UnixNano())
				req.OutputFormat = c.String("format")
				req.Tags = c.StringSlice("tag")
				req.Collection = c.String("collection")
				req.Folder = c.String("folder")
				req.CreatedAt = time.Now().Unix()
				req.UpdatedAt = time.Now().Unix()

				if err := db.SaveRequest(&req); err != nil {
					return fmt.Errorf("failed to save request: %w", err)
				}

				fmt.Printf("✓ Saved request '%s'\n", name)
				return nil
			}

			// Mode 4: Original behavior - name + URL as positional args (GET request)
			if args.Len() < 2 {
				return fmt.Errorf("name and URL arguments are required")
			}
			name := args.Get(0)
			url := args.Get(1)

			req := &types.SavedRequest{
				Name:         name,
				URL:          url,
				Method:       "GET",
				OutputFormat: c.String("format"),
				Tags:         c.StringSlice("tag"),
				Folder:       c.String("folder"),
				CreatedAt:    time.Now().Unix(),
				UpdatedAt:    time.Now().Unix(),
			}

			if err := db.SaveRequest(req); err != nil {
				return fmt.Errorf("failed to save request: %w", err)
			}

			fmt.Printf("✓ Saved request '%s'\n", name)
			return nil
		},
	}
}
