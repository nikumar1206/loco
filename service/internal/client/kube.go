package client

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"path/filepath"

	appsV1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	v1Gateway "sigs.k8s.io/gateway-api/apis/v1"
	gatewayCs "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
)

var ErrDeploymentNotFound = errors.New("deployment Not Found")

// constants
var (
	LocoGatewayName = "eg"
	LocoSetupNS     = "loco-setup"
)

type KubernetesClient struct {
	ClientSet  *kubernetes.Clientset
	GatewaySet gatewayCs.Interface
}

// NewKubernetesClient initializes a new Kubernetes client based on the application environment.
func NewKubernetesClient(appEnv string) *KubernetesClient {
	slog.Info("Initializing Kubernetes client", "env", appEnv)
	config := buildConfig(appEnv)

	clientSet := buildKubeClientSet(config)
	gatewaySet := buildGatewayClient(config)

	return &KubernetesClient{
		ClientSet:  clientSet,
		GatewaySet: gatewaySet,
	}
}

// CheckNSExists checks if a namespace exists in the Kubernetes cluster.
func (kc *KubernetesClient) CheckNSExists(c context.Context, namespace string) (bool, error) {
	slog.Debug("Checking if namespace exists", "namespace", namespace)
	namespaces, err := kc.ClientSet.CoreV1().Namespaces().List(c, metaV1.ListOptions{})
	if err != nil {
		slog.Error("Failed to list namespaces", "error", err)
		return false, err
	}

	for _, ns := range namespaces.Items {
		if ns.Name == namespace {
			slog.Info("Namespace exists", "namespace", namespace)
			return true, nil
		}
	}

	slog.Info("Namespace does not exist", "namespace", namespace)
	return false, nil
}

// CreateNS creates a new namespace in the Kubernetes cluster if it does not already exist.
func (kc *KubernetesClient) CreateNS(c context.Context, locoApp *LocoApp) (*v1.Namespace, error) {
	slog.Info("Creating namespace", "namespace", locoApp.Namespace)
	exists, err := kc.CheckNSExists(c, locoApp.Namespace)
	if err != nil {
		return nil, err
	}

	if exists {
		slog.Warn("Namespace already exists", "namespace", locoApp.Namespace)
		return nil, nil
	}

	nsConfig := &v1.Namespace{
		ObjectMeta: metaV1.ObjectMeta{
			Name:   locoApp.Namespace,
			Labels: locoApp.Labels,
		},
	}

	ns, err := kc.ClientSet.CoreV1().Namespaces().Create(c, nsConfig, metaV1.CreateOptions{})
	if err != nil {
		slog.Error("Failed to create namespace", "namespace", locoApp.Namespace, "error", err)
		return nil, err
	}

	slog.Info("Namespace created", "namespace", ns.Name)
	return ns, nil
}

// CheckDeploymentExists checks if a deployment exists in the specified namespace.
func (kc *KubernetesClient) CheckDeploymentExists(c context.Context, namespace string, deploymentName string) (bool, error) {
	slog.Debug("Checking if deployment exists", "namespace", namespace, "deployment", deploymentName)
	_, err := kc.ClientSet.AppsV1().Deployments(namespace).Get(c, deploymentName, metaV1.GetOptions{})
	if err != nil {
		slog.Error("Failed to get deployment", "deployment", deploymentName, "namespace", namespace, "error", err)
		return false, ErrDeploymentNotFound
	}
	slog.Info("Deployment exists", "deployment", deploymentName)
	return true, nil
}

