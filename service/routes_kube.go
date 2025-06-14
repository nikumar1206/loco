package main

import "github.com/gofiber/fiber/v2"

func buildKubeRouter(app *fiber.App, appConfig *AppConfig) {
	api := app.Group("/api/v1/kube/deploy")

	api.Get("/token", getKubeToken(appConfig))
}

func getKubeToken(appConfig *AppConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
			"error": "get punked kid",
		})
	}
}
