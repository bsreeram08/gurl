package commands

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/sreeram/gurl/internal/redact"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/urfave/cli/v3"
)

// PasteCommand creates the paste command
func PasteCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "paste",
		Aliases: []string{"clip", "copy"},
		Usage:   "Copy request as curl command to clipboard",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "reveal",
				Usage: "Include secret header and auth values instead of masking them",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			if agentMode(c) {
				return cli.Exit("paste is not allowed in agent mode (it copies to the clipboard); use: gurl show <name> --format curl", 2)
			}
			args := c.Args()
			if args.Len() < 1 {
				return fmt.Errorf("request name argument is required")
			}
			name := args.Get(0)

			req, err := db.GetRequestByName(name)
			if err != nil {
				return fmt.Errorf("request not found: %s", name)
			}
			if !c.Bool("reveal") {
				req = redact.SavedRequest(req)
			}

			// Join for display (safe - no shell execution)
			displayCmd := strings.Join(curlLines(req), " ")

			// Try to copy to clipboard using pbcopy (macOS) or xclip (Linux)
			var copyCmd *exec.Cmd
			switch {
			case isCommandAvailable("pbcopy"):
				copyCmd = exec.Command("pbcopy")
			case isCommandAvailable("xclip"):
				copyCmd = exec.Command("xclip", "-selection", "clipboard")
			case isCommandAvailable("wl-copy"):
				copyCmd = exec.Command("wl-copy")
			default:
				// Fallback: just print the command
				fmt.Println("Clipboard tools not available. Curl command:")
				fmt.Println(displayCmd)
				return nil
			}

			// Use array form - no shell=True
			copyCmd.Stdin = strings.NewReader(displayCmd)
			if err := copyCmd.Run(); err != nil {
				return fmt.Errorf("failed to copy to clipboard: %w", err)
			}

			fmt.Printf("✓ Copied curl command for '%s' to clipboard\n", name)
			return nil
		},
	}
}

// isCommandAvailable checks if a command is available in PATH
func isCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
