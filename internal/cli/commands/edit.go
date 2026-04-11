package commands

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/urfave/cli/v3"
)

// validMethods lists all valid HTTP methods
var validMethods = map[string]bool{
	"GET":     true,
	"POST":    true,
	"PUT":     true,
	"DELETE":  true,
	"PATCH":   true,
	"HEAD":    true,
	"OPTIONS": true,
	"CONNECT": true,
	"TRACE":   true,
}

// EditCommand creates the edit command
func EditCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "edit",
		Aliases: []string{"e"},
		Usage:   "Edit a saved request",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "method",
				Aliases: []string{"X"},
				Usage:   "Change HTTP method",
			},
			&cli.StringFlag{
				Name:    "url",
				Aliases: []string{"u"},
				Usage:   "Change request URL",
			},
			&cli.StringSliceFlag{
				Name:    "header",
				Aliases: []string{"H"},
				Usage:   "Add header (format: 'Key: Value')",
			},
			&cli.StringSliceFlag{
				Name:    "remove-header",
				Aliases: []string{"rm-header"},
				Usage:   "Remove header by key",
			},
			&cli.StringFlag{
				Name:    "body",
				Aliases: []string{"d", "data"},
				Usage:   "Change request body",
			},
			&cli.StringFlag{
				Name:    "collection",
				Aliases: []string{"c"},
				Usage:   "Move to collection",
			},
			&cli.StringSliceFlag{
				Name:    "tag",
				Aliases: []string{"t"},
				Usage:   "Add tag (can repeat)",
			},
			&cli.StringFlag{
				Name:    "pre-script",
				Aliases: []string{"pre"},
				Usage:   "Set pre-request script path",
			},
			&cli.StringFlag{
				Name:    "post-script",
				Aliases: []string{"post"},
				Usage:   "Set post-response script path",
			},
			&cli.StringSliceFlag{
				Name:    "assert",
				Aliases: []string{"a"},
				Usage:   "Add assertion (format: 'field=op=value')",
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

			// Track if any changes were made
			var changes []string

			// Handle method change
			if method := c.String("method"); method != "" {
				method = strings.ToUpper(method)
				if !validMethods[method] {
					return fmt.Errorf("invalid HTTP method: %s", method)
				}
				req.Method = method
				changes = append(changes, fmt.Sprintf("method → %s", method))
			}

			// Handle URL change
			if newURL := c.String("url"); newURL != "" {
				// Validate URL format
				if _, err := url.Parse(newURL); err != nil {
					return fmt.Errorf("invalid URL format: %s", newURL)
				}
				req.URL = newURL
				changes = append(changes, fmt.Sprintf("url → %s", newURL))
			}

			// Handle header additions
			for _, h := range c.StringSlice("header") {
				if idx := strings.Index(h, ":"); idx != -1 {
					req.Headers = append(req.Headers, types.Header{
						Key:   strings.TrimSpace(h[:idx]),
						Value: strings.TrimSpace(h[idx+1:]),
					})
					changes = append(changes, fmt.Sprintf("added header %s", strings.TrimSpace(h[:idx])))
				} else {
					return fmt.Errorf("invalid header format '%s': missing ':' (expected 'Key: Value')", h)
				}
			}

			// Handle header removals
			for _, key := range c.StringSlice("remove-header") {
				var newHeaders []types.Header
				for _, h := range req.Headers {
					if h.Key != key {
						newHeaders = append(newHeaders, h)
					}
				}
				if len(newHeaders) != len(req.Headers) {
					changes = append(changes, fmt.Sprintf("removed header %s", key))
				}
				req.Headers = newHeaders
			}

			// Handle body change
			if body := c.String("body"); body != "" {
				req.Body = body
				changes = append(changes, "body updated")
			}

			// Handle collection change
			if collection := c.String("collection"); collection != "" {
				req.Collection = collection
				changes = append(changes, fmt.Sprintf("collection → %s", collection))
			}

			// Handle tag additions (append, don't replace)
			for _, tag := range c.StringSlice("tag") {
				// Check if tag already exists
				found := false
				for _, t := range req.Tags {
					if t == tag {
						found = true
						break
					}
				}
				if !found {
					req.Tags = append(req.Tags, tag)
					changes = append(changes, fmt.Sprintf("added tag '%s'", tag))
				}
			}

			// Handle assertion additions
			for _, a := range c.StringSlice("assert") {
				assertion, err := parseAssertion(a)
				if err != nil {
					return fmt.Errorf("invalid assertion '%s': %w (expected format: field=op=value)", a, err)
				}
				req.Assertions = append(req.Assertions, *assertion)
				changes = append(changes, fmt.Sprintf("added assertion %s", a))
			}

			// Update timestamp
			req.UpdatedAt = time.Now().Unix()

			// Save the updated request
			if err := db.UpdateRequest(req); err != nil {
				return fmt.Errorf("failed to update request: %w", err)
			}

			// Print success message with changes
			fmt.Printf("✓ Updated request '%s'\n", name)
			if len(changes) > 0 {
				fmt.Println("Changes:")
				for _, change := range changes {
					fmt.Printf("  • %s\n", change)
				}
			}

			return nil
		},
	}
}

// parseAssertion parses an assertion string in format "field=op=value"
// Valid ops: =, !=, contains, startswith, endswith, matches
func parseAssertion(s string) (*types.Assertion, error) {
	// Find the first = to split field and rest
	eqIdx := strings.Index(s, "=")
	if eqIdx == -1 {
		return nil, fmt.Errorf("assertion must contain '=' (format: field=op=value)")
	}

	field := strings.TrimSpace(s[:eqIdx])
	rest := s[eqIdx+1:]

	var op, value string
	// Check for != operator
	if len(rest) >= 2 && rest[:2] == "!=" {
		op = "!="
		value = rest[2:]
	} else if len(rest) >= 1 && rest[0] == '=' {
		// Handle leading = after we already consumed one =
		op = "="
		value = rest[1:]
	} else {
		// For contains/startswith/endswith/matches, the value is the rest
		op = "="
		value = rest
	}

	value = strings.TrimSpace(value)

	if field == "" {
		return nil, fmt.Errorf("assertion field cannot be empty")
	}
	if value == "" {
		return nil, fmt.Errorf("assertion value cannot be empty")
	}

	return &types.Assertion{
		Field: field,
		Op:    op,
		Value: value,
	}, nil
}
