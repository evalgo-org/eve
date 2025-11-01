package kyma

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewClient_NilConfig tests that NewClient returns an error with nil config
func TestNewClient_NilConfig(t *testing.T) {
	client, err := NewClient(nil)
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.ErrorIs(t, err, ErrInvalidConfig)
}

// TestValidateApplication tests application validation
func TestValidateApplication(t *testing.T) {
	c := &client{
		config: &Config{},
	}

	tests := []struct {
		name    string
		app     *Application
		wantErr bool
		errType error
	}{
		{
			name:    "NilApplication",
			app:     nil,
			wantErr: true,
			errType: ErrInvalidApplication,
		},
		{
			name: "MissingName",
			app: &Application{
				Namespace:   "default",
				Image:       "nginx:latest",
				ServicePort: 80,
			},
			wantErr: true,
		},
		{
			name: "MissingNamespace",
			app: &Application{
				Name:        "test-app",
				Image:       "nginx:latest",
				ServicePort: 80,
			},
			wantErr: true,
		},
		{
			name: "MissingImage",
			app: &Application{
				Name:        "test-app",
				Namespace:   "default",
				ServicePort: 80,
			},
			wantErr: true,
		},
		{
			name: "InvalidServicePort",
			app: &Application{
				Name:        "test-app",
				Namespace:   "default",
				Image:       "nginx:latest",
				ServicePort: -1,
			},
			wantErr: true,
		},
		{
			name: "NegativeReplicas",
			app: &Application{
				Name:        "test-app",
				Namespace:   "default",
				Image:       "nginx:latest",
				ServicePort: 80,
				Replicas:    -1,
			},
			wantErr: true,
		},
		{
			name: "ValidApplication",
			app: &Application{
				Name:        "test-app",
				Namespace:   "default",
				Image:       "nginx:latest",
				ServicePort: 80,
				Replicas:    1,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.validateApplication(tt.app)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestApplyDefaults tests that default values are applied correctly
func TestApplyDefaults(t *testing.T) {
	c := &client{
		config: &Config{
			DefaultDomain: "example.kyma.cloud.sap",
		},
	}

	app := &Application{
		Name:        "test-app",
		Namespace:   "default",
		Image:       "nginx:latest",
		ServicePort: 80,
	}

	c.applyDefaults(app)

	// Check defaults were applied
	assert.Equal(t, int32(1), app.Replicas, "default replicas should be 1")
	assert.Equal(t, int32(80), app.ContainerPort, "container port should default to service port")
	assert.Equal(t, "example.kyma.cloud.sap", app.Domain, "domain should use config default")
	assert.Equal(t, "/*", app.PathPrefix, "default path prefix should be /*")
	assert.Equal(t, []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD"}, app.Methods)
	assert.NotNil(t, app.Labels)
	assert.Equal(t, "test-app", app.Labels["app"])
	assert.Equal(t, "kyma-client", app.Labels["managed-by"])
	assert.NotNil(t, app.Annotations)
	assert.NotNil(t, app.Env)
}

// TestApplyDefaults_PreservesExisting tests that existing values are not overwritten
func TestApplyDefaults_PreservesExisting(t *testing.T) {
	c := &client{
		config: &Config{
			DefaultDomain: "default.kyma.cloud.sap",
		},
	}

	app := &Application{
		Name:          "test-app",
		Namespace:     "default",
		Image:         "nginx:latest",
		ServicePort:   80,
		Replicas:      3,
		ContainerPort: 8080,
		Domain:        "custom.kyma.cloud.sap",
		PathPrefix:    "/api/*",
		Methods:       []string{"GET", "POST"},
		Labels: map[string]string{
			"custom": "label",
		},
	}

	c.applyDefaults(app)

	// Check existing values were preserved
	assert.Equal(t, int32(3), app.Replicas)
	assert.Equal(t, int32(8080), app.ContainerPort)
	assert.Equal(t, "custom.kyma.cloud.sap", app.Domain)
	assert.Equal(t, "/api/*", app.PathPrefix)
	assert.Equal(t, []string{"GET", "POST"}, app.Methods)
	assert.Equal(t, "label", app.Labels["custom"])
}

// TestGenerateDeploymentID tests deployment ID generation
func TestGenerateDeploymentID(t *testing.T) {
	id1 := generateDeploymentID("app1", "namespace1")
	id2 := generateDeploymentID("app1", "namespace1")
	id3 := generateDeploymentID("app2", "namespace1")

	// IDs should be unique (different timestamps)
	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEmpty(t, id3)

	// IDs for different apps should be different
	assert.NotEqual(t, id1, id3)

	// IDs should contain namespace and name
	assert.Contains(t, id1, "namespace1")
	assert.Contains(t, id1, "app1")
}

// TestBuildResourceRequirements tests resource requirements conversion
func TestBuildResourceRequirements(t *testing.T) {
	c := &client{}

	tests := []struct {
		name string
		reqs ResourceRequirements
		want map[string]string // simplified check
	}{
		{
			name: "AllFieldsSet",
			reqs: ResourceRequirements{
				RequestsCPU:    "100m",
				RequestsMemory: "128Mi",
				LimitsCPU:      "200m",
				LimitsMemory:   "256Mi",
			},
			want: map[string]string{
				"requests-cpu":    "100m",
				"requests-memory": "128Mi",
				"limits-cpu":      "200m",
				"limits-memory":   "256Mi",
			},
		},
		{
			name: "OnlyRequests",
			reqs: ResourceRequirements{
				RequestsCPU:    "100m",
				RequestsMemory: "128Mi",
			},
			want: map[string]string{
				"requests-cpu":    "100m",
				"requests-memory": "128Mi",
			},
		},
		{
			name: "OnlyLimits",
			reqs: ResourceRequirements{
				LimitsCPU:    "200m",
				LimitsMemory: "256Mi",
			},
			want: map[string]string{
				"limits-cpu":    "200m",
				"limits-memory": "256Mi",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.buildResourceRequirements(tt.reqs)

			if tt.reqs.RequestsCPU != "" {
				assert.NotNil(t, result.Requests["cpu"])
			}
			if tt.reqs.RequestsMemory != "" {
				assert.NotNil(t, result.Requests["memory"])
			}
			if tt.reqs.LimitsCPU != "" {
				assert.NotNil(t, result.Limits["cpu"])
			}
			if tt.reqs.LimitsMemory != "" {
				assert.NotNil(t, result.Limits["memory"])
			}
		})
	}
}

// TestBuildProbe tests health probe conversion
func TestBuildProbe(t *testing.T) {
	c := &client{}

	probe := &HealthProbe{
		Path:                "/health",
		Port:                8080,
		InitialDelaySeconds: 10,
		PeriodSeconds:       5,
		TimeoutSeconds:      3,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}

	result := c.buildProbe(probe)

	assert.NotNil(t, result)
	assert.NotNil(t, result.HTTPGet)
	assert.Equal(t, "/health", result.HTTPGet.Path)
	assert.Equal(t, int32(10), result.InitialDelaySeconds)
	assert.Equal(t, int32(5), result.PeriodSeconds)
	assert.Equal(t, int32(3), result.TimeoutSeconds)
	assert.Equal(t, int32(1), result.SuccessThreshold)
	assert.Equal(t, int32(3), result.FailureThreshold)
}

// TestBuildEnvVars tests environment variable conversion
func TestBuildEnvVars(t *testing.T) {
	c := &client{}

	env := map[string]string{
		"ENV1": "value1",
		"ENV2": "value2",
		"ENV3": "value3",
	}

	result := c.buildEnvVars(env)

	assert.Len(t, result, 3)

	// Check that all env vars are present
	envMap := make(map[string]string)
	for _, e := range result {
		envMap[e.Name] = e.Value
	}

	assert.Equal(t, "value1", envMap["ENV1"])
	assert.Equal(t, "value2", envMap["ENV2"])
	assert.Equal(t, "value3", envMap["ENV3"])
}

// TestBuildDeployment tests Deployment resource construction
func TestBuildDeployment(t *testing.T) {
	c := &client{}

	app := &Application{
		Name:        "test-app",
		Namespace:   "test-ns",
		Image:       "nginx:1.25-alpine",
		Replicas:    3,
		ServicePort: 80,
		Labels: map[string]string{
			"env": "test",
		},
		Annotations: map[string]string{
			"annotation": "value",
		},
	}

	deployment := c.buildDeployment(app)

	assert.NotNil(t, deployment)
	assert.Equal(t, "test-app", deployment.Name)
	assert.Equal(t, "test-ns", deployment.Namespace)
	assert.Equal(t, int32(3), *deployment.Spec.Replicas)
	assert.Equal(t, "test-app", deployment.Labels["app"])
	assert.Equal(t, "test", deployment.Labels["env"])
	assert.Equal(t, "value", deployment.Annotations["annotation"])
	assert.Len(t, deployment.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, "nginx:1.25-alpine", deployment.Spec.Template.Spec.Containers[0].Image)
}

// TestBuildService tests Service resource construction
func TestBuildService(t *testing.T) {
	c := &client{}

	app := &Application{
		Name:          "test-app",
		Namespace:     "test-ns",
		ServicePort:   80,
		ContainerPort: 8080,
		Labels: map[string]string{
			"env": "test",
		},
	}

	service := c.buildService(app)

	assert.NotNil(t, service)
	assert.Equal(t, "test-app", service.Name)
	assert.Equal(t, "test-ns", service.Namespace)
	assert.Equal(t, "test-app", service.Labels["app"])
	assert.Equal(t, "test", service.Labels["env"])
	assert.Len(t, service.Spec.Ports, 1)
	assert.Equal(t, int32(80), service.Spec.Ports[0].Port)
	assert.Equal(t, "test-app", service.Spec.Selector["app"])
}

// TestBuildAPIRule tests APIRule resource construction
func TestBuildAPIRule(t *testing.T) {
	c := &client{}

	app := &Application{
		Name:        "test-app",
		Namespace:   "test-ns",
		ServicePort: 80,
		Domain:      "example.kyma.cloud.sap",
		PathPrefix:  "/*",
		Methods:     []string{"GET", "POST"},
		AuthEnabled: false,
		Labels: map[string]string{
			"env": "test",
		},
	}

	apiRule := c.buildAPIRule(app)

	assert.NotNil(t, apiRule)
	assert.Equal(t, "test-app", apiRule.GetName())
	assert.Equal(t, "test-ns", apiRule.GetNamespace())
	assert.Equal(t, "gateway.kyma-project.io/v2alpha1", apiRule.GetAPIVersion())
	assert.Equal(t, "APIRule", apiRule.GetKind())

	// Check spec exists
	spec, ok := apiRule.Object["spec"]
	assert.True(t, ok)
	assert.NotNil(t, spec)
}

// BenchmarkBuildDeployment benchmarks Deployment construction
func BenchmarkBuildDeployment(b *testing.B) {
	c := &client{}
	app := &Application{
		Name:        "test-app",
		Namespace:   "test-ns",
		Image:       "nginx:1.25-alpine",
		Replicas:    3,
		ServicePort: 80,
		Resources: ResourceRequirements{
			RequestsCPU:    "100m",
			RequestsMemory: "128Mi",
			LimitsCPU:      "200m",
			LimitsMemory:   "256Mi",
		},
	}

	for i := 0; i < b.N; i++ {
		_ = c.buildDeployment(app)
	}
}

// BenchmarkBuildService benchmarks Service construction
func BenchmarkBuildService(b *testing.B) {
	c := &client{}
	app := &Application{
		Name:          "test-app",
		Namespace:     "test-ns",
		ServicePort:   80,
		ContainerPort: 8080,
	}

	for i := 0; i < b.N; i++ {
		_ = c.buildService(app)
	}
}

// BenchmarkBuildAPIRule benchmarks APIRule construction
func BenchmarkBuildAPIRule(b *testing.B) {
	c := &client{}
	app := &Application{
		Name:        "test-app",
		Namespace:   "test-ns",
		ServicePort: 80,
		Domain:      "example.kyma.cloud.sap",
		PathPrefix:  "/*",
		Methods:     []string{"GET", "POST", "PUT", "DELETE"},
	}

	for i := 0; i < b.N; i++ {
		_ = c.buildAPIRule(app)
	}
}
