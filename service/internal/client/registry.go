package client

import (
	"fmt"
	"log/slog"

	json "github.com/goccy/go-json"

	"github.com/gofiber/fiber/v3"
)

type gitlabResponse struct {
	Username  string   `json:"username"`
	Token     string   `json:"token"`
	ExpiresAt string   `json:"expires_at"`
	Revoked   bool     `json:"revoked"`
	Expired   bool     `json:"expired"`
	Scopes    []string `json:"scopes"`
}

func (c *Client) GetDeployToken(ctx fiber.Ctx, registryPat string, projectId string, payload map[string]any) (*gitlabResponse, error) {
	deployTokenPath := fmt.Sprintf("/api/v4/projects/%s/deploy_tokens", projectId)
	resp, err := c.Post(deployTokenPath, payload, map[string]string{
		"Content-Type":  "application/json",
		"PRIVATE-TOKEN": registryPat,
	})
	if err != nil {
		slog.ErrorContext(ctx, "Error creating deploy token", slog.String("err", err.Error()))
		return nil, err
	}

	gitlabResp := new(gitlabResponse)

	if err := json.Unmarshal(resp, &gitlabResp); err != nil {
		return nil, err
	}

	return gitlabResp, err
}
