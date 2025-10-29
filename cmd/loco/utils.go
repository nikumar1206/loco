package loco

import (
	"fmt"
	"log/slog"
	"os"
	"os/user"
	"time"

	"github.com/nikumar1206/loco/internal/keychain"
	"github.com/spf13/cobra"
)

const locoProdHost = "https://loco.deploy-app.com"

func getHost(cmd *cobra.Command) (string, error) {
	host, err := cmd.Flags().GetString("host")
	if err != nil {
		return "", fmt.Errorf("error reading host flag: %w", err)
	}
	if host != "" {
		slog.Debug("using host from flag")
		return host, nil
	}

	host = os.Getenv("LOCO__HOST")
	if host != "" {
		slog.Debug("using host from environment variable")
		return host, nil
	}

	slog.Debug("defaulting to prod url")
	return locoProdHost, nil
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

func parseLocoTomlPath(cmd *cobra.Command) (string, error) {
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return "", fmt.Errorf("error reading config flag: %w", err)
	}
	if configPath == "" {
		return "loco.toml", nil
	}
	return configPath, nil
}

func parseImageId(cmd *cobra.Command) (string, error) {
	imageId, err := cmd.Flags().GetString("image")
	if err != nil {
		return "", fmt.Errorf("error reading image flag: %w", err)
	}
	return imageId, nil
}
