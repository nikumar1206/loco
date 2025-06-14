package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var gitlabPAT = os.Getenv("GITLAB_PAT")

type AppConfig struct {
	Env             string `json:"env"`             // Environment (e.g., dev, prod)
	ProjectID       string `json:"projectId"`       // GitLab project ID
	RegistryURL     string `json:"registryUrl"`     // Container registry URL
	DeployTokenName string `json:"deployTokenName"` // Deploy token name
	GitlabPAT       string `json:"gitlabPAT"`       // GitLab Personal Access Token
}

func newAppConfig() *AppConfig {
	return &AppConfig{
		Env:             os.Getenv("APP_ENV"),
		ProjectID:       os.Getenv("GITLAB_PROJECT_ID"),
		RegistryURL:     os.Getenv("GITLAB_REGISTRY_URL"),
		DeployTokenName: os.Getenv("GITLAB_DEPLOY_TOKEN_NAME"),
		GitlabPAT:       gitlabPAT,
	}
}

// Response to CLI
type DeployTokenResponse struct {
	Username  string   `json:"username"`
	Password  string   `json:"password"`
	Registry  string   `json:"registry"`
	Image     string   `json:"image"`
	ExpiresAt string   `json:"expiresAt"`
	Revoked   bool     `json:"revoked"`
	Expired   bool     `json:"expired"`
	Scopes    []string `json:"scopes"`
}

func main() {
	app := fiber.New()
	ac := newAppConfig()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Loco Deploy Token Service is running")
	})

	clientSet := buildKubeClientSet(ac)

	pods := clientSet.CoreV1().Pods("loco-setup")

	pl, err := pods.List(context.Background(), metaV1.ListOptions{})
	if err != nil {
		log.Fatalf("Failed to list pods: %v", err)
	}
	log.Printf("Pods in loco-setup namespace: %d", len(pl.Items))

	for _, pod := range pl.Items {
		log.Printf("Pod Name: %s, Status: %s", pod.Name, pod.Status.Phase)
	}

	buildRegistryRouter(app, ac)
	buildKubeRouter(app, ac)

	log.Fatal(app.Listen(":8000"))
}

func buildKubeClientSet(ac *AppConfig) *kubernetes.Clientset {
	var config *rest.Config
	var err error

	if ac.Env == "production" {
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
