package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/internal/redact"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)

func EnvCommand(db env.Store) *cli.Command {
	return &cli.Command{
		Name:  "env",
		Usage: "Manage environments",
		Commands: []*cli.Command{
			{
				Name:    "create",
				Aliases: []string{"new", "add"},
				Usage:   "Create a new environment",
				UsageText: "gurl env create <name> [--var KEY=VALUE ...] [--secret KEY ...]\n" +
					"   Use KEY=- to read one value from stdin (prompted without echo on a terminal).",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name:    "var",
						Aliases: []string{"v"},
						Usage:   "Variable in KEY=VALUE format (can repeat); KEY=- reads the value from stdin",
					},
					&cli.StringSliceFlag{
						Name:    "secret",
						Aliases: []string{"s"},
						Usage:   "Mark variable as secret (KEY name, can repeat)",
					},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					args := c.Args()
					if args.Len() < 1 {
						return fmt.Errorf("environment name argument is required")
					}
					name := args.Get(0)

					newEnv := env.NewEnvironment(name, "")

					vars, err := parseVarAssignments(c, c.StringSlice("var"))
					if err != nil {
						return err
					}
					for _, kv := range vars {
						newEnv.SetVariable(kv[0], kv[1])
					}

					for _, key := range c.StringSlice("secret") {
						newEnv.SecretKeys[key] = true
					}

					if err := db.SaveEnv(newEnv); err != nil {
						return fmt.Errorf("failed to create environment: %w", err)
					}

					fmt.Printf("✓ Environment '%s' created\n", name)
					return nil
				},
			},
			{
				Name:    "list",
				Aliases: []string{"ls", "l"},
				Usage:   "List all environments",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "json",
						Usage: "Output as JSON",
					},
				},
				Action: jsonAction(func(ctx context.Context, c *cli.Command) error {
					envs, err := db.ListEnvs()
					if err != nil {
						return fmt.Errorf("failed to list environments: %w", err)
					}

					activeEnv, _ := db.GetActiveEnv()

					if jsonOutput(c) {
						type envSummary struct {
							Name      string `json:"name"`
							Active    bool   `json:"active"`
							Variables int    `json:"variables"`
						}
						out := make([]envSummary, 0, len(envs))
						for _, e := range envs {
							out = append(out, envSummary{Name: e.Name, Active: e.Name == activeEnv, Variables: len(e.Variables)})
						}
						data, err := json.MarshalIndent(out, "", "  ")
						if err != nil {
							return fmt.Errorf("failed to marshal environments: %w", err)
						}
						fmt.Println(string(data))
						return nil
					}

					if len(envs) == 0 {
						fmt.Println("No environments found.")
						return nil
					}

					fmt.Println("┌─ Environments ────────────────────────────────────────────┐")
					fmt.Println("│  NAME          ACTIVE   VARIABLES                        │")
					fmt.Println("├──────────────────────────────────────────────────────────┤")

					for _, e := range envs {
						activeMarker := " "
						if e.Name == activeEnv {
							activeMarker = "*"
						}
						varCount := len(e.Variables)
						fmt.Printf("│  %-13s %s        %d\n", e.Name, activeMarker, varCount)
					}

					fmt.Println("└──────────────────────────────────────────────────────────┘")
					return nil
				}),
			},
			{
				Name:    "switch",
				Aliases: []string{"use", "activate"},
				Usage:   "Switch to an environment",
				Action: func(ctx context.Context, c *cli.Command) error {
					args := c.Args()
					if args.Len() < 1 {
						return fmt.Errorf("environment name argument is required")
					}
					name := args.Get(0)

					env, err := db.GetEnvByName(name)
					if err != nil {
						return fmt.Errorf("failed to get environment '%s': %w", name, err)
					}
					if env == nil {
						return fmt.Errorf("environment '%s' not found", name)
					}

					if err := db.SetActiveEnv(name); err != nil {
						return fmt.Errorf("failed to switch environment: %w", err)
					}

					fmt.Printf("✓ Switched to environment '%s'\n", name)
					return nil
				},
			},
			{
				Name:    "delete",
				Aliases: []string{"rm", "remove", "del"},
				Usage:   "Delete an environment",
				Action: func(ctx context.Context, c *cli.Command) error {
					args := c.Args()
					if args.Len() < 1 {
						return fmt.Errorf("environment name argument is required")
					}
					name := args.Get(0)

					env, err := db.GetEnvByName(name)
					if err != nil {
						return fmt.Errorf("failed to get environment '%s': %w", name, err)
					}
					if env == nil {
						return fmt.Errorf("environment '%s' not found", name)
					}

					if err := db.DeleteEnv(env.ID); err != nil {
						return fmt.Errorf("failed to delete environment: %w", err)
					}

					fmt.Printf("✓ Deleted environment '%s'\n", name)
					return nil
				},
			},
			{
				Name:    "show",
				Aliases: []string{"display", "view"},
				Usage:   "Show environment details",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "json",
						Usage: "Output as JSON",
					},
					&cli.BoolFlag{
						Name:  "reveal",
						Usage: "Show secret values instead of masking them",
					},
				},
				Action: jsonAction(func(ctx context.Context, c *cli.Command) error {
					args := c.Args()
					if args.Len() < 1 {
						return fmt.Errorf("environment name argument is required")
					}
					name := args.Get(0)

					env, err := db.GetEnvByName(name)
					if err != nil {
						return fmt.Errorf("failed to get environment '%s': %w", name, err)
					}
					if env == nil {
						return fmt.Errorf("environment '%s' not found", name)
					}

					if err := rejectReveal(c); err != nil {
						return err
					}
					reveal := c.Bool("reveal")
					// Agent mode also masks sensitively named values that
					// are not marked secret.
					strict := agentMode(c)
					if jsonOutput(c) {
						out := struct {
							Name      string            `json:"name"`
							ID        string            `json:"id"`
							Variables map[string]string `json:"variables"`
							Secrets   []string          `json:"secrets"`
						}{Name: env.Name, ID: env.ID, Variables: make(map[string]string, len(env.Variables)), Secrets: []string{}}
						for k, v := range env.Variables {
							if env.IsSecret(k) {
								out.Secrets = append(out.Secrets, k)
								if !reveal {
									v = redact.Mask
								}
							}
							if strict && redact.SensitiveName(k) && v != "" {
								v = redact.Mask
							}
							out.Variables[k] = v
						}
						sort.Strings(out.Secrets)
						data, err := json.MarshalIndent(out, "", "  ")
						if err != nil {
							return fmt.Errorf("failed to marshal environment: %w", err)
						}
						fmt.Println(string(data))
						return nil
					}

					fmt.Printf("Environment: %s\n", env.Name)
					fmt.Printf("ID: %s\n", env.ID)
					if len(env.Variables) == 0 {
						fmt.Println("Variables: (none)")
					} else {
						fmt.Println("Variables:")
						for k, v := range env.Variables {
							if env.IsSecret(k) && !reveal {
								v = redact.Mask
							}
							fmt.Printf("  %s = %s\n", k, v)
						}
					}
					return nil
				}),
			},
			{
				Name:  "set",
				Usage: "Set a variable in an environment",
				UsageText: "gurl env set <env> KEY=VALUE [KEY=VALUE ...]\n" +
					"   gurl env set <env> KEY VALUE\n" +
					"   Use - as the value (KEY=-) to read it from stdin (prompted without echo on a terminal).",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name:    "var",
						Aliases: []string{"v"},
						Usage:   "Variable in KEY=VALUE format (can repeat); positional KEY=VALUE or KEY VALUE also supported; value - reads from stdin",
					},
					&cli.StringSliceFlag{
						Name:    "secret",
						Aliases: []string{"s"},
						Usage:   "Mark variable as secret (KEY name, can repeat)",
					},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					args := c.Args()
					if args.Len() < 1 {
						return fmt.Errorf("environment name argument is required")
					}
					name := args.Get(0)

					env, err := db.GetEnvByName(name)
					if err != nil {
						return fmt.Errorf("failed to get environment '%s': %w", name, err)
					}
					if env == nil {
						return fmt.Errorf("environment '%s' not found", name)
					}

					varPairs := c.StringSlice("var")
					positionalPairs, err := parseEnvSetArgs(args)
					if err != nil {
						return err
					}
					varPairs = append(varPairs, positionalPairs...)
					if len(varPairs) == 0 && len(c.StringSlice("secret")) == 0 {
						return fmt.Errorf("variable is required: use --var KEY=VALUE, KEY=VALUE, or KEY VALUE")
					}

					vars, err := parseVarAssignments(c, varPairs)
					if err != nil {
						return err
					}
					for _, kv := range vars {
						env.SetVariable(kv[0], kv[1])
					}

					for _, key := range c.StringSlice("secret") {
						env.SecretKeys[key] = true
					}

					if err := db.SaveEnv(env); err != nil {
						return fmt.Errorf("failed to update environment: %w", err)
					}

					fmt.Printf("✓ Updated environment '%s'\n", name)
					return nil
				},
			},
			{
				Name:  "unset",
				Usage: "Unset a variable in an environment",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name:     "var",
						Aliases:  []string{"v"},
						Usage:    "Variable key to remove (can repeat)",
						Required: true,
					},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					args := c.Args()
					if args.Len() < 1 {
						return fmt.Errorf("environment name argument is required")
					}
					name := args.Get(0)

					env, err := db.GetEnvByName(name)
					if err != nil {
						return fmt.Errorf("failed to get environment '%s': %w", name, err)
					}
					if env == nil {
						return fmt.Errorf("environment '%s' not found", name)
					}

					for _, key := range c.StringSlice("var") {
						env.DeleteVariable(key)
					}

					if err := db.SaveEnv(env); err != nil {
						return fmt.Errorf("failed to update environment: %w", err)
					}

					fmt.Printf("✓ Updated environment '%s'\n", name)
					return nil
				},
			},
			{
				Name:    "import",
				Aliases: []string{"imp"},
				Usage:   "Import variables from a .env file",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "file",
						Aliases:  []string{"f"},
						Usage:    "Path to .env file",
						Required: true,
					},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					args := c.Args()
					if args.Len() < 1 {
						return fmt.Errorf("environment name argument is required")
					}
					name := args.Get(0)

					filePath := c.String("file")

					vars, err := env.ParseDotenvFile(filePath)
					if err != nil {
						return fmt.Errorf("failed to parse .env file: %w", err)
					}

					envObj, err := db.GetEnvByName(name)
					if err != nil && envObj == nil {
						// Only create new if error indicates not found, not other errors
						envObj = env.NewEnvironment(name, "")
					} else if err != nil {
						return fmt.Errorf("failed to get environment: %w", err)
					}
					if envObj == nil {
						envObj = env.NewEnvironment(name, "")
					}

					for k, v := range vars {
						envObj.SetVariable(k, v)
					}

					if err := db.SaveEnv(envObj); err != nil {
						return fmt.Errorf("failed to save environment: %w", err)
					}

					fmt.Printf("✓ Imported %d variable(s) from '%s' into environment '%s'\n", len(vars), filePath, name)
					return nil
				},
			},
		},
	}
}

