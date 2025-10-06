package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"connectrpc.com/connect"
	"connectrpc.com/grpcreflect"
	charmLog "github.com/charmbracelet/log"
	"github.com/nikumar1206/loco/api/client"
	"github.com/nikumar1206/loco/api/middleware"
	"github.com/nikumar1206/loco/api/models"
	"github.com/nikumar1206/loco/api/service"
	"github.com/nikumar1206/loco/proto/app/v1/appv1connect"
	"github.com/nikumar1206/loco/proto/oauth/v1/oauthv1connect"
	"github.com/nikumar1206/loco/proto/registry/v1/registryv1connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func newAppConfig() *models.AppConfig {
	logLevel := Must(strconv.Atoi((os.Getenv("LOG_LEVEL"))))

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
	ac := newAppConfig()

	logger := slog.New(CustomHandler{Handler: getLoggerHandler(ac)})
	slog.SetDefault(logger)

	mux := http.NewServeMux()
	interceptors := connect.WithInterceptors(middleware.NewGithubAuthInterceptor())

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Loco Service is Running")
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Server is healthy.")
	})

	kubernetesClient := client.NewKubernetesClient(ac.Env)

	oAuthServiceHandler := &service.OAuthServer{}
	registryServiceHandler := &service.RegistryServer{AppConfig: *ac}
	appServiceHandler := &service.AppServer{AppConfig: *ac, Kc: *kubernetesClient}

	oauthPath, oauthHandler := oauthv1connect.NewOAuthServiceHandler(oAuthServiceHandler, interceptors)
	registryPath, registryHandler := registryv1connect.NewRegistryServiceHandler(registryServiceHandler, interceptors)
	appPath, appHandler := appv1connect.NewAppServiceHandler(appServiceHandler, interceptors)

	reflector := grpcreflect.NewStaticReflector(
		oauthv1connect.OAuthServiceGithubOAuthDetailsProcedure,
		registryv1connect.RegistryServiceGitlabTokenProcedure,
		appv1connect.AppServiceDeployAppProcedure,
		appv1connect.AppServiceLogsProcedure,
		appv1connect.AppServiceStatusProcedure,
	)

	// mount both old and new reflectors for backwards compatibility
	mux.Handle(grpcreflect.NewHandlerV1(reflector))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))

	mux.Handle(oauthPath, oauthHandler)
	mux.Handle(registryPath, registryHandler)
	mux.Handle(appPath, appHandler)

	muxWTiming := middleware.Timing(mux)
	muxWContext := middleware.SetContext(muxWTiming)

	log.Fatal(http.ListenAndServe(
		":8000",
		// use h2c so we can serve HTTP/2 without TLS.
		h2c.NewHandler(muxWContext, &http2.Server{}),
	))
}

func getLoggerHandler(ac *models.AppConfig) slog.Handler {
	if ac.Env == "PRODUCTION" {
		return slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:     ac.LogLevel,
			AddSource: true,
		})
	}
	return charmLog.New(os.Stderr)
}
