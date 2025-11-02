package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	appv1 "github.com/nikumar1206/loco/shared/proto/app/v1"
)

// this file needs to be cleaned up
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
	LabelAppName      = "app.loco.io/name"
	LabelAppInstance  = "app.loco.io/instance"
	LabelAppVersion   = "app.loco.io/version"
	LabelAppComponent = "app.loco.io/component"

	// todo: can we use this partof label for grouping apps into projects?
	LabelAppPartOf     = "app.loco.io/part-of"
	LabelAppManagedBy  = "app.loco.io/managed-by"
	LabelAppCreatedFor = "app.loco.io/created-for"
	LabelAppCreatedAt  = "app.loco.io/created-at"
	LabelAppCreatedBy  = "app.loco.io/created-by"
)

var Default = &appv1.LocoConfig{
	Metadata: &appv1.Metadata{
		ConfigVersion: "0.1",
		Description:   "Default Loco app configuration",
		Name:          "<ENTER_APP_NAME>",
	},
	Resources: &appv1.Resources{
		Cpu:    "100m",
		Memory: "256Mi",
		Replicas: &appv1.Replicas{
			Min: 1,
			Max: 1,
		},
		Scalers: &appv1.Scalers{
			Enabled:   true,
			CpuTarget: 70,
		},
	},
	Build: &appv1.Build{
		DockerfilePath: "Dockerfile",
		Type:           "docker",
	},
	Routing: &appv1.Routing{
		IdleTimeout: 60, // 1 minute
		PathPrefix:  "/",
		Port:        8000,
	},
	Health: &appv1.HealthCheck{
		Interval:           30,
		Path:               "/health",
		StartupGracePeriod: 0,
		Timeout:            5,
		FailThreshold:      3,
	},
	Obs: &appv1.Obs{
		Logging: &appv1.Logging{
			Enabled:         true,
			RetentionPeriod: "7d",
			Structured:      false,
		},
		Metrics: &appv1.Metrics{
			Enabled: false,
			Path:    "/metrics",
			Port:    9090,
		},
		Tracing: &appv1.Tracing{
			Enabled:    false,
			SampleRate: 0.1,
			Tags:       map[string]string{},
		},
	},
}

func FillSensibleDefaults(cfg *appv1.LocoConfig) {
	if cfg.Build.DockerfilePath == "" {
		cfg.Build.DockerfilePath = Default.Build.DockerfilePath
	}

	if cfg.Resources.Cpu == "" {
		cfg.Resources.Cpu = Default.Resources.Cpu
	}

	if cfg.Resources.Memory == "" {
		cfg.Resources.Memory = Default.Resources.Memory
	}
}

