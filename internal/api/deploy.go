package api

import (
	"fmt"
	"os"

	json "github.com/goccy/go-json"

	"github.com/joho/godotenv"
	"github.com/nikumar1206/loco/internal/config"
)

type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type DeployAppRequest struct {
	LocoConfig     config.Config `json:"locoConfig"`
	ContainerImage string        `json:"containerImage"`
	EnvVars        []EnvVar
}

type DeployAppResponse struct {
	Message string
}

func (c *Client) DeployApp(locoConfig config.Config, containerImage string, locoToken string, logf func(string)) error {
	// Create a new deployment
	envVars := map[string]string{}
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", locoToken),
	}

	if locoConfig.EnvFile != "" {
		f, err := os.Open(locoConfig.EnvFile)
		if err != nil {
			return err
		}

		envVars, err = godotenv.Parse(f)
		if err != nil {
			return err
		}

	}

	envVarList := []EnvVar{}

	for k, v := range envVars {
		envVarList = append(envVarList, EnvVar{Name: k, Value: v})
	}

	appReq := DeployAppRequest{
		LocoConfig:     locoConfig,
		ContainerImage: containerImage,
		EnvVars:        envVarList,
	}

	resp, err := c.Post("/api/v1/app", appReq, headers)
	if err != nil {
		return err
	}
	deployAppResponse := new(DeployAppResponse)

	if err := json.Unmarshal(resp, deployAppResponse); err != nil {
		return fmt.Errorf("error unmarshalling deploy token response: %v", err)
	}

	return nil
}
