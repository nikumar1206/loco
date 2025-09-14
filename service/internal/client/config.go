package client

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	app "github.com/nikumar1206/loco/proto/app/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var Default = app.LocoConfig{
	Name:           "myapp",
	Port:           8000,
	Subdomain:      "myapp",
	DockerfilePath: "Dockerfile",
	EnvFile:        ".env",
	Cpu:            "100m",
	Memory:         "100Mi",
	Replicas: &app.Replicas{
		Min: 1,
		Max: 1,
	},
	Scalers: &app.Scalers{
		CpuTarget: 70,
	},
	Health: &app.Health{
		Interval: 30,
		Path:     "/health",
		Timeout:  5,
	},
	Logs: &app.Logs{
		Structured:      true,
		RetentionPeriod: "7d",
	},
}

func Create(c *app.LocoConfig) error {
	file, err := os.Create("loco.toml")
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(c); err != nil {
		return err
	}
	return nil
}

func CreateDefault() error {
	return Create(&Default)
}

func Load(path string) (*app.LocoConfig, error) {
	var config app.LocoConfig
	file, err := os.Open(path)
	if err != nil {
		return &config, err
	}
	defer file.Close()

	decoder := toml.NewDecoder(file)
	_, err = decoder.Decode(&config)
	if err != nil {
		return &config, err
	}

	return &config, nil
}

func FillSensibleDefaults(cfg *app.LocoConfig) {
	if cfg.DockerfilePath == "" {
		cfg.DockerfilePath = Default.DockerfilePath
	}

	if cfg.Cpu == "" {
		cfg.Cpu = Default.Cpu
	}

	if cfg.Memory == "" {
		cfg.Memory = Default.Memory
	}
}

func Validate(cfg *app.LocoConfig) error {
	if cfg.Name == "" {
		return fmt.Errorf("name must be set")
	}

	if cfg.Port <= 1023 || cfg.Port > 65535 {
		return fmt.Errorf("port must be between 1024 and 65535")
	}

	if cfg.Subdomain == "" {
		return fmt.Errorf("subdomain must be set")
	}

	return validateResources(cfg.Cpu, cfg.Memory)
}

func validateResources(cpuStr, memStr string) error {
	cpuQty, err := resource.ParseQuantity(cpuStr)
	if err != nil {
		return fmt.Errorf("invalid CPU quantity: %w", err)
	}

	memQty, err := resource.ParseQuantity(memStr)
	if err != nil {
		return fmt.Errorf("invalid memory quantity: %w", err)
	}

	maxCPU := resource.MustParse("500m")
	maxMem := resource.MustParse("1Gi")

	if cpuQty.Cmp(maxCPU) == 1 {
		return fmt.Errorf("CPU exceeds 500m: got %s", cpuStr)
	}

	if memQty.Cmp(maxMem) == 1 {
		return fmt.Errorf("memory exceeds 1Gi: got %s", memStr)
	}

	return nil
}
