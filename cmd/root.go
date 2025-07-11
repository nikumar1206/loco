package cmd

import (
	"github.com/spf13/cobra"
)

var dev bool

var RootCmd = &cobra.Command{
	Use:     "loco",
	Short:   "A CLI for managing loco deployments",
	Version: "v0.0.1",
}

func init() {
	RootCmd.PersistentFlags().BoolVar(&dev, "dev", false, "Uses localhost. For development purposes only.")

	RootCmd.AddCommand(initCmd, deployCmd, logsCmd, statusCmd, destroyCmd, testCmd)
}
