package loco

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/nikumar1206/loco/internal/config"
	"github.com/nikumar1206/loco/internal/ui"
	appv1 "github.com/nikumar1206/loco/shared/proto/app/v1"
	appv1connect "github.com/nikumar1206/loco/shared/proto/app/v1/appv1connect"
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

		if output == "json" {
			return printEventsJSON(res.Msg.Events)
		}

		printEvents(res.Msg.Events)

		return nil
	},
}

func printEvents(events []*appv1.Event) {
	if len(events) == 0 {
		fmt.Println("No events found.")
		return
	}

	columns := []table.Column{
		{Title: "TIME", Width: 20},
		{Title: "REASON", Width: 20},
		{Title: "MESSAGE", Width: 80},
	}

	var rows []table.Row
	for _, event := range events {
		rows = append(rows, table.Row{
			event.Timestamp.AsTime().Format(time.RFC3339),
			event.Reason,
			simplifyMessage(event.Message),
		})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithHeight(len(rows)),
	)

	s := table.Styles{
		Header: lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ui.LocoMuted).
			BorderBottom(true).
			Bold(false),
		Cell: lipgloss.NewStyle().Padding(0, 1),
	}
	t.SetStyles(s)

	tableStyle := lipgloss.NewStyle().Margin(1, 2)
	fmt.Println(tableStyle.Render(t.View()))
}

func simplifyMessage(message string) string {
	if strings.Contains(message, "ImagePullBackOff") {
		return "Error: ImagePullBackOff"
	}
	if strings.Contains(message, "Failed to pull image") {
		return "Failed to pull image. Please check registry credentials and image path."
	}
	return message
}

func printEventsJSON(events []*appv1.Event) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(events)
}

func init() {
	eventsCmd.Flags().StringP("config", "c", "", "path to loco.toml config file")
	eventsCmd.Flags().StringP("output", "o", "table", "Output format: table | json")
	eventsCmd.Flags().String("host", "", "Loco API host")
}
