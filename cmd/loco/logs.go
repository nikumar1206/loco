package loco

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nikumar1206/loco/internal/client"
	"github.com/nikumar1206/loco/internal/config"
	"github.com/nikumar1206/loco/internal/ui"
	appv1 "github.com/nikumar1206/loco/proto/app/v1"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View application logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		output, _ := cmd.Flags().GetString("output")

		switch output {
		case "json":
			return streamLogsAsJson(cmd, args)
		case "table":
			return streamLogsInteractive(cmd, args)
		case "": // default
			return streamLogsInteractive(cmd, args)
		default:
			return fmt.Errorf("invalid output format: %s", output)
		}
	},
}

func streamLogsAsJson(cmd *cobra.Command, _ []string) error {
	parseAndSetDebugFlag(cmd)
	host := parseDevFlag(cmd)
	configPath := parseLocoTomlPath(cmd)

	locoToken, err := getLocoToken()
	if err != nil {
		slog.Debug("failed to get loco token", "error", err)
		return err
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Debug("failed to load config", "path", configPath, "error", err)
		return fmt.Errorf("failed to load config: %w", err)
	}

	c := client.NewClient(host)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logsChan := make(chan client.LogEntry)
	errChan := make(chan error)

	go c.StreamLogs(ctx, locoToken.Token, &appv1.LogsRequest{AppName: cfg.LocoConfig.Metadata.Name}, logsChan, errChan)

	for {
		select {
		case logEntry := <-logsChan:
			jsonLog, err := json.Marshal(logEntry)
			if err != nil {
				slog.Debug("failed to marshal log entry to json", "error", err)
				fmt.Fprintf(os.Stderr, "Error marshaling log: %v\n", err)
				continue
			}
			fmt.Println(string(jsonLog))
		case err := <-errChan:
			return err
		case <-ctx.Done():
			return nil
		}
	}
}

func streamLogsInteractive(cmd *cobra.Command, _ []string) error {
	parseAndSetDebugFlag(cmd)
	host := parseDevFlag(cmd)
	configPath := parseLocoTomlPath(cmd)

	locoToken, err := getLocoToken()
	if err != nil {
		slog.Debug("failed to get loco token", "error", err)
		return err
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Debug("failed to load config", "path", configPath, "error", err)
		return fmt.Errorf("failed to load config: %w", err)
	}

	c := client.NewClient(host)
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

	logsChan := make(chan client.LogEntry)
	errChan := make(chan error)

	go c.StreamLogs(ctx, locoToken.Token, &appv1.LogsRequest{AppName: cfg.LocoConfig.Metadata.Name}, logsChan, errChan)
	slog.Debug("streaming logs for app", "app_name", cfg.LocoConfig.Metadata.Name)

	m := logModel{
		table:     t,
		baseStyle: lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(ui.LocoGreyish),
		logs:      []table.Row{},
		logsChan:  logsChan,
		errChan:   errChan,
		ctx:       ctx,
		cancel:    cancel,
	}

	if finalModel, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running log viewer: %v\n", err)
		return err
	} else if fm, ok := finalModel.(logModel); ok && fm.err != nil {
		slog.Debug("log streaming failed", "error", fm.err)
		return fm.err
	}

	return nil
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

	logsChan chan client.LogEntry
	errChan  chan error
	err      error

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
				Time:    log.Timestamp.Local().Format(time.RFC3339),
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
		m.err = msg.error
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
	if m.err != nil {
		return lipgloss.NewStyle().Foreground(ui.LocoRed).Render(
			fmt.Sprintf("Error: %v", m.err),
		)
	}
	return m.baseStyle.Render(m.table.View()) +
		"\n[↑↓] Navigate • [esc] Toggle focus • [q] Quit"
}

func init() {
	logsCmd.Flags().StringP("config", "c", "", "Path to loco.toml config file")
	logsCmd.Flags().BoolP("follow", "f", false, "Follow log output")
	logsCmd.Flags().IntP("lines", "n", 0, "Number of lines to show")
	logsCmd.Flags().StringP("output", "o", "", "Output format (json, table). Defaults to table.")
}
