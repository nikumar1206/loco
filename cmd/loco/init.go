package loco

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"github.com/nikumar1206/loco/internal/config"
	"github.com/nikumar1206/loco/internal/ui"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Loco project",
	RunE: func(cmd *cobra.Command, args []string) error {
		parseAndSetDebugFlag(cmd)
		workingDir, err := os.Getwd()
		if err != nil {
			slog.Debug("failed to get working directory", "error", err)
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		_, dirName := filepath.Split(workingDir)
		err = config.CreateDefault(dirName)
		if err != nil {
			slog.Debug("failed to create default config", "error", err)
			return fmt.Errorf("failed to create loco.toml: %w", err)
		}

		style := lipgloss.NewStyle().Foreground(ui.LocoLightGreen).Bold(true)
		cmd.Printf("Created a %s in the working directory.\n", style.Render("`loco.toml`"))
		return nil
	},
}
