package main

import (
	"encoding/json"
	"fmt"

	"github.com/nikumar1206/loco/internal/api"
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

func getDeployToken() (DeployTokenResponse, error) {
	locoOut(LOCO__OK_PREFIX, "Fetching deploy token...")
	c := api.Client{
		BaseURL: "http://localhost:8000",
	}
	resp, err := c.Get("/api/v1/registry/token", nil)
	if err != nil {
		locoErr(LOCO__ERROR_PREFIX, "failed to get deploy token")
		locoErr(LOCO__ERROR_PREFIX, fmt.Sprintf("Error: %v", err)) // Changed from fmt.Print
		return DeployTokenResponse{}, err
	}

	var tokenResponse DeployTokenResponse
	if err := json.Unmarshal(resp, &tokenResponse); err != nil {
		// Assuming this error should also be logged, though it wasn't explicitly in the original
		locoErr(LOCO__ERROR_PREFIX, fmt.Sprintf("Error unmarshalling deploy token response: %v", err))
		return DeployTokenResponse{}, err
	}

	// Assuming a success message here might be useful, though not in the original
	locoOut(LOCO__OK_PREFIX, "Successfully fetched and unmarshalled deploy token.")
	return tokenResponse, nil
}
