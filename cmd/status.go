package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show application status",
	RunE: func(cmd *cobra.Command, args []string) error {
		file, _ := cmd.Flags().GetString("file")
		output, _ := cmd.Flags().GetString("output")

		// Example scaffolded status data
		status := appStatus{
			File:         file,
			AppName:      "my-app",
			Environment:  "production",
			Status:       "Running",
			Pods:         3,
			CPUUsage:     "210m",
			MemUsage:     "380Mi",
			AvgLatency:   "87ms",
			ExternalURL:  "https://your-app.loco.run",
			DeployedAt:   "2025-06-29 13:00 EST",
			DeployedBy:   "nikhil@company.com",
			TLSStatus:    "Secured (Expires: 2025-09-01)",
			HealthStatus: "Passing",
			Autoscaling:  "Enabled (Min: 1, Max: 5)",
			Replicas:     "2 desired / 2 ready",
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
}

// --- Data Model ---

type appStatus struct {
	File         string `json:"file"`
	AppName      string `json:"appName"`
	Environment  string `json:"environment"`
	Status       string `json:"status"`
	Pods         int    `json:"pods"`
	CPUUsage     string `json:"cpuUsage"`
	MemUsage     string `json:"memUsage"`
	AvgLatency   string `json:"avgLatency"`
	ExternalURL  string `json:"externalUrl"`
	DeployedAt   string `json:"deployedAt"`
	DeployedBy   string `json:"deployedBy"`
	TLSStatus    string `json:"tlsStatus"`
	HealthStatus string `json:"healthStatus"`
	Autoscaling  string `json:"autoscaling"`
	Replicas     string `json:"replicas"`
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
		Foreground(lipgloss.Color("#00FFD2")).
		Bold(true).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#AAAAAA")).
		Width(18)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true)

	blockStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#F57900")).
		Padding(1, 2).
		Margin(1, 2)

	content := fmt.Sprintf(
		"%s %s\n%s %s\n%s %s\n%s %s\n%s %s\n%s %s\n%s %s\n%s %s\n%s %s\n%s %s\n%s %s\n%s %s\n%s %s\n%s %s",
		labelStyle.Render("App:"), valueStyle.Render(m.status.AppName),
		labelStyle.Render("Environment:"), valueStyle.Render(m.status.Environment),
		labelStyle.Render("Status:"), valueStyle.Render(m.status.Status),
		labelStyle.Render("Pods:"), valueStyle.Render(fmt.Sprintf("%d", m.status.Pods)),
		labelStyle.Render("CPU Usage:"), valueStyle.Render(m.status.CPUUsage),
		labelStyle.Render("Memory:"), valueStyle.Render(m.status.MemUsage),
		labelStyle.Render("Latency:"), valueStyle.Render(m.status.AvgLatency),
		labelStyle.Render("URL:"), valueStyle.Render(m.status.ExternalURL),
		labelStyle.Render("Deployed At:"), valueStyle.Render(m.status.DeployedAt),
		labelStyle.Render("Deployed By:"), valueStyle.Render(m.status.DeployedBy),
		labelStyle.Render("TLS:"), valueStyle.Render(m.status.TLSStatus),
		labelStyle.Render("Health:"), valueStyle.Render(m.status.HealthStatus),
		labelStyle.Render("Autoscaling:"), valueStyle.Render(m.status.Autoscaling),
		labelStyle.Render("Replicas:"), valueStyle.Render(m.status.Replicas),
	)

	return titleStyle.Render("Application Status") + "\n" +
		blockStyle.Render(content) +
		"\n\nPress [q] or [Enter] to exit."
}
