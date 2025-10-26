package loco

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/nikumar1206/loco/internal/ui"
	"github.com/spf13/cobra"
)

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroy an application deployment",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := parseAndSetDebugFlag(cmd); err != nil {
			return fmt.Errorf("%w: %w", ErrCommandFailed, err)
		}

		yes, err := cmd.Flags().GetBool("yes")
		if err != nil {
			return fmt.Errorf("%w: %w", ErrFlagParsing, err)
		}

		// Lipgloss styles
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.LocoRed)
		labelStyle := lipgloss.NewStyle().Foreground(ui.LocoOrange)
		valueStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.LocoLightGreen)

		fmt.Println(titleStyle.Render("🔥 Destroy Command Called"))
		// fmt.Println(labelStyle.Render("File: ") + valueStyle.Render(file))

		fmt.Println(labelStyle.Render("Assume Yes: ") + valueStyle.Render(fmt.Sprintf("%t", yes)))
		return nil
	},
}

func init() {
	destroyCmd.Flags().StringP("file", "f", "", "Load configuration from FILE")
	destroyCmd.Flags().BoolP("yes", "y", false, "Assume yes to all prompts")
}
