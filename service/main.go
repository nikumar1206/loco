package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"

	"github.com/dusted-go/logging/prettylog"
	"github.com/gofiber/fiber/v3"
	"github.com/nikumar1206/loco/service/clients"
	"github.com/nikumar1206/loco/service/middlewares"
)

type AppConfig struct {
	Env             string `json:"env"`             // Environment (e.g., dev, prod)
	ProjectID       string `json:"projectId"`       // GitLab project ID
	RegistryURL     string `json:"registryUrl"`     // Container registry URL
	DeployTokenName string `json:"deployTokenName"` // Deploy token name
	GitlabPAT       string `json:"gitlabPAT"`       // GitLab Personal Access Token
	LogLevel        slog.Level
	PORT            string
}

func newAppConfig() *AppConfig {
	logLevel := Must(strconv.Atoi((os.Getenv("LOG_LEVEL"))))
	return &AppConfig{
		Env:             os.Getenv("APP_ENV"),
		ProjectID:       os.Getenv("GITLAB_PROJECT_ID"),
		RegistryURL:     os.Getenv("GITLAB_REGISTRY_URL"),
		DeployTokenName: os.Getenv("GITLAB_DEPLOY_TOKEN_NAME"),
		GitlabPAT:       os.Getenv("GITLAB_PAT"),
		LogLevel:        slog.Level(logLevel),
		PORT:            os.Getenv("PORT"),
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

	logger := slog.New(CustomHandler{Handler: getLoggerHandler(ac)})
	slog.SetDefault(logger)

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Loco Deploy Token Service is running")
	})

	app.Use(middlewares.SetContext())
	app.Use(middlewares.Timing())

	kubernetesClient := clients.NewKubernetesClient(ac.Env)

	buildRegistryRouter(app, ac)
	buildAppRouter(app, ac, kubernetesClient)

	routes := app.GetRoutes(true)

	for _, route := range routes {
		fmt.Printf("%s %s\n", route.Method, route.Path)
	}

	log.Fatal(app.Listen(ac.PORT))
}

func getLoggerHandler(ac *AppConfig) slog.Handler {
	if ac.Env == "PRODUCTION" {
		return slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:     ac.LogLevel,
			AddSource: true,
		})
	}

	return prettylog.NewHandler(&slog.HandlerOptions{
		Level:       ac.LogLevel,
		AddSource:   true,
		ReplaceAttr: nil,
	})
}
