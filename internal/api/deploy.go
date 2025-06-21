package api

import (
	"encoding/json"
	"fmt"

	"github.com/nikumar1206/loco/internal/config"
)

type DeployAppRequest struct {
	LocoConfig     config.Config `json:"locoConfig"`
	ContainerImage string        `json:"containerImage"`
}

type DeployAppResponse struct {
	Message string
}

func (c *Client) DeployApp(locoConfig config.Config, containerImage string) error {
	// Create a new deployment
	appReq := DeployAppRequest{
		LocoConfig:     locoConfig,
		ContainerImage: containerImage,
	}

	resp, err := c.Post("/api/v1/app/deploy", appReq, nil)
	if err != nil {
		return err
	}
	deployAppResponse := new(DeployAppResponse)

	if err := json.Unmarshal(resp, deployAppResponse); err != nil {
		return fmt.Errorf("error unmarshalling deploy token response: %v", err)
	}

	return nil
}
