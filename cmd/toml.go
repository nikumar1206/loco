package main

import (
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Name           string
	Port           int
	CPU            string
	Memory         string
	Replicas       Replicas
	Scalers        Scalers
	Logs           Logs
	DockerfilePath string
}

type Replicas struct {
	Min int
	Max int
}

type Scalers struct {
	CPUTarget    int
	MemoryTarget int
}

type Logs struct {
	Structured      bool
	RetentionPeriod string
}

func createConfig() error {
	config := Config{
		Name:   "myapp",
		Port:   8000,
		CPU:    "0.25",
		Memory: "512Mi",
		Replicas: Replicas{
			Min: 1,
			Max: 1,
		},
		Scalers: Scalers{
			CPUTarget:    80,
			MemoryTarget: 50,
		},
		Logs: Logs{
			Structured:      true,
			RetentionPeriod: "7d",
		},
		DockerfilePath: ".",
	}

	file, err := os.Create("loco.toml")
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(config); err != nil {
		return err
	}
	return nil
}
