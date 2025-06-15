package clients

import (
	"context"
	"flag"
	"fmt"
	"log"
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

type KubernetesClient struct {
	ClientSet *kubernetes.Clientset
}

// NewKubernetesClient initializes a new Kubernetes client based on the application environment.
func NewKubernetesClient(appEnv string) *KubernetesClient {
	clientSet := buildKubeClientSet(appEnv)
	return &KubernetesClient{
		ClientSet: clientSet,
	}
}

// CheckNSExists checks if a namespace exists in the Kubernetes cluster.
func (kc *KubernetesClient) CheckNSExists(c context.Context, namespace string) (bool, error) {
	namespaces, err := kc.ClientSet.CoreV1().Namespaces().List(c, metaV1.ListOptions{})
	if err != nil {
		return false, err
	}

	for _, ns := range namespaces.Items {
		if ns.Name == namespace {
			return true, nil
		}
	}

	return false, nil
}

// CreateNS creates a new namespace in the Kubernetes cluster if it does not already exist.
func (kc *KubernetesClient) CreateNS(c context.Context, namespace string) (*v1.Namespace, error) {
	exists, err := kc.CheckNSExists(c, namespace)
	if err != nil {
		return nil, err
	}

	if exists {
		return nil, nil
	}

	nsName := &v1.Namespace{
		ObjectMeta: metaV1.ObjectMeta{
			Name: namespace,
		},
	}

	return kc.ClientSet.CoreV1().Namespaces().Create(c, nsName, metaV1.CreateOptions{})
}

// CheckDeploymentExists checks if a deployment exists in the specified namespace.
func (kc *KubernetesClient) CheckDeploymentExists(c context.Context, namespace string, deploymentName string) (*appsV1.Deployment, error) {
	deployment, err := kc.ClientSet.AppsV1().Deployments(namespace).Get(c, deploymentName, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return deployment, nil
}

// CreateDeployment creates a Deployment if it doesn't exist.
func (kc *KubernetesClient) CreateDeployment(ctx context.Context, namespace string, deploymentName string) (*appsV1.Deployment, error) {
	existing, err := kc.CheckDeploymentExists(ctx, namespace, deploymentName)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
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

	return kc.ClientSet.AppsV1().Deployments(namespace).Create(ctx, deployment, metaV1.CreateOptions{})
}

// CreateService creates a Service for the specified deployment in the given namespace.
func (kc *KubernetesClient) CreateService(ctx context.Context, namespace, name string) (*v1.Service, error) {
	existing, err := kc.ClientSet.CoreV1().Services(namespace).Get(ctx, name, metaV1.GetOptions{})
	if err == nil {
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

	return kc.ClientSet.CoreV1().Services(namespace).Create(ctx, service, metaV1.CreateOptions{})
}

// GetPods retrieves a list of pod names in the specified namespace.
func (kc *KubernetesClient) GetPods(namespace string) ([]string, error) {
	pods, err := kc.ClientSet.CoreV1().Pods(namespace).List(context.Background(), metaV1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var podNames []string
	for _, pod := range pods.Items {
		podNames = append(podNames, pod.Name)
	}
	return podNames, nil
}

func buildKubeClientSet(appEnv string) *kubernetes.Clientset {
	var config *rest.Config
	var err error

	if appEnv == "production" {
		// Use in-cluster config in production
		config, err = rest.InClusterConfig()
		if err != nil {
			log.Fatalf("Failed to create in-cluster config: %v", err)
		}
	} else {
		// Use kubeconfig from file in other environments
		var kubeconfig *string
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}
		flag.Parse()

		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			log.Fatalf("Failed to build kubeconfig: %v", err)
		}
	}

	// Initialize Kubernetes client
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}
	return clientSet
}
