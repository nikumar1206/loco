package main

import (
	"github.com/gofiber/fiber/v3"

	c "github.com/nikumar1206/loco/service/clients"
)

func buildAppRouter(app *fiber.App, appConfig *AppConfig, kc *c.KubernetesClient) {
	api := app.Group("/api/v1/app")

	api.Get("/token", getKubeToken(appConfig))
	api.Post("/deploy", deployApp(appConfig, kc))
}

func getKubeToken(appConfig *AppConfig) fiber.Handler {
	return func(c fiber.Ctx) error {
		return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
			"error": "get punked kid",
		})
	}
}

func deployApp(appConfig *AppConfig, kc *c.KubernetesClient) fiber.Handler {
	return func(c fiber.Ctx) error {
		_, err := kc.CreateNS(c.Context(), "test-deploy-1")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to create namespace",
			})
		}
		_, err = kc.CreateDeployment(c.Context(), "test-deploy-1", "test-deploy-1-deployment")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to create deployment",
			})
		}

		_, err = kc.CreateService(c.Context(), "test-deploy-1", "test-deploy-1-service")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to create service",
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message":   "Deployment and service created successfully",
			"namespace": "loco-setup",
		})
	}
}
