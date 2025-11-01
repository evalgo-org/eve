package kyma

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestValidationError tests ValidationError creation and error interface
func TestValidationError(t *testing.T) {
	err := NewValidationError("Name", "name is required")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Name")
	assert.Contains(t, err.Error(), "name is required")

	// Check it's a ValidationError
	var validationErr *ValidationError
	assert.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "Name", validationErr.Field)
	assert.Equal(t, "name is required", validationErr.Message)
}

// TestResourceError tests ResourceError creation and error wrapping
func TestResourceError(t *testing.T) {
	underlyingErr := fmt.Errorf("connection refused")
	err := NewResourceError("create", "Deployment", "test-app", "default", underlyingErr)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create")
	assert.Contains(t, err.Error(), "Deployment")
	assert.Contains(t, err.Error(), "test-app")
	assert.Contains(t, err.Error(), "default")
	assert.Contains(t, err.Error(), "connection refused")

	// Check it's a ResourceError
	var resourceErr *ResourceError
	assert.True(t, errors.As(err, &resourceErr))
	assert.Equal(t, "create", resourceErr.Operation)
	assert.Equal(t, "Deployment", resourceErr.ResourceType)
	assert.Equal(t, "test-app", resourceErr.ResourceName)
	assert.Equal(t, "default", resourceErr.Namespace)

	// Check error unwrapping
	assert.ErrorIs(t, err, underlyingErr)
}

// TestErrorConstants tests that error constants are defined
func TestErrorConstants(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrInvalidConfig", ErrInvalidConfig},
		{"ErrInvalidApplication", ErrInvalidApplication},
		{"ErrKubeConfigNotFound", ErrKubeConfigNotFound},
		{"ErrInClusterConfigFailed", ErrInClusterConfigFailed},
		{"ErrDeploymentNotFound", ErrDeploymentNotFound},
		{"ErrServiceNotFound", ErrServiceNotFound},
		{"ErrAPIRuleNotFound", ErrAPIRuleNotFound},
		{"ErrResourceCreationFailed", ErrResourceCreationFailed},
		{"ErrResourceUpdateFailed", ErrResourceUpdateFailed},
		{"ErrResourceDeletionFailed", ErrResourceDeletionFailed},
		{"ErrNamespaceRequired", ErrNamespaceRequired},
		{"ErrNameRequired", ErrNameRequired},
		{"ErrImageRequired", ErrImageRequired},
		{"ErrDomainRequired", ErrDomainRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.err)
			assert.Error(t, tt.err)
			assert.NotEmpty(t, tt.err.Error())
		})
	}
}

// TestErrorIsChecks tests error type checking with errors.Is
func TestErrorIsChecks(t *testing.T) {
	err1 := fmt.Errorf("wrapped: %w", ErrInvalidConfig)
	assert.ErrorIs(t, err1, ErrInvalidConfig)

	err2 := fmt.Errorf("wrapped: %w", ErrNamespaceRequired)
	assert.ErrorIs(t, err2, ErrNamespaceRequired)

	resourceErr := NewResourceError("create", "Deployment", "test", "default", ErrResourceCreationFailed)
	assert.ErrorIs(t, resourceErr, ErrResourceCreationFailed)
}
