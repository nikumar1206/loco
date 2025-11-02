package loco

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"connectrpc.com/connect"
	"github.com/charmbracelet/lipgloss"
	"github.com/nikumar1206/loco/internal/ui"
	"github.com/nikumar1206/loco/shared/config"
	appv1 "github.com/nikumar1206/loco/shared/proto/app/v1"
	appv1connect "github.com/nikumar1206/loco/shared/proto/app/v1/appv1connect"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show application status",
	RunE: func(cmd *cobra.Command, args []string) error {
		host, err := getHost(cmd)
		if err != nil {
			return err
		}
		configPath, err := parseLocoTomlPath(cmd)
		if err != nil {
			return err
		}

		output, err := cmd.Flags().GetString("output")
		if err != nil {
			return fmt.Errorf("%w: %w", ErrFlagParsing, err)
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

		status := appStatus{
			StatusResponse: res.Msg,
			AppName:        cfg.LocoConfig.Metadata.Name,
		}

		if output == "json" {
			return printJSON(res.Msg)
		}

		m := newStatusModel(status)
		fmt.Println(m.View())
		return nil
	},
}

func init() {
	statusCmd.Flags().StringP("output", "o", "table", "Output format: table | json")
	statusCmd.Flags().StringP("config", "c", "", "path to loco.toml config file")
	statusCmd.Flags().String("host", "", "Set the host URL")
}

// --- Data Model ---

type appStatus struct {
	*appv1.StatusResponse
	AppName string `json:"appName"`
}

func printJSON(status *appv1.StatusResponse) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(status)
}

// --- TUI Model ---

type statusModel struct {
	status appStatus
}

func newStatusModel(s appStatus) statusModel {
	return statusModel{status: s}
}

func (m statusModel) View() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(ui.LocoCyan).
		Bold(true).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(ui.LocoDimGrey).
		Width(18)

	valueStyle := lipgloss.NewStyle().
		Foreground(ui.LocoWhite).
		Bold(true)

	blockStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.LocoOrange).
		Padding(1, 2).
		Margin(1, 2)

	content := fmt.Sprintf(
		"%s %s\n%s %s\n%s %s\n%s %s",
		labelStyle.Render("App:"), valueStyle.Render(m.status.AppName),
		labelStyle.Render("Status:"), valueStyle.Render(m.status.Health),
		labelStyle.Render("Replicas:"), valueStyle.Render(fmt.Sprintf("%d", m.status.Replicas)),
		labelStyle.Render("External URL:"), valueStyle.Render(m.status.Url),
	)

	return titleStyle.Render("Application Status") + "\n" + blockStyle.Render(content)
}
