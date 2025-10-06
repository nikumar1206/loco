package client

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"path/filepath"
	"time"

	"connectrpc.com/connect"
	json "github.com/goccy/go-json"
	appv1 "github.com/nikumar1206/loco/proto/app/v1"
	"google.golang.org/protobuf/types/known/timestamppb"

	appsV1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacV1 "k8s.io/api/rbac/v1"

	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/nikumar1206/loco/api/pkg/klogmux"
	locoConfig "github.com/nikumar1206/loco/internal/config"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	"k8s.io/client-go/util/homedir"
	v1Gateway "sigs.k8s.io/gateway-api/apis/v1"
	gatewayCs "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
)

var ErrDeploymentNotFound = errors.New("deployment Not Found")

// constants
var (
	LocoGatewayName = "eg"
	LocoNS          = "loco-system"
)

type KubernetesClient struct {
	ClientSet     *kubernetes.Clientset
	GatewaySet    gatewayCs.Interface
	CertManagerCs certmanagerv1.Interface
}

type PodLogLine struct {
	Timestamp time.Time `json:"timestamp"`
	PodName   string    `json:"podId"`
	Log       string    `json:"log"`
}
type DockerRegistryConfig struct {
	Server   string
	Username string
	Password string
	Email    string
}

// NewKubernetesClient initializes a new Kubernetes client based on the application environment.
// for local testing, it points to local kube config, otherwise will
func NewKubernetesClient(appEnv string) *KubernetesClient {
	slog.Info("Initializing Kubernetes client", "env", appEnv)
	config := buildConfig(appEnv)

	clientSet := buildKubeClientSet(config)
	gatewaySet := buildGatewayClient(config)
	certManagerClient := buildCertManagerClient(config)

	return &KubernetesClient{
		ClientSet:     clientSet,
		GatewaySet:    gatewaySet,
		CertManagerCs: certManagerClient,
	}
}

// CheckNSExists checks if a namespace exists in the Kubernetes cluster.
func (kc *KubernetesClient) CheckNSExists(c context.Context, namespace string) (bool, error) {
	slog.DebugContext(c, "Checking if namespace exists", "namespace", namespace)
	namespaces, err := kc.ClientSet.CoreV1().Namespaces().List(c, metaV1.ListOptions{})
	if err != nil {
		slog.Error("Failed to list namespaces", "error", err)
		return false, err
	}

	for _, ns := range namespaces.Items {
		if ns.Name == namespace {
			slog.InfoContext(c, "Namespace exists", "namespace", namespace)
			return true, nil
		}
	}

	slog.InfoContext(c, "Namespace does not exist", "namespace", namespace)
	return false, nil
}

// CreateNS creates a new namespace in the Kubernetes cluster if it does not already exist.
func (kc *KubernetesClient) CreateNS(c context.Context, locoApp *locoConfig.LocoApp) (*v1.Namespace, error) {
	slog.InfoContext(c, "Creating namespace", "namespace", locoApp.Namespace)
	exists, err := kc.CheckNSExists(c, locoApp.Namespace)
	if err != nil {
		return nil, err
	}

	if exists {
		slog.WarnContext(c, "Namespace already exists", "namespace", locoApp.Namespace)
		return nil, nil
	}
	// add label for allowing GW routes
	labels := locoApp.Labels
	labels["expose-via-gw"] = "true"

	nsConfig := &v1.Namespace{
		ObjectMeta: metaV1.ObjectMeta{
			Name:   locoApp.Namespace,
			Labels: labels,
		},
	}

	ns, err := kc.ClientSet.CoreV1().Namespaces().Create(c, nsConfig, metaV1.CreateOptions{})
	if err != nil {
		slog.ErrorContext(c, "Failed to create namespace", "namespace", locoApp.Namespace, "error", err)
		return nil, err
	}

	slog.InfoContext(c, "Namespace created", "namespace", ns.Name)
	return ns, nil
}

