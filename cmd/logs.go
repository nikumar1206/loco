package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nikumar1206/loco/internal/api"
	"github.com/nikumar1206/loco/internal/config"
	"github.com/nikumar1206/loco/internal/ui"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View application logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		isDev, err := cmd.Flags().GetBool("dev")
		if err != nil {
			return fmt.Errorf("error reading dev flag: %w", err)
		}

		var host string
		if isDev {
			host = "http://localhost:8000"
		} else {
			host = "https://loco.deploy-app.com"
		}

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
		fmt.Println("we got till here")
		client := api.NewClient(host)
		ctx, cancel := context.WithCancel(context.Background())

		columns := []table.Column{
			{Title: "Time", Width: 20},
			{Title: "Level", Width: 8},
			{Title: "Message", Width: 100},
		}

		t := table.New(
			table.WithColumns(columns),
			table.WithRows([]table.Row{}),
			table.WithFocused(true),
			table.WithHeight(20),
		)

		s := table.DefaultStyles()
		s.Header = s.Header.
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ui.LocoMuted).
			BorderBottom(true).
			Bold(false)
		s.Selected = s.Selected.
			Foreground(ui.LocoWhite).
			Background(ui.LocoGreen).
			Bold(false)
		t.SetStyles(s)

		logsChan := make(chan api.LogEntry)
		errChan := make(chan error)

		// start the http stream
		go client.StreamLogs(ctx, "", cfg.Name, logsChan, errChan)

		m := logModel{
			table:     t,
			baseStyle: lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(ui.LocoGreyish),
			logs:      []table.Row{},
			logsChan:  logsChan,
			errChan:   errChan,
			ctx:       ctx,
			cancel:    cancel,
		}

		if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running log viewer: %v\n", err)
			return err
		}

		return nil
	},
}

type logMsg struct {
	Time    string
	PodId   string
	Message string
}

type errMsg struct{ error }

type logModel struct {
	table     table.Model
	baseStyle lipgloss.Style
	logs      []table.Row

	logsChan chan api.LogEntry
	errChan  chan error

	ctx    context.Context
	cancel context.CancelFunc
}

func (m logModel) Init() tea.Cmd {
	return m.waitForLog()
}

// The polling command: checks for new logs or errors and emits tea.Msg
func (m logModel) waitForLog() tea.Cmd {
	return func() tea.Msg {
		select {
		case log := <-m.logsChan:
			return logMsg{
				Time: log.Timestamp.Local().Format(time.RFC3339),

				PodId:   log.PodName,
				Message: log.Log,
			}
		case err := <-m.errChan:
			return errMsg{err}
		case <-m.ctx.Done():
			return tea.Quit()
		case <-time.After(100 * time.Millisecond):
			// Timeout to keep the program responsive, even if no logs come in
			return m.waitForLog()
		}
	}
}

func (m logModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case logMsg:
		newRow := table.Row{msg.Time, msg.PodId, msg.Message}
		m.logs = append(m.logs, newRow)
		m.table.SetRows(m.logs)
		return m, m.waitForLog() // Continue polling

	case errMsg:
		fmt.Fprintf(os.Stderr, "Error: %v\n", msg)
		return m, tea.Quit

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
		case "q", "ctrl+c":
			m.cancel()
			return m, tea.Quit
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, tea.Batch(cmd, m.waitForLog())
}

func (m logModel) View() string {
	return m.baseStyle.Render(m.table.View()) +
		"\n[↑↓] Navigate • [esc] Toggle focus • [q] Quit"
}

func init() {
	logsCmd.Flags().StringP("config", "c", "", "Path to loco.toml config file")
	logsCmd.Flags().BoolP("follow", "f", false, "Follow log output")
	logsCmd.Flags().IntP("lines", "n", 0, "Number of lines to show")
}
