package main

import (
	"github.com/bsreeram08/valetkey/internal/brand"

	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bsreeram08/valetkey/internal/cli/commands"
	"github.com/bsreeram08/valetkey/internal/client"
	"github.com/bsreeram08/valetkey/internal/cookies"
	"github.com/bsreeram08/valetkey/internal/env"
	"github.com/bsreeram08/valetkey/internal/plugins"
	"github.com/bsreeram08/valetkey/internal/plugins/builtins"
	"github.com/bsreeram08/valetkey/internal/project"
	"github.com/bsreeram08/valetkey/internal/protocols/graphql"
	"github.com/bsreeram08/valetkey/internal/protocols/grpc"
	"github.com/bsreeram08/valetkey/internal/protocols/sse"
	"github.com/bsreeram08/valetkey/internal/protocols/websocket"
	"github.com/bsreeram08/valetkey/internal/runner"
	"github.com/bsreeram08/valetkey/internal/storage"
	"github.com/urfave/cli/v3"
)

var version = "dev"

// Global plugin registry - initialized at startup
var pluginRegistry *plugins.Registry

type cookieJarHolder struct {
	jar *cookies.CookieJar
}

func (h cookieJarHolder) GetCookieJar() *cookies.CookieJar {
	return h.jar
}

func getPluginDir() string {
	if dir := brand.Getenv("PLUGIN_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return brand.PluginDir(home)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Initialize database handles lazily so interactive commands don't hold a DB lock while idle.
	baseDB, err := storage.NewLazyDB()
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	proj, err := project.Discover("", projectDirFromArgs(os.Args))
	if err != nil {
		return err
	}
	var db storage.DB = baseDB
	if proj != nil {
		db = storage.NewProjectDB(baseDB, storage.NewFileStore(proj))
	}

	pluginDir := getPluginDir()
	loader := plugins.NewLoader(pluginDir, nil)
	pluginRegistry, err = loader.LoadAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: plugin discovery failed: %v\n", err)
		pluginRegistry = plugins.NewRegistry()
	}
	builtins.RegisterBuiltins(pluginRegistry)
	commands.SetPluginRegistry(pluginRegistry)

	cookieProvider := commands.CookieJarProvider(cookieJarHolder{})
	var runnerOpts []runner.Option
	if cookieDB, cookieErr := cookies.NewLMDBCookieDB(); cookieErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: cookie store unavailable: %v\n", cookieErr)
	} else if jar, jarErr := cookies.NewCookieJar(cookieDB); jarErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: cookie jar unavailable: %v\n", jarErr)
	} else {
		client.SetDefaultCookieJar(jar)
		cookieProvider = cookieJarHolder{jar: jar}
		runnerOpts = append(runnerOpts, runner.WithCookieJar(jar))
	}

	envStorage := env.NewEnvStorageWithPath(baseDB.Path())
	if proj != nil {
		envStorage = env.NewEnvStorageWithPathAndProject(baseDB.Path(), proj)
	}

	app := newApp(db, envStorage, cookieProvider, runnerOpts...)

	args := os.Args
	if len(args) > 0 && invokedAsAgent(args[0]) {
		args = append([]string{args[0], "--agent"}, args[1:]...)
	}
	agent := commands.AgentRequested(args)
	app.ExitErrHandler = commands.ExitErrHandler(agent)
	if agent {
		commands.QuietUsageErrors(app)
	}

	// Set up signal handling for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return app.Run(ctx, args)
}

// newApp builds the valetkey command tree.
func newApp(db storage.DB, envStorage env.Store, cookieProvider commands.CookieJarProvider, runnerOpts ...runner.Option) *cli.Command {
	return &cli.Command{
		Name:                  brand.Name,
		Usage:                 "Saved API requests your AI agents can run without ever seeing the keys",
		Version:               version,
		EnableShellCompletion: true,
		Flags: append([]cli.Flag{
			&cli.StringFlag{
				Name:  "project-dir",
				Usage: "Project root containing a .valetkey (or legacy .gurl) directory",
			},
		}, commands.GlobalFlags()...),
		Description: `valetkey is a library of named HTTP requests for people and AI agents.
You save requests and keep secrets in environments; agents run them by
name and get JSON back, but never see the keys.

Quick Start:
  valetkey save "health check" https://api.example.com/health
  valetkey run "health check" --env dev
  valetkey --agent run "health check" --env dev   # what an agent runs
  valetkey mcp                                    # serve saved requests to MCP clients

The old command name gurl, and agentgurl, still work.`,
		Commands: []*cli.Command{
			commands.InitCommand(),
			commands.SaveCommand(db),
			commands.RunCommand(db, envStorage, runnerOpts...),
			commands.ListCommand(db),
			commands.DeleteCommand(db),
			commands.RenameCommand(db),
			commands.HistoryCommand(db),
			commands.TimelineCommand(db),
			commands.DiffCommand(db, envStorage),
			commands.DiagCommand(db, envStorage),
			commands.DetectCommand(db),
			commands.EditCommand(db),
			commands.ShowCommand(db),
			commands.ExportCommand(db),
			commands.ImportCommand(db),
			commands.ShareCommand(db),
			commands.ReceiveCommand(db),
			commands.AuthCommand(),
			commands.EnvCommand(envStorage),
			commands.ExecCommand(envStorage),
			commands.ExtractCommand(),
			commands.PasteCommand(db),
			commands.CollectionCommand(db, envStorage, runnerOpts...),
			commands.SequenceCommand(db),
			commands.UpdateCommand(),
			commands.ShellCommand(db, envStorage),
			commands.CodegenCommand(db),
			graphql.GraphQLCommand(db),
			grpc.GRPCCommand(db),
			websocket.WSCommand(db),
			sse.SSECommand(db),
			commands.CookiesCommand(cookieProvider),
			commands.MigrateCommand(),
			commands.DocsCommand(),
		},
	}
}

// invokedAsAgent reports whether the binary was started as agentvaletkey
// or the pre-rename agentgurl (with or without .exe), so
// `ln -s "$(command -v valetkey)" ~/.local/bin/agentvaletkey` gives agents a
// binary that always runs with --agent.
func invokedAsAgent(arg0 string) bool {
	return brand.IsAgentBinary(arg0)
}

func projectDirFromArgs(args []string) string {
	for i, arg := range args {
		if arg == "--project-dir" && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(arg, "--project-dir=") {
			return strings.TrimPrefix(arg, "--project-dir=")
		}
	}
	return ""
}
