package kube

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"

	json "github.com/goccy/go-json"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateSecret creates a Kubernetes Secret for environment variables
func (kc *KubernetesClient) CreateSecret(ctx context.Context, ldc *LocoDeploymentContext, envVars map[string]string) (*v1.Secret, error) {
	slog.InfoContext(ctx, "Creating secret", "namespace", ldc.Namespace(), "name", ldc.EnvSecretName())

	secretData := make(map[string][]byte)
	for key, value := range envVars {
		secretData[key] = []byte(value)
	}

	secret := &v1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      ldc.EnvSecretName(),
			Namespace: ldc.Namespace(),
			Labels:    ldc.Labels(),
		},
		Data: secretData,
		Type: v1.SecretTypeOpaque,
	}

	result, err := kc.ClientSet.CoreV1().Secrets(ldc.Namespace()).Create(ctx, secret, metaV1.CreateOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create secret", "name", ldc.EnvSecretName(), "error", err)
		return nil, fmt.Errorf("failed to create secret: %w", err)
	}

	slog.InfoContext(ctx, "Secret created", "name", result.Name)
	return result, nil
}

// UpdateSecret updates a Kubernetes Secret for environment variables
func (kc *KubernetesClient) UpdateSecret(ctx context.Context, ldc *LocoDeploymentContext, envVars map[string]string) (*v1.Secret, error) {
	slog.InfoContext(ctx, "Updating secret", "namespace", ldc.Namespace(), "name", ldc.EnvSecretName())

	secretsClient := kc.ClientSet.CoreV1().Secrets(ldc.Namespace())
	secret, err := secretsClient.Get(ctx, ldc.EnvSecretName(), metaV1.GetOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get secret", "name", ldc.EnvSecretName(), "error", err)
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}

	for key, value := range envVars {
		secret.Data[key] = []byte(value)
	}

	result, err := secretsClient.Update(ctx, secret, metaV1.UpdateOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to update secret", "name", ldc.EnvSecretName(), "error", err)
		return nil, fmt.Errorf("failed to update secret: %w", err)
	}

	slog.InfoContext(ctx, "Secret updated", "name", result.Name)
	return result, nil
}

// CreateDockerPullSecret creates a Kubernetes Secret for Docker registry credentials
func (kc *KubernetesClient) CreateDockerPullSecret(ctx context.Context, ldc *LocoDeploymentContext, registry DockerRegistryConfig) error {
	slog.InfoContext(ctx, "Creating docker pull secret", "namespace", ldc.Namespace(), "registry", registry.Server)

	auth := map[string]any{
		"auths": map[string]any{
			registry.Server: map[string]string{
				"username": registry.Username,
				"password": registry.Password,
				"email":    registry.Email,
				"auth": base64.StdEncoding.EncodeToString(
					[]byte(fmt.Sprintf("%s:%s", registry.Username, registry.Password)),
				),
			},
		},
	}

	dockerConfigJSON, err := json.Marshal(auth)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to marshal docker config", "error", err)
		return fmt.Errorf("failed to marshal docker config: %w", err)
	}

	secret := &v1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      ldc.RegistrySecretName(),
			Namespace: ldc.Namespace(),
			Labels:    ldc.Labels(),
		},
		Type: v1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			".dockerconfigjson": dockerConfigJSON,
		},
	}

	_, err = kc.ClientSet.CoreV1().Secrets(ldc.Namespace()).Create(ctx, secret, metaV1.CreateOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create docker pull secret", "error", err)
		return fmt.Errorf("failed to create docker pull secret: %w", err)
	}

	slog.InfoContext(ctx, "Docker pull secret created", "name", ldc.RegistrySecretName())
	return nil
}

// UpdateDockerPullSecret updates a Kubernetes Secret for Docker registry credentials
func (kc *KubernetesClient) UpdateDockerPullSecret(ctx context.Context, ldc *LocoDeploymentContext, registry DockerRegistryConfig) error {
	slog.InfoContext(ctx, "Updating docker pull secret", "namespace", ldc.Namespace(), "registry", registry.Server)

	auth := map[string]any{
		"auths": map[string]any{
			registry.Server: map[string]string{
				"username": registry.Username,
				"password": registry.Password,
				"email":    registry.Email,
				"auth": base64.StdEncoding.EncodeToString(
					[]byte(fmt.Sprintf("%s:%s", registry.Username, registry.Password)),
				),
			},
		},
	}

	dockerConfigJSON, err := json.Marshal(auth)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to marshal docker config", "error", err)
		return fmt.Errorf("failed to marshal docker config: %w", err)
	}

	secret := &v1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      ldc.RegistrySecretName(),
			Namespace: ldc.Namespace(),
			Labels:    ldc.Labels(),
		},
		Type: v1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			".dockerconfigjson": dockerConfigJSON,
		},
	}

	_, err = kc.ClientSet.CoreV1().Secrets(ldc.Namespace()).Update(ctx, secret, metaV1.UpdateOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to update docker pull secret", "error", err)
		return fmt.Errorf("failed to update docker pull secret: %w", err)
	}

	slog.InfoContext(ctx, "Docker pull secret updated", "name", ldc.RegistrySecretName())
	return nil
}
