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

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Show application events",
	RunE: func(cmd *cobra.Command, args []string) error {
		host, err := getHost(cmd)
		if err != nil {
			return err
		}
		configPath, err := parseLocoTomlPath(cmd)
		if err != nil {
			return err
		}

		cfg, err := config.Load(configPath)
		if err != nil {
			slog.Debug("failed to load config", "path", configPath, "error", err)
			return fmt.Errorf("failed to load config: %w", err)
		}

		locoToken, err := getLocoToken()
		if err != nil {
			slog.Debug("Error retrieving loco token", "error", err)
			return fmt.Errorf("loco token not found. Please login via `loco login`")
		}
		client := appv1connect.NewAppServiceClient(http.DefaultClient, host)

		req := connect.NewRequest(&appv1.StatusRequest{
			AppName: cfg.LocoConfig.Metadata.Name,
		})
		req.Header().Set("Authorization", fmt.Sprintf("Bearer %s", locoToken.Token))

		res, err := client.Status(context.Background(), req)
		if err != nil {
			slog.Debug("failed to get app status", "app_name", cfg.LocoConfig.Metadata.Name, "error", err)
			return err
		}
		slog.Debug("retrieved app status", "status", res.Msg.Status)

		printEvents(cfg.LocoConfig.Metadata.Name, res.Msg.Events)

		return nil
	},
}

func printEvents(appName string, events []string) {
	titleStyle := lipgloss.NewStyle().
		Foreground(ui.LocoCyan).
		Bold(true).
		MarginBottom(1)

	fmt.Println(titleStyle.Render(fmt.Sprintf("Events for %s", appName)))

	if len(events) == 0 {
		fmt.Println("No events found.")
		return
	}

	for _, event := range events {
		fmt.Println(event)
	}
}

func init() {
	eventsCmd.Flags().StringP("config", "c", "", "path to loco.toml config file")
}
