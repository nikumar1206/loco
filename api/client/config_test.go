package client

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/nikumar1206/loco/internal/config"
	appv1 "github.com/nikumar1206/loco/proto/app/v1"
)

func TestCreateAndLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "loco.toml")

	cfg := appv1.LocoConfig{
		Metadata: &appv1.Metadata{
			Name:          "testapp",
			ConfigVersion: "0.1",
			Description:   "Test application",
		},
		Resources: &appv1.Resources{
			Cpu:      "250m",
			Memory:   "512Mi",
			Replicas: &appv1.Replicas{Min: 1, Max: 2},
			Scalers:  &appv1.Scalers{CpuTarget: 60, MemoryTarget: 70},
		},
		Build: &appv1.Build{
			DockerfilePath: "Dockerfile",
			Type:           "docker",
		},
		Routing: &appv1.Routing{
			Port:      8080,
			Subdomain: "testsub",
		},
		Env: &appv1.Env{
			File: ".env",
		},
		Health: &appv1.Health{
			Interval: 10,
			Timeout:  3,
			Path:     "/health",
		},
		Obs: &appv1.Obs{
			Logging: &appv1.Logging{RetentionPeriod: "7d", Structured: true},
		},
	}

	file, err := os.Create(configPath)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}
	defer file.Close()
	if err := toml.NewEncoder(file).Encode(&cfg); err != nil {
		t.Fatalf("failed to encode config: %v", err)
	}
	file.Close()

	loaded, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.LocoConfig.Metadata.Name != cfg.Metadata.Name ||
		loaded.LocoConfig.Routing.Port != cfg.Routing.Port ||
		loaded.LocoConfig.Routing.Subdomain != cfg.Routing.Subdomain {
		t.Errorf("loaded config does not match original")
	}
}

func TestCreateDefault(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	err := config.CreateDefault("testapp")
	if err != nil {
		t.Fatalf("CreateDefault failed: %v", err)
	}
	_, err = os.Stat("loco.toml")
	if err != nil {
		t.Fatalf("loco.toml not created: %v", err)
	}
}

func TestFillSensibleDefaults(t *testing.T) {
	cfg := &appv1.LocoConfig{}
	config.FillSensibleDefaults(cfg)
	if cfg.Build.DockerfilePath != config.Default.Build.DockerfilePath {
		t.Errorf("DockerfilePath not set to default")
	}

	if cfg.Resources.Cpu != config.Default.Resources.Cpu {
		t.Errorf("CPU not set to default")
	}
	if cfg.Resources.Memory != config.Default.Resources.Memory {
		t.Errorf("Memory not set to default")
	}
}

func TestValidate(t *testing.T) {
	valid := config.Default
	valid.Metadata.Name = "valid"
	valid.Routing.Port = 8080
	valid.Routing.Subdomain = "sub"
	if err := config.Validate(valid); err != nil {
		t.Errorf("expected valid config, got error: %v", err)
	}

	invalid := valid
	invalid.Metadata.Name = ""
	if err := config.Validate(invalid); err == nil {
		t.Errorf("expected error for empty name")
	}

	invalid = valid
	invalid.Routing.Port = 8080
	if err := config.Validate(invalid); err == nil {
		t.Errorf("expected error for invalid port")
	}

	invalid = valid
	invalid.Routing.Subdomain = ""
	if err := config.Validate(invalid); err == nil {
		t.Errorf("expected error for empty subdomain")
	}

	invalid = valid
	invalid.Resources.Cpu = "1000m"
	if err := config.Validate(invalid); err == nil {
		t.Errorf("expected error for CPU > 500m")
	}

	invalid = valid
	invalid.Resources.Memory = "2Gi"
	if err := config.Validate(invalid); err == nil {
		t.Errorf("expected error for Memory > 1Gi")
	}
}

func TestValidateResources(t *testing.T) {
	err := validateResources("250m", "512Mi")
	if err != nil {
		t.Errorf("expected valid resources, got error: %v", err)
	}

	err = validateResources("abc", "512Mi")
	if err == nil {
		t.Errorf("expected error for invalid CPU")
	}

	err = validateResources("250m", "xyz")
	if err == nil {
		t.Errorf("expected error for invalid Memory")
	}

	err = validateResources("600m", "512Mi")
	if err == nil {
		t.Errorf("expected error for CPU > 500m")
	}

	err = validateResources("250m", "2Gi")
	if err == nil {
		t.Errorf("expected error for Memory > 1Gi")
	}
}
