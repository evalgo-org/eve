package kyma

import (
	"errors"
	"fmt"
)

// Common errors returned by the Kyma client.
// These can be used with errors.Is() for error type checking.
var (
	// ErrInvalidConfig is returned when the client configuration is invalid.
	ErrInvalidConfig = errors.New("invalid client configuration")

	// ErrInvalidApplication is returned when application configuration is invalid.
	ErrInvalidApplication = errors.New("invalid application configuration")

	// ErrKubeConfigNotFound is returned when the kubeconfig file cannot be found.
	ErrKubeConfigNotFound = errors.New("kubeconfig file not found")

	// ErrInClusterConfigFailed is returned when in-cluster config fails.
	ErrInClusterConfigFailed = errors.New("in-cluster configuration failed")

	// ErrDeploymentNotFound is returned when a Deployment resource is not found.
	ErrDeploymentNotFound = errors.New("deployment not found")

	// ErrServiceNotFound is returned when a Service resource is not found.
	ErrServiceNotFound = errors.New("service not found")

	// ErrAPIRuleNotFound is returned when an APIRule resource is not found.
	ErrAPIRuleNotFound = errors.New("apirule not found")

	// ErrResourceCreationFailed is returned when resource creation fails.
	ErrResourceCreationFailed = errors.New("resource creation failed")

	// ErrResourceUpdateFailed is returned when resource update fails.
	ErrResourceUpdateFailed = errors.New("resource update failed")

	// ErrResourceDeletionFailed is returned when resource deletion fails.
	ErrResourceDeletionFailed = errors.New("resource deletion failed")

	// ErrNamespaceRequired is returned when namespace is not specified.
	ErrNamespaceRequired = errors.New("namespace is required")

	// ErrNameRequired is returned when application name is not specified.
	ErrNameRequired = errors.New("application name is required")

	// ErrImageRequired is returned when container image is not specified.
	ErrImageRequired = errors.New("container image is required")

	// ErrDomainRequired is returned when Kyma domain is not specified.
	ErrDomainRequired = errors.New("domain is required for APIRule")
)

// ValidationError represents an error that occurred during configuration validation.
type ValidationError struct {
	Field   string // Field that failed validation
	Message string // Detailed error message
}

// Error implements the error interface for ValidationError.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: field '%s': %s", e.Field, e.Message)
}

// ResourceError represents an error that occurred during a resource operation.
type ResourceError struct {
	Operation    string // Operation that failed (create, update, delete, get)
	ResourceType string // Type of resource (Deployment, Service, APIRule)
	ResourceName string // Name of the resource
	Namespace    string // Namespace of the resource
	Err          error  // Underlying error
}

// Error implements the error interface for ResourceError.
func (e *ResourceError) Error() string {
	return fmt.Sprintf("failed to %s %s '%s' in namespace '%s': %v",
		e.Operation, e.ResourceType, e.ResourceName, e.Namespace, e.Err)
}

// Unwrap implements error unwrapping for ResourceError.
func (e *ResourceError) Unwrap() error {
	return e.Err
}

// NewValidationError creates a new ValidationError.
func NewValidationError(field, message string) error {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// NewResourceError creates a new ResourceError.
func NewResourceError(operation, resourceType, resourceName, namespace string, err error) error {
	return &ResourceError{
		Operation:    operation,
		ResourceType: resourceType,
		ResourceName: resourceName,
		Namespace:    namespace,
		Err:          err,
	}
}
