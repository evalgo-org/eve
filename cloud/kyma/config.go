// Package kyma provides a client for deploying applications on SAP BTP Kyma.
// This package implements library-first design for easy integration into downstream applications.
//
// The package offers:
//   - Application lifecycle management (deploy, update, delete)
//   - Kubernetes Deployment configuration
//   - Service exposure
//   - Kyma APIRule management for external access
//
// All operations follow error-handling best practices (no panics) and return errors
// for proper handling by calling applications.
package kyma

import (
	"time"
)

// Config holds the configuration for creating a Kyma client.
// This structure provides flexible configuration options for connecting to
// Kubernetes/Kyma clusters either via kubeconfig file or in-cluster authentication.
type Config struct {
	// KubeconfigPath is the path to the kubeconfig file.
	// If empty, the client will attempt to use in-cluster configuration.
	// Falls back to ~/.kube/config if in-cluster config is unavailable.
	KubeconfigPath string

	// DefaultNamespace specifies the default namespace for operations.
	// Individual operations can override this on a per-request basis.
	DefaultNamespace string

	// DefaultDomain is the default Kyma domain for APIRules.
	// This is typically the cluster's ingress domain (e.g., "c-1234567.kyma.ondemand.com").
	DefaultDomain string
}

// Application defines the complete configuration for a Kyma application deployment.
// This includes Kubernetes resources (Deployment, Service) and Kyma-specific
// resources (APIRule for external access).
type Application struct {
	// Name is the application identifier, used for all created resources.
	Name string

	// Namespace is the Kubernetes namespace where resources will be created.
	Namespace string

	// Image is the container image to deploy (e.g., "nginx:1.25-alpine").
	Image string

	// Replicas specifies the number of pod replicas to run.
	Replicas int32

	// Resources defines CPU and memory requirements/limits.
	Resources ResourceRequirements

	// ServicePort is the port that the Service will expose.
	ServicePort int32

	// ContainerPort is the port that the container listens on.
	// Defaults to ServicePort if not specified.
	ContainerPort int32

	// Domain is the Kyma domain for the APIRule.
	// If empty, uses the client's DefaultDomain.
	Domain string

	// PathPrefix defines the URL path prefix for the APIRule (e.g., "/*").
	PathPrefix string

	// Methods specifies which HTTP methods are allowed through the APIRule.
	Methods []string

	// AuthEnabled determines whether authentication is required for the APIRule.
	// If false, the APIRule will use noAuth (public access).
	AuthEnabled bool

	// LivenessProbe defines the health check for container liveness.
	LivenessProbe *HealthProbe

	// ReadinessProbe defines the health check for container readiness.
	ReadinessProbe *HealthProbe

	// Labels are metadata labels applied to all created resources.
	Labels map[string]string

	// Annotations are metadata annotations applied to all created resources.
	Annotations map[string]string

	// Env specifies environment variables for the container.
	Env map[string]string
}

// ResourceRequirements defines CPU and memory resource constraints for containers.
// Follows Kubernetes resource specification format.
type ResourceRequirements struct {
	// RequestsCPU is the minimum CPU allocation (e.g., "100m" for 100 millicores).
	RequestsCPU string

	// RequestsMemory is the minimum memory allocation (e.g., "128Mi").
	RequestsMemory string

	// LimitsCPU is the maximum CPU allocation (e.g., "200m").
	LimitsCPU string

	// LimitsMemory is the maximum memory allocation (e.g., "256Mi").
	LimitsMemory string
}

// HealthProbe defines health check configuration for containers.
// Supports HTTP GET probes for liveness and readiness checks.
type HealthProbe struct {
	// Path is the HTTP endpoint to check (e.g., "/health", "/").
	Path string

	// Port is the container port to probe.
	Port int32

	// InitialDelaySeconds is the delay before the first probe.
	InitialDelaySeconds int32

	// PeriodSeconds is the interval between probes.
	PeriodSeconds int32

	// TimeoutSeconds is the probe timeout.
	TimeoutSeconds int32

	// SuccessThreshold is the minimum consecutive successes for the probe to be considered successful.
	SuccessThreshold int32

	// FailureThreshold is the minimum consecutive failures for the probe to be considered failed.
	FailureThreshold int32
}

// DeploymentResult contains information about a successful deployment.
// This is returned after deploying or updating an application.
type DeploymentResult struct {
	// Name is the application name.
	Name string

	// Namespace is the Kubernetes namespace.
	Namespace string

	// DeploymentID is a unique identifier for this deployment operation.
	DeploymentID string

	// ServiceURL is the internal cluster URL for the Service.
	ServiceURL string

	// APIRuleURL is the external public URL for accessing the application.
	APIRuleURL string

	// CreatedAt is the timestamp when the deployment was created/updated.
	CreatedAt time.Time

	// UpdatedAt is the timestamp when the deployment was last modified.
	UpdatedAt time.Time
}

// ApplicationStatus represents the current status of a deployed application.
// This includes the status of all associated resources.
type ApplicationStatus struct {
	// Name is the application name.
	Name string

	// Namespace is the Kubernetes namespace.
	Namespace string

	// DeploymentReady indicates if the Deployment has the desired number of ready replicas.
	DeploymentReady bool

	// ReadyReplicas is the number of pods that are ready.
	ReadyReplicas int32

	// DesiredReplicas is the target number of replicas.
	DesiredReplicas int32

	// ServiceExists indicates if the Service resource exists.
	ServiceExists bool

	// APIRuleExists indicates if the APIRule resource exists.
	APIRuleExists bool

	// APIRuleReady indicates if the APIRule is in a ready state.
	APIRuleReady bool

	// URL is the external access URL if the APIRule is ready.
	URL string

	// Conditions contains detailed status information.
	Conditions []StatusCondition
}

// StatusCondition represents a condition of a resource.
type StatusCondition struct {
	// Type is the type of condition (e.g., "Ready", "Available").
	Type string

	// Status is the status of the condition ("True", "False", "Unknown").
	Status string

	// Reason is a brief reason for the condition's status.
	Reason string

	// Message is a human-readable message describing the condition.
	Message string

	// LastTransitionTime is when the condition last changed.
	LastTransitionTime time.Time
}
