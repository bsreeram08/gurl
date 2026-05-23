package commands

import (
	"context"
	"fmt"
	"sort"
	"strings"

	authpkg "github.com/sreeram/gurl/internal/auth"
	"github.com/urfave/cli/v3"
)

// AuthCommand creates the auth discovery command.
func AuthCommand() *cli.Command {
	return &cli.Command{
		Name:  "auth",
		Usage: "Discover supported authentication types",
		Commands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List built-in authentication types",
				Action: func(ctx context.Context, c *cli.Command) error {
					registry := authpkg.BuiltinRegistry()
					fmt.Println("Built-in auth types:")
					for _, name := range builtinAuthTypeNames() {
						handler := registry.Get(name)
						fmt.Printf("  %-10s  %s\n", name, handler.Description())
					}
					return nil
				},
			},
			{
				Name:      "info",
				Usage:     "Show parameters for an authentication type",
				ArgsUsage: "<type>",
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.Args().Len() != 1 {
						return fmt.Errorf("auth type argument is required")
					}
					authType := strings.ToLower(strings.TrimSpace(c.Args().Get(0)))
					handler := authpkg.BuiltinRegistry().Get(authType)
					if handler == nil {
						return fmt.Errorf("unknown auth type %q", authType)
					}

					fmt.Printf("Auth type: %s\n", handler.Name())
					fmt.Println("Parameters:")
					for _, param := range handler.Params() {
						markers := []string{"optional"}
						if param.Required {
							markers[0] = "required"
						}
						if param.Secret {
							markers = append(markers, "secret")
						}
						if param.Default != "" {
							markers = append(markers, "default="+param.Default)
						}
						fmt.Printf("  %s\t%s\t%s\n", param.Name, strings.Join(markers, ","), param.Description)
					}
					return nil
				},
			},
		},
	}
}

func builtinAuthTypeNames() []string {
	registry := authpkg.BuiltinRegistry()
	names := []string{"basic", "bearer", "apikey", "oauth1", "oauth2", "awsv4", "digest", "ntlm"}
	filtered := names[:0]
	for _, name := range names {
		if registry.Get(name) != nil {
			filtered = append(filtered, name)
		}
	}
	sort.Strings(filtered)
	return filtered
}