// CheckDeploymentExists checks if a deployment exists in the specified namespace.
func (kc *KubernetesClient) CheckDeploymentExists(c context.Context, namespace string, deploymentName string) (bool, error) {
	slog.DebugContext(c, "Checking if deployment exists", "namespace", namespace, "deployment", deploymentName)
	_, err := kc.ClientSet.AppsV1().Deployments(namespace).Get(c, deploymentName, metaV1.GetOptions{})
	if err != nil {
		slog.ErrorContext(c, "Failed to get deployment", "deployment", deploymentName, "namespace", namespace, "error", err)
		return false, ErrDeploymentNotFound
	}
	slog.InfoContext(c, "Deployment exists", "deployment", deploymentName)
	return true, nil
}

// CreateDeployment creates a Deployment if it doesn't exist.
func (kc *KubernetesClient) CreateDeployment(ctx context.Context, locoApp *locoConfig.LocoApp, containerImage string, secrets *v1.Secret) (*appsV1.Deployment, error) {
	slog.InfoContext(ctx, "Creating deployment", "namespace", locoApp.Namespace, "deployment", locoApp.Name)
	existing, err := kc.CheckDeploymentExists(ctx, locoApp.Namespace, locoApp.Name)
	if err != nil {
		if errors.Is(err, ErrDeploymentNotFound) {
			slog.InfoContext(ctx, "deployment doesnt exist")
		} else {
			return nil, err
		}
	}

	if existing {
		slog.WarnContext(ctx, "Deployment already exists", "deployment", locoApp.Name)
		return nil, nil
	}

	replicas := int32(1)
	maxReplicaHistory := int32(2)

	deployment := &appsV1.Deployment{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      locoApp.Name,
			Namespace: locoApp.Namespace,
			Labels:    locoApp.Labels,
		},
		Spec: appsV1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metaV1.LabelSelector{
				MatchLabels: map[string]string{
					locoConfig.LabelAppName: locoApp.Name,
				},
			},
			Strategy: appsV1.DeploymentStrategy{
				Type: appsV1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsV1.RollingUpdateDeployment{
					MaxSurge:       &intstr.IntOrString{Type: intstr.String, StrVal: "25%"},
					MaxUnavailable: &intstr.IntOrString{Type: intstr.String, StrVal: "25%"},
				},
			},
			RevisionHistoryLimit: &maxReplicaHistory,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metaV1.ObjectMeta{
					Labels: locoApp.Labels,
				},
				Spec: v1.PodSpec{
					RestartPolicy: v1.RestartPolicyAlways,
					ImagePullSecrets: []v1.LocalObjectReference{
						{
							Name: fmt.Sprintf("%s-registry-credentials", locoApp.Name),
						},
					},
					ServiceAccountName: locoApp.Name,
					Containers: []v1.Container{
						{
							Name:  locoApp.Name,
							Image: containerImage,
							SecurityContext: &v1.SecurityContext{
								AllowPrivilegeEscalation: ptrToBool(false),
								Privileged:               ptrToBool(false),
								ReadOnlyRootFilesystem:   ptrToBool(true),
								RunAsNonRoot:             ptrToBool(true),
								Capabilities: &v1.Capabilities{
									Drop: []v1.Capability{"ALL"},
								},
							},
							Ports: []v1.ContainerPort{
								{
									ContainerPort: locoApp.Config.Routing.Port,
								},
							},

							EnvFrom: []v1.EnvFromSource{
								{
									SecretRef: &v1.SecretEnvSource{
										LocalObjectReference: v1.LocalObjectReference{
											Name: secrets.Name,
										},
									},
								},
							},
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    resourceMustParse(locoApp.Config.Resources.Cpu),
									v1.ResourceMemory: resourceMustParse(locoApp.Config.Resources.Memory),
								},
								Limits: v1.ResourceList{
									v1.ResourceCPU:    resourceMustParse(locoApp.Config.Resources.Cpu),
									v1.ResourceMemory: resourceMustParse(locoApp.Config.Resources.Memory),
								},
							},
						},
					},
				},
			},
		},
	}

	result, err := kc.ClientSet.AppsV1().Deployments(locoApp.Namespace).Create(ctx, deployment, metaV1.CreateOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create deployment", "deployment", locoApp.Name, "error", err)
		return nil, err
	}

	slog.InfoContext(ctx, "Deployment created", "deployment", result.Name)
	return result, nil
}

