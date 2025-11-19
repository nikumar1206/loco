package kube

import (
	"log/slog"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	gatewayCs "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
)

// Client implements Kubernetes operations for deployments
type Client struct {
	ClientSet  kubernetes.Interface
	GatewaySet gatewayCs.Interface
}

// NewClient initializes a new Kubernetes client based on the application environment.
// For local development, it uses the kubeconfig file. For production, it uses in-cluster config.
// todo: this is being called twice,
func NewClient(appEnv string) *Client {
	slog.Info("Initializing Kubernetes client", "env", appEnv)
	config := buildConfig(appEnv)

	clientSet := buildKubeClientSet(config)
	gatewaySet := buildGatewayClient(config)

	return &Client{
		ClientSet:  clientSet,
		GatewaySet: gatewaySet,
	}
}

// buildConfig creates a Kubernetes config using in-cluster config for production,
// or local kubeconfig for development.
func buildConfig(appEnv string) *rest.Config {
	if appEnv == "PRODUCTION" {
		slog.Info("Using in-cluster Kubernetes config")
		config, err := rest.InClusterConfig()
		if err != nil {
			slog.Error("Failed to create in-cluster config", "error", err)
			panic(err)
		}
		return config
	}

	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{}

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
	if err != nil {
		panic(err)
	}
	return config
}

func buildKubeClientSet(config *rest.Config) kubernetes.Interface {
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		slog.Error("Failed to create Kubernetes client", "error", err)
		panic(err)
	}

	slog.Info("Kubernetes client initialized")
	return clientSet
}

func buildGatewayClient(config *rest.Config) gatewayCs.Interface {
	gwcs, err := gatewayCs.NewForConfig(config)
	if err != nil {
		slog.Error("Failed to create gateway client", "error", err)
		panic(err)
	}

	slog.Info("Gateway client initialized")
	return gwcs
}
