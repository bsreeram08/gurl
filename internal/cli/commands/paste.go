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

			// Build curl command string
			curlCmd := fmt.Sprintf("curl -X %s", req.Method)

			for _, header := range req.Headers {
				curlCmd += fmt.Sprintf(" -H '%s: %s'", header.Key, header.Value)
			}

			if req.Body != "" {
				curlCmd += fmt.Sprintf(" -d '%s'", req.Body)
			}

			curlCmd += fmt.Sprintf(" '%s'", req.URL)

			// Try to copy to clipboard using pbcopy (macOS) or xclip (Linux)
			var cmd *exec.Cmd
			switch {
			case isCommandAvailable("pbcopy"):
				cmd = exec.Command("pbcopy")
			case isCommandAvailable("xclip"):
				cmd = exec.Command("xclip", "-selection", "clipboard")
			case isCommandAvailable("wl-copy"):
				cmd = exec.Command("wl-copy")
			default:
				// Fallback: just print the command
				fmt.Println("Clipboard tools not available. Curl command:")
				fmt.Println(curlCmd)
				return nil
			}

			cmd.Stdin = strings.NewReader(curlCmd)
			if err := cmd.Run(); err != nil {
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
