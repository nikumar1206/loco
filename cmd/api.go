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
	fmt.Print("Fetching deploy token... ")
	c := api.Client{
		BaseURL: "http://localhost:8000",
	}
	fmt.Print("calling the get ")
	resp, err := c.Get("/api/v1/registry/token", nil)
	if err != nil {
		fmt.Println("failed to get deploy token")
		fmt.Print("Error: ", err)
		return DeployTokenResponse{}, err
	}

	var tokenResponse DeployTokenResponse
	if err := json.Unmarshal(resp, &tokenResponse); err != nil {
		return DeployTokenResponse{}, err
	}

	return tokenResponse, nil
}
