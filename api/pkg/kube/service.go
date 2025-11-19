package kube

import (
	"context"
	"fmt"
	"log/slog"

	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// CheckServiceExists checks if a Service exists in the specified namespace
func (kc *Client) CheckServiceExists(ctx context.Context, namespace, serviceName string) (bool, error) {
	slog.DebugContext(ctx, "Checking if service exists", "namespace", namespace, "service", serviceName)
	_, err := kc.ClientSet.CoreV1().Services(namespace).Get(ctx, serviceName, metaV1.GetOptions{})
	if err == nil {
		slog.InfoContext(ctx, "Service already exists", "name", serviceName)
		return true, nil
	}
	return false, nil
}

// CreateService creates a Kubernetes Service for the deployment
func (kc *Client) CreateService(ctx context.Context, ldc *LocoDeploymentContext) (*v1.Service, error) {
	slog.InfoContext(ctx, "Creating service", "namespace", ldc.Namespace(), "name", ldc.ServiceName())

	exists, err := kc.CheckServiceExists(ctx, ldc.Namespace(), ldc.ServiceName())
	if err != nil {
		return nil, fmt.Errorf("failed to check service existence: %w", err)
	}
	if exists {
		slog.WarnContext(ctx, "Service already exists", "name", ldc.ServiceName())
		return nil, fmt.Errorf("service already exists: %s", ldc.ServiceName())
	}

	service := &v1.Service{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      ldc.ServiceName(),
			Namespace: ldc.Namespace(),
			Labels:    ldc.Labels(),
		},
		Spec: v1.ServiceSpec{
			Type: v1.ServiceTypeClusterIP,
			Selector: map[string]string{
				LabelAppName: ldc.App.Name,
			},
			SessionAffinity: v1.ServiceAffinityNone,
			SessionAffinityConfig: &v1.SessionAffinityConfig{
				ClientIP: &v1.ClientIPConfig{
					TimeoutSeconds: ptrToInt32(SessionAffinityTimeout),
				},
			},
			Ports: []v1.ServicePort{
				{
					Name:       ldc.ServicePort(),
					Protocol:   v1.ProtocolTCP,
					Port:       DefaultServicePort,
					TargetPort: intstr.FromInt32(ldc.Config.Routing.Port),
				},
			},
		},
	}

	result, err := kc.ClientSet.CoreV1().Services(ldc.Namespace()).Create(ctx, service, metaV1.CreateOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create service", "name", ldc.ServiceName(), "error", err)
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	slog.InfoContext(ctx, "Service created", "service", result.Name)
	return result, nil
}
