package cmd

import (
	"github.com/nikumar1206/loco/cmd/internal/color"
	"github.com/spf13/cobra"
)

var (
	LOCO__OK_PREFIX    = color.Colorize("LOCO: ", color.FgGreen)
	LOCO__ERROR_PREFIX = color.Colorize("LOCO: ", color.FgRed)
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