// CreateDeployment creates a Deployment if it doesn't exist.
func (kc *KubernetesClient) CreateDeployment(ctx context.Context, locoApp *LocoApp) (*appsV1.Deployment, error) {
	slog.Info("Creating deployment", "namespace", locoApp.Namespace, "deployment", locoApp.Name)
	existing, err := kc.CheckDeploymentExists(ctx, locoApp.Namespace, locoApp.Name)
	if err != nil {
		if errors.Is(err, ErrDeploymentNotFound) {
			slog.Info("deployment doesnt exist")
		} else {
			return nil, err
		}
	}

	if existing {
		slog.Warn("Deployment already exists", "deployment", locoApp.Name)
		return nil, nil
	}

	replicas := int32(1)

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
					LabelAppName: locoApp.Name,
				},
			},
			Strategy: appsV1.DeploymentStrategy{
				Type: appsV1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsV1.RollingUpdateDeployment{
					MaxSurge:       &intstr.IntOrString{Type: intstr.String, StrVal: "25%"},
					MaxUnavailable: &intstr.IntOrString{Type: intstr.String, StrVal: "25%"},
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metaV1.ObjectMeta{
					Labels: locoApp.Labels,
				},
				Spec: v1.PodSpec{
					RestartPolicy: v1.RestartPolicyAlways,
					// todo eventually replace with actual container image
					Containers: []v1.Container{
						{
							Name:    locoApp.Name,
							Image:   "alpine",
							Command: []string{"printenv"},
							Ports: []v1.ContainerPort{
								{
									ContainerPort: int32(locoApp.Config.Port),
								},
							},
							Env: createKubeEnvVars(locoApp.EnvVars),
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    resourceMustParse(locoApp.Config.CPU),
									v1.ResourceMemory: resourceMustParse(locoApp.Config.Memory),
								},
								Limits: v1.ResourceList{
									v1.ResourceCPU:    resourceMustParse(locoApp.Config.CPU),
									v1.ResourceMemory: resourceMustParse(locoApp.Config.Memory),
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
		slog.Error("Failed to create deployment", "deployment", locoApp.Name, "error", err)
		return nil, err
	}

	slog.Info("Deployment created", "deployment", result.Name)
	return result, nil
}

func (kc *KubernetesClient) CheckServiceExists(ctx context.Context, namespace, serviceName string) (bool, error) {
	_, err := kc.ClientSet.CoreV1().Services(namespace).Get(ctx, serviceName, metaV1.GetOptions{})
	if err == nil {
		slog.Warn("Service already exists", "name", serviceName)
		return true, nil
	}
	return false, nil
}

// CreateService creates a Service for the specified deployment in the given namespace.
func (kc *KubernetesClient) CreateService(ctx context.Context, locoApp *LocoApp) (*v1.Service, error) {
	slog.Info("Creating service", "namespace", locoApp.Namespace, "name", locoApp.Name)

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
				LabelAppName: locoApp.Name,
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
					TargetPort: intstr.FromInt(locoApp.Config.Port),
				},
			},
		},
	}

	result, err := kc.ClientSet.CoreV1().Services(locoApp.Namespace).Create(ctx, service, metaV1.CreateOptions{})
	if err != nil {
		slog.Error("Failed to create service", "name", locoApp.Name, "error", err)
		return nil, err
	}

	slog.Info("Service created", "service", result.Name)
	return result, nil
}

// Creates an HTTPRoute given the prov
func (kc *KubernetesClient) CreateHTTPRoute(ctx context.Context, locoApp *LocoApp) (*v1Gateway.HTTPRoute, error) {
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
						Namespace: ptrToNamespace(LocoSetupNS),
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
								Value: ptrToString("/"),
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

	return kc.GatewaySet.GatewayV1().HTTPRoutes(route.Namespace).Create(ctx, route, metaV1.CreateOptions{})
}

// GetPods retrieves a list of pod names in the specified namespace.
func (kc *KubernetesClient) GetPods(namespace string) ([]string, error) {
	slog.Debug("Fetching pods", "namespace", namespace)
	pods, err := kc.ClientSet.CoreV1().Pods(namespace).List(context.Background(), metaV1.ListOptions{})
	if err != nil {
		slog.Error("Failed to list pods", "namespace", namespace, "error", err)
		return nil, err
	}

	var podNames []string
	for _, pod := range pods.Items {
		podNames = append(podNames, pod.Name)
	}

	slog.Info("Retrieved pods", "namespace", namespace, "count", len(podNames))
	return podNames, nil
}

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

func createKubeEnvVars(envVars []EnvVar) []v1.EnvVar {
	kubeEnvVars := []v1.EnvVar{}

	for _, ev := range envVars {
		kubeEnvVars = append(kubeEnvVars, v1.EnvVar{
			Name:  ev.Name,
			Value: ev.Value,
		})
	}

	return kubeEnvVars
}
