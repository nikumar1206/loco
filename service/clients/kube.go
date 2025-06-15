package clients

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
)

var ErrDeploymentNotFound = errors.New("deployment Not Found")

type KubernetesClient struct {
	ClientSet *kubernetes.Clientset
}

// NewKubernetesClient initializes a new Kubernetes client based on the application environment.
func NewKubernetesClient(appEnv string) *KubernetesClient {
	slog.Info("Initializing Kubernetes client", "env", appEnv)
	clientSet := buildKubeClientSet(appEnv)
	return &KubernetesClient{
		ClientSet: clientSet,
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
func (kc *KubernetesClient) CreateNS(c context.Context, namespace string) (*v1.Namespace, error) {
	slog.Info("Creating namespace", "namespace", namespace)
	exists, err := kc.CheckNSExists(c, namespace)
	if err != nil {
		return nil, err
	}

	if exists {
		slog.Warn("Namespace already exists", "namespace", namespace)
		return nil, nil
	}

	nsName := &v1.Namespace{
		ObjectMeta: metaV1.ObjectMeta{
			Name: namespace,
		},
	}

	ns, err := kc.ClientSet.CoreV1().Namespaces().Create(c, nsName, metaV1.CreateOptions{})
	if err != nil {
		slog.Error("Failed to create namespace", "namespace", namespace, "error", err)
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
func (kc *KubernetesClient) CreateDeployment(ctx context.Context, namespace string, deploymentName string) (*appsV1.Deployment, error) {
	slog.Info("Creating deployment", "namespace", namespace, "deployment", deploymentName)
	existing, err := kc.CheckDeploymentExists(ctx, namespace, deploymentName)
	if err != nil {
		if errors.Is(err, ErrDeploymentNotFound) {
			slog.Info("deployment doesnt exist")
		} else {
			return nil, err
		}
	}

	if existing {
		slog.Warn("Deployment already exists", "deployment", deploymentName)
		return nil, nil
	}

	replicas := int32(1)

	deployment := &appsV1.Deployment{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      deploymentName,
			Namespace: namespace,
			Labels: map[string]string{
				"app": deploymentName,
			},
		},
		Spec: appsV1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metaV1.LabelSelector{
				MatchLabels: map[string]string{
					"app": deploymentName,
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
					Labels: map[string]string{
						"app": deploymentName,
					},
				},
				Spec: v1.PodSpec{
					RestartPolicy: v1.RestartPolicyAlways,
					Containers: []v1.Container{
						{
							Name:  deploymentName,
							Image: "hashicorp/http-echo",
							Args: []string{
								fmt.Sprintf("-text=Hello from %s!", deploymentName),
							},
							Ports: []v1.ContainerPort{
								{
									ContainerPort: 5678,
								},
							},
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    resourceMustParse("100m"),
									v1.ResourceMemory: resourceMustParse("100Mi"),
								},
								Limits: v1.ResourceList{
									v1.ResourceCPU:    resourceMustParse("100m"),
									v1.ResourceMemory: resourceMustParse("100Mi"),
								},
							},
						},
					},
				},
			},
		},
	}

	result, err := kc.ClientSet.AppsV1().Deployments(namespace).Create(ctx, deployment, metaV1.CreateOptions{})
	if err != nil {
		slog.Error("Failed to create deployment", "deployment", deploymentName, "error", err)
		return nil, err
	}

	slog.Info("Deployment created", "deployment", result.Name)
	return result, nil
}

// CreateService creates a Service for the specified deployment in the given namespace.
func (kc *KubernetesClient) CreateService(ctx context.Context, namespace, name string) (*v1.Service, error) {
	slog.Info("Creating service", "namespace", namespace, "name", name)
	existing, err := kc.ClientSet.CoreV1().Services(namespace).Get(ctx, name, metaV1.GetOptions{})
	if err == nil {
		slog.Warn("Service already exists", "name", name)
		return existing, nil
	}

	timeoutSeconds := int32(10800) // 3 hours

	service := &v1.Service{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.ServiceSpec{
			Type: v1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app": name,
			},
			SessionAffinity: v1.ServiceAffinityNone,
			SessionAffinityConfig: &v1.SessionAffinityConfig{
				ClientIP: &v1.ClientIPConfig{
					TimeoutSeconds: &timeoutSeconds,
				},
			},
			Ports: []v1.ServicePort{
				{
					Name:       name,
					Protocol:   v1.ProtocolTCP,
					Port:       80,
					TargetPort: intstr.FromInt(5678),
				},
			},
		},
	}

	result, err := kc.ClientSet.CoreV1().Services(namespace).Create(ctx, service, metaV1.CreateOptions{})
	if err != nil {
		slog.Error("Failed to create service", "name", name, "error", err)
		return nil, err
	}

	slog.Info("Service created", "service", result.Name)
	return result, nil
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

func buildKubeClientSet(appEnv string) *kubernetes.Clientset {
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

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		slog.Error("Failed to create Kubernetes client", "error", err)
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	slog.Info("Kubernetes client initialized")
	return clientSet
}
