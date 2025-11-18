package loco

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"github.com/charmbracelet/lipgloss"
	"github.com/nikumar1206/loco/internal/ui"
	appv1 "github.com/nikumar1206/loco/shared/proto/app/v1"
	appv1connect "github.com/nikumar1206/loco/shared/proto/app/v1/appv1connect"
	"github.com/spf13/cobra"
)

func init() {
	destroyCmd.Flags().StringP("app", "a", "", "Application name to destroy")
	destroyCmd.Flags().String("org", "", "organization ID")
	destroyCmd.Flags().String("workspace", "", "workspace ID")
	destroyCmd.Flags().BoolP("yes", "y", false, "Assume yes to all prompts")
	destroyCmd.Flags().String("host", "", "Set the host URL")
}

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroy an application deployment",
	RunE: func(cmd *cobra.Command, args []string) error {
		return destroyCmdFunc(cmd)
	},
}

func destroyCmdFunc(cmd *cobra.Command) error {
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

	yes, err := cmd.Flags().GetBool("yes")
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFlagParsing, err)
	}

	locoToken, err := getLocoToken()
	if err != nil {
		return ErrLoginRequired
	}

	appClient := appv1connect.NewAppServiceClient(http.DefaultClient, host)

	// List apps to find the one matching the name
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

	if !yes {
		confirmed, err := ui.AskYesNo(fmt.Sprintf("Are you sure you want to destroy the app '%s'?", appName))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("Aborted.")
			return nil
		}
	}

	slog.Debug("destroying app", "app_id", appID, "app_name", appName)

	destroyReq := connect.NewRequest(&appv1.DeleteAppRequest{
		Id: appID,
	})
	destroyReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", locoToken.Token))

	_, err = appClient.DeleteApp(ctx, destroyReq)
	if err != nil {
		slog.Error("failed to destroy app", "error", err)
		return fmt.Errorf("failed to destroy app '%s': %w", appName, err)
	}

	successMsg := fmt.Sprintf("\n🎉 App '%s' destroyed!", appName)
	s := lipgloss.NewStyle().
		Bold(true).
		Foreground(ui.LocoLightGreen).
		Render(successMsg)

	fmt.Println(s)

	return nil
}
