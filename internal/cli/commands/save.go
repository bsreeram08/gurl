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
		Flags: append([]cli.Flag{
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
				Name:  "name",
				Usage: "Name for the request (used with --curl)",
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
			&cli.StringSliceFlag{
				Name:  "extract",
				Usage: "Add extraction rule (format: VAR_NAME=METHOD:EXPRESSION)",
			},
			&cli.StringFlag{
				Name:    "pre-script",
				Aliases: []string{"pre"},
				Usage:   "Set pre-request script",
			},
			&cli.StringFlag{
				Name:    "post-script",
				Aliases: []string{"post"},
				Usage:   "Set post-response script",
			},
		}, authConfigFlags()...),
		Action: func(ctx context.Context, c *cli.Command) error {
			args := c.Args()
			nameFlag := c.String("name")
			extracts, err := parseExtractFlags(c.StringSlice("extract"))
			if err != nil {
				return cli.Exit(err.Error(), 2)
			}
			authConfig, _, err := parseAuthConfigFlags(c)
			if err != nil {
				if strings.Contains(err.Error(), "auth-param must be") {
					return cli.Exit(err.Error(), 2)
				}
				return err
			}

			// Mode 1: --curl flag provided
			if curlFlag := c.String("curl"); curlFlag != "" {
				name := nameFlag
				if name == "" {
					name = c.String("description")
				}
				if name == "" && args.Len() >= 1 {
					name = args.Get(0)
				} else if name == "" {
					return fmt.Errorf("name is required (use --name or provide as argument)")
				}

				parsed, err := curl.ParseCurl(curlFlag)
				if err != nil {
					return fmt.Errorf("failed to parse curl: %w", err)
				}

				req := types.ParsedCurlToSavedRequest(*parsed)
				req.Name = name
				req.ID = fmt.Sprintf("saved-%d", time.Now().UnixNano())
				format := c.String("format")
				if format != "auto" && format != "json" && format != "table" {
					return fmt.Errorf("invalid format '%s': must be one of auto, json, table", format)
				}
				req.OutputFormat = format
				req.Tags = c.StringSlice("tag")
				req.Collection = c.String("collection")
				req.Folder = c.String("folder")
				req.AuthConfig = authConfig
				applyFlowMetadata(&req, extracts, c.String("pre-script"), c.String("post-script"))
				req.CreatedAt = time.Now().Unix()
				req.UpdatedAt = time.Now().Unix()

				if err := db.SaveRequest(&req); err != nil {
					return fmt.Errorf("failed to save request: %w", err)
				}

				printSaveConfirmation(name, req.URL)
				return nil
			}

			// Mode 2: Individual flags (-X, -H, -d) provided
			if c.String("X") != "" || len(c.StringSlice("H")) > 0 || c.String("d") != "" {
				name, url, err := resolveSaveNameAndURL(args, nameFlag)
				if err != nil {
					return err
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
					Name:       name,
					URL:        url,
					Method:     method,
					Headers:    headerList,
					Body:       c.String("d"),
					Tags:       c.StringSlice("tag"),
					Collection: c.String("collection"),
					Folder:     c.String("folder"),
					AuthConfig: authConfig,
					CreatedAt:  time.Now().Unix(),
					UpdatedAt:  time.Now().Unix(),
				}
				format := c.String("format")
				if format != "auto" && format != "json" && format != "table" {
					return fmt.Errorf("invalid format '%s': must be one of auto, json, table", format)
				}
				req.OutputFormat = format
				applyFlowMetadata(req, extracts, c.String("pre-script"), c.String("post-script"))

				if err := db.SaveRequest(req); err != nil {
					return fmt.Errorf("failed to save request: %w", err)
				}

				printSaveConfirmation(name, req.URL)
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

				name := nameFlag
				if name == "" {
					name = generateNameFromURL(parsed.URL)
				}
				req := types.ParsedCurlToSavedRequest(*parsed)
				req.Name = name
				req.ID = fmt.Sprintf("saved-%d", time.Now().UnixNano())
				format := c.String("format")
				if format != "auto" && format != "json" && format != "table" {
					return fmt.Errorf("invalid format '%s': must be one of auto, json, table", format)
				}
				req.OutputFormat = format
				req.Tags = c.StringSlice("tag")
				req.Collection = c.String("collection")
				req.Folder = c.String("folder")
				req.AuthConfig = authConfig
				applyFlowMetadata(&req, extracts, c.String("pre-script"), c.String("post-script"))
				req.CreatedAt = time.Now().Unix()
				req.UpdatedAt = time.Now().Unix()

				if err := db.SaveRequest(&req); err != nil {
					return fmt.Errorf("failed to save request: %w", err)
				}

				printSaveConfirmation(name, req.URL)
				return nil
			}

			// Mode 4: Original behavior - name + URL as positional args (GET request)
			name, url, err := resolveSaveNameAndURL(args, nameFlag)
			if err != nil {
				return err
			}

			req := &types.SavedRequest{
				Name:         name,
				URL:          url,
				Method:       "GET",
				OutputFormat: c.String("format"),
				Tags:         c.StringSlice("tag"),
				Collection:   c.String("collection"),
				Folder:       c.String("folder"),
				AuthConfig:   authConfig,
				CreatedAt:    time.Now().Unix(),
				UpdatedAt:    time.Now().Unix(),
			}
			applyFlowMetadata(req, extracts, c.String("pre-script"), c.String("post-script"))

			if err := db.SaveRequest(req); err != nil {
				return fmt.Errorf("failed to save request: %w", err)
			}

			printSaveConfirmation(name, req.URL)
			return nil
		},
	}
}

