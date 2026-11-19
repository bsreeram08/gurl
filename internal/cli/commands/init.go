package commands

import (
	"context"
	"fmt"

	"github.com/bsreeram08/apigent/internal/project"
	"github.com/urfave/cli/v3"
)

func InitCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Initialize a apigent project",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "project-dir",
				Usage: "Project root for .apigent file storage",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			proj, err := project.Init(projectDirFlag(c))
			if err != nil {
				return err
			}
			fmt.Printf("✓ Initialized apigent project at %s\n", proj.Dir)
			return nil
		},
	}
}

func projectDirFlag(c *cli.Command) string {
	if c == nil {
		return ""
	}
	if value := c.String("project-dir"); value != "" {
		return value
	}
	if root := c.Root(); root != nil && root != c {
		return root.String("project-dir")
	}
	return ""
}
