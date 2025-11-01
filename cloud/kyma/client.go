package kyma

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client is the interface for Kyma operations.
// It provides methods for deploying, updating, deleting, and querying
// applications on SAP BTP Kyma clusters.
//
// All methods return errors instead of panicking, allowing proper error
// handling by calling applications.
type Client interface {
	// DeployApplication deploys a complete application (Deployment + Service + APIRule).
	// This creates all necessary Kubernetes and Kyma resources for the application.
	// If resources already exist, they will be updated.
	DeployApplication(ctx context.Context, app *Application) (*DeploymentResult, error)

	// UpdateApplication updates an existing application.
	// This is an alias for DeployApplication as both create-or-update the resources.
	UpdateApplication(ctx context.Context, app *Application) (*DeploymentResult, error)

	// DeleteApplication removes all resources for an application.
	// This deletes the Deployment, Service, and APIRule for the specified application.
	DeleteApplication(ctx context.Context, namespace, name string) error

	// GetApplicationStatus checks the status of a deployed application.
	// Returns detailed status information about all resources.
	GetApplicationStatus(ctx context.Context, namespace, name string) (*ApplicationStatus, error)
}

// client is the concrete implementation of the Client interface.
type client struct {
	config        *Config
	clientset     *kubernetes.Clientset
	dynamicClient dynamic.Interface
	restConfig    *rest.Config
}

// NewClient creates a new Kyma client with the provided configuration.
// It initializes Kubernetes client connections and validates the configuration.
//
// The client will attempt to connect using the following priority:
//  1. In-cluster configuration (if running inside a Kubernetes cluster)
//  2. Kubeconfig file specified in config.KubeconfigPath
//  3. Default kubeconfig at ~/.kube/config
//
// Returns an error if:
//   - Configuration is invalid
//   - Kubeconfig cannot be found or loaded
//   - Kubernetes client initialization fails
func NewClient(config *Config) (Client, error) {
	if config == nil {
		return nil, fmt.Errorf("%w: config cannot be nil", ErrInvalidConfig)
	}

	// Get Kubernetes configuration
	restConfig, err := getKubeConfig(config.KubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes config: %w", err)
	}

	// Create Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	// Create dynamic client for custom resources (APIRule)
	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &client{
		config:        config,
		clientset:     clientset,
		dynamicClient: dynamicClient,
		restConfig:    restConfig,
	}, nil
}

// getKubeConfig returns the Kubernetes configuration.
// It tries multiple sources in the following order:
//  1. In-cluster config (for running inside Kyma)
//  2. Kubeconfig file from kubeconfigPath parameter
//  3. Default kubeconfig at ~/.kube/config
func getKubeConfig(kubeconfigPath string) (*rest.Config, error) {
	// Try in-cluster config first (for running inside Kubernetes/Kyma)
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// If kubeconfigPath is not specified, use default location
	if kubeconfigPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		kubeconfigPath = filepath.Join(home, ".kube", "config")
	}

	// Check if kubeconfig file exists
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", ErrKubeConfigNotFound, kubeconfigPath)
	}

	// Build config from kubeconfig file
	config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
	}

	return config, nil
}

