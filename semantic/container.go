package semantic

import (
	"encoding/json"
	"fmt"
)

// Container Semantic Types for Docker/Podman/K8s Operations
// These types map container operations to Schema.org vocabulary for semantic orchestration

// ============================================================================
// Container Types (Schema.org: SoftwareApplication)
// ============================================================================

// Container represents a containerized application as Schema.org SoftwareApplication
type Container struct {
	Context             string                 `json:"@context,omitempty"`
	Type                string                 `json:"@type"` // "SoftwareApplication"
	Identifier          string                 `json:"identifier"`
	Name                string                 `json:"name,omitempty"`
	ApplicationCategory string                 `json:"applicationCategory,omitempty"` // "Container", "Pod", "Service"
	SoftwareVersion     string                 `json:"softwareVersion,omitempty"`
	Image               *ContainerImage        `json:"image,omitempty"`
	Runtime             string                 `json:"runtimePlatform,omitempty"` // "docker", "podman", "kubernetes"
	Properties          map[string]interface{} `json:"additionalProperty,omitempty"`
}

// ContainerImage represents a container image as Schema.org ImageObject
type ContainerImage struct {
	Type       string                 `json:"@type"` // "ImageObject"
	Identifier string                 `json:"identifier"`
	ContentUrl string                 `json:"contentUrl"`        // Registry URL
	Name       string                 `json:"name,omitempty"`    // Image name
	Version    string                 `json:"version,omitempty"` // Tag
	Properties map[string]interface{} `json:"additionalProperty,omitempty"`
}

// ContainerRegistry represents a container registry as Schema.org Service
type ContainerRegistry struct {
	Type       string                 `json:"@type"` // "Service"
	Identifier string                 `json:"identifier"`
	URL        string                 `json:"url"`
	Properties map[string]interface{} `json:"additionalProperty,omitempty"`
}

// ============================================================================
// Container Action Types
// ============================================================================

// ActivateAction represents container deployment/start operations
// Maps to Schema.org ActivateAction for starting containers
type ActivateAction struct {
	Context      string         `json:"@context,omitempty"`
	Type         string         `json:"@type"` // "ActivateAction"
	Identifier   string         `json:"identifier"`
	Name         string         `json:"name,omitempty"`
	Description  string         `json:"description,omitempty"`
	Object       *Container     `json:"object"`           // Container to deploy
	Target       *ComputeNode   `json:"target,omitempty"` // Where to deploy
	ActionStatus string         `json:"actionStatus,omitempty"`
	StartTime    string         `json:"startTime,omitempty"`
	EndTime      string         `json:"endTime,omitempty"`
	Error        *PropertyValue `json:"error,omitempty"`
}

// DeactivateAction represents container stop/removal operations
// Maps to Schema.org DeactivateAction for stopping containers
type DeactivateAction struct {
	Context      string         `json:"@context,omitempty"`
	Type         string         `json:"@type"` // "DeactivateAction"
	Identifier   string         `json:"identifier"`
	Name         string         `json:"name,omitempty"`
	Description  string         `json:"description,omitempty"`
	Object       *Container     `json:"object"` // Container to stop
	ActionStatus string         `json:"actionStatus,omitempty"`
	StartTime    string         `json:"startTime,omitempty"`
	EndTime      string         `json:"endTime,omitempty"`
	Error        *PropertyValue `json:"error,omitempty"`
}

// DownloadAction represents image pull operations
type DownloadAction struct {
	Context      string             `json:"@context,omitempty"`
	Type         string             `json:"@type"` // "DownloadAction"
	Identifier   string             `json:"identifier"`
	Name         string             `json:"name,omitempty"`
	Description  string             `json:"description,omitempty"`
	Object       *ContainerImage    `json:"object"`                 // Image to pull
	FromLocation *ContainerRegistry `json:"fromLocation,omitempty"` // Source registry
	ActionStatus string             `json:"actionStatus,omitempty"`
	StartTime    string             `json:"startTime,omitempty"`
	EndTime      string             `json:"endTime,omitempty"`
	Error        *PropertyValue     `json:"error,omitempty"`
}

// BuildAction represents container image build operations
type BuildAction struct {
	Context      string          `json:"@context,omitempty"`
	Type         string          `json:"@type"` // "CreateAction"
	Identifier   string          `json:"identifier"`
	Name         string          `json:"name,omitempty"`
	Description  string          `json:"description,omitempty"`
	Result       *ContainerImage `json:"result"`           // Resulting image
	Object       *SourceCode     `json:"object,omitempty"` // Source code/Dockerfile
	ActionStatus string          `json:"actionStatus,omitempty"`
	StartTime    string          `json:"startTime,omitempty"`
	EndTime      string          `json:"endTime,omitempty"`
	Error        *PropertyValue  `json:"error,omitempty"`
}

// SourceCode represents build context (Dockerfile, source code)
type SourceCode struct {
	Type           string `json:"@type"` // "SoftwareSourceCode"
	CodeRepository string `json:"codeRepository,omitempty"`
	ContentUrl     string `json:"contentUrl,omitempty"` // Path to Dockerfile
}

// ComputeNode represents a deployment target (host, cluster, namespace)
type ComputeNode struct {
	Type       string                 `json:"@type"` // "Place" or "Organization"
	Identifier string                 `json:"identifier"`
	Name       string                 `json:"name,omitempty"`
	URL        string                 `json:"url,omitempty"` // Docker socket, K8s API endpoint
	Properties map[string]interface{} `json:"additionalProperty,omitempty"`
}

// ============================================================================
// Network and Volume Types
// ============================================================================

