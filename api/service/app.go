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
) (*connect.Response[appv1.DeployAppResponse], error) {
	request := req.Msg

	// fill defaults and validate
	locoConfig.FillSensibleDefaults(request.LocoConfig)

	if err := locoConfig.Validate(request.LocoConfig); err != nil {
		slog.ErrorContext(ctx, "invalid locoConfig", "error", err.Error())
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid locoConfig: %w", err))
	}

	// check banned subdomain
	if locoConfig.IsBannedSubDomain(request.LocoConfig.Routing.Subdomain) {
		slog.ErrorContext(ctx, "banned subdomain", slog.String("subdomain", request.LocoConfig.Routing.Subdomain))
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("provided subdomain is not allowed"))
	}

	user, ok := ctx.Value("user").(string)
	if !ok {
		slog.ErrorContext(ctx, "could not determine user. should never happen")
		return nil, connect.NewError(connect.CodeUnauthenticated, ErrNoUser)
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
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check if service exists: %w", err))
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
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get deploy token: %w", err))
		}

		registry := client.DockerRegistryConfig{
			Server:   s.AppConfig.RegistryURL,
			Username: gitlabResp.Username,
			Password: gitlabResp.Token,
		}

		if err := s.Kc.UpdateDockerPullSecret(ctx, app, registry); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update docker pull secret: %w", err))
		}

		if err := s.Kc.UpdateContainer(ctx, app); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update container: %w", err))
		}

		return connect.NewResponse(&appv1.DeployAppResponse{
			Message: "App updated successfully",
		}), nil
	}

	// Create new app flow
	if _, err := s.Kc.CreateNS(ctx, app); err != nil {
		slog.ErrorContext(ctx, err.Error())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create namespace: %w", err))
	}

	expiry := time.Now().Add(5 * time.Minute).UTC().Format("2006-01-02T15:04:05-0700")
	payload := map[string]any{
		"name":       s.AppConfig.DeployTokenName,
		"scopes":     []string{"read_registry"},
		"expires_at": expiry,
	}

	gitlabResp, err := client.NewClient(s.AppConfig.GitlabURL).GetDeployToken(ctx, s.AppConfig.GitlabPAT, s.AppConfig.ProjectID, payload)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get deploy token: %w", err))
	}

	registry := client.DockerRegistryConfig{
		Server:   s.AppConfig.RegistryURL,
		Username: gitlabResp.Username,
		Password: gitlabResp.Token,
	}

	if err := s.Kc.CreateDockerPullSecret(ctx, app, registry); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create docker secret: %w", err))
	}

	envSecret, err := s.Kc.CreateSecret(ctx, app)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create secret: %w", err))
	}

	if _, err := s.Kc.CreateRole(ctx, app, envSecret); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create role: %w", err))
	}

	if _, err := s.Kc.CreateServiceAccount(ctx, app); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create service account: %w", err))
	}

	if _, err := s.Kc.CreateRoleBinding(ctx, app); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create role binding: %w", err))
	}

	if _, err := s.Kc.CreateDeployment(ctx, app, request.ContainerImage, envSecret); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create deployment: %w", err))
	}

	if _, err := s.Kc.CreateService(ctx, app); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create service: %w", err))
	}

	if _, err := s.Kc.CreateHTTPRoute(ctx, app); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create http route: %w", err))
	}

	return connect.NewResponse(&appv1.DeployAppResponse{
		Message: "Deployment and service created successfully",
	}), nil
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
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("app does not exist"))
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
