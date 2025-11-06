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
// Note: Legacy specific action structs have been removed.
// Use SemanticAction with NewSemantic* constructors instead.

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

// ParseContainerAction parses a JSON-LD container action as SemanticAction
func ParseContainerAction(data []byte) (*SemanticAction, error) {
	var typeCheck struct {
		Type string `json:"@type"`
	}

	if err := json.Unmarshal(data, &typeCheck); err != nil {
		return nil, fmt.Errorf("failed to parse @type: %w", err)
	}

	switch typeCheck.Type {
	case "ActivateAction", "DeactivateAction", "DownloadAction", "CreateAction", "ConnectAction", "AssignAction":
		var action SemanticAction
		if err := json.Unmarshal(data, &action); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", typeCheck.Type, err)
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

// ============================================================================
// SemanticAction Constructors for Container Operations
// ============================================================================

// NewSemanticActivateAction creates a container deployment action using SemanticAction
// This is the new recommended way - provides full semantic action capabilities
func NewSemanticActivateAction(id, name string, container *Container, target *ComputeNode) *SemanticAction {
	action := &SemanticAction{
		Context:      "https://schema.org",
		Type:         "ActivateAction",
		Identifier:   id,
		Name:         name,
		ActionStatus: "PotentialActionStatus",
		Properties:   make(map[string]interface{}),
	}

	// Store container-specific data in Properties
	if container != nil {
		action.Properties["object"] = container
	}
	if target != nil {
		action.Properties["target"] = target
	}

	return action
}

// NewSemanticBuildAction creates a container image build action using SemanticAction
func NewSemanticBuildAction(id, name string, resultImage *ContainerImage, sourceCode *SourceCode) *SemanticAction {
	action := &SemanticAction{
		Context:      "https://schema.org",
		Type:         "CreateAction", // BuildAction uses CreateAction in Schema.org
		Identifier:   id,
		Name:         name,
		ActionStatus: "PotentialActionStatus",
		Properties:   make(map[string]interface{}),
	}

	if resultImage != nil {
		action.Properties["result"] = resultImage
	}
	if sourceCode != nil {
		action.Properties["object"] = sourceCode
	}

	return action
}

// NewSemanticDeactivateAction creates a container stop action using SemanticAction
func NewSemanticDeactivateAction(id, name string, container *Container) *SemanticAction {
	action := &SemanticAction{
		Context:      "https://schema.org",
		Type:         "DeactivateAction",
		Identifier:   id,
		Name:         name,
		ActionStatus: "PotentialActionStatus",
		Properties:   make(map[string]interface{}),
	}

	if container != nil {
		action.Properties["object"] = container
	}

	return action
}

// NewSemanticDownloadAction creates an image pull action using SemanticAction
func NewSemanticDownloadAction(id, name string, image *ContainerImage, registry *ContainerRegistry) *SemanticAction {
	action := &SemanticAction{
		Context:      "https://schema.org",
		Type:         "DownloadAction",
		Identifier:   id,
		Name:         name,
		ActionStatus: "PotentialActionStatus",
		Properties:   make(map[string]interface{}),
	}

	if image != nil {
		action.Properties["object"] = image
	}
	if registry != nil {
		action.Properties["fromLocation"] = registry
	}

	return action
}

// NewSemanticNetworkAction creates a network connection action using SemanticAction
func NewSemanticNetworkAction(id, name string, object interface{}) *SemanticAction {
	action := &SemanticAction{
		Context:      "https://schema.org",
		Type:         "ConnectAction",
		Identifier:   id,
		Name:         name,
		ActionStatus: "PotentialActionStatus",
		Properties:   make(map[string]interface{}),
	}

	if object != nil {
		action.Properties["object"] = object
	}

	return action
}

// NewSemanticVolumeAction creates a volume mount action using SemanticAction
func NewSemanticVolumeAction(id, name string, volume *Volume, targetContainer *Container) *SemanticAction {
	action := &SemanticAction{
		Context:      "https://schema.org",
		Type:         "AssignAction",
		Identifier:   id,
		Name:         name,
		ActionStatus: "PotentialActionStatus",
		Properties:   make(map[string]interface{}),
	}

	if volume != nil {
		action.Properties["object"] = volume
	}
	if targetContainer != nil {
		action.Properties["target"] = targetContainer
	}

	return action
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

// GetContainerFromAction extracts Container from SemanticAction properties
func GetContainerFromAction(action *SemanticAction) (*Container, error) {
	if action == nil || action.Properties == nil {
		return nil, fmt.Errorf("action or properties is nil")
	}

	obj, ok := action.Properties["object"]
	if !ok {
		return nil, fmt.Errorf("no object found in action properties")
	}

	// Handle type assertion - could be Container or map[string]interface{}
	switch v := obj.(type) {
	case *Container:
		return v, nil
	case Container:
		return &v, nil
	case map[string]interface{}:
		// Convert map to Container via JSON marshaling
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal container: %w", err)
		}
		var container Container
		if err := json.Unmarshal(data, &container); err != nil {
			return nil, fmt.Errorf("failed to unmarshal container: %w", err)
		}
		return &container, nil
	default:
		return nil, fmt.Errorf("unexpected object type: %T", obj)
	}
}

// GetTargetFromAction extracts ComputeNode target from SemanticAction properties
func GetTargetFromAction(action *SemanticAction) (*ComputeNode, error) {
	if action == nil || action.Properties == nil {
		return nil, fmt.Errorf("action or properties is nil")
	}

	target, ok := action.Properties["target"]
	if !ok {
		return nil, nil // Target is optional
	}

	// Handle type assertion
	switch v := target.(type) {
	case *ComputeNode:
		return v, nil
	case ComputeNode:
		return &v, nil
	case map[string]interface{}:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal target: %w", err)
		}
		var node ComputeNode
		if err := json.Unmarshal(data, &node); err != nil {
			return nil, fmt.Errorf("failed to unmarshal target: %w", err)
		}
		return &node, nil
	default:
		return nil, fmt.Errorf("unexpected target type: %T", target)
	}
}

// GetImageFromAction extracts ContainerImage from SemanticAction properties
func GetImageFromAction(action *SemanticAction) (*ContainerImage, error) {
	if action == nil || action.Properties == nil {
		return nil, fmt.Errorf("action or properties is nil")
	}

	// Try "object" first (for DownloadAction), then "result" (for BuildAction)
	obj, hasObject := action.Properties["object"]
	if !hasObject {
		obj, hasObject = action.Properties["result"]
		if !hasObject {
			return nil, fmt.Errorf("no object or result found in action properties")
		}
	}

	switch v := obj.(type) {
	case *ContainerImage:
		return v, nil
	case ContainerImage:
		return &v, nil
	case map[string]interface{}:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal image: %w", err)
		}
		var image ContainerImage
		if err := json.Unmarshal(data, &image); err != nil {
			return nil, fmt.Errorf("failed to unmarshal image: %w", err)
		}
		return &image, nil
	default:
		return nil, fmt.Errorf("unexpected image type: %T", obj)
	}
}

