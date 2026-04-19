package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/sreeram/gurl/internal/cli/commands"
	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/internal/plugins"
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
	db, err := storage.NewLazyDB()
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	// Initialize plugin registry
	pluginDir := getPluginDir()
	loader := plugins.NewLoader(pluginDir, nil)
	pluginRegistry, _ = loader.LoadAll()
	envStorage := env.NewEnvStorageWithPath(db.Path())

	app := &cli.Command{
		Name:    "gurl",
		Usage:   "Smart curl saver - Your named request library",
		Version: version,
		Description: `gurl replaces your chaotic curl history with an intelligent, 
named request library. Save requests with memorable names 
and run them whenever you need.

Quick Start:
  gurl save "health check" https://api.example.com/health
  gurl list
  gurl run "health check"
  gurl delete "old request"`,
		Commands: []*cli.Command{
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