// NetworkAction represents network creation/connection
type NetworkAction struct {
	Context      string         `json:"@context,omitempty"`
	Type         string         `json:"@type"` // "ConnectAction"
	Identifier   string         `json:"identifier"`
	Name         string         `json:"name,omitempty"`
	Description  string         `json:"description,omitempty"`
	Object       interface{}    `json:"object"` // Container or network
	ActionStatus string         `json:"actionStatus,omitempty"`
	Error        *PropertyValue `json:"error,omitempty"`
}

// VolumeAction represents volume mounting/assignment
type VolumeAction struct {
	Context      string         `json:"@context,omitempty"`
	Type         string         `json:"@type"` // "AssignAction"
	Identifier   string         `json:"identifier"`
	Name         string         `json:"name,omitempty"`
	Description  string         `json:"description,omitempty"`
	Object       *Volume        `json:"object"` // Volume to mount
	Target       *Container     `json:"target"` // Container to mount to
	ActionStatus string         `json:"actionStatus,omitempty"`
	Error        *PropertyValue `json:"error,omitempty"`
}

// Volume represents a storage volume
type Volume struct {
	Type       string                 `json:"@type"` // "DataCatalog"
	Identifier string                 `json:"identifier"`
	Name       string                 `json:"name,omitempty"`
	URL        string                 `json:"url,omitempty"` // Mount path
	Properties map[string]interface{} `json:"additionalProperty,omitempty"`
}

// ============================================================================
// Helper Functions
// ============================================================================

// ParseContainerAction parses a JSON-LD container action
func ParseContainerAction(data []byte) (interface{}, error) {
	var typeCheck struct {
		Type string `json:"@type"`
	}

	if err := json.Unmarshal(data, &typeCheck); err != nil {
		return nil, fmt.Errorf("failed to parse @type: %w", err)
	}

	switch typeCheck.Type {
	case "ActivateAction":
		var action ActivateAction
		if err := json.Unmarshal(data, &action); err != nil {
			return nil, fmt.Errorf("failed to parse ActivateAction: %w", err)
		}
		return &action, nil

	case "DeactivateAction":
		var action DeactivateAction
		if err := json.Unmarshal(data, &action); err != nil {
			return nil, fmt.Errorf("failed to parse DeactivateAction: %w", err)
		}
		return &action, nil

	case "DownloadAction":
		var action DownloadAction
		if err := json.Unmarshal(data, &action); err != nil {
			return nil, fmt.Errorf("failed to parse DownloadAction: %w", err)
		}
		return &action, nil

	case "CreateAction": // BuildAction
		var action BuildAction
		if err := json.Unmarshal(data, &action); err != nil {
			return nil, fmt.Errorf("failed to parse BuildAction: %w", err)
		}
		return &action, nil

	case "ConnectAction": // NetworkAction
		var action NetworkAction
		if err := json.Unmarshal(data, &action); err != nil {
			return nil, fmt.Errorf("failed to parse NetworkAction: %w", err)
		}
		return &action, nil

	case "AssignAction": // VolumeAction
		var action VolumeAction
		if err := json.Unmarshal(data, &action); err != nil {
			return nil, fmt.Errorf("failed to parse VolumeAction: %w", err)
		}
		return &action, nil

	default:
		return nil, fmt.Errorf("unsupported container action type: %s", typeCheck.Type)
	}
}

// NewContainer creates a new semantic container representation
func NewContainer(name, image, runtime string) *Container {
	return &Container{
		Context:             "https://schema.org",
		Type:                "SoftwareApplication",
		Identifier:          name,
		Name:                name,
		ApplicationCategory: "Container",
		Runtime:             runtime,
		Image: &ContainerImage{
			Type:       "ImageObject",
			Identifier: image,
			ContentUrl: image,
		},
		Properties: make(map[string]interface{}),
	}
}

// NewActivateAction creates a new container deployment action
func NewActivateAction(id, name string, container *Container, target *ComputeNode) *ActivateAction {
	return &ActivateAction{
		Context:      "https://schema.org",
		Type:         "ActivateAction",
		Identifier:   id,
		Name:         name,
		Object:       container,
		Target:       target,
		ActionStatus: "PotentialActionStatus",
	}
}

// NewDeactivateAction creates a new container stop action
func NewDeactivateAction(id, name string, container *Container) *DeactivateAction {
	return &DeactivateAction{
		Context:      "https://schema.org",
		Type:         "DeactivateAction",
		Identifier:   id,
		Name:         name,
		Object:       container,
		ActionStatus: "PotentialActionStatus",
	}
}

// NewDownloadAction creates a new image pull action
func NewDownloadAction(id, name string, image *ContainerImage, registry *ContainerRegistry) *DownloadAction {
	return &DownloadAction{
		Context:      "https://schema.org",
		Type:         "DownloadAction",
		Identifier:   id,
		Name:         name,
		Object:       image,
		FromLocation: registry,
		ActionStatus: "PotentialActionStatus",
	}
}

// ExtractContainerConfig extracts container configuration from additionalProperty
func ExtractContainerConfig(container *Container) (map[string]interface{}, error) {
	if container == nil {
		return nil, fmt.Errorf("container is nil")
	}

	if container.Properties == nil {
		return make(map[string]interface{}), nil
	}

	return container.Properties, nil
}

// ExtractImageName extracts image name from ContainerImage
func ExtractImageName(image *ContainerImage) string {
	if image == nil {
		return ""
	}

	if image.ContentUrl != "" {
		return image.ContentUrl
	}

	return image.Identifier
}