func (kc *KubernetesClient) CheckServiceExists(ctx context.Context, namespace, serviceName string) (bool, error) {
	_, err := kc.ClientSet.CoreV1().Services(namespace).Get(ctx, serviceName, metaV1.GetOptions{})
	if err == nil {
		slog.WarnContext(ctx, "Service already exists", "name", serviceName)
		return true, nil
	}
	return false, nil
}

// CreateService creates a Service for the specified deployment in the given namespace.
func (kc *KubernetesClient) CreateService(ctx context.Context, locoApp *locoConfig.LocoApp) (*v1.Service, error) {
	slog.InfoContext(ctx, "Creating service", "namespace", locoApp.Namespace, "name", locoApp.Name)

	kc.CheckServiceExists(ctx, locoApp.Namespace, locoApp.Name)

	timeoutSeconds := int32(10800) // 3 hours

	service := &v1.Service{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      locoApp.Name,
			Namespace: locoApp.Namespace,
		},
		Spec: v1.ServiceSpec{
			Type: v1.ServiceTypeClusterIP,
			Selector: map[string]string{
				locoConfig.LabelAppName: locoApp.Name,
			},
			SessionAffinity: v1.ServiceAffinityNone,
			SessionAffinityConfig: &v1.SessionAffinityConfig{
				ClientIP: &v1.ClientIPConfig{
					TimeoutSeconds: &timeoutSeconds,
				},
			},
			Ports: []v1.ServicePort{
				{
					Name:       locoApp.Name,
					Protocol:   v1.ProtocolTCP,
					Port:       80,
					TargetPort: intstr.FromInt32(locoApp.Config.Routing.Port),
				},
			},
		},
	}

	result, err := kc.ClientSet.CoreV1().Services(locoApp.Namespace).Create(ctx, service, metaV1.CreateOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create service", "name", locoApp.Name, "error", err)
		return nil, err
	}

	slog.InfoContext(ctx, "Service created", "service", result.Name)
	return result, nil
}

func (kc *KubernetesClient) CreateSecret(ctx context.Context, locoApp *locoConfig.LocoApp) (*v1.Secret, error) {
	secretData := make(map[string][]byte)
	for _, envVar := range locoApp.EnvVars {
		secretData[envVar.Name] = []byte(envVar.Value)
	}

	secretConfig := &v1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      locoApp.Name,
			Namespace: locoApp.Namespace,
			Labels:    locoApp.Labels,
		},
		Data: secretData,
		Type: v1.SecretTypeOpaque,
	}

	return kc.ClientSet.CoreV1().Secrets(locoApp.Namespace).Create(ctx, secretConfig, metaV1.CreateOptions{})
}

func (kc *KubernetesClient) UpdateSecret(ctx context.Context, locoApp *locoConfig.LocoApp) (*v1.Secret, error) {
	secretData := make(map[string][]byte)
	for _, envVar := range locoApp.EnvVars {
		secretData[envVar.Name] = []byte(envVar.Value)
	}

	secretConfig := &v1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      locoApp.Name,
			Namespace: locoApp.Namespace,
			Labels:    locoApp.Labels,
		},
		Data: secretData,
		Type: v1.SecretTypeOpaque,
	}

	return kc.ClientSet.CoreV1().Secrets(locoApp.Namespace).Update(ctx, secretConfig, metaV1.UpdateOptions{})
}

