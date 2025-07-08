package handlers

import (
	"bufio"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	json "github.com/goccy/go-json"

	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"

	"github.com/nikumar1206/loco/service/internal/client"
	"github.com/nikumar1206/loco/service/internal/models"
	"github.com/nikumar1206/loco/service/internal/utils"
)

type DeployAppRequest struct {
	LocoConfig     client.LocoConfig `json:"locoConfig"`
	ContainerImage string            `json:"containerImage"`
	EnvVars        []client.EnvVar
}

func BuildAppRouter(app *fiber.App, ac *models.AppConfig, kc *client.KubernetesClient) {
	api := app.Group("/api/v1/app")

	api.Post("", deployApp(ac, kc))
	api.Get("/:appName/logs", appLogs(kc))
	api.Get("/:appName/status", appStatus(kc))
}

func deployApp(ac *models.AppConfig, kc *client.KubernetesClient) fiber.Handler {
	return func(c fiber.Ctx) error {
		var request DeployAppRequest

		if err := c.Bind().JSON(&request); err != nil {
			slog.ErrorContext(c.Context(), "invalid json body", "error", err.Error())
			return utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid Input")
		}
		request.LocoConfig.FillSensibleDefaults()
		// validate the locoConfig
		if err := request.LocoConfig.Validate(); err != nil {
			slog.ErrorContext(c.Context(), "Invalid locoConfig", "error", err.Error())
			return utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid locoConfig")
		}
		// need to only create the loco app if it already does not exist
		// replace with actual user name based on github login
		banned := client.IsBannedSubDomain(request.LocoConfig.Subdomain)

		if banned {
			slog.ErrorContext(c.Context(), "banned subdomain", slog.String("subdomain", request.LocoConfig.Subdomain))
			return utils.SendErrorResponse(c, http.StatusBadRequest, "Provided subdomain is not allowed. Please choose another")
		}
		user, _ := c.Locals("user").(string)

		app := client.NewLocoApp(request.LocoConfig.Name, request.LocoConfig.Subdomain, user, request.ContainerImage, request.EnvVars, request.LocoConfig)
		// we need to check if this service exists.
		exists, err := kc.CheckServiceExists(c.Context(), app.Namespace, app.Name)
		if err != nil {
			slog.ErrorContext(c.Context(), err.Error())
			return utils.SendErrorResponse(c, http.StatusInternalServerError, "failed to check if service already exists.")
		}

		if exists {
			slog.InfoContext(c.Context(), "service exists, we will update in-place")
			expiry := time.Now().Add(5 * time.Minute).UTC().Format("2006-01-02T15:04:05-0700")

			payload := map[string]any{
				"name":       ac.DeployTokenName,
				"scopes":     []string{"read_registry"},
				"expires_at": expiry,
			}
			gitlabResp, err := client.NewClient(ac.GitlabURL).GetDeployToken(c, ac.GitlabPAT, ac.ProjectID, payload)
			if err != nil {
				slog.ErrorContext(c.Context(), err.Error())
				return utils.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
			}

			registry := client.DockerRegistryConfig{
				Server:   ac.RegistryURL,
				Username: gitlabResp.Username,
				Password: gitlabResp.Token,
				Email:    "couldbeanything@gmail.com",
			}

			err = kc.UpdateDockerPullSecret(
				c.Context(),
				app,
				registry,
			)
			if err != nil {
				slog.ErrorContext(c.Context(), err.Error())
				return utils.SendErrorResponse(c, http.StatusInternalServerError, "failed to contact container registry")
			}

			err = kc.UpdateContainer(c.Context(), app)
			if err != nil {
				slog.ErrorContext(c.Context(), err.Error())
				return utils.SendErrorResponse(c, http.StatusInternalServerError, "failed to update Docker image of service")
			}

			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"message": "App updated successfully",
			})
		}

		_, err = kc.CreateNS(c.Context(), app)
		if err != nil {
			slog.ErrorContext(c.Context(), err.Error())
			return utils.SendErrorResponse(c, http.StatusInternalServerError, "failed to create namespace")
		}

		expiry := time.Now().Add(5 * time.Minute).UTC().Format("2006-01-02T15:04:05-0700")

		payload := map[string]any{
			"name":       ac.DeployTokenName,
			"scopes":     []string{"read_registry"},
			"expires_at": expiry,
		}

		// todo: clean up this new client nonsense that i need to do everytime
		gitlabResp, err := client.NewClient(ac.GitlabURL).GetDeployToken(c, ac.GitlabPAT, ac.ProjectID, payload)
		if err != nil {
			return utils.SendErrorResponse(
				c, fiber.StatusInternalServerError, err.Error(),
			)
		}

		registry := client.DockerRegistryConfig{
			Server:   ac.RegistryURL,
			Username: gitlabResp.Username,
			Password: gitlabResp.Token,
			Email:    "couldbeanything@gmail.com",
		}

		err = kc.CreateDockerPullSecret(
			c.Context(),
			app,
			registry,
		)
		if err != nil {
			slog.ErrorContext(c.Context(), err.Error())
			return utils.SendErrorResponse(c, http.StatusInternalServerError, "failed to generate credentials")
		}

		envSecret, err := kc.CreateSecret(c.Context(), app)
		if err != nil {
			slog.ErrorContext(c.Context(), err.Error())
			return utils.SendErrorResponse(c, http.StatusInternalServerError, "failed to create deployment")
		}

		_, err = kc.CreateRole(c.Context(), app, envSecret)
		if err != nil {
			slog.ErrorContext(c.Context(), err.Error())
			return utils.SendErrorResponse(c, http.StatusInternalServerError, "failed to create role")
		}

		_, err = kc.CreateServiceAccount(c.Context(), app)
		if err != nil {
			slog.ErrorContext(c.Context(), err.Error())
			return utils.SendErrorResponse(c, http.StatusInternalServerError, "failed to create service account")
		}

		_, err = kc.CreateRoleBinding(c.Context(), app)
		if err != nil {
			slog.ErrorContext(c.Context(), err.Error())
			return utils.SendErrorResponse(c, http.StatusInternalServerError, "failed to create role binding")
		}

		_, err = kc.CreateDeployment(c.Context(), app, request.ContainerImage, envSecret)
		if err != nil {
			slog.ErrorContext(c.Context(), err.Error())
			return utils.SendErrorResponse(c, http.StatusInternalServerError, "failed to create deployment")
		}

		_, err = kc.CreateService(c.Context(), app)
		if err != nil {
			return utils.SendErrorResponse(c, http.StatusInternalServerError, "failed to create service")
		}

		_, err = kc.CreateHTTPRoute(c.Context(), app)
		if err != nil {
			slog.ErrorContext(c.Context(), "oops something wrong", "error", err.Error())
			return utils.SendErrorResponse(c, http.StatusInternalServerError, "failed to create http route")
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "Deployment and service created successfully",
		})
	}
}

