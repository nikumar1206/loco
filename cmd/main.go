package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nikumar1206/loco/internal/color"
	"github.com/urfave/cli/v3"
)

var (
	LOCO__OK_PREFIX    = color.Colorize("LOCO: ", color.FgGreen)
	LOCO__ERROR_PREFIX = color.Colorize("LOCO: ", color.FgRed)
)

func main() {
	app := &cli.Command{
		Name:    "loco",
		Usage:   "A CLI for managing application deployments.",
		Version: "v0.0.1",
		Commands: []*cli.Command{
			{
				Name:  "init",
				Usage: "Initialize a new Loco project",
				Action: func(c context.Context, cmd *cli.Command) error {
					// create a loco.toml file if not already exists
					err := createConfig()
					if err != nil {
						return fmt.Errorf("failed to create loco.toml: %w", err)
					}

					fmt.Println("Loco project initialized")
					return nil
				},
			},
			{
				Name:  "deploy",
				Usage: "Deploy an application",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "file",
						Aliases: []string{"f"},
						Usage:   "Load configuration from `FILE`",
					},
					&cli.BoolFlag{
						Name:    "yes",
						Aliases: []string{"y"},
						Usage:   "Assume yes to all prompts",
					},
				},
				Action: func(c context.Context, cmd *cli.Command) error {
					locoOut(LOCO__OK_PREFIX, "Building Docker Image...")
					cli, err := createDockerClient()
					if err != nil {
						return fmt.Errorf("failed to create Docker client: %w", err)
					}
					defer cli.Close()

					if err := buildDockerImage(context.Background(), cli); err != nil {
						return fmt.Errorf("failed to build Docker image: %w", err)
					}

					return nil
				},
			},
			{
				Name:  "logs",
				Usage: "View application logs",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "file",
						Aliases: []string{"f"},
						Usage:   "Load configuration from `FILE`",
					},
					&cli.BoolFlag{
						Name:    "follow",
						Aliases: []string{"F"},
						Usage:   "Follow log output",
					},
					&cli.IntFlag{
						Name:    "lines",
						Aliases: []string{"n"},
						Usage:   "Number of lines to show",
					},
				},
				Action: func(c context.Context, cmd *cli.Command) error {
					fmt.Println("logs command called")
					fmt.Printf("file: %s\n", "file")
					fmt.Printf("follow: %t\n", "follow")
					fmt.Printf("lines: %d\n", "lines")
					return nil
				},
			},
			{
				Name:  "status",
				Usage: "Show application status",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "file",
						Aliases: []string{"f"},
						Usage:   "Load configuration from `FILE`",
					},
				},
				Action: func(c context.Context, cmd *cli.Command) error {
					fmt.Println("status command called")
					fmt.Printf("file: %s\n", "file")
					return nil
				},
			},
			{
				Name:  "destroy",
				Usage: "Destroy an application deployment",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "file",
						Aliases: []string{"f"},
						Usage:   "Load configuration from `FILE`",
					},
					&cli.BoolFlag{
						Name:    "yes",
						Aliases: []string{"y"},
						Usage:   "Assume yes to all prompts",
					},
				},
				Action: func(c context.Context, cmd *cli.Command) error {
					fmt.Println("destroy command called")
					fmt.Printf("file: %s\n", "file")
					fmt.Print("yes: %t\n", "yes")
					return nil
				},
			},
		},
	}

	err := app.Run(context.Background(), os.Args)
	if err != nil {
		fmt.Printf("Error running loco\n%v\n", err)
		os.Exit(1)
	}
}
