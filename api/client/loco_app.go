package client

import (
	"fmt"
	"slices"
	"strings"
	"time"

	appv1 "github.com/nikumar1206/loco/shared/proto/app/v1"
	appsV1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacV1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	v1Gateway "sigs.k8s.io/gateway-api/apis/v1"
)

var BannedSubdomains = []string{
	"api", "admin", "dashboard", "console",
	"login", "auth", "user", "users", "support", "help", "loco", "monitoring",
	"metrics", "stats", "status", "health", "system", "service", "services",
	"config", "configuration", "settings", "setup", "install", "uninstall",
}

const (
	LabelAppName       = "app.loco.io/name"
	LabelAppInstance   = "app.loco.io/instance"
	LabelAppVersion    = "app.loco.io/version"
	LabelAppComponent  = "app.loco.io/component"
	LabelAppPartOf     = "app.loco.io/part-of"
	LabelAppManagedBy  = "app.loco.io/managed-by"
	LabelAppCreatedFor = "app.loco.io/created-for"
	LabelAppCreatedAt  = "app.loco.io/created-at"
	LabelAppCreatedBy  = "app.loco.io/created-by"
)

type LocoApp struct {
	EnvVars        []*appv1.EnvVar
	CreatedAt      time.Time
	Name           string
	CreatedBy      string
	ContainerImage string
	Subdomain      string
	Labels         map[string]string
	Config         *appv1.LocoConfig
}

// todo: NewLocoApp should not require all this info. until then, generatenamespace will be exposed.
func NewLocoApp(config *appv1.LocoConfig, createdBy string, containerImage string, envVars []*appv1.EnvVar) *LocoApp {
	ns := GenerateNameSpace(config.Metadata.Name, createdBy)
	labels := generateLabels(config.Metadata.Name, ns, createdBy)
	return &LocoApp{
		Name:           config.Metadata.Name,
		Subdomain:      config.Routing.Subdomain,
		CreatedBy:      createdBy,
		CreatedAt:      time.Now(),
		Labels:         labels,
		EnvVars:        envVars,
		Config:         config,
		ContainerImage: containerImage,
	}
}

func (l *LocoApp) ContainerName() string {
	return l.Name
}

func (l *LocoApp) DeploymentName() string {
	return l.Name
}

func (l *LocoApp) ServicePort() string {
	return l.Name
}

func (l *LocoApp) ServiceName() string {
	return l.Name
}

func (l *LocoApp) EnvSecretName() string {
	return l.Name
}

func (l *LocoApp) ImagePullSecretName() string {
	return l.Name + "-registry"
}

func (l *LocoApp) RoleName() string {
	return l.Name
}

func (l *LocoApp) ServiceAccountName() string {
	return l.Name
}

func (l *LocoApp) RoleBindingName() string {
	return l.Name
}

func (l *LocoApp) HTTPRouteName() string {
	return l.Name
}

func (l *LocoApp) NamespaceName() string {
	return GenerateNameSpace(l.Name, l.CreatedBy)
}

// Resource spec methods

