package stacks

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStack_Validate(t *testing.T) {
	tests := []struct {
		name    string
		stack   Stack
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid stack",
			stack: Stack{
				Context: "https://schema.org",
				Type:    "ItemList",
				Name:    "test-stack",
				ItemListElement: []StackItemElement{
					{
						Type:     "SoftwareApplication",
						Position: 1,
						Name:     "postgres",
						Image:    "postgres:17",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty name",
			stack: Stack{
				Context: "https://schema.org",
				Type:    "ItemList",
				ItemListElement: []StackItemElement{
					{
						Type:     "SoftwareApplication",
						Position: 1,
						Name:     "postgres",
						Image:    "postgres:17",
					},
				},
			},
			wantErr: true,
			errMsg:  "stack name is required",
		},
		{
			name: "no containers",
			stack: Stack{
				Context:         "https://schema.org",
				Type:            "ItemList",
				Name:            "test-stack",
				ItemListElement: []StackItemElement{},
			},
			wantErr: true,
			errMsg:  "stack must contain at least one container",
		},
		{
			name: "duplicate container names",
			stack: Stack{
				Context: "https://schema.org",
				Type:    "ItemList",
				Name:    "test-stack",
				ItemListElement: []StackItemElement{
					{
						Type:     "SoftwareApplication",
						Position: 1,
						Name:     "postgres",
						Image:    "postgres:17",
					},
					{
						Type:     "SoftwareApplication",
						Position: 2,
						Name:     "postgres",
						Image:    "postgres:16",
					},
				},
			},
			wantErr: true,
			errMsg:  "duplicate container name: postgres",
		},
		{
			name: "duplicate positions",
			stack: Stack{
				Context: "https://schema.org",
				Type:    "ItemList",
				Name:    "test-stack",
				ItemListElement: []StackItemElement{
					{
						Type:     "SoftwareApplication",
						Position: 1,
						Name:     "postgres",
						Image:    "postgres:17",
					},
					{
						Type:     "SoftwareApplication",
						Position: 1,
						Name:     "redis",
						Image:    "redis:7",
					},
				},
			},
			wantErr: true,
			errMsg:  "duplicate position: 1",
		},
		{
			name: "invalid position",
			stack: Stack{
				Context: "https://schema.org",
				Type:    "ItemList",
				Name:    "test-stack",
				ItemListElement: []StackItemElement{
					{
						Type:     "SoftwareApplication",
						Position: 0,
						Name:     "postgres",
						Image:    "postgres:17",
					},
				},
			},
			wantErr: true,
			errMsg:  "position must be > 0",
		},
		{
			name: "missing image",
			stack: Stack{
				Context: "https://schema.org",
				Type:    "ItemList",
				Name:    "test-stack",
				ItemListElement: []StackItemElement{
					{
						Type:     "SoftwareApplication",
						Position: 1,
						Name:     "postgres",
					},
				},
			},
			wantErr: true,
			errMsg:  "image is required",
		},
		{
			name: "self dependency",
			stack: Stack{
				Context: "https://schema.org",
				Type:    "ItemList",
				Name:    "test-stack",
				ItemListElement: []StackItemElement{
					{
						Type:         "SoftwareApplication",
						Position:     1,
						Name:         "postgres",
						Image:        "postgres:17",
						Requirements: []string{"postgres"},
					},
				},
			},
			wantErr: true,
			errMsg:  "cannot depend on itself",
		},
		{
			name: "unknown dependency",
			stack: Stack{
				Context: "https://schema.org",
				Type:    "ItemList",
				Name:    "test-stack",
				ItemListElement: []StackItemElement{
					{
						Type:         "SoftwareApplication",
						Position:     1,
						Name:         "app",
						Image:        "app:latest",
						Requirements: []string{"postgres"},
					},
				},
			},
			wantErr: true,
			errMsg:  "unknown dependency: postgres",
		},
		{
			name: "circular dependency",
			stack: Stack{
				Context: "https://schema.org",
				Type:    "ItemList",
				Name:    "test-stack",
				ItemListElement: []StackItemElement{
					{
						Type:     "SoftwareApplication",
						Position: 1,
						Name:     "a",
						Image:    "a:latest",
						SoftwareRequirements: []SoftwareRequirement{
							{Name: "b"},
						},
					},
					{
						Type:     "SoftwareApplication",
						Position: 2,
						Name:     "b",
						Image:    "b:latest",
						SoftwareRequirements: []SoftwareRequirement{
							{Name: "a"},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "circular dependency detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.stack.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestStack_GetContainerByName(t *testing.T) {
	stack := Stack{
		Context: "https://schema.org",
		Type:    "ItemList",
		Name:    "test-stack",
		ItemListElement: []StackItemElement{
			{
				Type:     "SoftwareApplication",
				Position: 1,
				Name:     "postgres",
				Image:    "postgres:17",
			},
			{
				Type:     "SoftwareApplication",
				Position: 2,
				Name:     "redis",
				Image:    "redis:7",
			},
		},
	}

	// Test finding existing container
	container, err := stack.GetContainerByName("postgres")
	require.NoError(t, err)
	assert.Equal(t, "postgres", container.Name)
	assert.Equal(t, "postgres:17", container.Image)

	// Test finding non-existent container
	_, err = stack.GetContainerByName("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "container not found")
}

func TestStack_GetDependencies(t *testing.T) {
	stack := Stack{
		Context: "https://schema.org",
		Type:    "ItemList",
		Name:    "test-stack",
		ItemListElement: []StackItemElement{
			{
				Type:     "SoftwareApplication",
				Position: 1,
				Name:     "postgres",
				Image:    "postgres:17",
			},
			{
				Type:     "SoftwareApplication",
				Position: 2,
				Name:     "redis",
				Image:    "redis:7",
			},
			{
				Type:         "SoftwareApplication",
				Position:     3,
				Name:         "app",
				Image:        "app:latest",
				Requirements: []string{"postgres"},
				SoftwareRequirements: []SoftwareRequirement{
					{Name: "redis"},
				},
			},
		},
	}

	// Test container with dependencies
	deps, err := stack.GetDependencies("app")
	require.NoError(t, err)
	assert.Len(t, deps, 2)
	assert.Contains(t, deps, "postgres")
	assert.Contains(t, deps, "redis")

	// Test container without dependencies
	deps, err = stack.GetDependencies("postgres")
	require.NoError(t, err)
	assert.Len(t, deps, 0)

	// Test non-existent container
	_, err = stack.GetDependencies("nonexistent")
	require.Error(t, err)
}

func TestStack_GetDependencies_Transitive(t *testing.T) {
	// Test transitive dependencies (A -> B -> C)
	stack := Stack{
		Context: "https://schema.org",
		Type:    "ItemList",
		Name:    "test-stack",
		ItemListElement: []StackItemElement{
			{
				Type:     "SoftwareApplication",
				Position: 1,
				Name:     "c",
				Image:    "c:latest",
			},
			{
				Type:         "SoftwareApplication",
				Position:     2,
				Name:         "b",
				Image:        "b:latest",
				Requirements: []string{"c"},
			},
			{
				Type:         "SoftwareApplication",
				Position:     3,
				Name:         "a",
				Image:        "a:latest",
				Requirements: []string{"b"},
			},
		},
	}

	// A depends on B, B depends on C, so A should have both B and C as dependencies
	deps, err := stack.GetDependencies("a")
	require.NoError(t, err)
	assert.Len(t, deps, 2)
	assert.Contains(t, deps, "b")
	assert.Contains(t, deps, "c")
}

func TestStack_GetStartupOrder(t *testing.T) {
	stack := Stack{
		Context: "https://schema.org",
		Type:    "ItemList",
		Name:    "test-stack",
		ItemListElement: []StackItemElement{
			{
				Type:     "SoftwareApplication",
				Position: 3,
				Name:     "app",
				Image:    "app:latest",
			},
			{
				Type:     "SoftwareApplication",
				Position: 1,
				Name:     "postgres",
				Image:    "postgres:17",
			},
			{
				Type:     "SoftwareApplication",
				Position: 2,
				Name:     "redis",
				Image:    "redis:7",
			},
		},
	}

	ordered := stack.GetStartupOrder()
	require.Len(t, ordered, 3)

	// Verify order by position
	assert.Equal(t, "postgres", ordered[0].Name)
	assert.Equal(t, 1, ordered[0].Position)
	assert.Equal(t, "redis", ordered[1].Name)
	assert.Equal(t, 2, ordered[1].Position)
	assert.Equal(t, "app", ordered[2].Name)
	assert.Equal(t, 3, ordered[2].Position)
}

func TestStack_CircularDependencyDetection(t *testing.T) {
	tests := []struct {
		name    string
		stack   Stack
		wantErr bool
	}{
		{
			name: "no circular dependency",
			stack: Stack{
				Context: "https://schema.org",
				Type:    "ItemList",
				Name:    "test-stack",
				ItemListElement: []StackItemElement{
					{
						Type:     "SoftwareApplication",
						Position: 1,
						Name:     "postgres",
						Image:    "postgres:17",
					},
					{
						Type:         "SoftwareApplication",
						Position:     2,
						Name:         "app",
						Image:        "app:latest",
						Requirements: []string{"postgres"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "direct circular dependency (A -> B -> A)",
			stack: Stack{
				Context: "https://schema.org",
				Type:    "ItemList",
				Name:    "test-stack",
				ItemListElement: []StackItemElement{
					{
						Type:         "SoftwareApplication",
						Position:     1,
						Name:         "a",
						Image:        "a:latest",
						Requirements: []string{"b"},
					},
					{
						Type:         "SoftwareApplication",
						Position:     2,
						Name:         "b",
						Image:        "b:latest",
						Requirements: []string{"a"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "indirect circular dependency (A -> B -> C -> A)",
			stack: Stack{
				Context: "https://schema.org",
				Type:    "ItemList",
				Name:    "test-stack",
				ItemListElement: []StackItemElement{
					{
						Type:         "SoftwareApplication",
						Position:     1,
						Name:         "a",
						Image:        "a:latest",
						Requirements: []string{"b"},
					},
					{
						Type:         "SoftwareApplication",
						Position:     2,
						Name:         "b",
						Image:        "b:latest",
						Requirements: []string{"c"},
					},
					{
						Type:         "SoftwareApplication",
						Position:     3,
						Name:         "c",
						Image:        "c:latest",
						Requirements: []string{"a"},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.stack.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "circular dependency")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLoader_LoadStackFromJSON(t *testing.T) {
	jsonData := `{
		"@context": "https://schema.org",
		"@type": "ItemList",
		"name": "test-stack",
		"description": "Test stack",
		"network": {
			"name": "test-network",
			"driver": "bridge",
			"createIfNotExists": true
		},
		"itemListElement": [
			{
				"@type": "SoftwareApplication",
				"position": 1,
				"name": "postgres",
				"image": "postgres:17",
				"environment": {
					"POSTGRES_PASSWORD": "test"
				},
				"ports": [
					{
						"containerPort": 5432,
						"hostPort": 5432
					}
				],
				"healthCheck": {
					"type": "command",
					"command": ["pg_isready", "-U", "postgres"]
				}
			}
		]
	}`

	stack, err := LoadStackFromJSON([]byte(jsonData))
	require.NoError(t, err)
	assert.NotNil(t, stack)

	// Verify stack fields
	assert.Equal(t, "https://schema.org", stack.Context)
	assert.Equal(t, "ItemList", stack.Type)
	assert.Equal(t, "test-stack", stack.Name)
	assert.Equal(t, "Test stack", stack.Description)

	// Verify network
	assert.Equal(t, "test-network", stack.Network.Name)
	assert.Equal(t, "bridge", stack.Network.Driver)
	assert.True(t, stack.Network.CreateIfNotExists)

	// Verify container
	require.Len(t, stack.ItemListElement, 1)
	container := stack.ItemListElement[0]
	assert.Equal(t, "SoftwareApplication", container.Type)
	assert.Equal(t, 1, container.Position)
	assert.Equal(t, "postgres", container.Name)
	assert.Equal(t, "postgres:17", container.Image)

	// Verify environment
	assert.Equal(t, "test", container.Environment["POSTGRES_PASSWORD"])

	// Verify ports
	require.Len(t, container.Ports, 1)
	assert.Equal(t, 5432, container.Ports[0].ContainerPort)
	assert.Equal(t, 5432, container.Ports[0].HostPort)
	assert.Equal(t, "tcp", container.Ports[0].Protocol) // Default

	// Verify health check
	assert.Equal(t, "command", container.HealthCheck.Type)
	assert.Equal(t, []string{"pg_isready", "-U", "postgres"}, container.HealthCheck.Command)
	assert.Equal(t, 10, container.HealthCheck.Interval)    // Default
	assert.Equal(t, 5, container.HealthCheck.Timeout)      // Default
	assert.Equal(t, 3, container.HealthCheck.Retries)      // Default
	assert.Equal(t, 10, container.HealthCheck.StartPeriod) // Default
}

func TestLoader_LoadStackFromJSON_InvalidJSON(t *testing.T) {
	jsonData := `{invalid json`

	_, err := LoadStackFromJSON([]byte(jsonData))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse stack JSON")
}

func TestLoader_LoadStackFromJSON_InvalidStack(t *testing.T) {
	// Stack with no name (invalid)
	jsonData := `{
		"@context": "https://schema.org",
		"@type": "ItemList",
		"itemListElement": []
	}`

	_, err := LoadStackFromJSON([]byte(jsonData))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid stack")
}

func TestLoader_LoadStackFromJSON_Defaults(t *testing.T) {
	// Minimal stack definition - should fill in defaults
	jsonData := `{
		"name": "test-stack",
		"itemListElement": [
			{
				"position": 1,
				"name": "postgres",
				"image": "postgres:17"
			}
		]
	}`

	stack, err := LoadStackFromJSON([]byte(jsonData))
	require.NoError(t, err)

	// Verify defaults were set
	assert.Equal(t, "https://schema.org", stack.Context)
	assert.Equal(t, "ItemList", stack.Type)
	assert.Equal(t, "SoftwareApplication", stack.ItemListElement[0].Type)
}

func TestStackToJSON(t *testing.T) {
	stack := &Stack{
		Context: "https://schema.org",
		Type:    "ItemList",
		Name:    "test-stack",
		ItemListElement: []StackItemElement{
			{
				Type:     "SoftwareApplication",
				Position: 1,
				Name:     "postgres",
				Image:    "postgres:17",
			},
		},
	}

	jsonData, err := StackToJSON(stack)
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Verify it's valid JSON
	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	require.NoError(t, err)

	// Verify key fields
	assert.Equal(t, "https://schema.org", result["@context"])
	assert.Equal(t, "ItemList", result["@type"])
	assert.Equal(t, "test-stack", result["name"])
}

func TestHealthCheckConfig_Types(t *testing.T) {
	tests := []struct {
		name       string
		healthType string
		valid      bool
	}{
		{"command type", "command", true},
		{"http type", "http", true},
		{"tcp type", "tcp", true},
		{"postgres type", "postgres", true},
		{"redis type", "redis", true},
		{"unknown type", "unknown", true}, // Valid in structure, but would fail at runtime
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stack := Stack{
				Context: "https://schema.org",
				Type:    "ItemList",
				Name:    "test-stack",
				ItemListElement: []StackItemElement{
					{
						Type:     "SoftwareApplication",
						Position: 1,
						Name:     "test",
						Image:    "test:latest",
						HealthCheck: HealthCheckConfig{
							Type: tt.healthType,
						},
					},
				},
			}

			err := stack.Validate()
			if tt.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestPortMapping_Protocols(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		valid    bool
	}{
		{"tcp protocol", "tcp", true},
		{"udp protocol", "udp", true},
		{"empty protocol", "", true}, // Empty defaults to tcp
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stack := Stack{
				Context: "https://schema.org",
				Type:    "ItemList",
				Name:    "test-stack",
				ItemListElement: []StackItemElement{
					{
						Type:     "SoftwareApplication",
						Position: 1,
						Name:     "test",
						Image:    "test:latest",
						Ports: []PortMapping{
							{
								ContainerPort: 8080,
								Protocol:      tt.protocol,
							},
						},
					},
				},
			}

			err := stack.Validate()
			if tt.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestSoftwareRequirement_WaitForHealthy(t *testing.T) {
	stack := Stack{
		Context: "https://schema.org",
		Type:    "ItemList",
		Name:    "test-stack",
		ItemListElement: []StackItemElement{
			{
				Type:     "SoftwareApplication",
				Position: 1,
				Name:     "postgres",
				Image:    "postgres:17",
			},
			{
				Type:     "SoftwareApplication",
				Position: 2,
				Name:     "app",
				Image:    "app:latest",
				SoftwareRequirements: []SoftwareRequirement{
					{
						Type:           "SoftwareApplication",
						Name:           "postgres",
						WaitForHealthy: true,
					},
				},
			},
		},
	}

	err := stack.Validate()
	require.NoError(t, err)

	// Verify software requirement
	app := stack.ItemListElement[1]
	require.Len(t, app.SoftwareRequirements, 1)
	assert.Equal(t, "postgres", app.SoftwareRequirements[0].Name)
	assert.True(t, app.SoftwareRequirements[0].WaitForHealthy)
}

func TestAction_PostStartActions(t *testing.T) {
	stack := Stack{
		Context: "https://schema.org",
		Type:    "ItemList",
		Name:    "test-stack",
		ItemListElement: []StackItemElement{
			{
				Type:     "SoftwareApplication",
				Position: 1,
				Name:     "app",
				Image:    "app:latest",
				PotentialAction: []Action{
					{
						Type:       "Action",
						Name:       "Run migrations",
						ActionType: "migration",
						Command:    []string{"npm", "run", "migrate"},
						Timeout:    120,
					},
					{
						Type:       "Action",
						Name:       "Seed data",
						ActionType: "seed",
						Command:    []string{"npm", "run", "seed"},
						Timeout:    60,
					},
				},
			},
		},
	}

	err := stack.Validate()
	require.NoError(t, err)

	// Verify actions
	app := stack.ItemListElement[0]
	require.Len(t, app.PotentialAction, 2)

	assert.Equal(t, "Run migrations", app.PotentialAction[0].Name)
	assert.Equal(t, "migration", app.PotentialAction[0].ActionType)
	assert.Equal(t, []string{"npm", "run", "migrate"}, app.PotentialAction[0].Command)
	assert.Equal(t, 120, app.PotentialAction[0].Timeout)

	assert.Equal(t, "Seed data", app.PotentialAction[1].Name)
	assert.Equal(t, "seed", app.PotentialAction[1].ActionType)
}

func TestVolumeMount_Types(t *testing.T) {
	tests := []struct {
		name      string
		mountType string
		valid     bool
	}{
		{"volume type", "volume", true},
		{"bind type", "bind", true},
		{"tmpfs type", "tmpfs", true},
		{"empty type", "", true}, // Empty defaults to volume
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stack := Stack{
				Context: "https://schema.org",
				Type:    "ItemList",
				Name:    "test-stack",
				ItemListElement: []StackItemElement{
					{
						Type:     "SoftwareApplication",
						Position: 1,
						Name:     "test",
						Image:    "test:latest",
						Volumes: []VolumeMount{
							{
								Source: "test-data",
								Target: "/data",
								Type:   tt.mountType,
							},
						},
					},
				},
			}

			err := stack.Validate()
			if tt.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
