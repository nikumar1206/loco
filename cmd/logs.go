package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View application logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Dummy log data for demonstration
		logData := [][]string{
			{"2024-06-01 10:00:00", "INFO", "Application started"},
			{"2024-06-01 10:01:00", "WARN", "Low disk space"},
			{"2024-06-01 10:02:00", "ERROR", "Failed to connect to DB"},
		}

		columns := []table.Column{
			{Title: "Time", Width: 20},
			{Title: "Level", Width: 8},
			{Title: "Message", Width: 40},
		}

		var rows []table.Row
		for _, log := range logData {
			rows = append(rows, table.Row{log[0], log[1], log[2]})
		}

		t := table.New(
			table.WithColumns(columns),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(7),
		)

		s := table.DefaultStyles()
		s.Header = s.Header.
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			BorderBottom(true).
			Bold(false)
		s.Selected = s.Selected.
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(false)
		t.SetStyles(s)

		m := logModel{
			table: t,
			baseStyle: lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("240")),
		}

		if _, err := tea.NewProgram(m).Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running table: %v\n", err)
			return err
		}

		return nil
	},
}

// ---- Bubble Tea model for logs table ----

type logModel struct {
	table     table.Model
	baseStyle lipgloss.Style
}

func (m logModel) Init() tea.Cmd {
	return nil
}

func (m logModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m logModel) View() string {
	return m.baseStyle.Render(m.table.View()) +
		"\n[↑↓] Navigate • [esc] Toggle focus • [q] Quit"
}

// ---- CLI flag bindings ----

func init() {
	logsCmd.Flags().StringP("file", "f", "", "Load configuration from FILE")
	logsCmd.Flags().BoolP("follow", "F", false, "Follow log output")
	logsCmd.Flags().IntP("lines", "n", 0, "Number of lines to show")
}
