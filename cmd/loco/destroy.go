package loco

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"github.com/charmbracelet/lipgloss"
	"github.com/nikumar1206/loco/internal/config"
	"github.com/nikumar1206/loco/internal/ui"
	appv1 "github.com/nikumar1206/loco/proto/app/v1"
	appv1connect "github.com/nikumar1206/loco/proto/app/v1/appv1connect"
	"github.com/spf13/cobra"
)

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroy an application deployment",
	RunE: func(cmd *cobra.Command, args []string) error {
		return destroyCmdFunc(cmd, args)
	},
}

func destroyCmdFunc(cmd *cobra.Command, _ []string) error {
	yes, err := cmd.Flags().GetBool("yes")
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFlagParsing, err)
	}

	host, err := getHost(cmd)
	if err != nil {
		return err
	}

	configPath, err := parseLocoTomlPath(cmd)
	if err != nil {
		return err
	}

	locoToken, err := getLocoToken()
	if err != nil {
		return err
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if !yes {
		confirmed, err := ui.AskYesNo(fmt.Sprintf("Are you sure you want to destroy the app '%s'?", cfg.LocoConfig.Metadata.Name))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("Aborted.")
			return nil
		}
	}

	apiClient := appv1connect.NewAppServiceClient(http.DefaultClient, host)

	steps := []ui.Step{
		{
			Title: "Destroying App on Loco 🔥",
			Run: func(logf func(string)) error {
				slog.Debug("destroying app", "app", cfg.LocoConfig.Metadata.Name)
				logf(fmt.Sprintf("destroying app %s", cfg.LocoConfig.Metadata.Name))

				req := &appv1.DestroyAppRequest{
					Name: cfg.LocoConfig.Metadata.Name,
				}

				slog.Debug("destroy request", "req", req)

				destroyReq := connect.NewRequest(req)

				destroyReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", locoToken.Token))

				_, err := apiClient.DestroyApp(context.Background(), destroyReq)
				if err != nil {
					slog.Error("failed to destroy app", "error", err)
					return err
				}

				logf(fmt.Sprintf("app %s destroyed", cfg.LocoConfig.Metadata.Name))
				slog.Debug("app destroyed", "app", cfg.LocoConfig.Metadata.Name)
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
		Render("\n🎉 App destruction scheduled!")

	fmt.Println(s)

	return nil
}

func init() {
	destroyCmd.Flags().BoolP("yes", "y", false, "Assume yes to all prompts")
	destroyCmd.Flags().StringP("config", "c", "", "path to loco.toml config file")
}
