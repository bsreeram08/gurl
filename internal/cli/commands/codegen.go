package commands

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/sreeram/gurl/internal/codegen"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/urfave/cli/v3"
)

// CodegenCommand creates the codegen command
func CodegenCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "codegen",
		Aliases: []string{"cg"},
		Usage:   "Generate code in various languages from a saved request",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "lang",
				Aliases:  []string{"l"},
				Usage:    "Language to generate: go, python, javascript, curl",
				Required: true,
			},
			&cli.BoolFlag{
				Name:    "clipboard",
				Aliases: []string{"c"},
				Usage:   "Copy output to clipboard instead of printing to stdout",
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

			lang := c.String("lang")
			opts := &codegen.GenOptions{}

			code, err := codegen.Generate(lang, req, opts)
			if err != nil {
				return err
			}

			if c.Bool("clipboard") {
				return copyToClipboard(code)
			}

			fmt.Println(code)
			return nil
		},
	}
}

func copyToClipboard(text string) error {
	// Try pbcopy (macOS) first
	if isCommandAvailable("pbcopy") {
		cmd := exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}
	// Try xclip (Linux)
	if isCommandAvailable("xclip") {
		cmd := exec.Command("xclip", "-selection", "clipboard")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}
	// Try wl-copy (Wayland)
	if isCommandAvailable("wl-copy") {
		cmd := exec.Command("wl-copy")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}
	return fmt.Errorf("clipboard tools not available. Install xclip (Linux) or wl-copy (Wayland) to enable clipboard copy")
}
