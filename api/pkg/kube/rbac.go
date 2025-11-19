package kube

import (
	"context"
	"fmt"
	"log/slog"

	v1 "k8s.io/api/core/v1"
	rbacV1 "k8s.io/api/rbac/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateServiceAccount creates a Kubernetes ServiceAccount
func (kc *Client) CreateServiceAccount(ctx context.Context, ldc *LocoDeploymentContext) (*v1.ServiceAccount, error) {
	slog.InfoContext(ctx, "Creating service account", "namespace", ldc.Namespace(), "name", ldc.ServiceAccountName())

	sa := &v1.ServiceAccount{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      ldc.ServiceAccountName(),
			Namespace: ldc.Namespace(),
			Labels:    ldc.Labels(),
		},
	}

	result, err := kc.ClientSet.CoreV1().ServiceAccounts(ldc.Namespace()).Create(ctx, sa, metaV1.CreateOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create service account", "name", ldc.ServiceAccountName(), "error", err)
		return nil, fmt.Errorf("failed to create service account: %w", err)
	}

	slog.InfoContext(ctx, "Service account created", "name", result.Name)
	return result, nil
}

// CreateRole creates a Kubernetes Role for accessing secrets
func (kc *Client) CreateRole(ctx context.Context, ldc *LocoDeploymentContext, secret *v1.Secret) (*rbacV1.Role, error) {
	slog.InfoContext(ctx, "Creating role", "namespace", ldc.Namespace(), "name", ldc.RoleName())

	role := &rbacV1.Role{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      ldc.RoleName(),
			Namespace: ldc.Namespace(),
			Labels:    ldc.Labels(),
		},
		// todo: is this too liberal? we just need access to docker secret + env vars.
		Rules: []rbacV1.PolicyRule{
			{
				APIGroups:     []string{""},
				Resources:     []string{"secrets"},
				Verbs:         []string{"get", "list", "watch"},
				ResourceNames: []string{secret.Name},
			},
		},
	}

	result, err := kc.ClientSet.RbacV1().Roles(ldc.Namespace()).Create(ctx, role, metaV1.CreateOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create role", "name", ldc.RoleName(), "error", err)
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	slog.InfoContext(ctx, "Role created", "name", result.Name)
	return result, nil
}

// CreateRoleBinding creates a Kubernetes RoleBinding
func (kc *Client) CreateRoleBinding(ctx context.Context, ldc *LocoDeploymentContext) (*rbacV1.RoleBinding, error) {
	slog.InfoContext(ctx, "Creating role binding", "namespace", ldc.Namespace(), "name", ldc.RoleBindingName())

	rb := &rbacV1.RoleBinding{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      ldc.RoleBindingName(),
			Namespace: ldc.Namespace(),
			Labels:    ldc.Labels(),
		},
		Subjects: []rbacV1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      ldc.ServiceAccountName(),
				Namespace: ldc.Namespace(),
			},
		},
		RoleRef: rbacV1.RoleRef{
			Kind:     "Role",
			Name:     ldc.RoleName(),
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	result, err := kc.ClientSet.RbacV1().RoleBindings(ldc.Namespace()).Create(ctx, rb, metaV1.CreateOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create role binding", "name", ldc.RoleBindingName(), "error", err)
		return nil, fmt.Errorf("failed to create role binding: %w", err)
	}

	slog.InfoContext(ctx, "Role binding created", "name", result.Name)
	return result, nil
}