// Creates an HTTPRoute given the prov
func (kc *KubernetesClient) CreateHTTPRoute(ctx context.Context, locoApp *locoConfig.LocoApp) (*v1Gateway.HTTPRoute, error) {
	hostname := fmt.Sprintf("%s.deploy-app.com", locoApp.Subdomain)

	pathType := v1Gateway.PathMatchPathPrefix
	route := &v1Gateway.HTTPRoute{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      locoApp.Name,
			Namespace: locoApp.Namespace,
		},

		Spec: v1Gateway.HTTPRouteSpec{
			CommonRouteSpec: v1Gateway.CommonRouteSpec{
				ParentRefs: []v1Gateway.ParentReference{
					{
						Name:      v1Gateway.ObjectName(LocoGatewayName),
						Namespace: ptrToNamespace(LocoNS),
					},
				},
			},
			Hostnames: []v1Gateway.Hostname{v1Gateway.Hostname(hostname)},
			Rules: []v1Gateway.HTTPRouteRule{
				{
					Matches: []v1Gateway.HTTPRouteMatch{
						{
							Path: &v1Gateway.HTTPPathMatch{
								Type:  &pathType,
								Value: ptrToString(locoApp.Config.Routing.PathPrefix),
							},
						},
					},
					BackendRefs: []v1Gateway.HTTPBackendRef{
						{
							BackendRef: v1Gateway.BackendRef{
								BackendObjectReference: v1Gateway.BackendObjectReference{
									Name: v1Gateway.ObjectName(locoApp.Name),
									Port: ptrToPortNumber(80),
									Kind: ptrToKind("Service"),
								},
							},
						},
					},
				},
			},
		},
	}

	createdRoute, err := kc.GatewaySet.GatewayV1().HTTPRoutes(locoApp.Namespace).Create(ctx, route, metaV1.CreateOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create HTTPRoute", "name", locoApp.Name, "error", err)
		return nil, err
	}
	slog.InfoContext(ctx, "HTTPRoute created", "name", locoApp.Name, "hostname", hostname)
	return createdRoute, nil
}

func (kc *KubernetesClient) CreateDockerPullSecret(c context.Context, locoApp *locoConfig.LocoApp, registry DockerRegistryConfig) error {
	auth := map[string]any{
		"auths": map[string]any{
			registry.Server: map[string]string{
				"username": registry.Username,
				"password": registry.Password,
				"email":    registry.Email,
				"auth":     base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", registry.Username, registry.Password))),
			},
		},
	}

	dockerConfigJSON, err := json.Marshal(auth)
	if err != nil {
		slog.ErrorContext(c, err.Error())
		return err
	}

	secretName := fmt.Sprintf("%s-registry-credentials", locoApp.Name)
	secret := &v1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      secretName,
			Namespace: locoApp.Namespace,
			Labels:    locoApp.Labels,
		},
		Type: v1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			".dockerconfigjson": dockerConfigJSON,
		},
	}

	_, err = kc.ClientSet.CoreV1().Secrets(locoApp.Namespace).Create(c, secret, metaV1.CreateOptions{})
	if err != nil {
		slog.ErrorContext(c, err.Error())
		return fmt.Errorf("failed to create secret: %w", err)
	}
	return nil
}

func (kc *KubernetesClient) UpdateDockerPullSecret(c context.Context, locoApp *locoConfig.LocoApp, registry DockerRegistryConfig) error {
	auth := map[string]any{
		"auths": map[string]any{
			registry.Server: map[string]string{
				"username": registry.Username,
				"password": registry.Password,
				"email":    registry.Email,
				"auth":     base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", registry.Username, registry.Password))),
			},
		},
	}

	dockerConfigJSON, err := json.Marshal(auth)
	if err != nil {
		slog.ErrorContext(c, err.Error())
		return err
	}

	secretName := fmt.Sprintf("%s-registry-credentials", locoApp.Name)
	secret := &v1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      secretName,
			Namespace: locoApp.Namespace,
			Labels:    locoApp.Labels,
		},
		Type: v1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			".dockerconfigjson": dockerConfigJSON,
		},
	}

	_, err = kc.ClientSet.CoreV1().Secrets(locoApp.Namespace).Update(c, secret, metaV1.UpdateOptions{})
	if err != nil {
		slog.ErrorContext(c, err.Error())
		return fmt.Errorf("failed to update secret: %w", err)
	}
	return nil
}

