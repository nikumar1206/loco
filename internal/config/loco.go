package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
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

var Default = Config{
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

func Create(c Config) error {
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

func Load(path string) (Config, error) {
	var config Config
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

func FillSensibleDefaults(cfg Config) Config {
	if cfg.DockerfilePath == "" {
		cfg.DockerfilePath = Default.DockerfilePath
	}

	if cfg.EnvFile == "" {
		cfg.EnvFile = Default.EnvFile
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

	return cfg
}

func ErrorOnBadConfig(cfg Config) error {
	if cfg.Name == "" {
		return fmt.Errorf("name must be set")
	}

	if cfg.Port <= 1023 || cfg.Port > 65535 {
		return fmt.Errorf("port must be between 1024 and 65535")
	}

	if cfg.Subdomain == "" {
		return fmt.Errorf("subdomain must be set")
	}

	return nil
}
