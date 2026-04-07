package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v3"
	"github.com/sreeram/gurl/internal/cli/commands"
	"github.com/sreeram/gurl/internal/storage"
)

func main() {
	// Initialize database
	db, err := storage.NewLMDB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create database: %v\n", err)
		os.Exit(1)
	}

	if err := db.Open(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	app := &cli.Command{
		Name:    "gurl",
		Usage:   "Smart curl saver - Your named request library",
		Version: "0.1.0",
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
			commands.RunCommand(db),
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
		},
	}

	// Set up signal handling for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
