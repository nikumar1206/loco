package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"

	"github.com/dusted-go/logging/prettylog"
	"github.com/gofiber/fiber/v3"
	"github.com/nikumar1206/loco/service/internal/client"
	"github.com/nikumar1206/loco/service/internal/handlers"
	"github.com/nikumar1206/loco/service/internal/middlewares"
	"github.com/nikumar1206/loco/service/internal/models"
	"github.com/nikumar1206/loco/service/internal/utils"
)

func newAppConfig() *models.AppConfig {
	logLevel := utils.Must(strconv.Atoi((os.Getenv("LOG_LEVEL"))))
	return &models.AppConfig{
		Env:             os.Getenv("APP_ENV"),
		ProjectID:       os.Getenv("GITLAB_PROJECT_ID"),
		RegistryURL:     "https://gitlab.com",
		DeployTokenName: os.Getenv("GITLAB_DEPLOY_TOKEN_NAME"),
		GitlabPAT:       os.Getenv("GITLAB_PAT"),
		LogLevel:        slog.Level(logLevel),
		PORT:            os.Getenv("PORT"),
	}
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

	kubernetesClient := client.NewKubernetesClient(ac.Env)

	handlers.BuildRegistryRouter(app, ac)
	handlers.BuildAppRouter(app, ac, kubernetesClient)

	routes := app.GetRoutes(true)

	for _, route := range routes {
		fmt.Printf("%s %s\n", route.Method, route.Path)
	}

	log.Fatal(app.Listen(ac.PORT, fiber.ListenConfig{DisableStartupMessage: true}))
}

func getLoggerHandler(ac *models.AppConfig) slog.Handler {
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
