package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/urfave/cli/v3"
)

type timelineEntry struct {
	requestName string
	history     *types.ExecutionHistory
}

// TimelineCommand creates the timeline command
func TimelineCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "timeline",
		Aliases: []string{"tl", "log"},
		Usage:   "Show global execution timeline",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "since",
				Aliases: []string{"s"},
				Usage:   "Show since duration (e.g., 24h, 7d)",
			},
			&cli.StringFlag{
				Name:    "filter",
				Aliases: []string{"f"},
				Usage:   "Filter by name pattern",
			},
			&cli.IntFlag{
				Name:    "limit",
				Aliases: []string{"n", "l"},
				Usage:   "Limit number of entries",
				Value:   50,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			limit := c.Int("limit")
			filter := c.String("filter")

			// Get all requests and their history
			requests, err := db.ListRequests(&storage.ListOptions{})
			if err != nil {
				return fmt.Errorf("failed to list requests: %w", err)
			}

			var entries []timelineEntry
			for _, req := range requests {
				if filter != "" && !strings.Contains(req.Name, filter) {
					continue
				}
				history, err := db.GetHistory(req.ID, limit)
				if err != nil {
					continue
				}
				for _, h := range history {
					entries = append(entries, timelineEntry{
						requestName: req.Name,
						history:     h,
					})
				}
			}

			// Sort by timestamp (most recent first)
			sortByTimestamp(entries)

			if len(entries) == 0 {
				fmt.Println("No execution history found.")
				return nil
			}

			// Apply limit
			if len(entries) > limit {
				entries = entries[:limit]
			}

			fmt.Println("┌─ Execution Timeline ─────────────────────────────────────┐")

			for _, e := range entries {
				timestamp := time.Unix(e.history.Timestamp, 0).Format("15:04:05")
				status := fmt.Sprintf("%d", e.history.StatusCode)
				duration := fmt.Sprintf("%dms", e.history.DurationMs)

				statusIcon := "●"
				if e.history.StatusCode >= 400 {
					statusIcon = "✗"
				} else if e.history.StatusCode >= 300 {
					statusIcon = "→"
				}

				fmt.Printf("│  %s  %s %-20s %s %s\n",
					timestamp, statusIcon, e.requestName, status, duration)
			}

			fmt.Println("└────────────────────────────────────────────────────────────┘")

			return nil
		},
	}
}

// sortByTimestamp sorts entries by timestamp, most recent first
func sortByTimestamp(entries []timelineEntry) {
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].history.Timestamp < entries[j].history.Timestamp {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
}
