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
	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/internal/plugins"
	"github.com/sreeram/gurl/internal/project"
	"github.com/sreeram/gurl/internal/protocols/graphql"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/urfave/cli/v3"
)

var version = "dev"

// Global plugin registry - initialized at startup
var pluginRegistry *plugins.Registry

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

	// Initialize plugin registry
	pluginDir := getPluginDir()
	loader := plugins.NewLoader(pluginDir, nil)
	pluginRegistry, _ = loader.LoadAll()
	envStorage := env.NewEnvStorageWithPath(baseDB.Path())
	if proj != nil {
		envStorage = env.NewEnvStorageWithPathAndProject(baseDB.Path(), proj)
	}

	app := &cli.Command{
		Name:                  "gurl",
		Usage:                 "Smart curl saver - Your named request library",
		Version:               version,
		EnableShellCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "project-dir",
				Usage: "Project root containing .gurl file storage",
			},
		},
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
			commands.RunCommand(db, envStorage),
			commands.ListCommand(db),
			commands.DeleteCommand(db),
			commands.RenameCommand(db),
			commands.HistoryCommand(db),
			commands.TimelineCommand(db),
			commands.DiffCommand(db),
			commands.DetectCommand(db),
			commands.EditCommand(db),
			commands.ShowCommand(db),
			commands.ExportCommand(db),
			commands.ImportCommand(db),
			commands.AuthCommand(),
			commands.EnvCommand(envStorage),
			commands.PasteCommand(db),
			commands.CollectionCommand(db, envStorage),
			commands.SequenceCommand(db),
			commands.UpdateCommand(),
			commands.ShellCommand(db, envStorage),
			commands.CodegenCommand(db),
			graphql.GraphQLCommand(db),
		},
	}

	// Set up signal handling for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return app.Run(ctx, os.Args)
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
