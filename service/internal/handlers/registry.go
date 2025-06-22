package handlers

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/nikumar1206/loco/service/internal/client"
	"github.com/nikumar1206/loco/service/internal/models"
	"github.com/nikumar1206/loco/service/internal/utils"
)

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

// buildRegistryRouter houses APIs for interacting with container registry service (gitlab)
func BuildRegistryRouter(app *fiber.App, appConfig *models.AppConfig) {
	api := app.Group("/api/v1/registry")
	api.Get("/token", createGetTokenHandler(appConfig))
}

func createGetTokenHandler(appConfig *models.AppConfig) fiber.Handler {
	return func(c fiber.Ctx) error {
		projectId := appConfig.ProjectID
		tokenName := appConfig.DeployTokenName
		expiresInMin := 5
		expiry := time.Now().Add(time.Duration(expiresInMin) * time.Minute).UTC().Format("2006-01-02T15:04:05-0700")

		payload := map[string]any{
			"name":       tokenName,
			"scopes":     []string{"write_registry"},
			"expires_at": expiry,
		}

		// Call GitLab API

		gitlabResp, err := client.NewClient(appConfig.RegistryURL).GetDeployToken(c, appConfig.GitlabPAT, projectId, payload)
		if err != nil {
			return utils.SendErrorResponse(
				c, fiber.StatusInternalServerError, err.Error(),
			)
		}

		// Compose response
		res := DeployTokenResponse{
			Username:  gitlabResp.Username,
			Password:  gitlabResp.Token,
			Registry:  "registry.gitlab.com",
			Image:     "registry.gitlab.com/locomotive-group/loco-ecr",
			ExpiresAt: gitlabResp.ExpiresAt,
			Revoked:   gitlabResp.Revoked,
			Expired:   gitlabResp.Expired,
			Scopes:    gitlabResp.Scopes,
		}
		return c.JSON(res)
	}
}
