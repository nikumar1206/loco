package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sort"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	genDb "github.com/nikumar1206/loco/api/gen/db"
	"github.com/nikumar1206/loco/api/pkg/klogmux"
	"github.com/nikumar1206/loco/api/pkg/kube"
	"github.com/nikumar1206/loco/api/timeutil"
	appv1 "github.com/nikumar1206/loco/shared/proto/app/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var (
	ErrAppNotFound           = errors.New("app not found")
	ErrAppNameNotUnique      = errors.New("app name already exists in this workspace")
	ErrSubdomainNotAvailable = errors.New("subdomain already in use")
	ErrClusterNotFound       = errors.New("cluster not found")
	ErrClusterNotHealthy     = errors.New("cluster is not healthy")
	ErrInvalidAppType        = errors.New("invalid app type")
)

type AppServer struct {
	db         *pgxpool.Pool
	queries    *genDb.Queries
	kubeClient *kube.KubernetesClient
}

// NewAppServer creates a new AppServer instance
func NewAppServer(db *pgxpool.Pool, queries *genDb.Queries) *AppServer {
	// todo: move this out.
	appEnv := os.Getenv("APP_ENV")
	kubeClient := kube.NewKubernetesClient(appEnv)

	return &AppServer{
		db:         db,
		queries:    queries,
		kubeClient: kubeClient,
	}
}

// CreateApp creates a new app
func (s *AppServer) CreateApp(
	ctx context.Context,
	req *connect.Request[appv1.CreateAppRequest],
) (*connect.Response[appv1.CreateAppResponse], error) {
	r := req.Msg

	userID, ok := ctx.Value("user_id").(int64)
	if !ok {
		slog.ErrorContext(ctx, "user_id not found in context")
		return nil, connect.NewError(connect.CodeUnauthenticated, ErrUnauthorized)
	}

	// todo: revisit validating roles
	role, err := s.queries.GetWorkspaceMemberRole(ctx, genDb.GetWorkspaceMemberRoleParams{
		WorkspaceID: r.WorkspaceId,
		UserID:      userID,
	})
	if err != nil {
		slog.WarnContext(ctx, "user is not a member of workspace", "workspace_id", r.WorkspaceId, "user_id", userID)
		return nil, connect.NewError(connect.CodePermissionDenied, ErrNotWorkspaceMember)
	}

	if role != "admin" && role != "deploy" {
		slog.WarnContext(ctx, "user does not have permission to create app", "workspace_id", r.WorkspaceId, "user_id", userID, "role", role)
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("must be workspace admin or have deploy role"))
	}

	domain := r.GetDomain()
	if domain == "" {
		domain = "loco.deploy-app.com"
	}

	available, err := s.queries.CheckSubdomainAvailability(ctx, genDb.CheckSubdomainAvailabilityParams{
		Subdomain: r.Subdomain,
		Domain:    domain,
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to check subdomain availability", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	if !available {
		slog.WarnContext(ctx, "subdomain not available", "subdomain", r.Subdomain, "domain", domain)
		return nil, connect.NewError(connect.CodeAlreadyExists, ErrSubdomainNotAvailable)
	}

	// todo: Get cluster details and validate health
	// clusterDetails, err := s.queries.GetClusterDetails(ctx, r.ClusterId)
	// if err != nil {
	// 	slog.WarnContext(ctx, "cluster not found", "cluster_id", r.ClusterId)
	// 	return nil, connect.NewError(connect.CodeNotFound, ErrClusterNotFound)
	// }

	// if !clusterDetails.IsActive.Bool || clusterDetails.HealthStatus.String != "healthy" {
	// 	slog.WarnContext(ctx, "cluster is not healthy or active", "cluster_id", r.ClusterId, "is_active", clusterDetails.IsActive.Bool, "health_status", clusterDetails.HealthStatus.String)
	// 	return nil, connect.NewError(connect.CodeFailedPrecondition, ErrClusterNotHealthy)
	// }

	// todo: set namepsace after creating and saving app. or perhaps its set after first deployment on the app.
	app, err := s.queries.CreateApp(ctx, genDb.CreateAppParams{
		WorkspaceID: r.WorkspaceId,
		ClusterID:   1,
		Name:        r.Name,
		Type:        int32(r.Type.Number()),
		Subdomain:   r.Subdomain,
		Domain:      domain,
		CreatedBy:   userID,
		// ns empty until first deployment occurs on the app.
		// Namespace:   ns,
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to create app", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	return connect.NewResponse(&appv1.CreateAppResponse{
		App: dbAppToProto(app),
	}), nil
}

