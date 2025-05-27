package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nikumar1206/loco/internal/api"
)

var gitlabPAT = os.Getenv("GITLAB_PAT")

// Response to CLI
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

func main() {
	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Loco Deploy Token Service is running")
	})

	app.Get("/api/v1/registry/token", func(c *fiber.Ctx) error {
		projectId := "70221423"
		tokenName := "loco-ecr-deploy-token"
		expiresInMin := 5
		expiry := time.Now().Add(time.Duration(expiresInMin) * time.Minute).UTC().Format("2006-01-02T15:04:05-0700")

		// Create payload for GitLab API
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

		apiClient := api.NewClient("https://gitlab.com")

		resp, err := apiClient.Post(fmt.Sprintf("/api/v4/projects/%s/deploy_tokens", projectId), payloadBytes, map[string]string{
			"Content-Type":  "application/json",
			"PRIVATE-TOKEN": gitlabPAT,
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
	})

	log.Fatal(app.Listen(":8000"))
}
