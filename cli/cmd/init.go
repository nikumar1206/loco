package cmd

import (
	"fmt"

	"github.com/nikumar1206/loco/cli/pkg/config"
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

		cmd.Println("Loco project initialized successfully.")
		return nil
	},
}
