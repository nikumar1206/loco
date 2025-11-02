package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"

	"github.com/nikumar1206/loco/api/client"
	"github.com/nikumar1206/loco/api/models"
	locoConfig "github.com/nikumar1206/loco/shared/config"
	appv1 "github.com/nikumar1206/loco/shared/proto/app/v1"
)

var (
	ErrNoUser   = errors.New("user could not be determined")
	ErrNoStatus = errors.New("could not determine app status")
)

type AppServer struct {
	AppConfig models.AppConfig
	Kc        client.KubernetesClient
}

func (s *AppServer) DeployApp(
	ctx context.Context,
	req *connect.Request[appv1.DeployAppRequest],
	stream *connect.ServerStream[appv1.DeployAppResponse],
) error {
	request := req.Msg

	sendEvent := func(eventType, message string) error {
		return stream.Send(&appv1.DeployAppResponse{
			Message:   message,
			EventType: eventType,
		})
	}

	// fill defaults and validate
	locoConfig.FillSensibleDefaults(request.LocoConfig)

	if err := locoConfig.Validate(request.LocoConfig); err != nil {
		slog.ErrorContext(ctx, "invalid locoConfig", "error", err.Error())
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid locoConfig: %w", err))
	}

	// check banned subdomain
	if locoConfig.IsBannedSubDomain(request.LocoConfig.Routing.Subdomain) {
		slog.ErrorContext(ctx, "banned subdomain", slog.String("subdomain", request.LocoConfig.Routing.Subdomain))
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("provided subdomain is not allowed"))
	}

	err := client.ValidateResources(request.LocoConfig.Resources.Cpu, request.LocoConfig.Resources.Memory)
	if err != nil {
		slog.ErrorContext(ctx, "invalid resource requests", "error", err.Error())
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid resource requests: %w", err))
	}

	user, ok := ctx.Value("user").(string)
	if !ok {
		slog.ErrorContext(ctx, "could not determine user. should never happen")
		return connect.NewError(connect.CodeUnauthenticated, ErrNoUser)
	}

	app := locoConfig.NewLocoApp(
		request.LocoConfig,
		user,
		request.ContainerImage,
		request.EnvVars,
	)

	// check if service exists; if exists update in-place else create new
	serviceExists, err := s.Kc.CheckServiceExists(ctx, app.Namespace, app.Name)
	if err != nil {
		slog.ErrorContext(ctx, err.Error())
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check if service exists: %w", err))
	}

	if serviceExists {
		slog.InfoContext(ctx, "service exists, updating in-place")

		expiry := time.Now().Add(5 * time.Minute).UTC().Format("2006-01-02T15:04:05-0700")
		payload := map[string]any{
			"name":       s.AppConfig.DeployTokenName,
			"scopes":     []string{"read_registry"},
			"expires_at": expiry,
		}

		gitlabResp, err := client.NewClient(s.AppConfig.GitlabURL).GetDeployToken(ctx, s.AppConfig.GitlabPAT, s.AppConfig.ProjectID, payload)
		if err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get deploy token: %w", err))
		}

		registry := client.DockerRegistryConfig{
			Server:   s.AppConfig.RegistryURL,
			Username: gitlabResp.Username,
			Password: gitlabResp.Token,
		}

		if err := s.Kc.UpdateDockerPullSecret(ctx, app, registry); err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update docker pull secret: %w", err))
		}

		if err := s.Kc.UpdateContainer(ctx, app); err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update container: %w", err))
		}

		if err := sendEvent("info", "App updated successfully"); err != nil {
			return err
		}
		return nil
	}

	// Create new app flow
	if err := s.Kc.CheckNodesReady(ctx); err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("nodes not ready: %w", err))
	}

	if _, err := s.Kc.CreateNS(ctx, app); err != nil {
		slog.ErrorContext(ctx, err.Error())
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create namespace: %w", err))
	}

	if err := s.allocateResources(ctx, app, req, stream); err != nil {
		slog.InfoContext(ctx, "cleaning up namespace due to deployment failure", "namespace", app.Namespace)
		if delErr := s.Kc.DeleteNS(ctx, app.Namespace); delErr != nil {
			slog.ErrorContext(ctx, "failed to cleanup namespace", "namespace", app.Namespace, "error", delErr)
		}
		if sendErr := sendEvent("error", "App deployment failed"); sendErr != nil {
			return sendErr
		}
		return err
	}

	return nil
}

