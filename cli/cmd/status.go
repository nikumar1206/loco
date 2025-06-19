package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show application status",
	RunE: func(cmd *cobra.Command, args []string) error {
		file, _ := cmd.Flags().GetString("file")

		locoOut(LOCO__OK_PREFIX, "status command called")
		locoOut(LOCO__OK_PREFIX, fmt.Sprintf("file: %s", file))
		return nil
	},
}

func init() {
	statusCmd.Flags().StringP("file", "f", "", "Load configuration from FILE")
}
