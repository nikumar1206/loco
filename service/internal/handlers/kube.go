package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gofiber/fiber/v3"

	"github.com/nikumar1206/loco/service/internal/client"
	"github.com/nikumar1206/loco/service/internal/models"
	"github.com/nikumar1206/loco/service/internal/utils"
)

type DeployAppRequest struct {
	LocoConfig     client.LocoConfig `json:"locoConfig"`
	ContainerImage string            `json:"containerImage"`
}

func BuildAppRouter(app *fiber.App, ac *models.AppConfig, kc *client.KubernetesClient) {
	api := app.Group("/api/v1/app")

	api.Post("/deploy", deployApp(kc))
}

func deployApp(kc *client.KubernetesClient) fiber.Handler {
	return func(c fiber.Ctx) error {
		var request DeployAppRequest

		if err := c.Bind().JSON(&request); err != nil {
			return utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid Input")
		}

		// validate the locoConfig
		if err := request.LocoConfig.Validate(); err != nil {
			slog.Error("Invalid locoConfig", "error", err.Error())
			return utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid locoConfig")
		}
		// need to only create the loco app if it already does not exist
		// replace with actual user name based on github login
		banned := client.IsBannedSubDomain(request.LocoConfig.Subdomain)

		if banned {
			slog.Error("banned subdomain", slog.String("subdomain", request.LocoConfig.Subdomain))
			return utils.SendErrorResponse(c, http.StatusBadRequest, "Provided subdomain is not allowed. Please choose another")
		}

		app := client.NewLocoApp(request.LocoConfig.Name, request.LocoConfig.Subdomain, "nikhil", request.ContainerImage)
		slog.Info("Creating new loco app", "appName", app.Name, "namespace", app.Namespace)

		_, err := kc.CreateNS(c.Context(), app)
		if err != nil {
			return utils.SendErrorResponse(c, http.StatusInternalServerError, "failed to create namespace")
		}
		_, err = kc.CreateDeployment(c.Context(), app)
		if err != nil {
			return utils.SendErrorResponse(c, http.StatusInternalServerError, "failed to create deployment")
		}

		_, err = kc.CreateService(c.Context(), app)
		if err != nil {
			return utils.SendErrorResponse(c, http.StatusInternalServerError, "failed to create service")
		}

		_, err = kc.CreateHTTPRoute(c.Context(), app)
		if err != nil {
			slog.Error("oops something wrong", "error", err.Error())
			return utils.SendErrorResponse(c, http.StatusInternalServerError, "failed to create http route")
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message":   "Deployment and service created successfully",
			"namespace": "loco-setup",
		})
	}
}
