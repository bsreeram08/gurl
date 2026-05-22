package commands

import (
	"testing"

	"github.com/sreeram/gurl/internal/env"
	"github.com/urfave/cli/v3"
)

func TestFlowControlFlagsAppearInCommandHelp(t *testing.T) {
	db := newMockDB()
	envStorage := &env.EnvStorage{}

	tests := []struct {
		name  string
		cmd   *cli.Command
		flags []string
	}{
		{
			name:  "save",
			cmd:   SaveCommand(db),
			flags: []string{"extract", "pre-script", "post-script", "auth", "auth-param"},
		},
		{
			name:  "edit",
			cmd:   EditCommand(db),
			flags: []string{"extract", "remove-extract", "pre-script", "post-script", "run-if", "auth", "auth-param"},
		},
		{
			name:  "run",
			cmd:   RunCommand(db, envStorage),
			flags: []string{"persist"},
		},
		{
			name:  "collection run",
			cmd:   collectionRunSubcommand(t, CollectionCommand(db, envStorage)),
			flags: []string{"persist", "dry-run", "assert-bail"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, flagName := range tt.flags {
				count, usage := commandHelpFlagCountAndUsage(tt.cmd, flagName)
				if count != 1 {
					t.Fatalf("expected --%s to appear exactly once in %s help, got %d", flagName, tt.name, count)
				}
				if usage == "" {
					t.Fatalf("expected --%s in %s help to have a usage description", flagName, tt.name)
				}
			}
		})
	}
}

func collectionRunSubcommand(t *testing.T, cmd *cli.Command) *cli.Command {
	t.Helper()
	for _, subcommand := range cmd.Commands {
		if subcommand.Name == "run" {
			return subcommand
		}
	}
	t.Fatal("expected collection run subcommand")
	return nil
}

func commandHelpFlagCountAndUsage(cmd *cli.Command, name string) (int, string) {
	count := 0
	usage := ""
	for _, flag := range cmd.Flags {
		named, ok := flag.(interface{ Names() []string })
		if !ok {
			continue
		}
		for _, flagName := range named.Names() {
			if flagName == name {
				count++
				usage = flagUsage(flag)
			}
		}
	}
	return count, usage
}

func flagUsage(flag cli.Flag) string {
	switch f := flag.(type) {
	case *cli.BoolFlag:
		return f.Usage
	case *cli.DurationFlag:
		return f.Usage
	case *cli.IntFlag:
		return f.Usage
	case *cli.StringFlag:
		return f.Usage
	case *cli.StringSliceFlag:
		return f.Usage
	default:
		return ""
	}
}
