package client

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	appv1 "github.com/nikumar1206/loco/proto/app/v1"
)

func TestCreateAndLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "loco.toml")

	cfg := appv1.LocoConfig{
		Name:           "testapp",
		Port:           8080,
		Subdomain:      "testsub",
		DockerfilePath: "Dockerfile",
		EnvFile:        ".env",
		Cpu:            "200m",
		Memory:         "256Mi",
		Replicas:       &appv1.Replicas{Min: 1, Max: 2},
		Scalers:        &appv1.Scalers{CpuTarget: 60, MemoryTarget: 70},
		Health:         &appv1.Health{Interval: 10, Path: "/health", Timeout: 3},
		Logs:           &appv1.Logs{RetentionPeriod: "3d", Structured: false},
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
	cfg := &appv1.LocoConfig{}
	FillSensibleDefaults(cfg)
	if cfg.DockerfilePath != Default.DockerfilePath {
		t.Errorf("DockerfilePath not set to default")
	}

	if cfg.Cpu != Default.Cpu {
		t.Errorf("CPU not set to default")
	}
	if cfg.Memory != Default.Memory {
		t.Errorf("Memory not set to default")
	}
}

func TestValidate(t *testing.T) {
	valid := &Default
	valid.Name = "valid"
	valid.Port = 8080
	valid.Subdomain = "sub"
	if err := Validate(valid); err != nil {
		t.Errorf("expected valid config, got error: %v", err)
	}

	invalid := valid
	invalid.Name = ""
	if err := Validate(invalid); err == nil {
		t.Errorf("expected error for empty name")
	}

	invalid = valid
	invalid.Port = 80
	if err := Validate(invalid); err == nil {
		t.Errorf("expected error for invalid port")
	}

	invalid = valid
	invalid.Subdomain = ""
	if err := Validate(invalid); err == nil {
		t.Errorf("expected error for empty subdomain")
	}

	invalid = valid
	invalid.Cpu = "1000m"
	if err := Validate(invalid); err == nil {
		t.Errorf("expected error for CPU > 500m")
	}

	invalid = valid
	invalid.Memory = "2Gi"
	if err := Validate(invalid); err == nil {
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
