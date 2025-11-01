package loco

import (
	"fmt"

	"github.com/nikumar1206/loco/internal/client"
	"github.com/nikumar1206/loco/internal/config"
	"github.com/spf13/cobra"
)

var scaleCmd = &cobra.Command{
	Use:   "scale",
	Short: "Scale an application's resources.",
	Long:  `Scale an application's resources, such as replicas, CPU, or memory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return scaleCmdFunc(cmd)
	},
}

func init() {
	scaleCmd.Flags().StringP("config", "c", "", "path to loco.toml config file")
	scaleCmd.Flags().Int32P("replicas", "r", -1, "The number of replicas to scale to")
	scaleCmd.Flags().String("cpu", "", "The CPU to scale to (e.g. 100m, 0.5)")
	scaleCmd.Flags().String("memory", "", "The memory to scale to (e.g. 128Mi, 1Gi)")
	scaleCmd.Flags().String("host", "", "Set the host URL")
}

func scaleCmdFunc(cmd *cobra.Command) error {
	configPath, err := parseLocoTomlPath(cmd)
	if err != nil {
		return err
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	appName := cfg.LocoConfig.Metadata.Name

	replicas, err := cmd.Flags().GetInt32("replicas")
	if err != nil {
		return err
	}

	cpu, err := cmd.Flags().GetString("cpu")
	if err != nil {
		return err
	}

	memory, err := cmd.Flags().GetString("memory")
	if err != nil {
		return err
	}

	if replicas == -1 && cpu == "" && memory == "" {
		return fmt.Errorf("at least one of --replicas, --cpu, or --memory must be provided")
	}

	if replicas != -1 && replicas < 0 {
		return fmt.Errorf("replicas must be a non-negative integer")
	}

	host, err := getHost(cmd)
	if err != nil {
		return err
	}

	locoToken, err := getLocoToken()
	if err != nil {
		return err
	}

	var replicasPtr *int32
	if replicas != -1 {
		replicasPtr = &replicas
	}

	var cpuPtr *string
	if cpu != "" {
		cpuPtr = &cpu
	}

	var memoryPtr *string
	if memory != "" {
		memoryPtr = &memory
	}

	if err := client.ScaleApp(host, appName, replicasPtr, cpuPtr, memoryPtr, locoToken.Token); err != nil {
		return err
	}

	fmt.Printf("Scaling application %s:\n", appName)
	if replicas != -1 {
		fmt.Printf("  Replicas: %d\n", replicas)
	}
	if cpu != "" {
		fmt.Printf("  CPU: %s\n", cpu)
	}
	if memory != "" {
		fmt.Printf("  Memory: %s\n", memory)
	}

	return nil
}
