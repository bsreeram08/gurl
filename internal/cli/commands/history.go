package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
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
			&cli.BoolFlag{
				Name:    "json",
				Aliases: []string{"j"},
				Usage:   "JSON output",
			},
		},
		Action: jsonAction(func(ctx context.Context, c *cli.Command) error {
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

			if jsonOutput(c) {
				out := make([]recordSummary, 0, len(history))
				for _, h := range history {
					out = append(out, recordSummaryOf(h))
				}
				return writeJSON(os.Stdout, out)
			}

			if len(history) == 0 {
				fmt.Printf("No execution history for '%s'\n", name)
				return nil
			}

			// Print table header
			fmt.Printf("┌─ History: %s ───────────────────────────────────────────────────┐\n", name)
			fmt.Println("│  #   ID         STATUS   DURATION   SIZE     TIMESTAMP           │")
			fmt.Println("├────────────────────────────────────────────────────────────────────┤")

			for i, h := range history {
				timestamp := time.Unix(h.Timestamp, 0).Format("2006-01-02 15:04:05")
				status := fmt.Sprintf("%d", h.StatusCode)
				if h.StatusCode == 0 && h.Error != "" {
					status = "ERR"
				}
				duration := fmt.Sprintf("%dms", h.DurationMs)
				size := formatBytes(h.SizeBytes)

				fmt.Printf("│  %d   %-8s   %-6s   %-8s   %-8s %s\n",
					i+1, h.ShortID(), status, duration, size, timestamp)
				if h.Error != "" {
					fmt.Printf("│       ERROR: %s\n", h.Error)
				}

				if h.Response != "" {
					preview := h.Response
					if len(preview) > 60 {
						preview = preview[:60] + "..."
					}
					preview = strings.ReplaceAll(preview, "\n", " ")
					fmt.Printf("│       RESP: %s\n", preview)
				}
			}

			fmt.Println("└────────────────────────────────────────────────────────────────────┘")

			return nil
		}),
	}
}

// recordSummary is one execution record in `history --json` and
// `timeline --json`. Error is masked as stored.
type recordSummary struct {
	ID           string `json:"id"`
	ShortID      string `json:"short_id"`
	Request      string `json:"request,omitempty"`
	Time         string `json:"time"`
	OK           bool   `json:"ok"`
	Status       int    `json:"status,omitempty"`
	DurationMs   int64  `json:"duration_ms"`
	SizeBytes    int64  `json:"size_bytes"`
	Error        string `json:"error,omitempty"`
	FailurePhase string `json:"failure_phase,omitempty"`
}

func recordSummaryOf(h *types.ExecutionHistory) recordSummary {
	return recordSummary{
		ID:           h.ID,
		ShortID:      h.ShortID(),
		Request:      h.RequestName,
		Time:         h.Time().Format(time.RFC3339),
		OK:           !h.Failed(),
		Status:       h.StatusCode,
		DurationMs:   h.DurationMs,
		SizeBytes:    h.SizeBytes,
		Error:        h.Error,
		FailurePhase: h.FailurePhase,
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
