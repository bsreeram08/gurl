package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/sreeram/gurl/internal/env"
	"github.com/urfave/cli/v3"
)

// ExecCommand creates the exec command, which runs a child process with an
// environment's variables injected into its process environment. Values are
// never printed.
func ExecCommand(db env.Store) *cli.Command {
	// Stop flag parsing at the first positional so the child's own flags
	// (e.g. sh -c) are passed through even without "--".
	stopAtCommand := 1
	return &cli.Command{
		Name:  "exec",
		Usage: "Run a command with environment variables injected (values are never printed)",
		UsageText: "gurl exec [--env NAME] [--var KEY=VALUE ...] -- <command> [args...]\n\n" +
			"   gurl exec --env prod -- sh -c 'curl -H \"Authorization: Bearer $API_KEY\" \"$BASE_URL/v1/charges\"'",
		StopOnNthArg: &stopAtCommand,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "env",
				Aliases: []string{"e"},
				Usage:   "Environment to load (defaults to the active environment)",
			},
			&cli.StringSliceFlag{
				Name:    "var",
				Aliases: []string{"v"},
				Usage:   "Override variable in KEY=VALUE format (can repeat)",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			args := c.Args().Slice()
			if len(args) == 0 {
				return fmt.Errorf("command is required: gurl exec [--env NAME] -- <command> [args...]")
			}

			envName := c.String("env")
			if envName == "" {
				active, err := db.GetActiveEnv()
				if err != nil {
					return fmt.Errorf("failed to get active environment: %w", err)
				}
				if active == "" {
					return fmt.Errorf("no environment selected: use --env NAME or 'gurl env switch NAME'")
				}
				envName = active
			}

			e, err := db.GetEnvByName(envName)
			if err != nil {
				return fmt.Errorf("failed to get environment '%s': %w", envName, err)
			}
			if e == nil {
				return fmt.Errorf("environment '%s' not found", envName)
			}

			vars := e.ResolvedVariables()
			for _, v := range c.StringSlice("var") {
				key, value, ok := strings.Cut(v, "=")
				if !ok || key == "" {
					return fmt.Errorf("invalid variable format '%s': must be KEY=VALUE", v)
				}
				vars[key] = value
			}

			// Later entries win in exec.Cmd.Env, so injected vars override os.Environ().
			childEnv := os.Environ()
			for k, v := range vars {
				childEnv = append(childEnv, k+"="+v)
			}

			cmd := exec.CommandContext(ctx, args[0], args[1:]...)
			cmd.Env = childEnv
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) {
					code := exitErr.ExitCode()
					if code < 0 {
						code = 1 // killed by signal
					}
					return cli.Exit("", code)
				}
				return fmt.Errorf("failed to run %s: %w", args[0], err)
			}
			return nil
		},
	}
}
