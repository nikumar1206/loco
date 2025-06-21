package models

import "log/slog"

type AppConfig struct {
	Env             string `json:"env"`             // Environment (e.g., dev, prod)
	ProjectID       string `json:"projectId"`       // GitLab project ID
	RegistryURL     string `json:"registryUrl"`     // Container registry URL
	DeployTokenName string `json:"deployTokenName"` // Deploy token name
	GitlabPAT       string `json:"gitlabPAT"`       // GitLab Personal Access Token
	LogLevel        slog.Level
	PORT            string
}
