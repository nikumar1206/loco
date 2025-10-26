package loco

import (
	"fmt"
	"log/slog"

	"github.com/charmbracelet/lipgloss"
	"github.com/nikumar1206/loco/internal/config"
	"github.com/nikumar1206/loco/internal/ui"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate a loco.toml file.",
	Long:  "Validate a loco.toml file.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return validateCmdFunc(cmd, args)
	},
}

func init() {
	validateCmd.Flags().StringP("config", "c", "", "path to loco.toml config file")
}

func validateCmdFunc(cmd *cobra.Command, _ []string) error {
	configPath, err := parseLocoTomlPath(cmd)
	if err != nil {
		return err
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Debug("failed to load config", "path", configPath, "error", err)
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := config.Validate(cfg.LocoConfig); err != nil {
		slog.Debug("invalid configuration", "error", err)
		return fmt.Errorf("invalid configuration: %w", err)
	}

	cfgValid := lipgloss.NewStyle().
		Foreground(ui.LocoLightGreen).
		Render("\n🎉 loco.toml is valid!")
	fmt.Print(cfgValid)

	return nil
}