// Validate ensures the locoConfig is accurate.
// it also validates and resolves paths to env and Dockerfile
func Validate(cfg *appv1.LocoConfig) error {
	// --- Metadata ---
	if cfg.GetMetadata() == nil {
		return fmt.Errorf("metadata section is required")
	}

	if cfg.Metadata.Name == "" {
		return fmt.Errorf("metadata.name must be set")
	}

	if cfg.GetRouting() == nil {
		return fmt.Errorf("routing section is required")
	}

	if cfg.Routing.Subdomain == "" {
		return fmt.Errorf("routing.subdomain must be set")
	}

	// --- Routing ---
	if cfg.Routing.Port <= 1023 || cfg.Routing.Port > 65535 {
		return fmt.Errorf("routing.port must be between 1024 and 65535")
	}

	if cfg.Routing.PathPrefix == "" {
		cfg.Routing.PathPrefix = "/"
	} else if !strings.HasPrefix(cfg.Routing.PathPrefix, "/") {
		return fmt.Errorf("routing.pathprefix must start with '/'")
	}

	if cfg.Routing.IdleTimeout < 0 {
		return fmt.Errorf("routing.idletimeout cannot be negative")
	}

	// --- Build ---
	if cfg.Build.DockerfilePath == "" {
		cfg.Build.DockerfilePath = "Dockerfile"
	}

	// todo: re-enable this check, but its not needed if imageid is passed
	// if !fileExists(cfg.Build.DockerfilePath) {
	// 	return fmt.Errorf("provided Dockerfile path %q could not be resolved", cfg.Build.DockerfilePath)
	// }
	if cfg.Build.Type == "" {
		cfg.Build.Type = "docker"
	}

	// --- Env ---
	// Note: file validation removed as API doesn't need it

	// --- Resources ---
	if cfg.GetResources() == nil {
		return fmt.Errorf("resources section is required")
	}

	if cfg.Resources.Cpu == "" {
		return fmt.Errorf("resources.cpu must be set (e.g. '100m')")
	}
	if cfg.Resources.Memory == "" {
		return fmt.Errorf("resources.memory must be set (e.g. '512Mi')")
	}

	// Replicas
	if cfg.Resources.Replicas.Min <= 0 {
		return fmt.Errorf("resources.replicas.min must be greater than 0")
	}
	if cfg.Resources.Replicas.Max <= 0 {
		return fmt.Errorf("resources.replicas.max must be greater than 0")
	}
	if cfg.Resources.Replicas.Max < cfg.Resources.Replicas.Min {
		return fmt.Errorf("resources.replicas.max must be greater than or equal to min")
	}
	if cfg.Resources.Replicas.Max > 50 {
		return fmt.Errorf("resources.replicas.max cannot exceed 50 replicas")
	}

	// Scalers
	if cfg.Resources.Scalers.Enabled {
		if cfg.Resources.Scalers.CpuTarget == 0 && cfg.Resources.Scalers.MemoryTarget == 0 {
			return fmt.Errorf("when scalers.enabled=true, either cpu_target or memory_target must be provided")
		}
		if cfg.Resources.Scalers.CpuTarget != 0 && cfg.Resources.Scalers.MemoryTarget != 0 {
			return fmt.Errorf("only one of scalers.cpu_target or scalers.memory_target should be provided")
		}
		if cfg.Resources.Scalers.CpuTarget < 0 || cfg.Resources.Scalers.CpuTarget > 100 {
			return fmt.Errorf("scalers.cpu_target must be between 1 and 100")
		}
		if cfg.Resources.Scalers.MemoryTarget < 0 || cfg.Resources.Scalers.MemoryTarget > 100 {
			return fmt.Errorf("scalers.memory_target must be between 1 and 100")
		}
	}

	// --- Health ---
	if cfg.Health.Interval <= 0 {
		return fmt.Errorf("health.interval must be greater than 0")
	}
	if cfg.Health.Path == "" {
		return fmt.Errorf("health.path must be provided")
	}
	if !strings.HasPrefix(cfg.Health.Path, "/") {
		return fmt.Errorf("health.path must start with '/'")
	}
	if cfg.Health.Timeout <= 0 {
		return fmt.Errorf("health.timeout must be greater than 0")
	}
	if cfg.Health.StartupGracePeriod < 0 {
		return fmt.Errorf("health.startupgraceperiod cannot be negative")
	}

	// --- Observability ---
	// Logging
	if cfg.Obs.Logging.Enabled {
		if cfg.Obs.Logging.RetentionPeriod == "" {
			cfg.Obs.Logging.RetentionPeriod = "7d"
		}
		duration, err := parseRetention(cfg.Obs.Logging.RetentionPeriod)
		if err != nil || duration <= 0 {
			return fmt.Errorf("invalid logging.retentionperiod: %q", cfg.Obs.Logging.RetentionPeriod)
		}
	}

	// Metrics
	if cfg.Obs.Metrics.Enabled {
		if cfg.Obs.Metrics.Path == "" {
			cfg.Obs.Metrics.Path = "/metrics"
		}
		if !strings.HasPrefix(cfg.Obs.Metrics.Path, "/") {
			return fmt.Errorf("metrics.path must start with '/'")
		}
		if cfg.Obs.Metrics.Port <= 0 {
			cfg.Obs.Metrics.Port = 9090
		}
		if cfg.Obs.Metrics.Port <= 1023 || cfg.Obs.Metrics.Port > 65535 {
			return fmt.Errorf("metrics.port must be between 1024 and 65535")
		}
	}

	// Tracing
	if cfg.Obs.Tracing.Enabled {
		if cfg.Obs.Tracing.SampleRate < 0 || cfg.Obs.Tracing.SampleRate > 1 {
			return fmt.Errorf("tracing.samplerate must be between 0.0 and 1.0")
		}
	}

	return nil
}

// --- Helper utilities ---

func parseRetention(value string) (time.Duration, error) {
	// supports formats like "7d", "24h"
	if strings.HasSuffix(value, "d") {
		daysStr := strings.TrimSuffix(value, "d")
		days, err := strconv.Atoi(daysStr)
		if err != nil {
			return 0, err
		}
		return time.Hour * 24 * time.Duration(days), nil
	}
	return time.ParseDuration(value)
}

type LocoApp struct {
	EnvVars        []*appv1.EnvVar
	CreatedAt      time.Time
	Name           string
	Namespace      string
	CreatedBy      string
	ContainerImage string
	Subdomain      string
	Labels         map[string]string
	Config         *appv1.LocoConfig
}

func NewLocoApp(config *appv1.LocoConfig, createdBy string, containerImage string, envVars []*appv1.EnvVar) *LocoApp {
	ns := GenerateNameSpace(config.Metadata.Name, createdBy)
	labels := generateLabels(config.Metadata.Name, ns, createdBy)
	return &LocoApp{
		Name:           config.Metadata.Name,
		Namespace:      ns,
		Subdomain:      config.Routing.Subdomain,
		CreatedBy:      createdBy,
		CreatedAt:      time.Now(),
		Labels:         labels,
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

func generateLabels(name, namespace, createdBy string) map[string]string {
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

type Config struct {
	LocoConfig  *appv1.LocoConfig
	ProjectPath string
}

func Create(c *appv1.LocoConfig, dirName string) error {
	file, err := os.Create("loco.toml")
	if err != nil {
		return err
	}
	defer file.Close()

	c.Metadata.Name = dirName
	c.Routing.Subdomain = dirName

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(c); err != nil {
		return err
	}
	return nil
}

func CreateDefault(dirName string) error {
	return Create(Default, dirName)
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

	if config.LocoConfig.GetBuild() == nil {
		config.LocoConfig.Build = &appv1.Build{
			DockerfilePath: "Dockerfile",
			Type:           "docker",
		}
	}

	if config.LocoConfig.Env == nil {
		config.LocoConfig.Env = &appv1.Env{}
	}

	config.LocoConfig.Build.DockerfilePath = resolvePath(config.LocoConfig.Build.DockerfilePath, cfgPathAbs)
	config.LocoConfig.Env.File = resolvePath(config.LocoConfig.Env.File, cfgPathAbs)

	return config, nil
}
