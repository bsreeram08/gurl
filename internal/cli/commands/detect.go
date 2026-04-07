package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/sreeram/gurl/internal/core/curl"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/urfave/cli/v3"
)

// DetectCommand creates the detect command
func DetectCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "detect",
		Aliases: []string{"parse", "d"},
		Usage:   "Parse curl from stdin or file",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "file",
				Aliases: []string{"f"},
				Usage:   "Read from file instead of stdin",
			},
			&cli.StringFlag{
				Name:    "name",
				Aliases: []string{"n"},
				Usage:   "Name for the detected request",
			},
			&cli.StringFlag{
				Name:    "collection",
				Aliases: []string{"c"},
				Usage:   "Assign to collection",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			filePath := c.String("file")

			var input string
			switch {
			case filePath != "":
				data, err := os.ReadFile(filePath)
				if err != nil {
					return fmt.Errorf("failed to read file: %w", err)
				}
				input = string(data)
			default:
				data, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("failed to read stdin: %w", err)
				}
				input = string(data)
			}

			input = strings.TrimSpace(input)
			if input == "" {
				return errors.New("empty input")
			}

			parsed, err := curl.ParseCurl(input)
			if err != nil {
				return fmt.Errorf("failed to parse curl: %w", err)
			}

			savedReq := types.ParsedCurlToSavedRequest(*parsed)
			savedReq.ID = fmt.Sprintf("detected-%d", time.Now().UnixNano())
			savedReq.CreatedAt = time.Now().Unix()
			savedReq.UpdatedAt = time.Now().Unix()
			savedReq.OutputFormat = "auto"

			name := c.String("name")
			if name == "" {
				name = generateNameFromURL(parsed.URL)
			}
			savedReq.Name = name

			if collection := c.String("collection"); collection != "" {
				savedReq.Collection = collection
			}

			if err := db.SaveRequest(&savedReq); err != nil {
				return fmt.Errorf("failed to save request: %w", err)
			}

			fmt.Printf("✓ Detected and saved request '%s'\n", savedReq.Name)
			fmt.Printf("  URL: %s\n", savedReq.URL)
			fmt.Printf("  Method: %s\n", savedReq.Method)
			if len(savedReq.Headers) > 0 {
				fmt.Printf("  Headers: %d\n", len(savedReq.Headers))
			}
			if savedReq.Body != "" {
				fmt.Printf("  Body: %d bytes\n", len(savedReq.Body))
			}
			if savedReq.Collection != "" {
				fmt.Printf("  Collection: %s\n", savedReq.Collection)
			}

			return nil
		},
	}
}

// generateNameFromURL creates a name from the URL when --name is not provided
func generateNameFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Sprintf("request-%d", time.Now().Unix())
	}

	host := parsed.Host
	path := strings.TrimPrefix(parsed.Path, "/")

	var name string
	if path != "" {
		parts := strings.Split(path, "/")
		if len(parts) > 0 && parts[0] != "" {
			name = fmt.Sprintf("%s-%s", host, parts[0])
		} else {
			name = host
		}
	} else {
		name = host
	}

	name = strings.ReplaceAll(name, ".", "-")
	name = strings.ReplaceAll(name, "/", "-")

	if len(name) < 3 {
		name = fmt.Sprintf("request-%d", time.Now().Unix())
	}

	return name
}
