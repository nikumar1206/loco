package cmd

import (
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:     "loco",
	Short:   "A CLI for managing loco deployments",
	Version: "v0.0.1",
}

func init() {
	RootCmd.AddCommand(initCmd)
	RootCmd.AddCommand(deployCmd)
	RootCmd.AddCommand(logsCmd)
	RootCmd.AddCommand(statusCmd)
	RootCmd.AddCommand(destroyCmd)
}
