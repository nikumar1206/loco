package client

import (
	"slices"
	"strings"
	"time"
)

// goal is to house logic required for creating and managing the app

// Users cannot use these subdomains for their apps
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

type LocoApp struct {
	Name           string
	Namespace      string
	CreatedBy      string
	CreatedAt      time.Time
	Labels         map[string]string
	ContainerImage string
	Subdomain      string
	EnvVars        []EnvVar
	Config         LocoConfig
}

func NewLocoApp(name, subdomain, createdBy string, containerImage string, envVars []EnvVar, config LocoConfig) *LocoApp {
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