// GetPods retrieves a list of pod names in the specified namespace.
// comment in if needed
// func (kc *KubernetesClient) GetPods(namespace string) ([]string, error) {
// 	slog.Debug(ctx, "Fetching pods", "namespace", namespace)
// 	pods, err := kc.ClientSet.CoreV1().Pods(namespace).List(ctx, metaV1.ListOptions{})
// 	if err != nil {
// 		slog.ErrorContext(ctx, "Failed to list pods", "namespace", namespace, "error", err)
// 		return nil, err
// 	}

// 	var podNames []string
// 	for _, pod := range pods.Items {
// 		podNames = append(podNames, pod.Name)
// 	}

// 	slog.InfoContext(ctx, "Retrieved pods", "namespace", namespace, "count", len(podNames))
// 	return podNames, nil
// }

// GetPodLogs retrieves a list of pod names in the specified namespace.
func (kc *KubernetesClient) GetPodLogs(ctx context.Context, namespace, podName string, tailLines *int64) ([]PodLogLine, error) {
	podLogOpts := &v1.PodLogOptions{}
	if tailLines != nil {
		podLogOpts.TailLines = tailLines
	}

	req := kc.ClientSet.CoreV1().Pods(namespace).GetLogs(podName, podLogOpts)

	podLogs, err := req.Stream(ctx)
	if err != nil {
		return nil, fmt.Errorf("error streaming logs for pod %s: %v", podName, err)
	}
	defer podLogs.Close()

	var logs []PodLogLine
	scanner := bufio.NewScanner(podLogs)
	for scanner.Scan() {
		logs = append(logs, PodLogLine{
			Timestamp: time.Now(),
			PodName:   podName,
			Log:       scanner.Text(),
		})
	}

	if err := scanner.Err(); err != nil {
		slog.ErrorContext(ctx, "error reading logs for pod", "pod", podName, "error", err)
		return nil, fmt.Errorf("error reading logs for pod %s: %v", podName, err)
	}

	return logs, nil
}

func (kc *KubernetesClient) GetLogs(
	ctx context.Context,
	namespace,
	serviceName string,
	username string,
	tailLines *int64,
	stream *connect.ServerStream[appv1.LogsResponse],
) error {
	// Build the log stream
	builder := klogmux.NewBuilder(kc.ClientSet).
		Namespace(namespace).
		Follow(true).
		Timestamps(true)

	if tailLines != nil {
		builder = builder.TailLines(*tailLines)
	}

	selector := "app.loco.io/instance=" + serviceName + "-" + username
	if serviceName != "" {
		builder = builder.LabelSelector(selector)
	}

	logStream := builder.Build()

	if err := logStream.Start(ctx); err != nil {
		return fmt.Errorf("failed to start log stream: %w", err)
	}
	defer logStream.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case entry, ok := <-logStream.Entries():
			if !ok {
				// chan closed, stream done
				return nil
			}

			if err := stream.Send(&appv1.LogsResponse{
				Timestamp: timestamppb.New(entry.Timestamp),
				PodName:   entry.PodName,
				Log:       entry.Message,
			}); err != nil {
				return fmt.Errorf("failed to send log entry: %w", err)
			}

		case err, ok := <-logStream.Errors():
			if !ok {
				// err chan closed
				continue
			}
			// todo: this does mean that one error in stream interrupts response
			// maybe later we can handle this better
			return err
		}
	}
}

func (kc *KubernetesClient) UpdateContainer(ctx context.Context, locoApp *locoConfig.LocoApp) error {
	deploymentsClient := kc.ClientSet.AppsV1().Deployments(locoApp.Namespace)

	deployment, err := deploymentsClient.Get(ctx, locoApp.Name, metaV1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get deployment: %v", err)
	}

	if len(deployment.Spec.Template.Spec.Containers) == 0 {
		return fmt.Errorf("deployment has no containers")
	}

	// update secrets
	_, err = kc.UpdateSecret(ctx, locoApp)
	if err != nil {
		return err
	}

	// update container image
	deployment.Spec.Template.Spec.Containers[0].Image = locoApp.ContainerImage

	_, err = deploymentsClient.Update(ctx, deployment, metaV1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update deployment: %v", err)
	}

	return nil
}

