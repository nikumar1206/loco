package loco

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"github.com/nikumar1206/loco/internal/ui"
	"github.com/nikumar1206/loco/shared/config"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Loco project",
	RunE: func(cmd *cobra.Command, args []string) error {
		force, err := cmd.Flags().GetBool("force")
		if err != nil {
			return fmt.Errorf("%w: %w", ErrFlagParsing, err)
		}

		if _, statErr := os.Stat("loco.toml"); statErr == nil && !force {
			overwrite, askErr := ui.AskYesNo("A loco.toml file already exists. Do you want to overwrite it?")
			if askErr != nil {
				return fmt.Errorf("%w: %w", ErrCommandFailed, askErr)
			}
			if !overwrite {
				fmt.Println("Aborted.")
				return nil
			}
		}

		appName, err := ui.AskForString("Enter the name of your application: ")
		if err != nil {
			return fmt.Errorf("%w: %w", ErrCommandFailed, err)
		}

		if appName == "" {
			workingDir, getwdErr := os.Getwd()
			if getwdErr != nil {
				return fmt.Errorf("%w: %w", ErrFileAccess, getwdErr)
			}
			_, dirName := filepath.Split(workingDir)
			appName = dirName
		}

		if err := config.CreateDefault(appName); err != nil {
			return fmt.Errorf("%w: %w", ErrConfigLoad, err)
		}

		style := lipgloss.NewStyle().Foreground(ui.LocoLightGreen).Bold(true)
		cmd.Printf("Created a %s in the working directory.\n", style.Render("`loco.toml`"))
		return nil
	},
}

func init() {
	initCmd.Flags().BoolP("force", "f", false, "Force overwrite of existing loco.toml file")
}
