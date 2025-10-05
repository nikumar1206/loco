package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	appv1 "github.com/nikumar1206/loco/proto/app/v1"
)

var ALLOWED_SCHEMA_VERSIONS = []string{
	"0.1",
}

var BannedSubdomains = []string{
	"api", "admin", "dashboard", "console",
	"login", "auth", "user", "users", "support", "help", "loco", "monitoring",
	"metrics", "stats", "status", "health", "system", "service", "services",
	"config", "configuration", "settings", "setup", "install", "uninstall",
}

const (
	LabelAppName       = "app.loco.io/name"
	LabelAppInstance   = "app.loco.io/instance"
	LabelAppVersion    = "app.loco.io/version"
	LabelAppComponent  = "app.loco.io/component"
	LabelAppPartOf     = "app.loco.io/part-of"
	LabelAppManagedBy  = "app.loco.io/managed-by"
	LabelAppCreatedFor = "app.loco.io/created-for"
	LabelAppCreatedAt  = "app.loco.io/created-at"
	LabelAppCreatedBy  = "app.loco.io/created-by"
)

// todo: this file needs cleanup and better structuring.
type Config struct {
	LocoConfig  *appv1.LocoConfig
	ProjectPath string
}

var Default = &appv1.LocoConfig{
	Name:           "myapp",
	Port:           8000,
	Subdomain:      "myapp",
	DockerfilePath: "Dockerfile",
	EnvFile:        ".env",
	Cpu:            "100m",
	Memory:         "100Mi",
	Replicas: &appv1.Replicas{
		Min: 1,
		Max: 1,
	},
	Scalers: &appv1.Scalers{
		CpuTarget: 70,
	},
	Health: &appv1.Health{
		Interval: 30,
		Path:     "/health",
		Timeout:  5,
	},
	Logs: &appv1.Logs{
		Structured:      true,
		RetentionPeriod: "7d",
	},
}

func Create(c *appv1.LocoConfig) error {
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
	config.ProjectPath = filepath.Dir(cfgPathAbs)

	file, err := os.Open(cfgPath)
	if err != nil {
		return config, err
	}
	defer file.Close()

	decoder := toml.NewDecoder(file)
	_, err = decoder.Decode(&config.LocoConfig)
	if err != nil {
		return config, err
	}

	config.LocoConfig.DockerfilePath = resolvePath(config.LocoConfig.DockerfilePath, cfgPathAbs)
	config.LocoConfig.EnvFile = resolvePath(config.LocoConfig.EnvFile, cfgPathAbs)

	return config, nil
}

func FillSensibleDefaults(cfg *appv1.LocoConfig) {
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
func Validate(cfg *appv1.LocoConfig) error {
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

	if cfg.Scalers.CpuTarget != 0 && cfg.Scalers.MemoryTarget != 0 {
		return fmt.Errorf("only one scaler config should be provided")
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

type LocoApp struct {
	Name           string
	Namespace      string
	CreatedBy      string
	CreatedAt      time.Time
	Labels         map[string]string
	ContainerImage string
	Subdomain      string
	EnvVars        []*appv1.EnvVar
	Config         *appv1.LocoConfig
}

func NewLocoApp(name, subdomain, createdBy string, containerImage string, envVars []*appv1.EnvVar, config *appv1.LocoConfig) *LocoApp {
	ns := GenerateNameSpace(name, createdBy)
	return &LocoApp{
		Name:           name,
		Namespace:      ns,
		Subdomain:      subdomain,
		CreatedBy:      createdBy,
		CreatedAt:      time.Now(),
		Labels:         GenerateLabels(name, ns, createdBy),
		EnvVars:        envVars,
		Config:         config,
		ContainerImage: containerImage,
	}
}

func IsBannedSubDomain(subdomain string) bool {
	return slices.Contains(BannedSubdomains, subdomain) || strings.Contains(subdomain, "loco")
}

func GenerateNameSpace(name string, username string) string {
	appName := strings.ToLower(strings.TrimSpace(name))
	userName := strings.ToLower(strings.TrimSpace(username))

	return appName + "-" + userName
}

func GenerateLabels(name, namespace, createdBy string) map[string]string {
	return map[string]string{
		LabelAppName:       name,
		LabelAppInstance:   namespace,
		LabelAppVersion:    "1.0.0",
		LabelAppComponent:  "backend",
		LabelAppPartOf:     "loco-platform",
		LabelAppManagedBy:  "loco",
		LabelAppCreatedFor: createdBy,
		LabelAppCreatedAt:  time.Now().UTC().Format("20060102T150405Z"),
	}
}
