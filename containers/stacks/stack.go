package stacks

import (
	"fmt"
	"time"
)

// Stack represents a schema.org ItemList containing multiple containers.
//
// This implements the schema.org ItemList structure for defining multi-container
// deployments with dependency ordering and orchestration.
//
// Schema.org Reference: https://schema.org/ItemList
type Stack struct {
	// Context is the JSON-LD context (schema.org)
	Context string `json:"@context"`
	// Type is the schema.org type (ItemList)
	Type string `json:"@type"`
	// Name is the stack name
	Name string `json:"name"`
	// Description provides a human-readable description
	Description string `json:"description,omitempty"`
	// ItemListElement contains the containers in the stack
	ItemListElement []StackItemElement `json:"itemListElement"`
	// Network configuration for the stack
	Network NetworkConfig `json:"network,omitempty"`
	// Volumes shared across containers in the stack
	Volumes []VolumeConfig `json:"volumes,omitempty"`
}

// StackItemElement represents a container in the stack as a schema.org SoftwareApplication.
//
// Each container is modeled as a SoftwareApplication with dependencies, health checks,
// and post-start actions (migrations, initialization).
//
// Schema.org Reference: https://schema.org/SoftwareApplication
type StackItemElement struct {
	// Type is the schema.org type (SoftwareApplication)
	Type string `json:"@type"`
	// Position determines startup order (lower numbers start first)
	Position int `json:"position"`
	// Name is the container name (unique identifier)
	Name string `json:"name"`
	// ApplicationCategory categorizes the application (Database, Cache, etc.)
	ApplicationCategory string `json:"applicationCategory,omitempty"`
	// SoftwareVersion is the version/tag of the software
	SoftwareVersion string `json:"softwareVersion,omitempty"`
	// Image is the Docker image (e.g., "postgres:17")
	Image string `json:"image"`

	// Dependencies and ordering
	// Requirements lists the names of containers that must start before this one
	Requirements []string `json:"requirements,omitempty"`
	// SoftwareRequirements provides detailed dependency specifications
	SoftwareRequirements []SoftwareRequirement `json:"softwareRequirements,omitempty"`

	// Container configuration
	// Environment variables for the container
	Environment map[string]string `json:"environment,omitempty"`
	// Ports to expose and map
	Ports []PortMapping `json:"ports,omitempty"`
	// Volumes to mount
	Volumes []VolumeMount `json:"volumeMounts,omitempty"`
	// HealthCheck configuration
	HealthCheck HealthCheckConfig `json:"healthCheck,omitempty"`
	// Command overrides the default container command
	Command []string `json:"command,omitempty"`

	// Post-start actions (migrations, initialization)
	// PotentialAction uses schema.org Action for post-start commands
	// Schema.org Reference: https://schema.org/potentialAction
	PotentialAction []Action `json:"potentialAction,omitempty"`
}

// SoftwareRequirement specifies a dependency on another container.
//
// Schema.org Reference: https://schema.org/SoftwareApplication (softwareRequirements)
type SoftwareRequirement struct {
	// Type is the schema.org type (SoftwareApplication)
	Type string `json:"@type"`
	// Name is the name of the required container
	Name string `json:"name"`
	// WaitForHealthy indicates whether to wait for the container to be healthy
	WaitForHealthy bool `json:"waitForHealthy"`
}

// Action represents a post-start command (migration, initialization, etc.).
//
// Schema.org Reference: https://schema.org/Action
type Action struct {
	// Type is the schema.org type (Action)
	Type string `json:"@type"`
	// Name is a human-readable name for the action
	Name string `json:"name"`
	// ActionType categorizes the action (migration, init, seed, etc.)
	ActionType string `json:"actionType,omitempty"`
	// Target specifies which container to run the command in
	// If empty, runs in the container this action belongs to
	Target string `json:"target,omitempty"`
	// Command is the command to execute
	Command []string `json:"command"`
	// Timeout in seconds (0 = no timeout)
	Timeout int `json:"timeout,omitempty"`
	// WorkingDirectory for command execution
	WorkingDirectory string `json:"workingDirectory,omitempty"`
}

