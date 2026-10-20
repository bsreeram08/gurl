package commands

import (
	"context"
	"fmt"

	"github.com/sreeram/gurl/internal/codegen"
	"github.com/sreeram/gurl/internal/redact"
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
			&cli.BoolFlag{
				Name:  "reveal",
				Usage: "Include secret header and auth values instead of masking them (Authorization always becomes a placeholder)",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			args := c.Args()
			if args.Len() < 1 {
				return fmt.Errorf("request name argument is required")
			}
			name := args.Get(0)
			if err := rejectReveal(c); err != nil {
				return err
			}
			if agentMode(c) && c.Bool("clipboard") {
				return cli.Exit("--clipboard is not allowed in agent mode; codegen prints masked code to stdout", 2)
			}

			req, err := db.GetRequestByName(name)
			if err != nil {
				return fmt.Errorf("request not found: %s", name)
			}
			if !c.Bool("reveal") {
				req = redact.SavedRequest(req)
			}

			lang := c.String("lang")
			opts := &codegen.GenOptions{}

			code, err := codegen.Generate(lang, req, opts)
			if err != nil {
				return err
			}

			if c.Bool("clipboard") {
				if err := clipboardWrite(code); err != nil {
					return fmt.Errorf("failed to copy to clipboard: %w", err)
				}
				return nil
			}

			fmt.Println(code)
			return nil
		},
	}
}
