package loco

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var (
	dev     bool
	debug   bool
	logFile *os.File
	logPath string
)

var RootCmd = &cobra.Command{
	Use:   "loco",
	Short: "A CLI for managing loco deployments",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		var logger *slog.Logger
		if debug {
			home, err := os.UserHomeDir()
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to get user home directory: %v\n", err)
				os.Exit(1)
			}

			logsDir := filepath.Join(home, ".loco", "logs")
			if err := os.MkdirAll(logsDir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "failed to create logs directory: %v\n", err)
				os.Exit(1)
			}

			timestamp := time.Now().Format("20060102_150405")
			logPath = filepath.Join(logsDir, fmt.Sprintf("loco_%s_%s.log", cmd.Name(), timestamp))

			logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to open log file: %v\n", err)
				os.Exit(1)
			}

			logger = slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}))
			slog.SetDefault(logger)
			slog.Info("Debug logging enabled.")
		} else {
			logger = slog.New(slog.NewTextHandler(io.Discard, nil))
			slog.SetDefault(logger)
		}
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if logFile != nil {
			logFile.Close()
			fmt.Fprintf(os.Stderr, "Debug logs written to: %s\n", logPath)
		}
	},
	Version: "v0.0.1",
}

func init() {
	RootCmd.PersistentFlags().BoolVar(&dev, "dev", false, "Uses localhost. For development purposes only.")
	RootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enables debug logging.")

	RootCmd.AddCommand(initCmd, deployCmd, logsCmd, statusCmd, destroyCmd, testCmd, validateCmd)
}
