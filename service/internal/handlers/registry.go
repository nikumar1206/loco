package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/nikumar1206/loco/service/internal/client"
	"github.com/nikumar1206/loco/service/internal/models"
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
			"scopes":     []string{"read_registry", "write_registry"},
			"expires_at": expiry,
		}
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			log.Printf("Error marshalling payload: %v", err)
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create deploy token payload",
			})
		}

		// Call GitLab API

		apiClient := client.NewAPIClient(appConfig.RegistryURL)

		resp, err := apiClient.Post(fmt.Sprintf("/api/v4/projects/%s/deploy_tokens", projectId), payloadBytes, map[string]string{
			"Content-Type":  "application/json",
			"PRIVATE-TOKEN": appConfig.GitlabPAT,
		})
		if err != nil {
			log.Printf("Error creating deploy token: %v", err)
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create deploy token",
			})
		}

		var gitlabResp struct {
			Username  string   `json:"username"`
			Token     string   `json:"token"`
			ExpiresAt string   `json:"expires_at"`
			Revoked   bool     `json:"revoked"`
			Expired   bool     `json:"expired"`
			Scopes    []string `json:"scopes"`
		}
		if err := json.Unmarshal(resp, &gitlabResp); err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to parse GitLab response",
			})
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
