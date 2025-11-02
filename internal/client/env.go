package client

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	appv1 "github.com/nikumar1206/loco/shared/proto/app/v1"
	appv1connect "github.com/nikumar1206/loco/shared/proto/app/v1/appv1connect"
)

func UpdateEnvVars(host string, appName string, envVars []*appv1.EnvVar, restart bool, token string) error {
	appClient := appv1connect.NewAppServiceClient(http.DefaultClient, host)

	req := &appv1.UpdateEnvVarsRequest{
		Name:    appName,
		EnvVars: envVars,
		Restart: restart,
	}

	request := connect.NewRequest(req)
	request.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))

	_, err := appClient.UpdateEnvVars(context.Background(), request)
	if err != nil {
		return fmt.Errorf("failed to update env vars: %w", err)
	}

	return nil
}
