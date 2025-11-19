package loco

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"connectrpc.com/connect"
	"github.com/charmbracelet/lipgloss"
	"github.com/nikumar1206/loco/internal/client"
	"github.com/nikumar1206/loco/internal/ui"
	"github.com/nikumar1206/loco/shared"
	appv1 "github.com/nikumar1206/loco/shared/proto/app/v1"
	appv1connect "github.com/nikumar1206/loco/shared/proto/app/v1/appv1connect"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show application status",
	RunE: func(cmd *cobra.Command, args []string) error {
		return statusCmdFunc(cmd)
	},
}

func statusCmdFunc(cmd *cobra.Command) error {
	ctx := context.Background()

	host, err := getHost(cmd)
	if err != nil {
		return err
	}

	workspaceID, err := getWorkspaceId(cmd)
	if err != nil {
		return err
	}

	appName, err := cmd.Flags().GetString("app")
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFlagParsing, err)
	}
	if appName == "" {
		return fmt.Errorf("app name is required. Use --app flag")
	}

	output, err := cmd.Flags().GetString("output")
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFlagParsing, err)
	}

	locoToken, err := getLocoToken()
	if err != nil {
		return ErrLoginRequired
	}

	appClient := appv1connect.NewAppServiceClient(shared.NewHTTPClient(), host)

	slog.Debug("listing apps to find app by name", "workspace_id", workspaceID, "app_name", appName)

	listAppsReq := connect.NewRequest(&appv1.ListAppsRequest{
		WorkspaceId: workspaceID,
	})
	listAppsReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", locoToken.Token))

	listAppsResp, err := appClient.ListApps(ctx, listAppsReq)
	if err != nil {
		slog.Debug("failed to list apps", "error", err)
		return fmt.Errorf("failed to list apps: %w", err)
	}

	var appID int64
	for _, app := range listAppsResp.Msg.Apps {
		if app.Name == appName {
			appID = app.Id
			slog.Debug("found app by name", "app_name", appName, "app_id", appID)
			break
		}
	}

	if appID == 0 {
		return fmt.Errorf("app '%s' not found in workspace", appName)
	}

	apiClient := client.NewClient(host, locoToken.Token)

	slog.Debug("retrieving app status", "app_id", appID, "app_name", appName)

	statusResp, err := apiClient.GetAppStatus(ctx, appID)
	if err != nil {
		slog.Error("failed to get app status", "error", err)
		return fmt.Errorf("failed to get app status: %w", err)
	}

	if output == "json" {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(statusResp)
	}

	m := newStatusModel(statusResp)
	fmt.Println(m.View())
	return nil
}

func init() {
	statusCmd.Flags().StringP("app", "a", "", "Application name")
	statusCmd.Flags().String("org", "", "organization ID")
	statusCmd.Flags().String("workspace", "", "workspace ID")
	statusCmd.Flags().StringP("output", "", "table", "Output format: table | json")
	statusCmd.Flags().String("host", "", "Set the host URL")
}

type statusModel struct {
	response *appv1.GetAppStatusResponse
}

func newStatusModel(resp *appv1.GetAppStatusResponse) statusModel {
	return statusModel{response: resp}
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

	appName := m.response.App.Name
	var status, replicas, subdomain, domain, deploymentID string

	if m.response.CurrentDeployment != nil {
		status = m.response.CurrentDeployment.Status
		replicas = fmt.Sprintf("%d", m.response.CurrentDeployment.Replicas)
		deploymentID = fmt.Sprintf("%d", m.response.CurrentDeployment.Id)
	} else {
		status = "no deployment"
		replicas = "0"
		deploymentID = "N/A"
	}

	subdomain = m.response.App.Subdomain
	domain = m.response.App.Domain
	url := fmt.Sprintf("%s.%s", subdomain, domain)

	content := fmt.Sprintf(
		"%s %s\n%s %s\n%s %s\n%s %s\n%s %s",
		labelStyle.Render("App:"), valueStyle.Render(appName),
		labelStyle.Render("Status:"), valueStyle.Render(status),
		labelStyle.Render("Replicas:"), valueStyle.Render(replicas),
		labelStyle.Render("Deployment ID:"), valueStyle.Render(deploymentID),
		labelStyle.Render("URL:"), valueStyle.Render(url),
	)

	return titleStyle.Render("Application Status") + "\n" + blockStyle.Render(content)
}
