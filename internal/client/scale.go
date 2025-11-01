package client

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	appv1 "github.com/nikumar1206/loco/proto/app/v1"
	appv1connect "github.com/nikumar1206/loco/proto/app/v1/appv1connect"
)

func ScaleApp(host string, appName string, replicas *int32, cpu, memory *string, token string) error {
	appClient := appv1connect.NewAppServiceClient(http.DefaultClient, host)

	req := &appv1.ScaleAppRequest{
		Name:     appName,
		Replicas: replicas,
		Cpu:      cpu,
		Memory:   memory,
	}

	request := connect.NewRequest(req)
	request.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))

	_, err := appClient.ScaleApp(context.Background(), request)
	if err != nil {
		return fmt.Errorf("failed to scale app: %w", err)
	}

	return nil
}