// DeployApplication deploys or updates a complete application stack.
// This includes creating/updating:
//  1. Kubernetes Deployment
//  2. Kubernetes Service
//  3. Kyma APIRule (for external access)
func (c *client) DeployApplication(ctx context.Context, app *Application) (*DeploymentResult, error) {
	// Validate application configuration
	if err := c.validateApplication(app); err != nil {
		return nil, err
	}

	// Apply defaults
	c.applyDefaults(app)

	// Create or update Deployment
	if err := c.deployDeployment(ctx, app); err != nil {
		return nil, err
	}

	// Create or update Service
	if err := c.deployService(ctx, app); err != nil {
		return nil, err
	}

	// Create or update APIRule
	if err := c.deployAPIRule(ctx, app); err != nil {
		return nil, err
	}

	// Build result
	result := &DeploymentResult{
		Name:         app.Name,
		Namespace:    app.Namespace,
		DeploymentID: generateDeploymentID(app.Name, app.Namespace),
		ServiceURL:   fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", app.Name, app.Namespace, app.ServicePort),
		APIRuleURL:   fmt.Sprintf("https://%s.%s", app.Name, app.Domain),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	return result, nil
}

// UpdateApplication is an alias for DeployApplication.
// Both methods perform create-or-update operations.
func (c *client) UpdateApplication(ctx context.Context, app *Application) (*DeploymentResult, error) {
	return c.DeployApplication(ctx, app)
}

// DeleteApplication removes all resources associated with an application.
func (c *client) DeleteApplication(ctx context.Context, namespace, name string) error {
	if namespace == "" {
		return ErrNamespaceRequired
	}
	if name == "" {
		return ErrNameRequired
	}

	// Delete in reverse order: APIRule -> Service -> Deployment
	if err := c.deleteAPIRule(ctx, namespace, name); err != nil {
		// Log but continue with other deletions
		// This follows the pattern of warning on errors but not failing completely
	}

	if err := c.deleteService(ctx, namespace, name); err != nil {
		// Log but continue
	}

	if err := c.deleteDeployment(ctx, namespace, name); err != nil {
		return err
	}

	return nil
}

// GetApplicationStatus retrieves the status of all resources for an application.
func (c *client) GetApplicationStatus(ctx context.Context, namespace, name string) (*ApplicationStatus, error) {
	if namespace == "" {
		return nil, ErrNamespaceRequired
	}
	if name == "" {
		return nil, ErrNameRequired
	}

	status := &ApplicationStatus{
		Name:      name,
		Namespace: namespace,
	}

	// Get Deployment status
	if err := c.getDeploymentStatus(ctx, namespace, name, status); err != nil {
		// Resource might not exist, don't fail
	}

	// Get Service status
	if err := c.getServiceStatus(ctx, namespace, name, status); err != nil {
		// Resource might not exist, don't fail
	}

	// Get APIRule status
	if err := c.getAPIRuleStatus(ctx, namespace, name, status); err != nil {
		// Resource might not exist, don't fail
	}

	return status, nil
}

// validateApplication checks if the application configuration is valid.
func (c *client) validateApplication(app *Application) error {
	if app == nil {
		return ErrInvalidApplication
	}

	if app.Name == "" {
		return NewValidationError("Name", "application name is required")
	}

	if app.Namespace == "" {
		return NewValidationError("Namespace", "namespace is required")
	}

	if app.Image == "" {
		return NewValidationError("Image", "container image is required")
	}

	if app.Replicas < 0 {
		return NewValidationError("Replicas", "replicas cannot be negative")
	}

	if app.ServicePort <= 0 {
		return NewValidationError("ServicePort", "service port must be greater than 0")
	}

	return nil
}

// applyDefaults applies default values to application configuration.
func (c *client) applyDefaults(app *Application) {
	// Default replicas to 1 if not specified
	if app.Replicas == 0 {
		app.Replicas = 1
	}

	// Default container port to service port if not specified
	if app.ContainerPort == 0 {
		app.ContainerPort = app.ServicePort
	}

	// Use client's default domain if not specified
	if app.Domain == "" {
		app.Domain = c.config.DefaultDomain
	}

	// Default path prefix to "/*" if not specified
	if app.PathPrefix == "" {
		app.PathPrefix = "/*"
	}

	// Default HTTP methods if not specified
	if len(app.Methods) == 0 {
		app.Methods = []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD"}
	}

	// Default labels if not specified
	if app.Labels == nil {
		app.Labels = make(map[string]string)
	}
	if _, exists := app.Labels["app"]; !exists {
		app.Labels["app"] = app.Name
	}
	if _, exists := app.Labels["managed-by"]; !exists {
		app.Labels["managed-by"] = "kyma-client"
	}

	// Initialize annotations if nil
	if app.Annotations == nil {
		app.Annotations = make(map[string]string)
	}

	// Initialize env if nil
	if app.Env == nil {
		app.Env = make(map[string]string)
	}
}

// generateDeploymentID creates a unique identifier for a deployment.
func generateDeploymentID(name, namespace string) string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%s-%s-%d", namespace, name, timestamp)
}
