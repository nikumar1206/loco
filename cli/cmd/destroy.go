package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroy an application deployment",
	RunE: func(cmd *cobra.Command, args []string) error {
		file, _ := cmd.Flags().GetString("file")
		yes, _ := cmd.Flags().GetBool("yes")

		locoOut(LOCO__OK_PREFIX, "destroy command called")
		locoOut(LOCO__OK_PREFIX, fmt.Sprintf("file: %s", file))
		locoOut(LOCO__OK_PREFIX, fmt.Sprintf("yes: %t", yes))
		return nil
	},
}

func init() {
	destroyCmd.Flags().StringP("file", "f", "", "Load configuration from FILE")
	destroyCmd.Flags().BoolP("yes", "y", false, "Assume yes to all prompts")
}
