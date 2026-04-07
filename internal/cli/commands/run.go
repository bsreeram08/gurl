package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sreeram/gurl/internal/core/template"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/urfave/cli/v3"
)

// RunCommand creates the run command
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

			// Build curl command
			cmdParts := []string{"curl", "-s", "-w", "\\n%{http_code}"}

			if req.Method != "GET" {
				cmdParts = append(cmdParts, "-X", req.Method)
			}

			for _, header := range req.Headers {
				cmdParts = append(cmdParts, "-H", fmt.Sprintf("%s: %s", header.Key, header.Value))
			}

			if substitutedBody != "" {
				cmdParts = append(cmdParts, "-d", substitutedBody)
			}

			cmdParts = append(cmdParts, substitutedURL)

			cmd := exec.Command("curl", cmdParts[1:]...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			start := time.Now()
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("curl execution failed: %w", err)
			}
			duration := time.Since(start)

			history := &types.ExecutionHistory{
				RequestID:  req.ID,
				DurationMs: duration.Milliseconds(),
				Timestamp:  time.Now().Unix(),
			}
			db.SaveHistory(history)

			return nil
		},
	}
}