// NetworkConfig defines network configuration for the stack.
type NetworkConfig struct {
	// Name is the network name
	Name string `json:"name"`
	// Driver is the network driver (bridge, overlay, etc.)
	Driver string `json:"driver,omitempty"`
	// CreateIfNotExists creates the network if it doesn't exist
	CreateIfNotExists bool `json:"createIfNotExists,omitempty"`
}

// VolumeConfig defines a shared volume for the stack.
type VolumeConfig struct {
	// Name is the volume name
	Name string `json:"name"`
	// Driver is the volume driver
	Driver string `json:"driver,omitempty"`
	// CreateIfNotExists creates the volume if it doesn't exist
	CreateIfNotExists bool `json:"createIfNotExists,omitempty"`
}

// PortMapping defines a port mapping for a container.
type PortMapping struct {
	// ContainerPort is the port inside the container
	ContainerPort int `json:"containerPort"`
	// HostPort is the port on the host (0 = random port)
	HostPort int `json:"hostPort,omitempty"`
	// Protocol is the protocol (tcp, udp)
	Protocol string `json:"protocol,omitempty"`
}

// VolumeMount defines a volume mount for a container.
type VolumeMount struct {
	// Source is the volume name or host path
	Source string `json:"source"`
	// Target is the mount path inside the container
	Target string `json:"target"`
	// ReadOnly indicates if the mount is read-only
	ReadOnly bool `json:"readOnly,omitempty"`
	// Type is the mount type (volume, bind, tmpfs)
	Type string `json:"type,omitempty"`
}

// HealthCheckConfig defines health check configuration.
type HealthCheckConfig struct {
	// Type is the health check type (http, tcp, command, postgres, redis, etc.)
	Type string `json:"type"`
	// Command for command-based health checks
	Command []string `json:"command,omitempty"`
	// Interval in seconds between health checks
	Interval int `json:"interval,omitempty"`
	// Timeout in seconds for each health check
	Timeout int `json:"timeout,omitempty"`
	// Retries before considering unhealthy
	Retries int `json:"retries,omitempty"`
	// StartPeriod grace period before health checks start
	StartPeriod int `json:"startPeriod,omitempty"`
	// Path for HTTP health checks
	Path string `json:"path,omitempty"`
	// Port for TCP/HTTP health checks (0 = use first exposed port)
	Port int `json:"port,omitempty"`
}

// StackDeployment represents a deployed stack with container IDs.
type StackDeployment struct {
	// Stack is the stack definition
	Stack Stack
	// Containers maps container names to their IDs
	Containers map[string]string
	// Network is the network ID
	Network string
	// Volumes maps volume names to their IDs
	Volumes map[string]string
	// StartTime when the stack was deployed
	StartTime time.Time
}

// Validate validates the stack configuration.
func (s *Stack) Validate() error {
	if s.Context != "https://schema.org" && s.Context != "" {
		return fmt.Errorf("invalid @context: must be 'https://schema.org' or empty")
	}
	if s.Type != "ItemList" && s.Type != "" {
		return fmt.Errorf("invalid @type: must be 'ItemList' or empty")
	}
	if s.Name == "" {
		return fmt.Errorf("stack name is required")
	}
	if len(s.ItemListElement) == 0 {
		return fmt.Errorf("stack must contain at least one container")
	}

	// Validate each container
	names := make(map[string]bool)
	positions := make(map[int]bool)

	for i, item := range s.ItemListElement {
		// Validate type
		if item.Type != "SoftwareApplication" && item.Type != "" {
			return fmt.Errorf("container %d: invalid @type: must be 'SoftwareApplication' or empty", i)
		}

		// Validate name
		if item.Name == "" {
			return fmt.Errorf("container %d: name is required", i)
		}
		if names[item.Name] {
			return fmt.Errorf("duplicate container name: %s", item.Name)
		}
		names[item.Name] = true

		// Validate position
		if item.Position <= 0 {
			return fmt.Errorf("container %s: position must be > 0", item.Name)
		}
		if positions[item.Position] {
			return fmt.Errorf("duplicate position: %d", item.Position)
		}
		positions[item.Position] = true

		// Validate image
		if item.Image == "" {
			return fmt.Errorf("container %s: image is required", item.Name)
		}

		// Validate dependencies exist
		for _, req := range item.Requirements {
			if req == item.Name {
				return fmt.Errorf("container %s: cannot depend on itself", item.Name)
			}
		}

		// Validate software requirements
		for _, swReq := range item.SoftwareRequirements {
			if swReq.Name == "" {
				return fmt.Errorf("container %s: software requirement name is required", item.Name)
			}
			if swReq.Name == item.Name {
				return fmt.Errorf("container %s: cannot depend on itself", item.Name)
			}
		}
	}

	// Validate that all dependencies exist
	for _, item := range s.ItemListElement {
		for _, req := range item.Requirements {
			if !names[req] {
				return fmt.Errorf("container %s: unknown dependency: %s", item.Name, req)
			}
		}
		for _, swReq := range item.SoftwareRequirements {
			if !names[swReq.Name] {
				return fmt.Errorf("container %s: unknown software requirement: %s", item.Name, swReq.Name)
			}
		}
	}

	// Check for circular dependencies
	if err := s.checkCircularDependencies(); err != nil {
		return err
	}

	return nil
}

