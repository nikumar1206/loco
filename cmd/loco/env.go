package loco

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/nikumar1206/loco/internal/client"
	"github.com/nikumar1206/loco/shared/config"
	appv1 "github.com/nikumar1206/loco/shared/proto/app/v1"
	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Sync environment variables for an application.",
	Long:  `Sync environment variables for an application without redeploying.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return envCmdFunc(cmd)
	},
}

func init() {
	envCmd.Flags().StringP("config", "c", "", "path to loco.toml config file")
	envCmd.Flags().String("env-file", "", "path to .env file (optional, overrides config)")
	envCmd.Flags().StringSlice("set", []string{}, "set environment variables (e.g. --set KEY1=VALUE1 --set KEY2=VALUE2)")
	envCmd.Flags().Bool("restart", false, "restart the deployment after updating env vars")
	envCmd.Flags().String("host", "", "Set the host URL")
}

func envCmdFunc(cmd *cobra.Command) error {
	configPath, err := parseLocoTomlPath(cmd)
	if err != nil {
		return err
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	appName := cfg.LocoConfig.Metadata.Name

	envVars := map[string]string{}

	// First, load from config's env file if exists, else check for .env
	envFilePath := cfg.LocoConfig.Env.File
	if envFilePath == "" {
		if _, err := os.Stat(".env"); err == nil {
			envFilePath = ".env"
		}
	}

	if envFilePath != "" {
		if _, err := os.Stat(envFilePath); err == nil {
			f, err := os.Open(envFilePath)
			if err != nil {
				return fmt.Errorf("failed to open env file %s: %w", envFilePath, err)
			}
			defer f.Close()
			parsed, err := godotenv.Parse(f)
			if err != nil {
				return fmt.Errorf("failed to parse env file %s: %w", envFilePath, err)
			}
			for k, v := range parsed {
				envVars[k] = v
			}
		}
	}

	// Also load from config's Variables
	for _, envVar := range cfg.LocoConfig.Env.Variables {
		envVars[envVar.Name] = envVar.Value
	}

	// Override with --env-file if provided
	envFile, err := cmd.Flags().GetString("env-file")
	if err != nil {
		return err
	}
	if envFile != "" {
		f, err := os.Open(envFile)
		if err != nil {
			return fmt.Errorf("failed to open specified env file: %w", err)
		}
		defer f.Close()
		parsed, err := godotenv.Parse(f)
		if err != nil {
			return fmt.Errorf("failed to parse specified env file: %w", err)
		}
		for k, v := range parsed {
			envVars[k] = v
		}
	}

	// Override with --set values
	setVars, err := cmd.Flags().GetStringSlice("set")
	if err != nil {
		return err
	}
	for _, setVar := range setVars {
		parts := strings.SplitN(setVar, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid --set format: %s, expected KEY=VALUE", setVar)
		}
		envVars[parts[0]] = parts[1]
	}

	if len(envVars) == 0 {
		return fmt.Errorf("no environment variables to sync")
	}

	host, err := getHost(cmd)
	if err != nil {
		return err
	}

	locoToken, err := getLocoToken()
	if err != nil {
		return err
	}

	restart, err := cmd.Flags().GetBool("restart")
	if err != nil {
		return err
	}

	envVarList := []*appv1.EnvVar{}
	for k, v := range envVars {
		envVarList = append(envVarList, &appv1.EnvVar{Name: k, Value: v})
	}

	if err := client.UpdateEnvVars(host, appName, envVarList, restart, locoToken.Token); err != nil {
		return err
	}

	fmt.Printf("Environment variables synced for application %s\n", appName)
	return nil
}
