package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/nikumar1206/loco/internal/config"
	"github.com/nikumar1206/loco/internal/ui"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Loco project",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := config.CreateDefault()
		if err != nil {
			return fmt.Errorf("failed to create loco.toml: %w", err)
		}

		style := lipgloss.NewStyle().Foreground(ui.LocoLightGreen).Bold(true)
		cmd.Printf("Created a %s in the working directory.\n", style.Render("`loco.toml`"))
		return nil
	},
}
