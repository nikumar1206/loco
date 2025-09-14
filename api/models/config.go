package models

import (
	"log/slog"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

type AppConfig struct {
	Env             string `json:"env"`             // Environment (e.g., dev, prod)
	ProjectID       string `json:"projectId"`       // GitLab project ID
	GitlabURL       string `json:"gitlabUrl"`       // Container registry URL
	RegistryURL     string `json:"registryUrl"`     // Container registry URL
	DeployTokenName string `json:"deployTokenName"` // Deploy token name
	GitlabPAT       string `json:"gitlabPAT"`       // GitLab Personal Access Token
	LogLevel        slog.Level
	PORT            string
}

var OAuthConf = &oauth2.Config{
	ClientID:     os.Getenv("GH_OAUTH_CLIENT_ID"),
	ClientSecret: os.Getenv("GH_OAUTH_CLIENT_SECRET"),
	Scopes:       []string{"read:user"},
	Endpoint:     github.Endpoint,
	RedirectURL:  os.Getenv("GH_OAUTH_REDIRECT_URL"),
}

var OAuthTokenTTL = time.Duration(8 * time.Hour)
