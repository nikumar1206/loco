package cmd

import (
	"context"
	"fmt"
	"os/user"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/nikumar1206/loco/internal/api"
	"github.com/nikumar1206/loco/internal/config"
	"github.com/nikumar1206/loco/internal/docker"
	"github.com/nikumar1206/loco/internal/keychain"
	"github.com/nikumar1206/loco/internal/ui"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy a new application to Loco.",
	Long:  "Deploy a new application to Loco.\nThis builds and pushes a Docker image to the Loco registry and deploys it onto the Loco platform under the specified subdomain.",

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

	isDev, err := cmd.Flags().GetBool("dev")
	if err != nil {
		return fmt.Errorf("error reading dev flag: %w", err)
	}

	var host string
	if isDev {
		host = "http://localhost:8000"
	} else {
		host = "https://loco.deploy-app.com"
	}

	apiClient := api.NewClient(host)

	usr, err := user.Current()
	if err != nil {
		return err
	}

	locoToken, err := keychain.GetGithubToken(usr.Name)
	if err != nil {
		return err
	}

	if locoToken.ExpiresAt.Before(time.Now().Add(5 * time.Minute)) {
		return fmt.Errorf("token is expired or about to expire soon. Please re-login via `loco deploy`")
	}

	cfgValid := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00CC66")).
		Render("\n🎉 Validated loco.toml. Beginning deployment!") + "\n"

	fmt.Print(cfgValid)

	steps := []ui.Step{
		{
			Title: "Fetch deploy token",
			Run: func(logf func(string)) error {
				tokenResponse, err = apiClient.GetDeployToken(locoToken.Token)
				dockerCli.GenerateImageTag(tokenResponse.Image)
				if err != nil {
					return err
				}
				return nil
			},
		},
		{
			Title: "Build Docker image",
			Run: func(logf func(string)) error {
				if err := dockerCli.BuildImage(context.Background(), logf); err != nil {
					return err
				}
				return nil
			},
		},
		{
			Title: "Push image to registry",
			Run: func(logf func(string)) error {
				if err := dockerCli.PushImage(context.Background(), logf, tokenResponse.Username, tokenResponse.Password); err != nil {
					return fmt.Errorf("failed to push Docker image: %w", err)
				}
				return nil
			},
		},
		{
			Title: "Create Kubernetes deployment",
			Run: func(logf func(string)) error {
				return apiClient.DeployApp(cfg, dockerCli.ImageName, locoToken.Token, logf)
			},
		},
	}
	if err := ui.RunSteps(steps); err != nil {
		return err
	}

	s := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00CC66")).
		Render("\n🎉 Deployment complete!") + "\n"

	fmt.Print(s)
	return nil
}
