package api

import (
	"encoding/json"
	"fmt"
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

func (c *Client) GetDeployToken() (DeployTokenResponse, error) {
	resp, err := c.Get("/api/v1/registry/token", nil)
	if err != nil {
		return DeployTokenResponse{}, fmt.Errorf("failed to get deploy token: %v", err)
	}

	var tokenResponse DeployTokenResponse
	if err := json.Unmarshal(resp, &tokenResponse); err != nil {
		return DeployTokenResponse{}, fmt.Errorf("error unmarshalling deploy token response: %v", err)
	}

	return tokenResponse, nil
}
