package kube

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	appsV1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// CheckDeploymentExists checks if a Deployment exists in the specified namespace
func (kc *KubernetesClient) CheckDeploymentExists(ctx context.Context, namespace string, deploymentName string) (bool, error) {
	slog.DebugContext(ctx, "Checking if deployment exists", "namespace", namespace, "deployment", deploymentName)
	_, err := kc.ClientSet.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metaV1.GetOptions{})
	if err != nil {
		slog.DebugContext(ctx, "Deployment does not exist", "deployment", deploymentName, "namespace", namespace)
		return false, nil
	}
	slog.InfoContext(ctx, "Deployment exists", "deployment", deploymentName)
	return true, nil
}

// CreateDeployment creates a Kubernetes Deployment
func (kc *KubernetesClient) CreateDeployment(ctx context.Context, ldc *LocoDeploymentContext) (*appsV1.Deployment, error) {
	slog.InfoContext(ctx, "Creating deployment", "namespace", ldc.Namespace(), "deployment", ldc.DeploymentName())

	existing, err := kc.CheckDeploymentExists(ctx, ldc.Namespace(), ldc.DeploymentName())
	if err != nil {
		return nil, fmt.Errorf("failed to check deployment existence: %w", err)
	}

	if existing {
		slog.WarnContext(ctx, "Deployment already exists", "deployment", ldc.DeploymentName())
		return nil, fmt.Errorf("deployment already exists: %s", ldc.DeploymentName())
	}

	replicas := int32(ldc.Deployment.Replicas)
	if replicas == 0 {
		replicas = DefaultReplicas
	}

	cpuQuantity, err := resource.ParseQuantity(ldc.Config.Resources.CPU)
	if err != nil {
		slog.ErrorContext(ctx, "Invalid CPU value", "cpu", ldc.Config.Resources.CPU, "error", err)
		return nil, fmt.Errorf("invalid cpu value: %w", err)
	}

	memoryQuantity, err := resource.ParseQuantity(ldc.Config.Resources.Memory)
	if err != nil {
		slog.ErrorContext(ctx, "Invalid memory value", "memory", ldc.Config.Resources.Memory, "error", err)
		return nil, fmt.Errorf("invalid memory value: %w", err)
	}

	deployment := &appsV1.Deployment{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      ldc.DeploymentName(),
			Namespace: ldc.Namespace(),
			Labels:    ldc.Labels(),
		},
		Spec: appsV1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metaV1.LabelSelector{
				MatchLabels: map[string]string{
					LabelAppName: ldc.App.Name,
				},
			},
			Strategy: appsV1.DeploymentStrategy{
				Type: appsV1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsV1.RollingUpdateDeployment{
					MaxSurge:       &intstr.IntOrString{Type: intstr.String, StrVal: MaxSurgePercent},
					MaxUnavailable: &intstr.IntOrString{Type: intstr.String, StrVal: MaxUnavailablePercent},
				},
			},
			RevisionHistoryLimit: ptrToInt32(MaxReplicaHistory),
			Template: v1.PodTemplateSpec{
				ObjectMeta: metaV1.ObjectMeta{
					Labels: ldc.Labels(),
				},
				Spec: v1.PodSpec{
					RestartPolicy: v1.RestartPolicyAlways,
					ImagePullSecrets: []v1.LocalObjectReference{
						{
							Name: ldc.RegistrySecretName(),
						},
					},
					ServiceAccountName: ldc.ServiceAccountName(),
					Containers: []v1.Container{
						{
							Name:  ldc.ContainerName(),
							Image: ldc.Deployment.Image,
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
									ContainerPort: ldc.Config.Routing.Port,
								},
							},
							EnvFrom: []v1.EnvFromSource{
								{
									SecretRef: &v1.SecretEnvSource{
										LocalObjectReference: v1.LocalObjectReference{
											Name: ldc.EnvSecretName(),
										},
									},
								},
							},
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    cpuQuantity,
									v1.ResourceMemory: memoryQuantity,
								},
								Limits: v1.ResourceList{
									v1.ResourceCPU:    cpuQuantity,
									v1.ResourceMemory: memoryQuantity,
								},
							},
							LivenessProbe: &v1.Probe{
								InitialDelaySeconds:           ldc.Config.Health.StartupGracePeriod,
								TimeoutSeconds:                ldc.Config.Health.Timeout,
								PeriodSeconds:                 ldc.Config.Health.Interval,
								TerminationGracePeriodSeconds: ptrToInt64(TerminationGracePeriod),
								SuccessThreshold:              1,
								FailureThreshold:              ldc.Config.Health.FailThreshold,
								ProbeHandler: v1.ProbeHandler{
									HTTPGet: &v1.HTTPGetAction{
										Path: ldc.Config.Health.Path,
										Port: intstr.FromInt32(ldc.Config.Routing.Port),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	result, err := kc.ClientSet.AppsV1().Deployments(ldc.Namespace()).Create(ctx, deployment, metaV1.CreateOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create deployment", "deployment", ldc.DeploymentName(), "error", err)
		return nil, fmt.Errorf("failed to create deployment: %w", err)
	}

	slog.InfoContext(ctx, "Deployment created", "deployment", result.Name)
	return result, nil
}

// UpdateContainer updates the container image in an existing Deployment
func (kc *KubernetesClient) UpdateContainer(ctx context.Context, ldc *LocoDeploymentContext) error {
	slog.InfoContext(ctx, "Updating container image", "namespace", ldc.Namespace(), "deployment", ldc.DeploymentName())

	deploymentsClient := kc.ClientSet.AppsV1().Deployments(ldc.Namespace())

	deployment, err := deploymentsClient.Get(ctx, ldc.DeploymentName(), metaV1.GetOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get deployment", "error", err)
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	if len(deployment.Spec.Template.Spec.Containers) == 0 {
		return fmt.Errorf("deployment has no containers")
	}

	deployment.Spec.Template.Spec.Containers[0].Image = ldc.Deployment.Image

	_, err = deploymentsClient.Update(ctx, deployment, metaV1.UpdateOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to update deployment", "error", err)
		return fmt.Errorf("failed to update deployment: %w", err)
	}

	slog.InfoContext(ctx, "Container image updated", "image", ldc.Deployment.Image)
	return nil
}

// ScaleDeployment scales a Deployment to a specific number of replicas
func (kc *KubernetesClient) ScaleDeployment(ctx context.Context, namespace, appName string, replicas *int32, cpu, memory *string) error {
	slog.InfoContext(ctx, "Scaling deployment", "namespace", namespace, "app", appName, "replicas", replicas)

	deploymentsClient := kc.ClientSet.AppsV1().Deployments(namespace)

	deployment, err := deploymentsClient.Get(ctx, appName, metaV1.GetOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get deployment", "error", err)
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	if replicas != nil {
		deployment.Spec.Replicas = replicas
	}

	if cpu != nil || memory != nil {
		if len(deployment.Spec.Template.Spec.Containers) > 0 {
			container := &deployment.Spec.Template.Spec.Containers[0]
			if cpu != nil {
				cpuQuantity, err := resource.ParseQuantity(*cpu)
				if err != nil {
					slog.ErrorContext(ctx, "Invalid CPU value", "error", err)
					return fmt.Errorf("invalid cpu value: %w", err)
				}
				container.Resources.Requests[v1.ResourceCPU] = cpuQuantity
				container.Resources.Limits[v1.ResourceCPU] = cpuQuantity
			}
			if memory != nil {
				memoryQuantity, err := resource.ParseQuantity(*memory)
				if err != nil {
					slog.ErrorContext(ctx, "Invalid memory value", "error", err)
					return fmt.Errorf("invalid memory value: %w", err)
				}
				container.Resources.Requests[v1.ResourceMemory] = memoryQuantity
				container.Resources.Limits[v1.ResourceMemory] = memoryQuantity
			}
		}
	}

	_, err = deploymentsClient.Update(ctx, deployment, metaV1.UpdateOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to update deployment", "error", err)
		return fmt.Errorf("failed to update deployment: %w", err)
	}

	slog.InfoContext(ctx, "Deployment scaled successfully")
	return nil
}

// RestartDeployment triggers a rollout restart by updating pod template annotations
func (kc *KubernetesClient) RestartDeployment(ctx context.Context, namespace, deploymentName string) error {
	slog.InfoContext(ctx, "Restarting deployment", "namespace", namespace, "deployment", deploymentName)

	deploymentsClient := kc.ClientSet.AppsV1().Deployments(namespace)
	deployment, err := deploymentsClient.Get(ctx, deploymentName, metaV1.GetOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get deployment", "error", err)
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = make(map[string]string)
	}
	deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	_, err = deploymentsClient.Update(ctx, deployment, metaV1.UpdateOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to restart deployment", "error", err)
		return fmt.Errorf("failed to restart deployment: %w", err)
	}

	slog.InfoContext(ctx, "Deployment restarted successfully")
	return nil
}
