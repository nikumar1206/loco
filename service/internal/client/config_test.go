package client

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestCreateAndLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "loco.toml")

	cfg := LocoConfig{
		Name:           "testapp",
		Port:           8080,
		Subdomain:      "testsub",
		DockerfilePath: "Dockerfile",
		EnvFile:        ".env",
		CPU:            "200m",
		Memory:         "256Mi",
		Replicas:       Replicas{Min: 1, Max: 2},
		Scalers:        Scalers{CPUTarget: 60, MemoryTarget: 70},
		Health:         Health{Interval: 10, Path: "/health", Timeout: 3},
		Logs:           Logs{RetentionPeriod: "3d", Structured: false},
	}

	file, err := os.Create(configPath)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}
	defer file.Close()
	if err := toml.NewEncoder(file).Encode(cfg); err != nil {
		t.Fatalf("failed to encode config: %v", err)
	}
	file.Close()

	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if loaded.Name != cfg.Name || loaded.Port != cfg.Port || loaded.Subdomain != cfg.Subdomain {
		t.Errorf("loaded config does not match original")
	}
}

func TestCreateDefault(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	err := CreateDefault()
	if err != nil {
		t.Fatalf("CreateDefault failed: %v", err)
	}
	_, err = os.Stat("loco.toml")
	if err != nil {
		t.Fatalf("loco.toml not created: %v", err)
	}
}

func TestFillSensibleDefaults(t *testing.T) {
	cfg := LocoConfig{}
	cfg.FillSensibleDefaults()
	if cfg.DockerfilePath != Default.DockerfilePath {
		t.Errorf("DockerfilePath not set to default")
	}

	if cfg.CPU != Default.CPU {
		t.Errorf("CPU not set to default")
	}
	if cfg.Memory != Default.Memory {
		t.Errorf("Memory not set to default")
	}
}

func TestValidate(t *testing.T) {
	valid := Default
	valid.Name = "valid"
	valid.Port = 8080
	valid.Subdomain = "sub"
	if err := valid.Validate(); err != nil {
		t.Errorf("expected valid config, got error: %v", err)
	}

	invalid := valid
	invalid.Name = ""
	if err := invalid.Validate(); err == nil {
		t.Errorf("expected error for empty name")
	}

	invalid = valid
	invalid.Port = 80
	if err := invalid.Validate(); err == nil {
		t.Errorf("expected error for invalid port")
	}

	invalid = valid
	invalid.Subdomain = ""
	if err := invalid.Validate(); err == nil {
		t.Errorf("expected error for empty subdomain")
	}

	invalid = valid
	invalid.CPU = "1000m"
	if err := invalid.Validate(); err == nil {
		t.Errorf("expected error for CPU > 500m")
	}

	invalid = valid
	invalid.Memory = "2Gi"
	if err := invalid.Validate(); err == nil {
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
