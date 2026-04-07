package commands

import (
	"context"
	"fmt"

	"github.com/sreeram/gurl/internal/cookies"
	"github.com/urfave/cli/v3"
)

type CookieJarProvider interface {
	GetCookieJar() *cookies.CookieJar
}

func CookiesCommand(provider CookieJarProvider) *cli.Command {
	return &cli.Command{
		Name:    "cookies",
		Usage:   "Manage cookies",
		Aliases: []string{"cookie", "ck"},
		Commands: []*cli.Command{
			{
				Name:    "list",
				Aliases: []string{"ls", "l"},
				Usage:   "List all stored cookies",
				Action: func(ctx context.Context, c *cli.Command) error {
					jar := provider.GetCookieJar()
					if jar == nil {
						fmt.Println("Cookie jar not initialized.")
						return nil
					}

					entries, err := jar.List()
					if err != nil {
						return fmt.Errorf("failed to list cookies: %w", err)
					}

					if len(entries) == 0 {
						fmt.Println("No cookies stored.")
						return nil
					}

					fmt.Println("┌─ Cookies ──────────────────────────────────────────────────┐")
					fmt.Println("│  DOMAIN            NAME          VALUE                  │")
					fmt.Println("├─────────────────────────────────────────────────────────┤")

					for _, e := range entries {
						domain := e.Domain
						if len(domain) > 17 {
							domain = domain[:14] + "..."
						}
						name := e.Name
						if len(name) > 13 {
							name = name[:10] + "..."
						}
						value := e.Value
						if len(value) > 20 {
							value = value[:17] + "..."
						}
						fmt.Printf("│  %-17s %-13s %-20s\n", domain, name, value)
					}

					fmt.Println("└─────────────────────────────────────────────────────────┘")
					fmt.Printf("  %d cookie(s)\n", len(entries))
					return nil
				},
			},
			{
				Name:    "clear",
				Aliases: []string{"cl", "clean"},
				Usage:   "Clear all stored cookies",
				Action: func(ctx context.Context, c *cli.Command) error {
					jar := provider.GetCookieJar()
					if jar == nil {
						fmt.Println("Cookie jar not initialized.")
						return nil
					}

					if err := jar.ClearAll(); err != nil {
						return fmt.Errorf("failed to clear cookies: %w", err)
					}

					fmt.Println("✓ All cookies cleared")
					return nil
				},
			},
			{
				Name:    "delete",
				Aliases: []string{"rm", "del"},
				Usage:   "Delete a specific cookie",
				Action: func(ctx context.Context, c *cli.Command) error {
					args := c.Args()
					if args.Len() < 2 {
						return fmt.Errorf("usage: cookies delete <domain> <name>")
					}
					domain := args.Get(0)
					name := args.Get(1)

					jar := provider.GetCookieJar()
					if jar == nil {
						fmt.Println("Cookie jar not initialized.")
						return nil
					}

					if err := jar.DeleteCookie(domain, name); err != nil {
						return fmt.Errorf("failed to delete cookie: %w", err)
					}

					fmt.Printf("✓ Deleted cookie '%s' for domain '%s'\n", name, domain)
					return nil
				},
			},
		},
	}
}
