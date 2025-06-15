package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/nikumar1206/loco/service/clients"
)

type AppConfig struct {
	Env             string `json:"env"`             // Environment (e.g., dev, prod)
	ProjectID       string `json:"projectId"`       // GitLab project ID
	RegistryURL     string `json:"registryUrl"`     // Container registry URL
	DeployTokenName string `json:"deployTokenName"` // Deploy token name
	GitlabPAT       string `json:"gitlabPAT"`       // GitLab Personal Access Token
}

func newAppConfig() *AppConfig {
	return &AppConfig{
		Env:             os.Getenv("APP_ENV"),
		ProjectID:       os.Getenv("GITLAB_PROJECT_ID"),
		RegistryURL:     os.Getenv("GITLAB_REGISTRY_URL"),
		DeployTokenName: os.Getenv("GITLAB_DEPLOY_TOKEN_NAME"),
		GitlabPAT:       os.Getenv("GITLAB_PAT"),
	}
}

// Response to CLI
type DeployTokenResponse struct {
	Username  string   `json:"username"`
	Password  string   `json:"password"`
	Registry  string   `json:"registry"`
	Image     string   `json:"image"`
	ExpiresAt string   `json:"expiresAt"`
	Revoked   bool     `json:"revoked"`
	Expired   bool     `json:"expired"`
	Scopes    []string `json:"scopes"`
}

func main() {
	app := fiber.New()
	ac := newAppConfig()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Loco Deploy Token Service is running")
	})

	kubernetesClient := clients.NewKubernetesClient(ac.Env)

	buildRegistryRouter(app, ac)
	buildAppRouter(app, ac, kubernetesClient)

	log.Fatal(app.Listen(":8000"))
}