// checkCircularDependencies detects circular dependencies in the stack.
func (s *Stack) checkCircularDependencies() error {
	// Build dependency graph
	deps := make(map[string][]string)
	for _, item := range s.ItemListElement {
		var allDeps []string
		allDeps = append(allDeps, item.Requirements...)
		for _, swReq := range item.SoftwareRequirements {
			allDeps = append(allDeps, swReq.Name)
		}
		deps[item.Name] = allDeps
	}

	// DFS to detect cycles
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(string) bool
	hasCycle = func(node string) bool {
		visited[node] = true
		recStack[node] = true

		for _, dep := range deps[node] {
			if !visited[dep] {
				if hasCycle(dep) {
					return true
				}
			} else if recStack[dep] {
				return true
			}
		}

		recStack[node] = false
		return false
	}

	for _, item := range s.ItemListElement {
		if !visited[item.Name] {
			if hasCycle(item.Name) {
				return fmt.Errorf("circular dependency detected involving: %s", item.Name)
			}
		}
	}

	return nil
}

// GetContainerByName returns a container by name.
func (s *Stack) GetContainerByName(name string) (*StackItemElement, error) {
	for i := range s.ItemListElement {
		if s.ItemListElement[i].Name == name {
			return &s.ItemListElement[i], nil
		}
	}
	return nil, fmt.Errorf("container not found: %s", name)
}

// GetDependencies returns all dependencies for a container (direct and transitive).
func (s *Stack) GetDependencies(name string) ([]string, error) {
	container, err := s.GetContainerByName(name)
	if err != nil {
		return nil, err
	}

	deps := make(map[string]bool)
	var collectDeps func(string) error

	collectDeps = func(containerName string) error {
		c, err := s.GetContainerByName(containerName)
		if err != nil {
			return err
		}

		// Add direct dependencies
		for _, req := range c.Requirements {
			if !deps[req] {
				deps[req] = true
				if err := collectDeps(req); err != nil {
					return err
				}
			}
		}

		// Add software requirements
		for _, swReq := range c.SoftwareRequirements {
			if !deps[swReq.Name] {
				deps[swReq.Name] = true
				if err := collectDeps(swReq.Name); err != nil {
					return err
				}
			}
		}

		return nil
	}

	if err := collectDeps(container.Name); err != nil {
		return nil, err
	}

	// Convert map to slice
	result := make([]string, 0, len(deps))
	for dep := range deps {
		result = append(result, dep)
	}

	return result, nil
}

// GetStartupOrder returns containers in the order they should be started.
func (s *Stack) GetStartupOrder() []StackItemElement {
	// Sort by position
	items := make([]StackItemElement, len(s.ItemListElement))
	copy(items, s.ItemListElement)

	// Simple insertion sort by position
	for i := 1; i < len(items); i++ {
		key := items[i]
		j := i - 1
		for j >= 0 && items[j].Position > key.Position {
			items[j+1] = items[j]
			j--
		}
		items[j+1] = key
	}

	return items
}
