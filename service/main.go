package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"

	"github.com/dusted-go/logging/prettylog"
	json "github.com/goccy/go-json"
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
		GitlabURL:       os.Getenv("GITLAB_URL"),
		RegistryURL:     os.Getenv("GITLAB_REGISTRY_URL"),
		DeployTokenName: os.Getenv("GITLAB_DEPLOY_TOKEN_NAME"),
		GitlabPAT:       os.Getenv("GITLAB_PAT"),
		PORT:            os.Getenv("PORT"),
		LogLevel:        slog.Level(logLevel),
	}
}

func main() {
	app := fiber.New(fiber.Config{
		JSONEncoder: json.Marshal,
		JSONDecoder: json.Unmarshal,
	})
	ac := newAppConfig()

	logger := slog.New(CustomHandler{Handler: getLoggerHandler(ac)})
	slog.SetDefault(logger)

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Loco Deploy Token Service is running")
	})

	app.Use(middlewares.SetContext())
	app.Use(middlewares.Timing())
	app.Use(middlewares.GithubTokenValidator())

	app.Get("/secure", func(c fiber.Ctx) error {
		user, _ := c.Locals("user").(string)

		fmt.Println("hello user", user)
		return c.SendString("on the secure endpoint")
	})

	kubernetesClient := client.NewKubernetesClient(ac.Env)

	handlers.BuildRegistryRouter(app, ac)
	handlers.BuildAppRouter(app, ac, kubernetesClient)
	handlers.BuildOauthRouter(app, ac)

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
