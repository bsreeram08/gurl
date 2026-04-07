package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/sreeram/gurl/internal/env"
	"github.com/urfave/cli/v3"
)

type EnvStorage interface {
	SaveEnv(e *env.Environment) error
	GetEnv(id string) (*env.Environment, error)
	DeleteEnv(id string) error
	ListEnvs() ([]*env.Environment, error)
	GetEnvByName(name string) (*env.Environment, error)
	GetActiveEnv() (string, error)
	SetActiveEnv(name string) error
}

func EnvCommand(db EnvStorage) *cli.Command {
	return &cli.Command{
		Name:  "env",
		Usage: "Manage environments",
		Commands: []*cli.Command{
			{
				Name:    "create",
				Aliases: []string{"new", "add"},
				Usage:   "Create a new environment",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name:    "var",
						Aliases: []string{"v"},
						Usage:   "Variable in KEY=VALUE format (can repeat)",
					},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					args := c.Args()
					if args.Len() < 1 {
						return fmt.Errorf("environment name argument is required")
					}
					name := args.Get(0)

					newEnv := env.NewEnvironment(name, "")

					for _, v := range c.StringSlice("var") {
						parts := strings.SplitN(v, "=", 2)
						if len(parts) == 2 {
							newEnv.SetVariable(parts[0], parts[1])
						}
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
				Action: func(ctx context.Context, c *cli.Command) error {
					envs, err := db.ListEnvs()
					if err != nil {
						return fmt.Errorf("failed to list environments: %w", err)
					}

					activeEnv, _ := db.GetActiveEnv()

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
				},
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
					if err != nil || env == nil {
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
					if err != nil || env == nil {
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
				Action: func(ctx context.Context, c *cli.Command) error {
					args := c.Args()
					if args.Len() < 1 {
						return fmt.Errorf("environment name argument is required")
					}
					name := args.Get(0)

					env, err := db.GetEnvByName(name)
					if err != nil || env == nil {
						return fmt.Errorf("environment '%s' not found", name)
					}

					fmt.Printf("Environment: %s\n", env.Name)
					fmt.Printf("ID: %s\n", env.ID)
					if len(env.Variables) == 0 {
						fmt.Println("Variables: (none)")
					} else {
						fmt.Println("Variables:")
						for k, v := range env.Variables {
							if isSecretVariable(k) {
								v = "********"
							}
							fmt.Printf("  %s = %s\n", k, v)
						}
					}
					return nil
				},
			},
			{
				Name:  "set",
				Usage: "Set a variable in an environment",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name:     "var",
						Aliases:  []string{"v"},
						Usage:    "Variable in KEY=VALUE format (can repeat)",
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
					if err != nil || env == nil {
						return fmt.Errorf("environment '%s' not found", name)
					}

					for _, v := range c.StringSlice("var") {
						parts := strings.SplitN(v, "=", 2)
						if len(parts) == 2 {
							env.SetVariable(parts[0], parts[1])
						}
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
					if err != nil || env == nil {
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
					if filePath == "" {
						return fmt.Errorf("--file flag is required")
					}

					vars, err := env.ParseDotenvFile(filePath)
					if err != nil {
						return fmt.Errorf("failed to parse .env file: %w", err)
					}

					envObj, err := db.GetEnvByName(name)
					if err != nil {
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

func isSecretVariable(key string) bool {
	lowerKey := strings.ToLower(key)
	secretIndicators := []string{"key", "secret", "password", "token", "api_key", "apikey", "auth"}
	for _, indicator := range secretIndicators {
		if strings.Contains(lowerKey, indicator) {
			return true
		}
	}
	return false
}
