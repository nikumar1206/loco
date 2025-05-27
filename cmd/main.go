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
						// Error returned here will be handled by urfave/cli or the main app.Run() handler
						return fmt.Errorf("failed to create loco.toml: %w", err)
					}

					locoOut(LOCO__OK_PREFIX, "Loco project initialized") // Already correct
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
					locoOut(LOCO__OK_PREFIX, "Building Docker Image...") // Already correct
					dockerCli, err := createDockerClient() // Renamed cli to dockerCli for clarity
					if err != nil {
						return fmt.Errorf("failed to create Docker client: %w", err)
					}
					defer dockerCli.Close()

					tokenResponse, err := getDeployToken()
					if err != nil {
						return fmt.Errorf("failed to get deploy token: %w", err)
					}
					// Changed fmt.Println to locoOut with Sprintf
					locoOut(LOCO__OK_PREFIX, fmt.Sprintf("Get token response: %+v", tokenResponse))

					if err := buildDockerImage(context.Background(), dockerCli, tokenResponse.Image); err != nil {
						return fmt.Errorf("failed to build Docker image: %w", err)
					}

					err = dockerPush(dockerCli, tokenResponse.Username, tokenResponse.Password, "registry.gitlab.com", tokenResponse.Image)
					if err != nil {
						return fmt.Errorf("failed to push Docker image: %w", err)
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
					locoOut(LOCO__OK_PREFIX, "logs command called")
					locoOut(LOCO__OK_PREFIX, fmt.Sprintf("file: %s", cmd.String("file")))
					locoOut(LOCO__OK_PREFIX, fmt.Sprintf("follow: %t", cmd.Bool("follow")))
					locoOut(LOCO__OK_PREFIX, fmt.Sprintf("lines: %d", cmd.Int("lines")))
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
					locoOut(LOCO__OK_PREFIX, "status command called")
					locoOut(LOCO__OK_PREFIX, fmt.Sprintf("file: %s", cmd.String("file")))
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
					locoOut(LOCO__OK_PREFIX, "destroy command called")
					locoOut(LOCO__OK_PREFIX, fmt.Sprintf("file: %s", cmd.String("file")))
					locoOut(LOCO__OK_PREFIX, fmt.Sprintf("yes: %t", cmd.Bool("yes")))
					return nil
				},
			},
		},
	}

	err := app.Run(context.Background(), os.Args)
	if err != nil {
		// Changed main error printing to use locoErr
		locoErr(LOCO__ERROR_PREFIX, fmt.Sprintf("Error running loco: %v", err))
		os.Exit(1)
	}
}
