package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const ConfigFileName = "config.toml"

// CLIState represents the CLI's local state and authentication configuration
// gets written and loaded from ~/.loco/config.toml
type CLIState struct {
	CurrentOrg         string `toml:"current_org"`
	CurrentOrgID       int64  `toml:"current_org_id"`
	CurrentWorkspace   string `toml:"current_workspace"`
	CurrentWorkspaceID int64  `toml:"current_workspace_id"`
}

func GetConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	locoDir := filepath.Join(home, ".loco")
	if err := os.MkdirAll(locoDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create .loco directory: %w", err)
	}

	return filepath.Join(locoDir, ConfigFileName), nil
}

func Load() (*CLIState, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	var cfg CLIState
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &CLIState{}, nil
	}

	if _, err := toml.DecodeFile(configPath, &cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	return &cfg, nil
}

func (c *CLIState) Save() error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	f, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(c); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	return nil
}
