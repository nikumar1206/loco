package kube

import (
	"context"
	"fmt"
	"log/slog"

	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CheckNSExists checks if a namespace exists in the Kubernetes cluster.
func (kc *KubernetesClient) CheckNSExists(ctx context.Context, namespace string) (bool, error) {
	slog.DebugContext(ctx, "Checking if namespace exists", "namespace", namespace)
	namespaces, err := kc.ClientSet.CoreV1().Namespaces().List(ctx, metaV1.ListOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to list namespaces", "error", err)
		return false, err
	}

	for _, ns := range namespaces.Items {
		if ns.Name == namespace {
			slog.InfoContext(ctx, "Namespace exists", "namespace", namespace)
			return true, nil
		}
	}

	slog.InfoContext(ctx, "Namespace does not exist", "namespace", namespace)
	return false, nil
}

// CreateNS creates a new namespace in the Kubernetes cluster if it does not already exist.
func (kc *KubernetesClient) CreateNS(ctx context.Context, ldc *LocoDeploymentContext) (*v1.Namespace, error) {
	namespace := ldc.Namespace()
	slog.InfoContext(ctx, "Creating namespace", "namespace", namespace)

	exists, err := kc.CheckNSExists(ctx, namespace)
	if err != nil {
		return nil, err
	}

	if exists {
		slog.WarnContext(ctx, "Namespace already exists", "namespace", namespace)
		return nil, fmt.Errorf("namespace already exists: %s", namespace)
	}

	// required gw label
	// todo: fragile? can we move it to some constant?
	labels := ldc.Labels()
	labels["expose-via-gw"] = "true"

	nsConfig := &v1.Namespace{
		ObjectMeta: metaV1.ObjectMeta{
			Name:   namespace,
			Labels: labels,
		},
	}

	ns, err := kc.ClientSet.CoreV1().Namespaces().Create(ctx, nsConfig, metaV1.CreateOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create namespace", "namespace", namespace, "error", err)
		return nil, fmt.Errorf("failed to create namespace: %w", err)
	}

	slog.InfoContext(ctx, "Namespace created", "namespace", ns.Name)
	return ns, nil
}

// DeleteNS deletes a namespace in the Kubernetes cluster.
func (kc *KubernetesClient) DeleteNS(ctx context.Context, namespace string) error {
	slog.InfoContext(ctx, "Deleting namespace", "namespace", namespace)
	err := kc.ClientSet.CoreV1().Namespaces().Delete(ctx, namespace, metaV1.DeleteOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to delete namespace", "namespace", namespace, "error", err)
		return fmt.Errorf("failed to delete namespace: %w", err)
	}
	slog.InfoContext(ctx, "Namespace deleted", "namespace", namespace)
	return nil
}