// does container image name need to be parameterized here?
func (l *LocoApp) DeploymentSpec(containerImage string) *appsV1.Deployment {
	return &appsV1.Deployment{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      l.DeploymentName(),
			Namespace: l.NamespaceName(),
			Labels:    l.Labels,
		},
		Spec: appsV1.DeploymentSpec{
			Replicas: &[]int32{DefaultReplicas}[0],
			Selector: &metaV1.LabelSelector{
				MatchLabels: map[string]string{
					LabelAppName: l.Name,
				},
			},
			Strategy: appsV1.DeploymentStrategy{
				Type: appsV1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsV1.RollingUpdateDeployment{
					MaxSurge:       &intstr.IntOrString{Type: intstr.String, StrVal: MaxSurgePercent},
					MaxUnavailable: &intstr.IntOrString{Type: intstr.String, StrVal: MaxUnavailablePercent},
				},
			},
			RevisionHistoryLimit: &[]int32{MaxReplicaHistory}[0],
			Template: v1.PodTemplateSpec{
				ObjectMeta: metaV1.ObjectMeta{
					Labels: l.Labels,
				},
				Spec: v1.PodSpec{
					RestartPolicy: v1.RestartPolicyAlways,
					ImagePullSecrets: []v1.LocalObjectReference{
						{
							Name: fmt.Sprintf("%s-registry-credentials", l.ImagePullSecretName()),
						},
					},
					ServiceAccountName: l.ServiceAccountName(),
					Containers: []v1.Container{
						{
							Name:  l.ContainerName(),
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
									ContainerPort: l.Config.Routing.Port,
								},
							},
							EnvFrom: []v1.EnvFromSource{
								{
									SecretRef: &v1.SecretEnvSource{
										LocalObjectReference: v1.LocalObjectReference{
											Name: l.EnvSecretName(),
										},
									},
								},
							},
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    resourceMustParse(l.Config.Resources.Cpu),
									v1.ResourceMemory: resourceMustParse(l.Config.Resources.Memory),
								},
								Limits: v1.ResourceList{
									v1.ResourceCPU:    resourceMustParse(l.Config.Resources.Cpu),
									v1.ResourceMemory: resourceMustParse(l.Config.Resources.Memory),
								},
							},
							LivenessProbe: &v1.Probe{
								InitialDelaySeconds:           l.Config.Health.StartupGracePeriod,
								TimeoutSeconds:                l.Config.Health.Timeout,
								PeriodSeconds:                 l.Config.Health.Interval,
								TerminationGracePeriodSeconds: ptrtoInt64(TerminationGracePeriod),
								SuccessThreshold:              1,
								FailureThreshold:              l.Config.Health.FailThreshold,
								ProbeHandler: v1.ProbeHandler{
									HTTPGet: &v1.HTTPGetAction{
										Path: l.Config.Health.Path,
										Port: intstr.FromInt32(l.Config.Routing.Port),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (l *LocoApp) ServiceSpec() *v1.Service {
	return &v1.Service{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      l.ServiceName(),
			Namespace: l.NamespaceName(),
		},
		Spec: v1.ServiceSpec{
			Type: v1.ServiceTypeClusterIP,
			Selector: map[string]string{
				LabelAppName: l.Name,
			},
			SessionAffinity: v1.ServiceAffinityNone,
			SessionAffinityConfig: &v1.SessionAffinityConfig{
				ClientIP: &v1.ClientIPConfig{
					TimeoutSeconds: &[]int32{SessionAffinityTimeout}[0],
				},
			},
			Ports: []v1.ServicePort{
				{
					Name:       l.ServicePort(),
					Protocol:   v1.ProtocolTCP,
					Port:       DefaultServicePort,
					TargetPort: intstr.FromInt32(l.Config.Routing.Port),
				},
			},
		},
	}
}

func (l *LocoApp) SecretSpec() *v1.Secret {
	secretData := make(map[string][]byte)
	for _, envVar := range l.EnvVars {
		secretData[envVar.Name] = []byte(envVar.Value)
	}

	return &v1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      l.EnvSecretName(),
			Namespace: l.NamespaceName(),
			Labels:    l.Labels,
		},
		Data: secretData,
		Type: v1.SecretTypeOpaque,
	}
}

func (l *LocoApp) ServiceAccountSpec() *v1.ServiceAccount {
	return &v1.ServiceAccount{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      l.ServiceAccountName(),
			Namespace: l.NamespaceName(),
			Labels:    l.Labels,
		},
	}
}

func (l *LocoApp) RoleSpec(secret *v1.Secret) *rbacV1.Role {
	return &rbacV1.Role{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      l.RoleName(),
			Namespace: l.NamespaceName(),
			Labels:    l.Labels,
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
}

func (l *LocoApp) RoleBindingSpec() *rbacV1.RoleBinding {
	return &rbacV1.RoleBinding{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      l.RoleBindingName(),
			Namespace: l.NamespaceName(),
			Labels:    l.Labels,
		},
		Subjects: []rbacV1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      l.ServiceAccountName(),
				Namespace: l.NamespaceName(),
			},
		},
		RoleRef: rbacV1.RoleRef{
			Kind:     "Role",
			Name:     l.RoleName(),
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
}

func (l *LocoApp) HTTPRouteSpec() *v1Gateway.HTTPRoute {
	hostname := fmt.Sprintf("%s.deploy-app.com", l.Subdomain)
	pathType := v1Gateway.PathMatchPathPrefix
	timeout := DefaultRequestTimeout

	return &v1Gateway.HTTPRoute{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      l.HTTPRouteName(),
			Namespace: l.NamespaceName(),
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
								Value: ptrToString(l.Config.Routing.PathPrefix),
							},
						},
					},
					Timeouts: &v1Gateway.HTTPRouteTimeouts{
						Request: ptrToDuration(timeout),
					},
					BackendRefs: []v1Gateway.HTTPBackendRef{
						{
							BackendRef: v1Gateway.BackendRef{
								BackendObjectReference: v1Gateway.BackendObjectReference{
									Name: v1Gateway.ObjectName(l.DeploymentName()),
									Port: ptrToPortNumber(int(DefaultServicePort)),
									Kind: ptrToKind("Service"),
								},
							},
						},
					},
				},
			},
		},
	}
}

// todo: cleanup; should not be exported. namespace should only pullable via locoApp interface.
func GenerateNameSpace(name string, username string) string {
	appName := strings.ToLower(strings.TrimSpace(name))
	userName := strings.ToLower(strings.TrimSpace(username))

	return appName + "-" + userName
}

func generateLabels(name, namespace, createdBy string) map[string]string {
	return map[string]string{
		LabelAppName:       name,
		LabelAppInstance:   namespace,
		LabelAppVersion:    "1.0.0",
		LabelAppComponent:  "backend",
		LabelAppPartOf:     "loco-platform",
		LabelAppManagedBy:  "loco",
		LabelAppCreatedFor: createdBy,
		LabelAppCreatedAt:  time.Now().UTC().Format("20060102T150405Z"),
	}
}

func resourceMustParse(value string) resource.Quantity {
	q, err := resource.ParseQuantity(value)
	if err != nil {
		panic(err)
	}
	return q
}

func IsBannedSubDomain(subdomain string) bool {
	return slices.Contains(BannedSubdomains, subdomain) || strings.Contains(subdomain, "loco")
}
