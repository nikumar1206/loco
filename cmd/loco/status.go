package loco

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"connectrpc.com/connect"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nikumar1206/loco/internal/config"
	"github.com/nikumar1206/loco/internal/ui"
	appv1 "github.com/nikumar1206/loco/proto/app/v1"
	appv1connect "github.com/nikumar1206/loco/proto/app/v1/appv1connect"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show application status",
	RunE: func(cmd *cobra.Command, args []string) error {
		parseAndSetDebugFlag(cmd)
		host := parseDevFlag(cmd)
		configPath := parseLocoTomlPath(cmd)

		file, _ := cmd.Flags().GetString("file")
		output, _ := cmd.Flags().GetString("output")

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
			File:           file,
			AppName:        cfg.LocoConfig.Metadata.Name,
			Environment:    "production",
		}

		if output == "json" {
			return printJSON(status)
		}

		p := tea.NewProgram(newStatusModel(status))
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
			return err
		}
		return nil
	},
}

func init() {
	statusCmd.Flags().StringP("file", "f", "", "Load configuration from FILE")
	statusCmd.Flags().StringP("output", "o", "ux", "Output format: ux | json")
	statusCmd.Flags().StringP("config", "c", "", "path to loco.toml config file")
}

// --- Data Model ---

type appStatus struct {
	*appv1.StatusResponse
	File        string `json:"file"`
	AppName     string `json:"appName"`
	Environment string `json:"environment"`
	// Status       string `json:"status"`
	// Pods         int    `json:"pods"`
	// CPUUsage     string `json:"cpuUsage"`
	// MemUsage     string `json:"memUsage"`
	// AvgLatency   string `json:"avgLatency"`
	// ExternalURL  string `json:"externalUrl"`
	// DeployedAt   string `json:"deployedAt"`
	// DeployedBy   string `json:"deployedBy"`
	// TLSStatus    string `json:"tlsStatus"`
	// HealthStatus string `json:"healthStatus"`
	// Autoscaling  string `json:"autoscaling"`
	// Replicas     string `json:"replicas"`
}

func printJSON(status appStatus) error {
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

func (m statusModel) Init() tea.Cmd {
	return nil
}

func (m statusModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "enter":
			return m, tea.Quit
		}
	}
	return m, nil
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

	readyReplicas := strconv.Itoa(int(m.status.ReadyReplicas))

	replicaSummary := fmt.Sprintf("%d / %d / %d", m.status.MinReplicas, m.status.DesiredReplicas, m.status.MaxReplicas)

	content := fmt.Sprintf(
		"%s %s\n%s %s\n%s %s\n%s %d\n%s %s\n%s %s\n%s %s\n%s %s\n%s %s\n%s %s\n%s %s\n%s %s\n%s %s\n%s %s\n%s %s",
		labelStyle.Render("App:"), valueStyle.Render(m.status.AppName),
		labelStyle.Render("Environment:"), valueStyle.Render(m.status.Environment),
		labelStyle.Render("Status:"), valueStyle.Render(m.status.Status),
		labelStyle.Render("Pods:"), m.status.Pods,
		labelStyle.Render("CPU Usage:"), valueStyle.Render(m.status.CpuUsage),
		labelStyle.Render("Memory:"), valueStyle.Render(m.status.MemoryUsage),
		labelStyle.Render("Latency:"), valueStyle.Render(m.status.Latency),
		labelStyle.Render("URL:"), valueStyle.Render(m.status.Url),
		labelStyle.Render("Deployed At:"), valueStyle.Render(m.status.DeployedAt.String()),
		labelStyle.Render("Deployed By:"), valueStyle.Render(m.status.DeployedBy),
		labelStyle.Render("TLS:"), valueStyle.Render(m.status.Tls),
		labelStyle.Render("Health:"), valueStyle.Render(m.status.Health),
		labelStyle.Render("Autoscaling:"), valueStyle.Render(strconv.FormatBool(m.status.Autoscaling)),
		labelStyle.Render("Ready Replicas:"), valueStyle.Render(readyReplicas),
		labelStyle.Render("Replicas (Min/Desired/Max):"), valueStyle.Render(replicaSummary),
	)

	return titleStyle.Render("Application Status") + "\n" +
		blockStyle.Render(content) +
		"\n\nPress [q] or [Enter] to exit."
}
