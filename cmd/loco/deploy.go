package loco

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"github.com/charmbracelet/lipgloss"
	"github.com/nikumar1206/loco/internal/client"
	"github.com/nikumar1206/loco/internal/config"
	"github.com/nikumar1206/loco/internal/docker"
	"github.com/nikumar1206/loco/internal/ui"
	registryv1 "github.com/nikumar1206/loco/proto/registry/v1"
	registryv1connect "github.com/nikumar1206/loco/proto/registry/v1/registryv1connect"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy/Update an application to Loco.",
	Long:  "Deploy/Update an application to Loco.\nThis builds and pushes a Docker image to the Loco registry and deploys it onto the Loco platform under the specified subdomain. This cmd can also discover all loco.toml files in the current directory using the -r flag.",

	RunE: func(cmd *cobra.Command, args []string) error {
		return deployCmdFunc(cmd, args)
	},
}

func init() {
	deployCmd.Flags().StringP("config", "c", "", "path to loco.toml config file")
	deployCmd.Flags().BoolP("yes", "y", false, "Assume yes to all prompts")
	deployCmd.Flags().StringP("image", "i", "", "image tag to use for deployment")
}

func deployCmdFunc(cmd *cobra.Command, _ []string) error {
	host, err := getHost(cmd)
	if err != nil {
		return err
	}
	configPath, err := parseLocoTomlPath(cmd)
	if err != nil {
		return err
	}
	imageId, err := parseImageId(cmd)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFlagParsing, err)
	}

	var tokenResponse *connect.Response[registryv1.GitlabTokenResponse]

	locoToken, err := getLocoToken()
	if err != nil {
		return err
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if validateErr := config.Validate(cfg.LocoConfig); validateErr != nil {
		return fmt.Errorf("%w: %w", ErrValidation, validateErr)
	}
	config.FillSensibleDefaults(cfg.LocoConfig)

	cfgValid := lipgloss.NewStyle().
		Foreground(ui.LocoLightGreen).
		Render("\n🎉 Validated loco.toml. Beginning deployment!")
	fmt.Print(cfgValid)

	dockerCli, err := docker.NewDockerClient(cfg)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrDockerClient, err)
	}
	defer func() {
		if closeErr := dockerCli.Close(); closeErr != nil {
			slog.Debug("failed to close docker client", "error", closeErr)
		}
	}()

	apiClient := client.NewClient(host)
	registryClient := registryv1connect.NewRegistryServiceClient(http.DefaultClient, host)

	buildStep := ui.Step{
		Title: "Build Docker image",
		Run: func(logf func(string)) error {
			if buildErr := dockerCli.BuildImage(context.Background(), logf); buildErr != nil {
				return fmt.Errorf("%w: %w", ErrDockerBuild, buildErr)
			}
			return nil
		},
	}

	validateStep := ui.Step{
		Title: "Validate and Tag Docker image",
		Run: func(logf func(string)) error {
			if validateErr := dockerCli.ValidateImage(context.Background(), imageId, logf); validateErr != nil {
				return fmt.Errorf("%w: %w", ErrDockerValidate, validateErr)
			}
			if tagErr := dockerCli.ImageTag(context.Background(), imageId); tagErr != nil {
				return fmt.Errorf("failed to tag image: %w", tagErr)
			}
			return nil
		},
	}

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
				// improve below function
				imageTag := dockerCli.GenerateImageTag(tokenResponse.Msg.Image, imageId)
				dockerCli.ImageName = imageTag
				slog.Debug("generated image tag", "tag", dockerCli.ImageName)
				return nil
			},
		},
	}

	if imageId != "" {
		steps = append(steps, validateStep)
	} else {
		steps = append(steps, buildStep)
	}

	steps = append(steps, ui.Step{
		Title: "Push image to registry",
		Run: func(logf func(string)) error {
			if pushErr := dockerCli.PushImage(context.Background(), logf, tokenResponse.Msg.GetUsername(), tokenResponse.Msg.GetToken()); pushErr != nil {
				return fmt.Errorf("%w: %w", ErrDockerPush, pushErr)
			}
			return nil
		},
	})

	steps = append(steps, ui.Step{
		Title: "Deploying App on Loco 🔥",
		Run: func(logf func(string)) error {
			// todo: cleanup how we pass variables around, why should this be dockercli.image?
			// and why would this be generated client side?
			if err := apiClient.DeployApp(cfg, dockerCli.ImageName, locoToken.Token, logf); err != nil {
				return err
			}
			return nil
		},
	})

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
