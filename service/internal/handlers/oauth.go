package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/nikumar1206/loco/service/internal/models"
)

// BuildOauthRouter houses APIs for interacting with OAuth services. Currently github, but potentially google as well
func BuildOauthRouter(app *fiber.App, appConfig *models.AppConfig) {
	githubOAuthGroup := app.Group("/api/v1/oauth/github")

	githubOAuthGroup.Get("", getTokenDetails())
}

func getTokenDetails() fiber.Handler {
	return func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"clientId": models.OAuthConf.ClientID,
			"tokenTTL": models.OAuthTokenTTL.Seconds(),
		})
	}
}
