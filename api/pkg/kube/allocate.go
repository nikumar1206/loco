package kube

import (
	"context"
	"fmt"
	"log/slog"

	v1 "k8s.io/api/core/v1"
	rbacV1 "k8s.io/api/rbac/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AllocateResources orchestrates the creation of all Kubernetes resources for a deployment.
// If any step fails, it attempts to clean up by deleting the namespace.
func (kc *Client) AllocateResources(
	ctx context.Context,
	ldc *LocoDeploymentContext,
	envVars map[string]string,
	registryConfig *DockerRegistryConfig,
) error {
	namespace := ldc.Namespace()
	slog.InfoContext(ctx, "Starting resource allocation", "namespace", namespace, "app", ldc.App.Name)

	_, err := kc.CreateNS(ctx, ldc)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create namespace", "error", err)
		return fmt.Errorf("failed to allocate resources: %w", err)
	}

	defer func() {
		if err != nil {
			slog.WarnContext(ctx, "Cleaning up namespace due to allocation failure", "namespace", namespace)
			if deleteErr := kc.DeleteNS(ctx, namespace); deleteErr != nil {
				slog.ErrorContext(ctx, "Failed to delete namespace during cleanup", "error", deleteErr)
			}
		}
	}()

	_, err = kc.CreateSecret(ctx, ldc, envVars)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create secret", "error", err)
		return fmt.Errorf("failed to create secret: %w", err)
	}

	if registryConfig != nil {
		err = kc.CreateDockerPullSecret(ctx, ldc, *registryConfig)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to create docker pull secret", "error", err)
			return fmt.Errorf("failed to create docker pull secret: %w", err)
		}
	}

	_, err = kc.CreateServiceAccount(ctx, ldc)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create service account", "error", err)
		return fmt.Errorf("failed to create service account: %w", err)
	}

	_, err = kc.createRoleWithSecretName(ctx, ldc, ldc.EnvSecretName())
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create role", "error", err)
		return fmt.Errorf("failed to create role: %w", err)
	}

	_, err = kc.CreateRoleBinding(ctx, ldc)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create role binding", "error", err)
		return fmt.Errorf("failed to create role binding: %w", err)
	}

	_, err = kc.CreateService(ctx, ldc)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create service", "error", err)
		return fmt.Errorf("failed to create service: %w", err)
	}

	_, err = kc.CreateDeployment(ctx, ldc)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create deployment", "error", err)
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	_, err = kc.CreateHTTPRoute(ctx, ldc)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create HTTPRoute", "error", err)
		return fmt.Errorf("failed to create HTTPRoute: %w", err)
	}

	slog.InfoContext(ctx, "Resource allocation completed successfully", "namespace", namespace, "app", ldc.App.Name)
	return nil
}

// createRoleWithSecretName is a helper that creates a role referencing a secret name
func (kc *Client) createRoleWithSecretName(ctx context.Context, ldc *LocoDeploymentContext, secretName string) (*rbacV1.Role, error) {
	slog.InfoContext(ctx, "Creating role", "namespace", ldc.Namespace(), "name", ldc.RoleName())

	placeholderSecret := &v1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name: secretName,
		},
	}

	return kc.CreateRole(ctx, ldc, placeholderSecret)
}
