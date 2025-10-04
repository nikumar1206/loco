package main

import (
	"fmt"
	"log"
	"log/slog"
	"os/user"
	"time"

	"github.com/nikumar1206/loco/internal/keychain"
	"github.com/spf13/cobra"
)

func parseDevFlag(cmd *cobra.Command) string {
	isDev, err := cmd.Flags().GetBool("dev")
	if err != nil {
		log.Fatalf("Error getting dev flag: %v", err)
	}

	if isDev {
		slog.Debug("using dev host")
		return "http://localhost:8000"
	}
	return "https://loco.deploy-app.com"
}

func getLocoToken() (*keychain.UserToken, error) {
	usr, err := user.Current()
	if err != nil {
		slog.Debug("failed to get current user", "error", err)
		return nil, err
	}
	locoToken, err := keychain.GetGithubToken(usr.Name)
	if err != nil {
		slog.Debug("failed to get github token", "error", err)
		return nil, err
	}

	if locoToken.ExpiresAt.Before(time.Now().Add(5 * time.Minute)) {
		slog.Debug("token is expired or will expire soon", "expires_at", locoToken.ExpiresAt)
		return nil, fmt.Errorf("token is expired or will expire soon. Please re-login via `loco login`")
	}

	return locoToken, err
}

func parseAndSetDebugFlag(cmd *cobra.Command) {
	isDebug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		log.Fatalf("Error getting debug flag: %v", err)
	}
	if isDebug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}
}