func (s *AppServer) allocateResources(ctx context.Context, app *locoConfig.LocoApp, req *connect.Request[appv1.DeployAppRequest], stream *connect.ServerStream[appv1.DeployAppResponse]) error {
	sendEvent := func(eventType, message string) error {
		return stream.Send(&appv1.DeployAppResponse{
			Message:   message,
			EventType: eventType,
		})
	}

	expiry := time.Now().Add(5 * time.Minute).UTC().Format("2006-01-02T15:04:05-0700")
	payload := map[string]any{
		"name":       s.AppConfig.DeployTokenName,
		"scopes":     []string{"read_registry"},
		"expires_at": expiry,
	}

	gitlabResp, err := client.NewClient(s.AppConfig.GitlabURL).GetDeployToken(ctx, s.AppConfig.GitlabPAT, s.AppConfig.ProjectID, payload)
	if err != nil {
		return fmt.Errorf("failed to get deploy token: %w", err)
	}

	registry := client.DockerRegistryConfig{
		Server:   s.AppConfig.RegistryURL,
		Username: gitlabResp.Username,
		Password: gitlabResp.Token,
	}

	if err := s.Kc.CreateDockerPullSecret(ctx, app, registry); err != nil {
		return fmt.Errorf("failed to create docker secret: %w", err)
	}

	if err := sendEvent("progress", "Creating necessary roles and policies"); err != nil {
		return err
	}

	envSecret, err := s.Kc.CreateSecret(ctx, app)
	if err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}

	if _, err := s.Kc.CreateRole(ctx, app, envSecret); err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	if _, err := s.Kc.CreateServiceAccount(ctx, app); err != nil {
		return fmt.Errorf("failed to create service account: %w", err)
	}

	if _, err := s.Kc.CreateRoleBinding(ctx, app); err != nil {
		return fmt.Errorf("failed to create role binding: %w", err)
	}

	if err := sendEvent("progress", "Scheduling compute"); err != nil {
		return err
	}
	if _, err := s.Kc.CreateDeployment(ctx, app, req.Msg.ContainerImage, envSecret); err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	if _, err := s.Kc.CreateService(ctx, app); err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	if err := sendEvent("progress", "Exposing your app to the internet"); err != nil {
		return err
	}

	if _, err := s.Kc.CreateHTTPRoute(ctx, app); err != nil {
		return fmt.Errorf("failed to create http route: %w", err)
	}

	if !req.Msg.Wait {
		return nil
	}

	if err := sendEvent("progress", "Waiting for rollout"); err != nil {
		return err
	}

	// wait for rollout to complete if provided wait flag.
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("rollout timed out")
		case <-ticker.C:
			status, err := s.Kc.GetDeploymentStatus(ctx, app.Namespace, app.Name)
			if err != nil {
				return fmt.Errorf("failed to get deployment status: %w", err)
			}
			if status.Status == "Running" && status.Health == "Passing" {
				if err := sendEvent("info", "Rollout completed successfully"); err != nil {
					return err
				}
				return nil
			} else {
				if err := sendEvent("progress", "Waiting for rollout"); err != nil {
					return err
				}
			}
		}
	}
}

func (s *AppServer) Logs(
	ctx context.Context,
	req *connect.Request[appv1.LogsRequest],
	stream *connect.ServerStream[appv1.LogsResponse],
) error {
	appName := req.Msg.AppName
	user, ok := ctx.Value("user").(string)
	if !ok {
		slog.ErrorContext(ctx, "could not determine user. should never happen")
		return connect.NewError(connect.CodeUnauthenticated, ErrNoUser)
	}

	namespace := locoConfig.GenerateNameSpace(appName, user)
	err := s.Kc.GetLogs(ctx, namespace, appName, user, nil, stream)
	if err != nil {
		slog.ErrorContext(ctx, err.Error())
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to fetch logs: %w", err))
	}

	return nil
}

func (s *AppServer) Status(
	ctx context.Context, req *connect.Request[appv1.StatusRequest],
) (*connect.Response[appv1.StatusResponse], error) {
	appName := req.Msg.AppName

	user, ok := ctx.Value("user").(string)

	if !ok {
		slog.ErrorContext(ctx, "could not determine user. should never happen")
		return nil, connect.NewError(connect.CodeUnauthenticated, ErrNoUser)
	}

	namespace := locoConfig.GenerateNameSpace(appName, user)

	deploymentStatus, err := s.Kc.GetDeploymentStatus(ctx, namespace, appName)
	if err != nil {
		slog.ErrorContext(ctx, err.Error())
		return nil, connect.NewError(connect.CodeInternal, ErrNoStatus)
	}

	return connect.NewResponse(deploymentStatus), nil
}

func (s *AppServer) DestroyApp(
	ctx context.Context,
	req *connect.Request[appv1.DestroyAppRequest],
) (*connect.Response[appv1.DestroyAppResponse], error) {
	appName := req.Msg.Name

	user, ok := ctx.Value("user").(string)
	if !ok {
		slog.ErrorContext(ctx, "could not determine user. should never happen")
		return nil, connect.NewError(connect.CodeUnauthenticated, ErrNoUser)
	}

	namespace := locoConfig.GenerateNameSpace(appName, user)

	if err := s.Kc.DeleteNS(ctx, namespace); err != nil {
		slog.ErrorContext(ctx, err.Error())
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("app does not exist. It may have already been deleted"))
	}

	return connect.NewResponse(&appv1.DestroyAppResponse{
		Message: "App destruction initiated successfully",
	}), nil
}

func (s *AppServer) ScaleApp(
	ctx context.Context,
	req *connect.Request[appv1.ScaleAppRequest],
) (*connect.Response[appv1.ScaleAppResponse], error) {
	appName := req.Msg.Name

	user, ok := ctx.Value("user").(string)
	if !ok {
		slog.ErrorContext(ctx, "could not determine user. should never happen")
		return nil, connect.NewError(connect.CodeUnauthenticated, ErrNoUser)
	}

	namespace := locoConfig.GenerateNameSpace(appName, user)

	if err := s.Kc.ScaleDeployment(ctx, namespace, appName, req.Msg.Replicas, req.Msg.Cpu, req.Msg.Memory); err != nil {
		slog.ErrorContext(ctx, err.Error())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to scale app: %w", err))
	}

	return connect.NewResponse(&appv1.ScaleAppResponse{
		Message: "App scaled successfully",
	}), nil
}
