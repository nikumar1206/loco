package cmd

import (
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

		// Example scaffolded status data
		status := appStatus{
			File:        file,
			Pods:        3,
			CPUUsage:    "210m",
			MemUsage:    "380Mi",
			AvgLatency:  "87ms",
			ExternalURL: "https://your-app.loco.run",
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
}

// --- Data Model ---

type appStatus struct {
	File        string
	Pods        int
	CPUUsage    string
	MemUsage    string
	AvgLatency  string
	ExternalURL string
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
		Width(14)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true)

	blockStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#F57900")).
		Padding(1, 2).
		Margin(1, 2)

	content := fmt.Sprintf(
		"%s %s\n%s %s\n%s %s\n%s %s\n%s %s",
		labelStyle.Render("Pods:"), valueStyle.Render(fmt.Sprintf("%d", m.status.Pods)),
		labelStyle.Render("CPU Usage:"), valueStyle.Render(m.status.CPUUsage),
		labelStyle.Render("Memory:"), valueStyle.Render(m.status.MemUsage),
		labelStyle.Render("Latency:"), valueStyle.Render(m.status.AvgLatency),
		labelStyle.Render("URL:"), valueStyle.Render(m.status.ExternalURL),
	)

	return titleStyle.Render("Application Status") + "\n" +
		blockStyle.Render(content) +
		"\n\nPress [q] or [Enter] to exit."
}