func applyFlowMetadata(req *types.SavedRequest, extracts []types.Extract, preScript, postScript string) {
	req.Extracts = append(req.Extracts, extracts...)
	req.PreScript = preScript
	req.PostScript = postScript
}

func parseExtractFlags(values []string) ([]types.Extract, error) {
	extracts := make([]types.Extract, 0, len(values))
	for _, value := range values {
		extract, err := parseExtractFlag(value)
		if err != nil {
			return nil, err
		}
		extracts = append(extracts, extract)
	}
	return extracts, nil
}

func parseExtractFlag(value string) (types.Extract, error) {
	name, source, ok := strings.Cut(value, "=")
	if !ok {
		return types.Extract{}, fmt.Errorf("extract must be VAR_NAME=METHOD:EXPRESSION")
	}

	name = strings.TrimSpace(name)
	method, expression, ok := strings.Cut(source, ":")
	if name == "" || !ok {
		return types.Extract{}, fmt.Errorf("extract must be VAR_NAME=METHOD:EXPRESSION")
	}

	method = strings.TrimSpace(method)
	expression = strings.TrimSpace(expression)
	if method == "" || expression == "" {
		return types.Extract{}, fmt.Errorf("extract must be VAR_NAME=METHOD:EXPRESSION")
	}

	switch method {
	case "jsonpath", "header", "regex":
		return types.Extract{Name: name, Source: method + ":" + expression}, nil
	case "jq":
		return types.Extract{Name: name, Source: "jsonpath:" + expression}, nil
	default:
		return types.Extract{}, fmt.Errorf("extract method must be one of jsonpath, header, regex, jq")
	}
}

func printSaveConfirmation(name string, url string) {
	if url != "" {
		fmt.Printf("✓ Saved request '%s' (%s)\n", name, url)
		return
	}
	fmt.Printf("✓ Saved request '%s'\n", name)
}

func resolveSaveNameAndURL(args cli.Args, nameFlag string) (string, string, error) {
	if nameFlag != "" {
		if args.Len() != 1 {
			return "", "", fmt.Errorf("URL argument is required when using --name (usage: gurl save --name <name> <url>)")
		}
		return nameFlag, args.Get(0), nil
	}

	if args.Len() < 2 {
		return "", "", fmt.Errorf("name and URL arguments are required")
	}
	if args.Len() > 2 {
		return "", "", fmt.Errorf("too many arguments: expected name and URL")
	}
	return args.Get(0), args.Get(1), nil
}
