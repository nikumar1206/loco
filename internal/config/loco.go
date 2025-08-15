package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Name           string   `toml:"Name"`
	Port           int      `toml:"Port"`
	Subdomain      string   `toml:"Subdomain"`
	DockerfilePath string   `toml:"DockerfilePath"`
	EnvFile        string   `toml:"EnvFile"`
	CPU            string   `toml:"CPU"`
	Memory         string   `toml:"Memory"`
	Replicas       Replicas `toml:"Replicas"`
	Scalers        Scalers  `toml:"Scalers"`
	Health         Health   `toml:"Health"`
	Logs           Logs     `toml:"Logs"`
	projectPath    string
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
	if _, err := os.Stat("loco.toml"); !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("file already exists")
	}
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

func Load(cfgPath string) (Config, error) {
	var config Config

	cfgPathAbs, err := filepath.Abs(cfgPath)
	if err != nil {
		return config, err
	}
	config.projectPath = filepath.Dir(cfgPathAbs)

	file, err := os.Open(cfgPath)
	if err != nil {
		return config, err
	}
	defer file.Close()

	decoder := toml.NewDecoder(file)
	_, err = decoder.Decode(&config)
	if err != nil {
		return config, err
	}

	config.DockerfilePath = resolvePath(config.DockerfilePath, cfgPathAbs)
	config.EnvFile = resolvePath(config.EnvFile, cfgPathAbs)

	return config, nil
}

func (cfg *Config) FillSensibleDefaults() {
	if cfg.DockerfilePath == "" {
		cfg.DockerfilePath = Default.DockerfilePath
	}

	if cfg.CPU == "" {
		cfg.CPU = Default.CPU
	}

	if cfg.Memory == "" {
		cfg.Memory = Default.Memory
	}
}

func resolvePath(path, baseDir string) string {
	if path == "" {
		return ""
	}

	if filepath.IsAbs(path) {
		return path
	}

	projectFolder := filepath.Dir(baseDir)

	return filepath.Join(projectFolder, path)
}

// Validate ensures the locoConfig is accurate.
// it also validates and resolves paths to env and Dockerfile
func (cfg *Config) Validate() error {
	if cfg.Name == "" {
		return fmt.Errorf("name must be set")
	}

	if cfg.Port <= 1023 || cfg.Port > 65535 {
		return fmt.Errorf("port must be between 1024 and 65535")
	}

	if cfg.Subdomain == "" {
		return fmt.Errorf("subdomain must be set")
	}

	if !fileExists(cfg.DockerfilePath) {
		return fmt.Errorf("provided Dockerfile path could not be resolved. Please provide path to a valid Dockerfile")
	}

	if cfg.EnvFile != "" && !fileExists(cfg.EnvFile) {
		return fmt.Errorf("provided env path could not be resolved. Please provide path to a valid environments file")
	}
	return nil
}

func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	if err != nil {
		return !errors.Is(err, os.ErrNotExist)
	}
	return true
}

func (cfg Config) GetProjectPath() string {
	return cfg.projectPath
}
