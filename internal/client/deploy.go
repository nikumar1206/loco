package client

import (
	"context"
	"fmt"
	"os"

	"connectrpc.com/connect"
	"github.com/joho/godotenv"
	"github.com/nikumar1206/loco/shared/config"
	appv1 "github.com/nikumar1206/loco/shared/proto/app/v1"
	appv1connect "github.com/nikumar1206/loco/shared/proto/app/v1/appv1connect"
)

func (c *Client) DeployApp(config config.Config, containerImage string, locoToken string, logf func(string), wait bool) error {
	envVars := map[string]string{}
	if config.LocoConfig.Env.File != "" {
		f, err := os.Open(config.LocoConfig.Env.File)
		if err != nil {
			return err
		}
		envVars, err = godotenv.Parse(f)
		if err != nil {
			return err
		}
	}

	envVarList := []*appv1.EnvVar{}
	for k, v := range envVars {
		envVarList = append(envVarList, &appv1.EnvVar{Name: k, Value: v})
	}

	deployClient := appv1connect.NewAppServiceClient(&c.HTTPClient, c.BaseURL)

	req := connect.NewRequest(&appv1.DeployAppRequest{
		LocoConfig:     config.LocoConfig,
		ContainerImage: containerImage,
		EnvVars:        envVarList,
		Wait:           wait,
	})
	req.Header().Set("Authorization", fmt.Sprintf("Bearer %s", locoToken))

	stream, err := deployClient.DeployApp(context.Background(), req)
	if err != nil {
		return err
	}

	for stream.Receive() {
		resp := stream.Msg()
		logf(resp.Message)
		if resp.EventType == "error" {
			return fmt.Errorf("deployment failed: %s", resp.Message)
		}
	}

	if err := stream.Err(); err != nil {
		return err
	}

	return nil
}
