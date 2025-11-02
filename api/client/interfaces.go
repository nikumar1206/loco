package client

import (
	"context"
	"time"

	"connectrpc.com/connect"
	appv1 "github.com/nikumar1206/loco/shared/proto/app/v1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	v1Gateway "sigs.k8s.io/gateway-api/apis/v1"
)

// NamespaceManager handles namespace operations
type NamespaceManager interface {
	CheckNSExists(ctx context.Context, namespace string) (bool, error)
	CreateNS(ctx context.Context, locoApp *LocoApp) (*corev1.Namespace, error)
	DeleteNS(ctx context.Context, namespace string) error
}

// DeploymentManager handles deployment operations
type DeploymentManager interface {
	CheckDeploymentExists(ctx context.Context, namespace string, deploymentName string) (bool, error)
	CreateDeployment(ctx context.Context, locoApp *LocoApp, containerImage string, secrets *corev1.Secret) (*v1.Deployment, error)
	UpdateContainer(ctx context.Context, locoApp *LocoApp) error
	ScaleDeployment(ctx context.Context, namespace, appName string, replicas *int32, cpu, memory *string) error
	UpdateEnvVars(ctx context.Context, locoApp *LocoApp, envVars []*appv1.EnvVar, restart bool) error
	GetDeploymentStatus(ctx context.Context, namespace, appName string) (*appv1.StatusResponse, error)
}

// ServiceManager handles service operations
type ServiceManager interface {
	CheckServiceExists(ctx context.Context, namespace, serviceName string) (bool, error)
	CreateService(ctx context.Context, locoApp *LocoApp) (*corev1.Service, error)
}

// SecretManager handles secret operations
type SecretManager interface {
	CreateSecret(ctx context.Context, locoApp *LocoApp) (*corev1.Secret, error)
	UpdateSecret(ctx context.Context, locoApp *LocoApp) (*corev1.Secret, error)
	CreateDockerPullSecret(ctx context.Context, locoApp *LocoApp, registry DockerRegistryConfig) error
	UpdateDockerPullSecret(ctx context.Context, locoApp *LocoApp, registry DockerRegistryConfig) error
}

// RBACManager handles RBAC operations
type RBACManager interface {
	CreateServiceAccount(ctx context.Context, locoApp *LocoApp) (*corev1.ServiceAccount, error)
	CreateRole(ctx context.Context, locoApp *LocoApp, secret *corev1.Secret) (*rbacv1.Role, error)
	CreateRoleBinding(ctx context.Context, locoApp *LocoApp) (*rbacv1.RoleBinding, error)
}

// GatewayManager handles gateway operations
type GatewayManager interface {
	CreateHTTPRoute(ctx context.Context, locoApp *LocoApp) (*v1Gateway.HTTPRoute, error)
}

// CertificateManager handles certificate operations
type CertificateManager interface {
	GetCertificateExpiry(ctx context.Context, namespace, certName string) (time.Time, error)
}

// LogManager handles logging operations
type LogManager interface {
	GetPodLogs(ctx context.Context, namespace, podName string, tailLines *int64) ([]PodLogLine, error)
	GetLogs(ctx context.Context, namespace, serviceName, username string, tailLines *int64, stream *connect.ServerStream[appv1.LogsResponse]) error
}

// KubernetesClientInterface combines all managers for convenience
type KubernetesClientInterface interface {
	NamespaceManager
	DeploymentManager
	ServiceManager
	SecretManager
	RBACManager
	GatewayManager
	CertificateManager
	LogManager
}
