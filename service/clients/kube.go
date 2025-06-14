package main

import (
	"context"
	"flag"
	"log"
	"path/filepath"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type KubernetesClient struct {
	ClientSet *kubernetes.Clientset
}

func NewKubernetesClient(appEnv string) *KubernetesClient {
	clientSet := buildKubeClientSet(appEnv)
	return &KubernetesClient{
		ClientSet: clientSet,
	}
}

func (kc *KubernetesClient) CheckNSExists(namespace string) (bool, error) {
	namespaces, err := kc.ClientSet.CoreV1().Namespaces().List(context.Background(), metaV1.ListOptions{})
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
