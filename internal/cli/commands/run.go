package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sreeram/gurl/internal/client"
	"github.com/sreeram/gurl/internal/core/template"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/urfave/cli/v3"
)

func RunCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "run",
		Aliases: []string{"r", "execute"},
		Usage:   "Execute a saved request",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:    "var",
				Aliases: []string{"v"},
				Usage:   "Variable substitution (--var key=value)",
			},
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Usage:   "Output format (auto|json|table)",
				Value:   "auto",
			},
			&cli.BoolFlag{
				Name:    "cache",
				Aliases: []string{"c"},
				Usage:   "Use cached response if fresh",
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

			varSlice := c.StringSlice("var")
			vars := make(map[string]string)
			for _, pair := range varSlice {
				if idx := strings.Index(pair, "="); idx != -1 {
					vars[pair[:idx]] = pair[idx+1:]
				}
			}

			substitutedURL, err := template.Substitute(req.URL, vars)
			if err != nil {
				return fmt.Errorf("variable substitution failed: %w", err)
			}

			substitutedBody, _ := template.Substitute(req.Body, vars)

			clientReq := client.Request{
				Method:  req.Method,
				URL:     substitutedURL,
				Headers: convertHeaders(req.Headers),
				Body:    substitutedBody,
			}

			resp, err := client.Execute(clientReq)
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}

			history := types.NewExecutionHistory(
				req.ID,
				string(resp.Body),
				resp.StatusCode,
				resp.Duration.Milliseconds(),
				resp.Size,
			)
			if err := db.SaveHistory(history); err != nil {
				return fmt.Errorf("failed to save history: %w", err)
			}

			format := c.String("format")
			return printResponse(os.Stdout, resp.Body, format)

		},
	}
}

func convertHeaders(headers []types.Header) []client.Header {
	result := make([]client.Header, len(headers))
	for i, h := range headers {
		result[i] = client.Header{Key: h.Key, Value: h.Value}
	}
	return result
}

func printResponse(out *os.File, body []byte, format string) error {
	switch format {
	case "json":
		var data interface{}
		if json.Unmarshal(body, &data) == nil {
			enc := json.NewEncoder(out)
			enc.SetIndent("", "  ")
			return enc.Encode(data)
		}
		_, err := out.Write(body)
		return err
	case "table":
		var data interface{}
		if json.Unmarshal(body, &data) == nil {
			enc := json.NewEncoder(out)
			enc.SetIndent("  ", "")
			return enc.Encode(data)
		}
		_, err := out.Write(body)
		return err
	default:
		_, err := out.Write(body)
		return err
	}
}
