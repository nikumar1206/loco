package client

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"k8s.io/apimachinery/pkg/api/resource"
)

type LocoConfig struct {
	Name           string   `toml:"Name"`
	Port           int      `toml:"Port"`
	Subdomain      string   `toml:"Subdomain"`
	DockerfilePath string   `toml:"DockerfilePath"`
	EnvFile        string   `toml:"EnvFile"`
	ProjectPath    string   `toml:"ProjectPath"`
	CPU            string   `toml:"CPU"`
	Memory         string   `toml:"Memory"`
	Replicas       Replicas `toml:"Replicas"`
	Scalers        Scalers  `toml:"Scalers"`
	Health         Health   `toml:"Health"`
	Logs           Logs     `toml:"Logs"`
}

type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Replicas struct {
	Max int `toml:"Max"`
	Min int `toml:"Min"`
}

type Scalers struct {
	CPUTarget    int `toml:"CPUTarget"`
	MemoryTarget int `toml:"MemoryTarget"`
}

type Health struct {
	Interval int    `toml:"Interval"`
	Path     string `toml:"Path"`
	Timeout  int    `toml:"Timeout"`
}

type Logs struct {
	RetentionPeriod string `toml:"RetentionPeriod"`
	Structured      bool   `toml:"Structured"`
}

var Default = LocoConfig{
	Name:           "myapp",
	Port:           8000,
	Subdomain:      "myapp",
	DockerfilePath: "Dockerfile",
	EnvFile:        ".env",
	ProjectPath:    ".",
	CPU:            "100m",
	Memory:         "100Mi",
	Replicas: Replicas{
		Min: 1,
		Max: 1,
	},
	Scalers: Scalers{
		CPUTarget: 70,
	},
	Health: Health{
		Interval: 30,
		Path:     "/health",
		Timeout:  5,
	},
	Logs: Logs{
		Structured:      true,
		RetentionPeriod: "7d",
	},
}

func Create(c LocoConfig) error {
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
	return Create(Default)
}

func Load(path string) (LocoConfig, error) {
	var config LocoConfig
	file, err := os.Open(path)
	if err != nil {
		return config, err
	}
	defer file.Close()

	decoder := toml.NewDecoder(file)
	_, err = decoder.Decode(&config)
	if err != nil {
		return config, err
	}

	return config, nil
}

func (cfg *LocoConfig) FillSensibleDefaults() {
	if cfg.DockerfilePath == "" {
		cfg.DockerfilePath = Default.DockerfilePath
	}

	if cfg.ProjectPath == "" {
		cfg.ProjectPath = Default.ProjectPath
	}

	if cfg.CPU == "" {
		cfg.CPU = Default.CPU
	}

	if cfg.Memory == "" {
		cfg.Memory = Default.Memory
	}
}

func (cfg *LocoConfig) Validate() error {
	if cfg.Name == "" {
		return fmt.Errorf("name must be set")
	}

	if cfg.Port <= 1023 || cfg.Port > 65535 {
		return fmt.Errorf("port must be between 1024 and 65535")
	}

	if cfg.Subdomain == "" {
		return fmt.Errorf("subdomain must be set")
	}

	return validateResources(cfg.CPU, cfg.Memory)
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
