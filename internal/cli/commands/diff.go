package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/internal/formatter"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/urfave/cli/v3"
)

func DiffCommand(db storage.DB, envs env.Store) *cli.Command {
	return &cli.Command{
		Name:    "diff",
		Aliases: nil,
		Usage:   "Compare responses for a request",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "limit",
				Aliases: []string{"n", "l"},
				Usage:   "Number of responses to compare",
				Value:   2,
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

			if len(history) < 2 {
				return fmt.Errorf("not enough execution history to diff (need at least 2). Use --limit to specify how many history entries to fetch")
			}

			histA := history[0]
			histB := history[1]

			if jsonOutput(c) {
				before, after := histB, histA
				if agentMode(c) {
					secrets := currentSecrets(db, envs, histA)
					before, after = maskedRecord(before, secrets), maskedRecord(after, secrets)
				}
				return writeJSON(os.Stdout, diffReport{
					Request:    name,
					Before:     diffSideOf(before),
					After:      diffSideOf(after),
					DiffResult: formatter.Diff([]byte(before.Response), []byte(after.Response)),
				})
			}

			diff, err := formatter.DiffResponses(*histA, *histB)
			if err != nil {
				return fmt.Errorf("failed to diff responses: %w", err)
			}

			fmt.Println(diff)
			return nil
		}),
	}
}

// diffReport is `diff --json` output: the response body of the newest
// execution (after) compared with the one before it (before).
type diffReport struct {
	Request string   `json:"request"`
	Before  diffSide `json:"before"`
	After   diffSide `json:"after"`
	formatter.DiffResult
}

type diffSide struct {
	ID      string `json:"id"`
	ShortID string `json:"short_id"`
	Time    string `json:"time"`
	Status  int    `json:"status"`
}

func diffSideOf(rec *types.ExecutionHistory) diffSide {
	return diffSide{ID: rec.ID, ShortID: rec.ShortID(), Time: rec.Time().Format(time.RFC3339), Status: rec.StatusCode}
}