// GetSourceCodeFromAction extracts SourceCode from SemanticAction properties
func GetSourceCodeFromAction(action *SemanticAction) (*SourceCode, error) {
	if action == nil || action.Properties == nil {
		return nil, fmt.Errorf("action or properties is nil")
	}

	obj, ok := action.Properties["object"]
	if !ok {
		return nil, fmt.Errorf("no object found in action properties")
	}

	switch v := obj.(type) {
	case *SourceCode:
		return v, nil
	case SourceCode:
		return &v, nil
	case map[string]interface{}:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal source code: %w", err)
		}
		var sourceCode SourceCode
		if err := json.Unmarshal(data, &sourceCode); err != nil {
			return nil, fmt.Errorf("failed to unmarshal source code: %w", err)
		}
		return &sourceCode, nil
	default:
		return nil, fmt.Errorf("unexpected source code type: %T", obj)
	}
}

// GetVolumeFromAction extracts Volume from SemanticAction properties
func GetVolumeFromAction(action *SemanticAction) (*Volume, error) {
	if action == nil || action.Properties == nil {
		return nil, fmt.Errorf("action or properties is nil")
	}

	obj, ok := action.Properties["object"]
	if !ok {
		return nil, fmt.Errorf("no object found in action properties")
	}

	switch v := obj.(type) {
	case *Volume:
		return v, nil
	case Volume:
		return &v, nil
	case map[string]interface{}:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal volume: %w", err)
		}
		var volume Volume
		if err := json.Unmarshal(data, &volume); err != nil {
			return nil, fmt.Errorf("failed to unmarshal volume: %w", err)
		}
		return &volume, nil
	default:
		return nil, fmt.Errorf("unexpected volume type: %T", obj)
	}
}

// GetRegistryFromAction extracts ContainerRegistry from SemanticAction properties
func GetRegistryFromAction(action *SemanticAction) (*ContainerRegistry, error) {
	if action == nil || action.Properties == nil {
		return nil, fmt.Errorf("action or properties is nil")
	}

	loc, ok := action.Properties["fromLocation"]
	if !ok {
		return nil, nil // Registry is optional
	}

	switch v := loc.(type) {
	case *ContainerRegistry:
		return v, nil
	case ContainerRegistry:
		return &v, nil
	case map[string]interface{}:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal registry: %w", err)
		}
		var registry ContainerRegistry
		if err := json.Unmarshal(data, &registry); err != nil {
			return nil, fmt.Errorf("failed to unmarshal registry: %w", err)
		}
		return &registry, nil
	default:
		return nil, fmt.Errorf("unexpected registry type: %T", loc)
	}
}
