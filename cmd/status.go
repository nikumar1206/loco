package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nikumar1206/loco/internal/api"
	"github.com/nikumar1206/loco/internal/config"
	"github.com/nikumar1206/loco/internal/ui"
	"github.com/spf13/cobra"
)

type DeploymentStatus struct {
	Status          string    `json:"status"`
	Pods            int       `json:"pods"`
	CPUUsage        string    `json:"cpuUsage"`
	MemoryUsage     string    `json:"memoryUsage"`
	Latency         string    `json:"latency"`
	URL             string    `json:"url"`
	DeployedAt      time.Time `json:"deployedAt"`
	DeployedBy      string    `json:"deployedBy"`
	TLS             string    `json:"tls"`
	Health          string    `json:"health"`
	Autoscaling     bool      `json:"autoscaling"`
	MinReplicas     int32     `json:"minReplicas"`
	MaxReplicas     int32     `json:"maxReplicas"`
	DesiredReplicas int32     `json:"desiredReplicas"`
	ReadyReplicas   int32     `json:"readyReplicas"`
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show application status",
	RunE: func(cmd *cobra.Command, args []string) error {
		file, _ := cmd.Flags().GetString("file")
		output, _ := cmd.Flags().GetString("output")

		isDev, err := cmd.Flags().GetBool("dev")
		if err != nil {
			return fmt.Errorf("error reading dev flag: %w", err)
		}

		host := determineHost(isDev)

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

		locoToken, err := getLocoToken()
		if err != nil {
			return err
		}

		deploymentStatus := new(DeploymentStatus)
		path := fmt.Sprintf("/api/v1/app/%s/status", cfg.Name)

		statusUrl := host + path

		resp, err := api.Resty.R().
			SetHeader("Authorization", fmt.Sprintf("Bearer %s", locoToken.Token)).
			SetResult(&deploymentStatus).
			Get(statusUrl)
		if err != nil {
			slog.Error(err.Error())
			return err
		}

		if resp.IsError() {
			return fmt.Errorf("client/server error: %s", resp.String())
		}

		status := appStatus{
			DeploymentStatus: *deploymentStatus,
			File:             file,
			AppName:          cfg.Name,
			Environment:      "production",
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
	DeploymentStatus
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
		labelStyle.Render("CPU Usage:"), valueStyle.Render(m.status.CPUUsage),
		labelStyle.Render("Memory:"), valueStyle.Render(m.status.MemoryUsage),
		labelStyle.Render("Latency:"), valueStyle.Render(m.status.Latency),
		labelStyle.Render("URL:"), valueStyle.Render(m.status.URL),
		labelStyle.Render("Deployed At:"), valueStyle.Render(m.status.DeployedAt.String()),
		labelStyle.Render("Deployed By:"), valueStyle.Render(m.status.DeployedBy),
		labelStyle.Render("TLS:"), valueStyle.Render(m.status.TLS),
		labelStyle.Render("Health:"), valueStyle.Render(m.status.Health),
		labelStyle.Render("Autoscaling:"), valueStyle.Render(strconv.FormatBool(m.status.Autoscaling)),
		labelStyle.Render("Ready Replicas:"), valueStyle.Render(readyReplicas),
		labelStyle.Render("Replicas (Min/Desired/Max):"), valueStyle.Render(replicaSummary),
	)

	return titleStyle.Render("Application Status") + "\n" +
		blockStyle.Render(content) +
		"\n\nPress [q] or [Enter] to exit."
}
