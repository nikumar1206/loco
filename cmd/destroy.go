package cmd

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
		// file, _ := cmd.Flags().GetString("file")
		yes, _ := cmd.Flags().GetBool("yes")

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
