package commands

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/urfave/cli/v3"
	"github.com/sreeram/gurl/internal/storage"
)

// PasteCommand creates the paste command
func PasteCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "paste",
		Aliases: []string{"clip", "copy"},
		Usage:   "Copy request as curl command to clipboard",
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

			// Build curl command using array form to prevent shell injection
			curlCmd := []string{"curl", "-X", req.Method}

			for _, header := range req.Headers {
				curlCmd = append(curlCmd, "-H", fmt.Sprintf("%s: %s", shellEscape(header.Key), shellEscape(header.Value)))
			}

			if req.Body != "" {
				curlCmd = append(curlCmd, "-d", shellEscape(req.Body))
			}

			curlCmd = append(curlCmd, req.URL)

			// Join for display (safe - no shell execution)
			displayCmd := strings.Join(curlCmd, " ")

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