func appLogs(kc *client.KubernetesClient) fiber.Handler {
	return func(c fiber.Ctx) error {
		appName := c.Params("appName")

		user, ok := c.Locals("user").(string)

		if !ok {
			slog.ErrorContext(c.Context(), "could not determine user. should never happen")
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "could not determine user")
		}

		namespace := client.GenerateNameSpace(appName, user)

		tailStr := c.Query("tail")
		var tailLines *int64
		if tailStr != "" {
			if tail, err := strconv.ParseInt(tailStr, 10, 64); err == nil {
				tailLines = &tail
			}
		}

		c.Set("Content-Type", "text/event-stream")
		c.Set("Cache-Control", "no-cache")
		c.Set("Connection", "keep-alive")
		c.Set("Transfer-Encoding", "chunked")

		logLines, err := kc.GetServiceLogs(c.Context(), namespace, appName, tailLines)
		if err != nil {
			slog.ErrorContext(c.Context(), err.Error())
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
		}

		c.Status(fiber.StatusOK).Response().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
			for _, line := range logLines {
				jsonData, err := json.Marshal(line)
				if err != nil {
					slog.ErrorContext(c.Context(), err.Error())
					fmt.Fprintf(w, "data: %s\n\n", err.Error())
					break
				}
				fmt.Fprintf(w, "data: %s\n\n", jsonData)
				err = w.Flush()
				if err != nil {
					slog.ErrorContext(c.Context(), err.Error())
					break
				}
			}
		}))

		return nil
	}
}

func appStatus(kc *client.KubernetesClient) fiber.Handler {
	return func(c fiber.Ctx) error {
		appName := c.Params("appName")

		user, ok := c.Locals("user").(string)

		if !ok {
			slog.ErrorContext(c.Context(), "could not determine user. should never happen")
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "could not determine user")
		}

		namespace := client.GenerateNameSpace(appName, user)

		deploymentStatus, err := kc.GetDeploymentStatus(c.Context(), namespace, appName)
		if err != nil {
			slog.ErrorContext(c.Context(), err.Error())
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "could not fetch deployment status")
		}

		return c.Status(fiber.StatusOK).JSON(deploymentStatus)
	}
}