// GetApp retrieves an app by ID
func (s *AppServer) GetApp(
	ctx context.Context,
	req *connect.Request[appv1.GetAppRequest],
) (*connect.Response[appv1.GetAppResponse], error) {
	r := req.Msg

	// todo: role checks should actually be done first.
	userID, ok := ctx.Value("user_id").(int64)
	if !ok {
		slog.ErrorContext(ctx, "user_id not found in context")
		return nil, connect.NewError(connect.CodeUnauthenticated, ErrUnauthorized)
	}

	app, err := s.queries.GetAppByID(ctx, r.Id)
	if err != nil {
		slog.WarnContext(ctx, "app not found", "id", r.Id)
		return nil, connect.NewError(connect.CodeNotFound, ErrAppNotFound)
	}

	_, err = s.queries.GetWorkspaceMember(ctx, genDb.GetWorkspaceMemberParams{
		WorkspaceID: app.WorkspaceID,
		UserID:      userID,
	})
	if err != nil {
		slog.WarnContext(ctx, "user is not a member of app's workspace", "workspace_id", app.WorkspaceID, "user_id", userID)
		return nil, connect.NewError(connect.CodePermissionDenied, ErrNotWorkspaceMember)
	}

	return connect.NewResponse(&appv1.GetAppResponse{
		App: dbAppToProto(app),
	}), nil
}

// ListApps lists all apps in a workspace
func (s *AppServer) ListApps(
	ctx context.Context,
	req *connect.Request[appv1.ListAppsRequest],
) (*connect.Response[appv1.ListAppsResponse], error) {
	r := req.Msg

	userID, ok := ctx.Value("user_id").(int64)
	if !ok {
		slog.ErrorContext(ctx, "user_id not found in context")
		return nil, connect.NewError(connect.CodeUnauthenticated, ErrUnauthorized)
	}

	_, err := s.queries.GetWorkspaceMember(ctx, genDb.GetWorkspaceMemberParams{
		WorkspaceID: r.WorkspaceId,
		UserID:      userID,
	})
	if err != nil {
		slog.WarnContext(ctx, "user is not a member of workspace", "workspace_id", r.WorkspaceId, "user_id", userID)
		return nil, connect.NewError(connect.CodePermissionDenied, ErrNotWorkspaceMember)
	}

	dbApps, err := s.queries.ListAppsForWorkspace(ctx, r.WorkspaceId)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list apps", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	var apps []*appv1.App
	for _, dbApp := range dbApps {
		apps = append(apps, dbAppToProto(dbApp))
	}

	return connect.NewResponse(&appv1.ListAppsResponse{
		Apps: apps,
	}), nil
}

