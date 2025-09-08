package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"connectrpc.com/grpcreflect"
	"github.com/dusted-go/logging/prettylog"
	json "github.com/goccy/go-json"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/nikumar1206/loco/proto/oauth/v1/oauthv1connect"
	"github.com/nikumar1206/loco/service/internal/client"
	"github.com/nikumar1206/loco/service/internal/handlers"
	"github.com/nikumar1206/loco/service/internal/middlewares"
	"github.com/nikumar1206/loco/service/internal/models"
	"github.com/nikumar1206/loco/service/internal/utils"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
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

	kubernetesClient := client.NewKubernetesClient(ac.Env)

	handlers.BuildRegistryRouter(app, ac)
	handlers.BuildAppRouter(app, ac, kubernetesClient)
	handlers.BuildOauthRouter(app, ac)

	routes := app.GetRoutes(true)

	path, handler := oauthv1connect.NewOAuthServiceHandler(new(handlers.OAuthServer))

	newOAuthHandler := adaptor.HTTPHandler(handler)
	app.Use(path, newOAuthHandler)

	// todo: cleanup, but leaving here for e2e testing
	go func() {
		mux := http.NewServeMux()
		reflector := grpcreflect.NewStaticReflector(
			oauthv1connect.OAuthServiceGithubOAuthDetailsProcedure,
		)
		mux.Handle(grpcreflect.NewHandlerV1(reflector))
		// Many tools still expect the older version of the server reflection API, so
		// most servers should mount both handlers.
		mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))
		mux.Handle(path, handler)
		http.ListenAndServe(
			"localhost:8080",
			// Use h2c so we can serve HTTP/2 without TLS.
			h2c.NewHandler(mux, &http2.Server{}),
		)
	}()

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
