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
	host    string
	debug   bool
	logFile *os.File
	logPath string
)

var RootCmd = &cobra.Command{
	Use:   "loco",
	Short: "A CLI for managing loco deployments",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		isDebug, err := cmd.Flags().GetBool("debug")
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to parse debug flag: %v\n", err)
			os.Exit(1)
		}
		if !isDebug {
			return
		}
		if err := initLogger(cmd); err != nil {
			fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
			os.Exit(1)
		}
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if logFile != nil {
			if closeErr := logFile.Close(); closeErr != nil {
				fmt.Fprintf(os.Stderr, "failed to close log file: %v\n", closeErr)
			}
			fmt.Fprintf(os.Stderr, "Debug logs written to: %s\n", logPath)
		}
	},
}

func initLogger(cmd *cobra.Command) error {
	var logger *slog.Logger
	if debug {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}

		logsDir := filepath.Join(home, ".loco", "logs")
		if err = os.MkdirAll(logsDir, 0o755); err != nil {
			return fmt.Errorf("failed to create logs directory: %w", err)
		}

		timestamp := time.Now().Format("20060102_150405")
		logPath = filepath.Join(logsDir, fmt.Sprintf("loco_%s_%s.log", cmd.Name(), timestamp))

		logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
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
	return nil
}

func init() {
	RootCmd.PersistentFlags().StringVar(&host, "host", "", "Set the host URL")
	RootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enables debug logging.")
	RootCmd.AddCommand(initCmd, deployCmd, logsCmd, statusCmd, destroyCmd, loginCmd, validateCmd)
}
