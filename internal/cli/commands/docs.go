package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/urfave/cli/v3"
)

// DocsCommand prints one compact reference of gurl: every command and flag,
// generated from the command tree at runtime, plus the static sections in
// docSections. `gurl docs <command>...` narrows it to one subtree.
func DocsCommand() *cli.Command {
	return &cli.Command{
		Name:      "docs",
		Usage:     "Print a compact reference of every command, flag and concept (start here, agents)",
		ArgsUsage: "[command [subcommand]]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "json",
				Aliases: []string{"j"},
				Usage:   "JSON output",
			},
		},
		Action: jsonAction(func(ctx context.Context, c *cli.Command) error {
			root := c.Root()
			target, path, err := findCommand(root, c.Args().Slice())
			if err != nil {
				return err
			}
			ref := docsReference{Commands: docCommands(target, path)}
			// The static sections describe gurl as a whole; a narrowed
			// reference keeps only the commands.
			if target == root {
				ref.Sections = make(map[string]string, len(docSections))
				for _, s := range docSections {
					ref.Sections[s.Key] = s.Body
				}
			}
			if jsonOutput(c) {
				return writeJSONLine(os.Stdout, ref)
			}
			return writeDocsText(os.Stdout, ref, target == root)
		}),
	}
}

type docsReference struct {
	Commands []docCommand      `json:"commands"`
	Sections map[string]string `json:"sections,omitempty"`
}

type docCommand struct {
	// Path is the full invocation, e.g. "gurl collection run".
	Path    string    `json:"path"`
	Aliases []string  `json:"aliases,omitempty"`
	Usage   string    `json:"usage,omitempty"`
	Args    string    `json:"args,omitempty"`
	Flags   []docFlag `json:"flags,omitempty"`
}

type docFlag struct {
	Name    string   `json:"name"`
	Aliases []string `json:"aliases,omitempty"`
	// Type is the value type; empty (omitted) for boolean switches.
	Type    string `json:"type,omitempty"`
	Default string `json:"default,omitempty"`
	Usage   string `json:"usage,omitempty"`
}

// findCommand walks names (or aliases) from root.
func findCommand(root *cli.Command, names []string) (*cli.Command, string, error) {
	cmd, path := root, root.Name
	for _, name := range names {
		sub := cmd.Command(name)
		if sub == nil || sub.Hidden {
			return nil, "", fmt.Errorf("unknown command %q: run gurl docs for the list", strings.TrimSpace(path+" "+name))
		}
		cmd, path = sub, path+" "+sub.Name
	}
	return cmd, path, nil
}

// docCommands lists cmd and its visible subcommands, depth first.
func docCommands(cmd *cli.Command, path string) []docCommand {
	out := []docCommand{{
		Path:    path,
		Aliases: cmd.Aliases,
		Usage:   cmd.Usage,
		Args:    cmd.ArgsUsage,
		Flags:   docFlags(cmd.Flags),
	}}
	for _, sub := range cmd.Commands {
		if sub.Hidden || sub.Name == "help" {
			continue
		}
		out = append(out, docCommands(sub, path+" "+sub.Name)...)
	}
	return out
}

func docFlags(flags []cli.Flag) []docFlag {
	var out []docFlag
	for _, f := range flags {
		names := f.Names()
		if len(names) == 0 || names[0] == "help" {
			continue
		}
		if v, ok := f.(cli.VisibleFlag); ok && !v.IsVisible() {
			continue
		}
		flag := docFlag{Name: names[0], Aliases: names[1:]}
		if d, ok := f.(cli.DocGenerationFlag); ok {
			flag.Usage = d.GetUsage()
			if d.TakesValue() {
				flag.Type = d.TypeName()
				if m, ok := f.(cli.DocGenerationMultiValueFlag); ok && m.IsMultiValueFlag() {
					flag.Type = "[]" + flag.Type
				}
				// Only this command's flags are parsed, so the value of any
				// other flag is its default.
				if def := d.GetValue(); def != "" && def != "0" && def != `""` && def != "[]" {
					flag.Default = strings.Trim(def, `"`)
				}
			}
		}
		out = append(out, flag)
	}
	return out
}

// writeDocsText prints ref as compact plain text: one line per command,
// one per flag, then the sections.
func writeDocsText(w io.Writer, ref docsReference, withSections bool) error {
	var b strings.Builder
	for _, cmd := range ref.Commands {
		b.WriteString(cmd.Path)
		if cmd.Args != "" {
			b.WriteString(" " + cmd.Args)
		}
		if cmd.Usage != "" {
			b.WriteString("  — " + cmd.Usage)
		}
		if len(cmd.Aliases) > 0 {
			b.WriteString(" (aliases: " + strings.Join(cmd.Aliases, ", ") + ")")
		}
		b.WriteString("\n")
		for _, f := range cmd.Flags {
			b.WriteString("  --" + f.Name)
			for _, a := range f.Aliases {
				if len(a) == 1 {
					b.WriteString(", -" + a)
				} else {
					b.WriteString(", --" + a)
				}
			}
			if f.Type != "" {
				b.WriteString(" <" + f.Type + ">")
			}
			if f.Usage != "" {
				b.WriteString("  " + f.Usage)
			}
			if f.Default != "" {
				b.WriteString(" (default " + f.Default + ")")
			}
			b.WriteString("\n")
		}
	}
	if withSections {
		for _, s := range docSections {
			fmt.Fprintf(&b, "\n## %s\n%s\n", s.Title, s.Body)
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}
