package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/sreeram/gurl/internal/cli/commands"
	"github.com/sreeram/gurl/internal/client"
	"github.com/sreeram/gurl/internal/cookies"
	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/internal/plugins"
	"github.com/sreeram/gurl/internal/plugins/builtins"
	"github.com/sreeram/gurl/internal/project"
	"github.com/sreeram/gurl/internal/protocols/graphql"
	"github.com/sreeram/gurl/internal/protocols/grpc"
	"github.com/sreeram/gurl/internal/protocols/sse"
	"github.com/sreeram/gurl/internal/protocols/websocket"
	"github.com/sreeram/gurl/internal/runner"
	"github.com/sreeram/gurl/internal/storage"
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
	if dir := os.Getenv("GURL_PLUGIN_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".gurl", "plugins")
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

// newApp builds the gurl command tree.
func newApp(db storage.DB, envStorage env.Store, cookieProvider commands.CookieJarProvider, runnerOpts ...runner.Option) *cli.Command {
	return &cli.Command{
		Name:                  "gurl",
		Usage:                 "Smart curl saver - Your named request library",
		Version:               version,
		EnableShellCompletion: true,
		Flags: append([]cli.Flag{
			&cli.StringFlag{
				Name:  "project-dir",
				Usage: "Project root containing .gurl file storage",
			},
		}, commands.GlobalFlags()...),
		Description: `gurl replaces your chaotic curl history with an intelligent, 
named request library. Save requests with memorable names 
and run them whenever you need.

Quick Start:
  gurl save "health check" https://api.example.com/health
  gurl list
  gurl run "health check"
  gurl delete "old request"`,
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
			commands.DocsCommand(),
		},
	}
}

// agentBinaryName is the program name that turns on agent mode, so
// `ln -s "$(command -v gurl)" ~/.local/bin/agentgurl` gives agents a gurl
// that always runs with --agent.
const agentBinaryName = "agentgurl"

// invokedAsAgent reports whether gurl was started as agentgurl (or
// agentgurl.exe).
func invokedAsAgent(arg0 string) bool {
	name := filepath.Base(arg0)
	if ext := filepath.Ext(name); strings.EqualFold(ext, ".exe") {
		name = strings.TrimSuffix(name, ext)
	}
	return strings.EqualFold(name, agentBinaryName)
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
