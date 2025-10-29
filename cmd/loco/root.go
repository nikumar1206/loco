package loco

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	host    string
	logPath string
)

var RootCmd = &cobra.Command{
	Use:   "loco",
	Short: "A CLI for managing loco deployments",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if err := initLogger(cmd); err != nil {
			fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
			os.Exit(1)
		}
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(os.Stderr, "Logs written to: %s\n", logPath)
	},
}

func initLogger(cmd *cobra.Command) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	logsDir := filepath.Join(home, ".loco", "logs")
	logPath = filepath.Join(logsDir, "loco.log")

	output := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    10, // megabytes
		MaxBackups: 3,
		MaxAge:     30, // days
		Compress:   false,
	}

	logger := slog.New(slog.NewTextHandler(output, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)
	initLog := fmt.Sprintf("new loco run; cmd name: %s", cmd.Use)
	slog.Info(initLog)
	return nil
}

func init() {
	RootCmd.PersistentFlags().StringVar(&host, "host", "", "Set the host URL")
	RootCmd.AddCommand(initCmd, deployCmd, logsCmd, statusCmd, destroyCmd, loginCmd, validateCmd)
}
