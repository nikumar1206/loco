package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View application logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		file, _ := cmd.Flags().GetString("file")
		follow, _ := cmd.Flags().GetBool("follow")
		lines, _ := cmd.Flags().GetInt("lines")

		locoOut(LOCO__OK_PREFIX, "logs command called")
		locoOut(LOCO__OK_PREFIX, fmt.Sprintf("file: %s", file))
		locoOut(LOCO__OK_PREFIX, fmt.Sprintf("follow: %t", follow))
		locoOut(LOCO__OK_PREFIX, fmt.Sprintf("lines: %d", lines))
		return nil
	},
}

func init() {
	logsCmd.Flags().StringP("file", "f", "", "Load configuration from FILE")
	logsCmd.Flags().BoolP("follow", "F", false, "Follow log output")
	logsCmd.Flags().IntP("lines", "n", 0, "Number of lines to show")
}