func (kc *KubernetesClient) CreateServiceAccount(ctx context.Context, locoApp *locoConfig.LocoApp) (*v1.ServiceAccount, error) {
	sa := &v1.ServiceAccount{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      locoApp.Name,
			Namespace: locoApp.Namespace,
			Labels:    locoApp.Labels,
		},
	}
	return kc.ClientSet.CoreV1().ServiceAccounts(locoApp.Namespace).Create(ctx, sa, metaV1.CreateOptions{})
}

func (kc *KubernetesClient) CreateRole(ctx context.Context, locoApp *locoConfig.LocoApp, secret *v1.Secret) (*rbacV1.Role, error) {
	role := &rbacV1.Role{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      locoApp.Name,
			Namespace: locoApp.Namespace,
			Labels:    locoApp.Labels,
		},
		Rules: []rbacV1.PolicyRule{
			{
				APIGroups:     []string{""},
				Resources:     []string{"secrets"},
				Verbs:         []string{"get", "list", "watch"},
				ResourceNames: []string{secret.Name},
			},
		},
	}
	return kc.ClientSet.RbacV1().Roles(locoApp.Namespace).Create(ctx, role, metaV1.CreateOptions{})
}

func (kc *KubernetesClient) CreateRoleBinding(ctx context.Context, locoApp *locoConfig.LocoApp) (*rbacV1.RoleBinding, error) {
	rb := &rbacV1.RoleBinding{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      locoApp.Name,
			Namespace: locoApp.Namespace,
			Labels:    locoApp.Labels,
		},

		Subjects: []rbacV1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      locoApp.Name,
				Namespace: locoApp.Namespace,
			},
		},
		RoleRef: rbacV1.RoleRef{
			Kind:     "Role",
			Name:     locoApp.Name,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
	return kc.ClientSet.RbacV1().RoleBindings(locoApp.Namespace).Create(ctx, rb, metaV1.CreateOptions{})
}

func (kc *KubernetesClient) GetCertificateExpiry(ctx context.Context, namespace, certName string) (time.Time, error) {
	cert, err := kc.CertManagerCs.CertmanagerV1().Certificates(namespace).Get(ctx, certName, metaV1.GetOptions{})
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get certificate: %v", err)
	}

	if cert.Status.NotAfter == nil {
		return time.Time{}, fmt.Errorf("certificate does not have a NotAfter (expiry) field set")
	}

	return cert.Status.NotAfter.Time, nil
}

