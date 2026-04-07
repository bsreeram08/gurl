package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sreeram/gurl/internal/cli/commands"
	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/internal/protocols/graphql"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/urfave/cli/v3"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Initialize database
	db, err := storage.NewLMDB()
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	defer db.Close() // Will run even on os.Exit

	if err := db.Open(); err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

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
			commands.RunCommand(db, env.NewEnvStorage(db)),
			commands.ListCommand(db),
			commands.DeleteCommand(db),
			commands.RenameCommand(db),
			commands.HistoryCommand(db),
			commands.TimelineCommand(db),
			commands.DiffCommand(db),
			commands.DetectCommand(db),
			commands.EditCommand(db),
			commands.ExportCommand(db),
			commands.ImportCommand(db),
			commands.PasteCommand(db),
			commands.CollectionCommand(db),
			commands.UpdateCommand(),
			graphql.GraphQLCommand(db),
		},
	}

	// Set up signal handling for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return app.Run(ctx, os.Args)
}