func parseEnvSetArgs(args cli.Args) ([]string, error) {
	if args.Len() <= 1 {
		return nil, nil
	}

	positional := make([]string, 0, args.Len()-1)
	for i := 1; i < args.Len(); i++ {
		positional = append(positional, args.Get(i))
	}

	allAssignments := true
	for _, arg := range positional {
		if !strings.Contains(arg, "=") {
			allAssignments = false
			break
		}
	}
	if allAssignments {
		return positional, nil
	}

	if len(positional) == 2 && !strings.Contains(positional[0], "=") {
		key := positional[0]
		if positional[1] == stdinValueMarker {
			// The flag parser silently drops everything after a lone "-",
			// so a trailing --secret would be lost without this guard.
			return nil, fmt.Errorf("use %s=- to read the value from stdin", key)
		}
		if key == "" {
			return nil, fmt.Errorf("invalid variable format: KEY cannot be empty")
		}
		return []string{key + "=" + positional[1]}, nil
	}

	return nil, fmt.Errorf("invalid env set syntax: expected 'env set <env> KEY=VALUE [KEY=VALUE...]' or 'env set <env> KEY VALUE'")
}

// stdinValueMarker is the variable value that means "read the value from stdin".
const stdinValueMarker = "-"

// parseVarAssignments splits KEY=VALUE pairs. A value of "-" is read from the
// command's stdin so secrets stay out of argv and shell history; at most one
// key per invocation may use it.
func parseVarAssignments(c *cli.Command, pairs []string) ([][2]string, error) {
	out := make([][2]string, 0, len(pairs))
	stdinKey := ""
	for _, v := range pairs {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid variable format '%s': must be KEY=VALUE", v)
		}
		if parts[0] == "" {
			return nil, fmt.Errorf("invalid variable format '%s': KEY cannot be empty", v)
		}
		if parts[1] == stdinValueMarker {
			if stdinKey != "" {
				return nil, fmt.Errorf("only one variable may read its value from stdin (got '%s' and '%s')", stdinKey, parts[0])
			}
			stdinKey = parts[0]
		}
		out = append(out, [2]string{parts[0], parts[1]})
	}
	if stdinKey == "" {
		return out, nil
	}

	value, err := readStdinValue(commandReader(c), stdinKey, !noInput(c))
	if err != nil {
		return nil, err
	}
	for i := range out {
		if out[i][0] == stdinKey && out[i][1] == stdinValueMarker {
			out[i][1] = value
		}
	}
	return out, nil
}

func commandReader(c *cli.Command) io.Reader {
	if r := c.Root().Reader; r != nil {
		return r
	}
	return os.Stdin
}

// readStdinValue reads a single value for key. On a terminal it prompts on
// stderr without echo, or fails when prompt is false (non-interactive mode);
// otherwise it reads all input and trims one trailing newline.
func readStdinValue(r io.Reader, key string, prompt bool) (string, error) {
	if f, ok := r.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		if !prompt {
			return "", fmt.Errorf("value for %s must be piped on stdin with --no-input (e.g. printf %%s \"$VALUE\" | gurl env set <env> %s=-)", key, key)
		}
		fmt.Fprintf(os.Stderr, "Value for %s: ", key)
		b, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", fmt.Errorf("failed to read value for %s: %w", key, err)
		}
		return string(b), nil
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("failed to read value for %s from stdin: %w", key, err)
	}
	value := string(b)
	if strings.HasSuffix(value, "\r\n") {
		value = strings.TrimSuffix(value, "\r\n")
	} else {
		value = strings.TrimSuffix(value, "\n")
	}
	return value, nil
}