// UpdateApp updates an app
func (s *AppServer) UpdateApp(
	ctx context.Context,
	req *connect.Request[appv1.UpdateAppRequest],
) (*connect.Response[appv1.UpdateAppResponse], error) {
	r := req.Msg

	userID, ok := ctx.Value("user_id").(int64)
	if !ok {
		slog.ErrorContext(ctx, "user_id not found in context")
		return nil, connect.NewError(connect.CodeUnauthenticated, ErrUnauthorized)
	}

	workspaceID, err := s.queries.GetAppWorkspaceID(ctx, r.Id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, connect.NewError(connect.CodeNotFound, ErrAppNotFound)
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	role, err := s.queries.GetWorkspaceMemberRole(ctx, genDb.GetWorkspaceMemberRoleParams{
		WorkspaceID: workspaceID,
		UserID:      userID,
	})
	if err != nil {
		slog.WarnContext(ctx, "user is not a member of workspace", "workspace_id", fmt.Sprintf("%d", workspaceID), "user_id", userID)
		return nil, connect.NewError(connect.CodePermissionDenied, ErrNotWorkspaceMember)
	}

	if role != genDb.WorkspaceRoleAdmin && role != genDb.WorkspaceRoleDeploy {
		slog.WarnContext(ctx, "user does not have permission to update app", "workspace_id", fmt.Sprintf("%d", workspaceID), "user_id", userID, "role", string(role))
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("must be workspace admin or deploy role to update app"))
	}

	updateParams := genDb.UpdateAppParams{
		ID: r.Id,
	}

	if r.GetName() != "" {
		updateParams.Name = pgtype.Text{String: r.GetName(), Valid: true}
	}

	if r.GetSubdomain() != "" {
		updateParams.Subdomain = pgtype.Text{String: r.GetSubdomain(), Valid: true}
	}

	if r.GetDomain() != "" {
		updateParams.Domain = pgtype.Text{String: r.GetDomain(), Valid: true}
	}

	app, err := s.queries.UpdateApp(ctx, updateParams)
	if err != nil {
		slog.ErrorContext(ctx, "failed to update app", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	return connect.NewResponse(&appv1.UpdateAppResponse{
		App: dbAppToProto(app),
	}), nil
}

// DeleteApp deletes an app
func (s *AppServer) DeleteApp(
	ctx context.Context,
	req *connect.Request[appv1.DeleteAppRequest],
) (*connect.Response[appv1.DeleteAppResponse], error) {
	r := req.Msg

	userID, ok := ctx.Value("user_id").(int64)
	if !ok {
		slog.ErrorContext(ctx, "user_id not found in context")
		return nil, connect.NewError(connect.CodeUnauthenticated, ErrUnauthorized)
	}

	workspaceID, err := s.queries.GetAppWorkspaceID(ctx, r.Id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, connect.NewError(connect.CodeNotFound, ErrAppNotFound)
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	role, err := s.queries.GetWorkspaceMemberRole(ctx, genDb.GetWorkspaceMemberRoleParams{
		WorkspaceID: workspaceID,
		UserID:      userID,
	})
	if err != nil {
		slog.WarnContext(ctx, "user is not a member of workspace", "workspace_id", fmt.Sprintf("%d", workspaceID), "user_id", userID)
		return nil, connect.NewError(connect.CodePermissionDenied, ErrNotWorkspaceMember)
	}

	if role != genDb.WorkspaceRoleAdmin {
		slog.WarnContext(ctx, "user is not an admin of workspace", "workspace_id", fmt.Sprintf("%d", workspaceID), "user_id", userID, "role", string(role))
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("must be workspace admin to delete app"))
	}

	err = s.queries.DeleteApp(ctx, r.Id)
	if err != nil {
		slog.ErrorContext(ctx, "failed to delete app", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	return connect.NewResponse(&appv1.DeleteAppResponse{
		Success: true,
	}), nil
}

// CheckSubdomainAvailability checks if a subdomain is available
func (s *AppServer) CheckSubdomainAvailability(
	ctx context.Context,
	req *connect.Request[appv1.CheckSubdomainAvailabilityRequest],
) (*connect.Response[appv1.CheckSubdomainAvailabilityResponse], error) {
	r := req.Msg

	available, err := s.queries.CheckSubdomainAvailability(ctx, genDb.CheckSubdomainAvailabilityParams{
		Subdomain: r.Subdomain,
		Domain:    r.Domain,
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to check subdomain availability", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	return connect.NewResponse(&appv1.CheckSubdomainAvailabilityResponse{
		Available: available,
	}), nil
}

// GetAppStatus retrieves an app and its current deployment status
func (s *AppServer) GetAppStatus(
	ctx context.Context,
	req *connect.Request[appv1.GetAppStatusRequest],
) (*connect.Response[appv1.GetAppStatusResponse], error) {
	r := req.Msg

	userID, ok := ctx.Value("user_id").(int64)
	if !ok {
		slog.ErrorContext(ctx, "user_id not found in context")
		return nil, connect.NewError(connect.CodeUnauthenticated, ErrUnauthorized)
	}

	app, err := s.queries.GetAppByID(ctx, r.AppId)
	if err != nil {
		slog.WarnContext(ctx, "app not found", "app_id", r.AppId)
		return nil, connect.NewError(connect.CodeNotFound, ErrAppNotFound)
	}

	_, err = s.queries.GetWorkspaceMember(ctx, genDb.GetWorkspaceMemberParams{
		WorkspaceID: app.WorkspaceID,
		UserID:      userID,
	})
	if err != nil {
		slog.WarnContext(ctx, "user is not a member of app's workspace", "workspace_id", app.WorkspaceID, "user_id", userID)
		return nil, connect.NewError(connect.CodePermissionDenied, ErrNotWorkspaceMember)
	}

	deploymentList, err := s.queries.ListDeploymentsForApp(ctx, genDb.ListDeploymentsForAppParams{
		AppID:  r.AppId,
		Limit:  1,
		Offset: 0,
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to list deployments", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	var deploymentStatus *appv1.DeploymentStatus
	if len(deploymentList) > 0 {
		deployment := deploymentList[0]
		deploymentStatus = &appv1.DeploymentStatus{
			Id:       deployment.ID,
			Status:   string(deployment.Status),
			Replicas: deployment.Replicas,
		}
		if deployment.Message.Valid {
			deploymentStatus.Message = &deployment.Message.String
		}
		if deployment.ErrorMessage.Valid {
			deploymentStatus.ErrorMessage = &deployment.ErrorMessage.String
		}
	}

	return connect.NewResponse(&appv1.GetAppStatusResponse{
		App:               dbAppToProto(app),
		CurrentDeployment: deploymentStatus,
	}), nil
}

// StreamLogs streams logs for an app
func (s *AppServer) StreamLogs(
	ctx context.Context,
	req *connect.Request[appv1.StreamLogsRequest],
	stream *connect.ServerStream[appv1.LogEntry],
) error {
	r := req.Msg

	userID, ok := ctx.Value("user_id").(int64)
	if !ok {
		slog.ErrorContext(ctx, "user_id not found in context")
		return connect.NewError(connect.CodeUnauthenticated, ErrUnauthorized)
	}

	app, err := s.queries.GetAppByID(ctx, r.AppId)
	if err != nil {
		slog.WarnContext(ctx, "app not found", "app_id", r.AppId)
		return connect.NewError(connect.CodeNotFound, ErrAppNotFound)
	}

	_, err = s.queries.GetWorkspaceMember(ctx, genDb.GetWorkspaceMemberParams{
		WorkspaceID: app.WorkspaceID,
		UserID:      userID,
	})
	if err != nil {
		slog.WarnContext(ctx, "user is not a member of app's workspace", "workspace_id", app.WorkspaceID, "user_id", userID)
		return connect.NewError(connect.CodePermissionDenied, ErrNotWorkspaceMember)
	}

	if app.Namespace == "" {
		slog.WarnContext(ctx, "app has no namespace assigned", "app_id", r.AppId)
		return connect.NewError(connect.CodeFailedPrecondition, errors.New("app has not been deployed yet"))
	}

	slog.InfoContext(ctx, "streaming logs for app", "app_id", r.AppId, "app_namespace", app.Namespace)

	// build label selector to find pods for this app
	selector := labels.SelectorFromSet(labels.Set{"app": app.Name})

	// build the log stream
	builder := klogmux.NewBuilder(s.kubeClient.ClientSet).
		Namespace(app.Namespace).
		LabelSelector(selector.String()).
		Follow(r.GetFollow())

	if r.Limit != nil {
		builder.TailLines(int64(*r.Limit))
	}

	logStream := builder.Build()

	// start the log stream
	logStream.Start(ctx)
	defer logStream.Stop()

	slog.DebugContext(ctx, "log stream started", "app_id", r.AppId, "namespace", app.Namespace)

	// stream log entries to client
	for entry := range logStream.Entries() {
		logProto := &appv1.LogEntry{
			PodName:   entry.PodName,
			Namespace: entry.Namespace,
			Container: entry.Container,
			Timestamp: timestamppb.New(entry.Timestamp),
			Log:       entry.Message,
		}

		if entry.IsError {
			logProto.Level = "ERROR"
		} else {
			logProto.Level = "INFO"
		}

		if err := stream.Send(logProto); err != nil {
			slog.ErrorContext(ctx, "failed to send log entry", "error", err)
			return err
		}
	}

	// check for stream errors
	for err := range logStream.Errors() {
		if err != nil {
			slog.ErrorContext(ctx, "log stream error", "error", err)
			return connect.NewError(connect.CodeInternal, fmt.Errorf("log stream error: %w", err))
		}
	}

	slog.DebugContext(ctx, "log stream completed", "app_id", r.AppId)
	return nil
}

// GetEvents retrieves Kubernetes events for an app
func (s *AppServer) GetEvents(
	ctx context.Context,
	req *connect.Request[appv1.GetEventsRequest],
) (*connect.Response[appv1.GetEventsResponse], error) {
	r := req.Msg

	userID, ok := ctx.Value("user_id").(int64)
	if !ok {
		slog.ErrorContext(ctx, "user_id not found in context")
		return nil, connect.NewError(connect.CodeUnauthenticated, ErrUnauthorized)
	}

	app, err := s.queries.GetAppByID(ctx, r.AppId)
	if err != nil {
		slog.WarnContext(ctx, "app not found", "app_id", r.AppId)
		return nil, connect.NewError(connect.CodeNotFound, ErrAppNotFound)
	}

	_, err = s.queries.GetWorkspaceMember(ctx, genDb.GetWorkspaceMemberParams{
		WorkspaceID: app.WorkspaceID,
		UserID:      userID,
	})
	if err != nil {
		slog.WarnContext(ctx, "user is not a member of app's workspace", "workspace_id", app.WorkspaceID, "user_id", userID)
		return nil, connect.NewError(connect.CodePermissionDenied, ErrNotWorkspaceMember)
	}

	if app.Namespace == "" {
		slog.WarnContext(ctx, "app has no namespace assigned", "app_id", r.AppId)
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("app has not been deployed yet"))
	}

	slog.InfoContext(ctx, "fetching events for app", "app_id", r.AppId, "app_namespace", app.Namespace)

	eventList, err := s.kubeClient.ClientSet.CoreV1().Events(app.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "failed to list events from kubernetes", "error", err, "namespace", app.Namespace)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to fetch events: %w", err))
	}

	var protoEvents []*appv1.Event
	for _, k8sEvent := range eventList.Items {
		// filter events to those related to this app's pods
		if k8sEvent.InvolvedObject.Kind != "Pod" {
			continue
		}

		protoEvent := &appv1.Event{
			Timestamp: timestamppb.New(k8sEvent.FirstTimestamp.Time),
			Reason:    k8sEvent.Reason,
			Message:   k8sEvent.Message,
			Type:      k8sEvent.Type,
			PodName:   k8sEvent.InvolvedObject.Name,
		}
		protoEvents = append(protoEvents, protoEvent)
	}

	// sort by timestamp descending (newest first)
	sort.Slice(protoEvents, func(i, j int) bool {
		return protoEvents[i].Timestamp.AsTime().After(protoEvents[j].Timestamp.AsTime())
	})

	// apply limit if specified
	if r.Limit != nil && *r.Limit > 0 && int(*r.Limit) < len(protoEvents) {
		protoEvents = protoEvents[:*r.Limit]
	}

	slog.DebugContext(ctx, "fetched events for app", "app_id", r.AppId, "event_count", len(protoEvents))

	return connect.NewResponse(&appv1.GetEventsResponse{
		Events: protoEvents,
	}), nil
}

// dbAppToProto converts a database App to the proto App
// to be returned to client.
func dbAppToProto(app genDb.App) *appv1.App {
	appType := appv1.AppType(app.Type)
	return &appv1.App{
		Id:          app.ID,
		WorkspaceId: app.WorkspaceID,
		Name:        app.Name,
		Namespace:   app.Namespace,
		Type:        appType,
		Subdomain:   app.Subdomain,
		Domain:      app.Domain,
		CreatedBy:   app.CreatedBy,
		CreatedAt:   timeutil.ParsePostgresTimestamp(app.CreatedAt.Time),
		UpdatedAt:   timeutil.ParsePostgresTimestamp(app.UpdatedAt.Time),
	}
}