func (kc *KubernetesClient) GetDeploymentStatus(ctx context.Context, namespace, appName string) (*appv1.StatusResponse, error) {
	deployment, err := kc.ClientSet.AppsV1().Deployments(namespace).Get(ctx, appName, metaV1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %v", err)
	}

	createdBy := deployment.Labels[locoConfig.LabelAppCreatedFor]
	createdAtStr := deployment.Labels[locoConfig.LabelAppCreatedAt]

	var createdAt time.Time
	if createdAtStr != "" {
		parsedTime, err := time.Parse("20060102T150405Z", createdAtStr)
		if err == nil {
			createdAt = parsedTime
		} else {
			createdAt = deployment.CreationTimestamp.Time // fallback
		}
	} else {
		createdAt = deployment.CreationTimestamp.Time // fallback
	}

	pods, err := kc.ClientSet.CoreV1().Pods(namespace).List(ctx, metaV1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", locoConfig.LabelAppName, appName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %v", err)
	}

	httpRoute, err := kc.GatewaySet.GatewayV1().HTTPRoutes(namespace).Get(ctx, appName, metaV1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get HTTPRoute: %v", err)
	}
	hostname := ""
	if len(httpRoute.Spec.Hostnames) > 0 {
		hostname = string(httpRoute.Spec.Hostnames[0])
	}

	hpa, err := kc.ClientSet.AutoscalingV2().HorizontalPodAutoscalers(namespace).Get(ctx, appName, metaV1.GetOptions{})
	autoscalingEnabled := false
	var minReplicas, maxReplicas int32
	if err == nil {
		autoscalingEnabled = true
		if hpa.Spec.MinReplicas != nil {
			minReplicas = *hpa.Spec.MinReplicas
		}
		maxReplicas = hpa.Spec.MaxReplicas
	}

	status := "Running"
	if deployment.Status.ReadyReplicas != *deployment.Spec.Replicas {
		status = "Pending"
	}

	health := "Passing"
	for _, pod := range pods.Items {
		for _, condition := range pod.Status.Conditions {
			if condition.Type == v1.PodReady && condition.Status != v1.ConditionTrue {
				health = "Degraded"
				break
			}
		}
	}

	certExpiry := "Unknown"
	expiryTime, err := kc.GetCertificateExpiry(ctx, namespace, appName)
	if err == nil {
		certExpiry = expiryTime.Format("2006-01-02")
	} else {
		slog.WarnContext(ctx, "Failed to get certificate expiry", "error", err)
	}

	return &appv1.StatusResponse{
		Status:          status,
		Pods:            int32(len(pods.Items)),
		CpuUsage:        "N/A",
		MemoryUsage:     "N/A",
		Latency:         "N/A",
		Url:             fmt.Sprintf("https://%s", hostname),
		DeployedAt:      timestamppb.New(createdAt),
		DeployedBy:      createdBy,
		Tls:             fmt.Sprintf("Secured (Expires: %s)", certExpiry),
		Health:          health,
		Autoscaling:     autoscalingEnabled,
		MinReplicas:     minReplicas,
		MaxReplicas:     maxReplicas,
		DesiredReplicas: *deployment.Spec.Replicas,
		ReadyReplicas:   deployment.Status.ReadyReplicas,
	}, nil
}

// buildConfig uses in-cluster kube config in production, otherwise uses local.
func buildConfig(appEnv string) *rest.Config {
	var config *rest.Config
	var err error

	if appEnv == "PRODUCTION" {
		slog.Info("Using in-cluster Kubernetes config")
		config, err = rest.InClusterConfig()
		if err != nil {
			slog.Error("Failed to create in-cluster config", "error", err)
			log.Fatalf("Failed to create in-cluster config: %v", err)
		}
	} else {
		slog.Info("Using local kubeconfig")
		var kubeconfig *string
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "kubeconfig path")
		} else {
			kubeconfig = flag.String("kubeconfig", "", "kubeconfig path")
		}
		flag.Parse()

		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			slog.Error("Failed to build kubeconfig", "error", err)
			log.Fatalf("Failed to build kubeconfig: %v", err)
		}
	}

	return config
}

func buildKubeClientSet(config *rest.Config) *kubernetes.Clientset {
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		slog.Error("Failed to create Kubernetes client", "error", err)
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	slog.Info("Kubernetes client initialized")
	return clientSet
}

func buildGatewayClient(config *rest.Config) *gatewayCs.Clientset {
	gwcs, err := gatewayCs.NewForConfig(config)
	if err != nil {
		slog.Error("Failed to create gateway client", "error", err)
		log.Fatalf("Failed to create gateway client: %v", err)
	}

	slog.Info("Gateway client initialized")
	return gwcs
}

func buildCertManagerClient(config *rest.Config) certmanagerv1.Interface {
	certClient, err := certmanagerv1.NewForConfig(config)
	if err != nil {
		slog.Error("Failed to create cert-manager client", "error", err)
		log.Fatalf("Failed to create cert-manager client: %v", err)
	}
	return certClient
}

func resourceMustParse(value string) resource.Quantity {
	q, err := resource.ParseQuantity(value)
	if err != nil {
		panic(err)
	}
	return q
}

func ptrToString(s string) *string { return &s }

func ptrToPortNumber(p int) *v1Gateway.PortNumber {
	n := v1Gateway.PortNumber(p)
	return &n
}

func ptrToNamespace(n string) *v1Gateway.Namespace {
	ns := v1Gateway.Namespace(n)
	return &ns
}

func ptrToKind(k string) *v1Gateway.Kind {
	t := v1Gateway.Kind(k)
	return &t
}

func ptrToBool(b bool) *bool { return &b }
