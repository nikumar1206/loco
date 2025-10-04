package loco

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os/user"
	"time"

	"connectrpc.com/connect"
	"github.com/charmbracelet/lipgloss"
	"github.com/nikumar1206/loco/internal/client"
	"github.com/nikumar1206/loco/internal/config"
	"github.com/nikumar1206/loco/internal/docker"
	"github.com/nikumar1206/loco/internal/keychain"
	"github.com/nikumar1206/loco/internal/ui"
	registryv1 "github.com/nikumar1206/loco/proto/registry/v1"
	registryv1connect "github.com/nikumar1206/loco/proto/registry/v1/registryv1connect"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy a new application to Loco.",
	Long:  "Deploy a new application to Loco.\nThis builds and pushes a Docker image to the Loco registry and deploys it onto the Loco platform under the specified subdomain. Note, loco will attempt to autodiscover all loco.toml and deploy them. To deploy a subset, use the -c flag.",

	RunE: func(cmd *cobra.Command, args []string) error {
		return deployCmdFunc(cmd, args)
	},
}

func init() {
	deployCmd.Flags().StringP("config", "c", "", "path to loco.toml config file")
	deployCmd.Flags().BoolP("yes", "y", false, "Assume yes to all prompts")
}

func deployCmdFunc(cmd *cobra.Command, _ []string) error {
	parseAndSetDebugFlag(cmd)
	host := parseDevFlag(cmd)

	var err error
	var tokenResponse *connect.Response[registryv1.GitlabTokenResponse]

	usr, err := user.Current()
	if err != nil {
		slog.Debug("failed to get current user", "error", err)
		return err
	}
	locoToken, err := keychain.GetGithubToken(usr.Name)
	if err != nil {
		slog.Debug("failed to get github token", "error", err)
		return err
	}

	if locoToken.ExpiresAt.Before(time.Now().Add(5 * time.Minute)) {
		slog.Debug("token is expired or will expire soon", "expires_at", locoToken.ExpiresAt)
		return fmt.Errorf("token is expired or will expire soon. Please re-login via `loco login`")
	}
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return fmt.Errorf("failed to read config flag: %w", err)
	}
	if configPath == "" {
		configPath = "loco.toml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Debug("failed to load config", "path", configPath, "error", err)
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := config.Validate(cfg.LocoConfig); err != nil {
		slog.Debug("invalid configuration", "error", err)
		return fmt.Errorf("invalid configuration: %w", err)
	}
	config.FillSensibleDefaults(cfg.LocoConfig)

	cfgValid := lipgloss.NewStyle().
		Foreground(ui.LocoLightGreen).
		Render("\n🎉 Validated loco.toml. Beginning deployment!")
	fmt.Print(cfgValid)

	dockerCli, err := docker.NewDockerClient(cfg)
	if err != nil {
		slog.Debug("failed to create docker client", "error", err)
		return err
	}
	defer dockerCli.Close()

	apiClient := client.NewClient(host)
	registryClient := registryv1connect.NewRegistryServiceClient(http.DefaultClient, host)

	steps := []ui.Step{
		{
			Title: "Fetch deploy token",
			Run: func(logf func(string)) error {
				req := connect.NewRequest(&registryv1.GitlabTokenRequest{})
				req.Header().Set("Authorization", fmt.Sprintf("Bearer %s", locoToken.Token))

				tokenResponse, err = registryClient.GitlabToken(context.Background(), req)
				if err != nil {
					slog.Debug("failed to fetch deploy token", "error", err)
					return err
				}
				dockerCli.GenerateImageTag(tokenResponse.Msg.Image)
				slog.Debug("generated image tag", "tag", dockerCli.ImageName)
				return nil
			},
		},
		{
			Title: "Build Docker image",
			Run: func(logf func(string)) error {
				if err := dockerCli.BuildImage(context.Background(), logf); err != nil {
					slog.Debug("failed to build docker image", "error", err)
					return err
				}
				return nil
			},
		},
		{
			Title: "Push image to registry",
			Run: func(logf func(string)) error {
				if err := dockerCli.PushImage(context.Background(), logf, tokenResponse.Msg.GetUsername(), tokenResponse.Msg.GetToken()); err != nil {
					slog.Debug("failed to push docker image", "error", err)
					return fmt.Errorf("failed to push Docker image: %w", err)
				}
				return nil
			},
		},
		{
			Title: "Create Kubernetes deployment",
			Run: func(logf func(string)) error {
				// todo: cleanup how we pass variables around, why should this be dockercli.image?
				// and why would this be generated client side?
				if err := apiClient.DeployApp(cfg, dockerCli.ImageName, locoToken.Token, logf); err != nil {
					slog.Debug("failed to create kubernetes deployment", "error", err)
					return err
				}
				return nil
			},
		},
	}
	if err := ui.RunSteps(steps); err != nil {
		return err
	}

	s := lipgloss.NewStyle().
		Bold(true).
		Foreground(ui.LocoLightGreen).
		Render("\n🎉 Deployment scheduled!")

	fmt.Println(s)

	s = lipgloss.NewStyle().
		Foreground(ui.LocoOrange).
		Render("\nYou can track deployment status by running `loco status`")
	fmt.Println(s)

	return nil
}
