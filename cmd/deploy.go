package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/nikumar1206/loco/cmd/pkg/api"
	"github.com/nikumar1206/loco/cmd/pkg/config"
	"github.com/nikumar1206/loco/cmd/pkg/docker"
	"github.com/nikumar1206/loco/cmd/pkg/progress"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy a new application to Loco.\nThis builds and pushes a Docker image to the Loco registry and deploys it onto the Loco platform under the specified subdomain.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return deployCmdFunc(cmd, args)
	},
}

func init() {
	deployCmd.Flags().StringP("config", "c", "", "path to loco.toml config file")
	deployCmd.Flags().BoolP("yes", "y", false, "Assume yes to all prompts")
}

func deployCmdFunc(cmd *cobra.Command, args []string) error {
	var err error
	var tokenResponse api.DeployTokenResponse

	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return fmt.Errorf("failed to read config flag: %w", err)
	}
	if configPath == "" {
		configPath = "loco.toml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := config.ErrorOnBadConfig(cfg); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	cfg = config.FillSensibleDefaults(cfg)

	dockerCli, err := docker.NewDockerClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer dockerCli.Close()

	cfgValid := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00CC66")).
		Render("\n🎉 Validated loco.toml. Beginning deployment!") + "\n"
	fmt.Print(cfgValid)

	steps := []progress.Step{
		{
			Title: "Fetching deploy token",
			Run: func(logf func(string)) error {
				// TODO: remove this sleep
				time.Sleep(1 * time.Second)
				tokenResponse, err = api.GetDeployToken()
				dockerCli.GenerateImageTag(tokenResponse.Image)
				if err != nil {
					return fmt.Errorf("failed to get deploy token: %w", err)
				}
				return nil
			},
		},
		{
			Title: "Building Docker image",
			Run: func(logf func(string)) error {
				// todo: have this function output logs to the progress bar
				if err := dockerCli.BuildImage(context.Background(), logf); err != nil {
					return fmt.Errorf("failed to build Docker image: %w", err)
				}
				return nil
			},
		},
		{
			Title: "Pushing image to registry",
			Run: func(logf func(string)) error {
				if err := dockerCli.PushImage(context.Background(), logf, tokenResponse.Username, tokenResponse.Password); err != nil {
					return fmt.Errorf("failed to push Docker image: %w", err)
				}
				return nil
			},
		},
		// {
		// 	Title: "Creating Kubernetes deployment",
		// 	Run: func() error {
		// 		err = docker.PushImage(dockerCli, tokenResponse.Username, tokenResponse.Password, "registry.gitlab.com", tokenResponse.Image)
		// 		if err != nil {
		// 			return fmt.Errorf("failed to push Docker image: %w", err)
		// 		}
		// 		return nil
		// 	},
		// },
	}
	if err := progress.RunSteps(steps); err != nil {
		return err
	}

	s := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00CC66")).
		Render("\n🎉 Deployment complete!") + "\n"

	fmt.Print(s)
	return nil
}
