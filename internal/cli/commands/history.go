package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sreeram/gurl/internal/storage"
	"github.com/urfave/cli/v3"
)

// HistoryCommand creates the history command
func HistoryCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "history",
		Aliases: []string{"hist", "h"},
		Usage:   "Show execution history for a request",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "limit",
				Aliases: []string{"n", "l"},
				Usage:   "Limit number of entries",
				Value:   10,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			args := c.Args()
			if args.Len() < 1 {
				return fmt.Errorf("request name argument is required")
			}
			name := args.Get(0)
			limit := c.Int("limit")

			req, err := db.GetRequestByName(name)
			if err != nil {
				return fmt.Errorf("request not found: %s", name)
			}

			history, err := db.GetHistory(req.ID, limit)
			if err != nil {
				return fmt.Errorf("failed to get history: %w", err)
			}

			if len(history) == 0 {
				fmt.Printf("No execution history for '%s'\n", name)
				return nil
			}

			// Print table header
			fmt.Printf("┌─ History: %s ─────────────────────────────────────────┐\n", name)
			fmt.Println("│  #   STATUS   DURATION   SIZE     TIMESTAMP           │")
			fmt.Println("├──────────────────────────────────────────────────────────┤")

			for i, h := range history {
				timestamp := time.Unix(h.Timestamp, 0).Format("2006-01-02 15:04:05")
				status := fmt.Sprintf("%d", h.StatusCode)
				duration := fmt.Sprintf("%dms", h.DurationMs)
				size := formatBytes(h.SizeBytes)

				fmt.Printf("│  %d   %-6s   %-8s   %-8s %s\n",
					i+1, status, duration, size, timestamp)

				if h.Response != "" {
					preview := h.Response
					if len(preview) > 60 {
						preview = preview[:60] + "..."
					}
					preview = strings.ReplaceAll(preview, "\n", " ")
					fmt.Printf("│       RESP: %s\n", preview)
				}
			}

			fmt.Println("└──────────────────────────────────────────────────────────┘")

			return nil
		},
	}
}

// formatBytes formats byte size to human readable string
func formatBytes(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%dB", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.1fKB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.1fMB", float64(bytes)/(1024*1024))
}
