package semantic

import (
	"encoding/json"
	"fmt"
)

// GraphDB Semantic Types for Repository and Graph Operations
// These types map GraphDB operations to Schema.org vocabulary for semantic orchestration

// ============================================================================
// Repository Types (Schema.org: SoftwareSourceCode)
// ============================================================================

// GraphDBRepository represents a GraphDB repository as Schema.org SoftwareSourceCode
// This allows repositories to be represented semantically in workflows
type GraphDBRepository struct {
	Context        string                 `json:"@context,omitempty"`
	Type           string                 `json:"@type"` // "SoftwareSourceCode"
	Identifier     string                 `json:"identifier"`
	Name           string                 `json:"name,omitempty"`
	CodeRepository string                 `json:"codeRepository"` // Full URL to repository
	Author         *Person                `json:"author,omitempty"`
	AccessMode     string                 `json:"accessMode,omitempty"` // "authenticated", "public"
	Properties     map[string]interface{} `json:"additionalProperty,omitempty"`
}

// Person represents a Schema.org Person (for repository authors)
type Person struct {
	Type string `json:"@type"` // "Person"
	Name string `json:"name"`
}

// ============================================================================
// Graph Types (Schema.org: Dataset)
// ============================================================================

// GraphDBGraph represents a named graph as Schema.org Dataset
// Named graphs are collections of RDF triples within a repository
type GraphDBGraph struct {
	Context               string                 `json:"@context,omitempty"`
	Type                  string                 `json:"@type"` // "Dataset"
	Identifier            string                 `json:"identifier"`
	Name                  string                 `json:"name,omitempty"`
	Description           string                 `json:"description,omitempty"`
	IncludedInDataCatalog *DataCatalog           `json:"includedInDataCatalog,omitempty"`
	EncodingFormat        string                 `json:"encodingFormat,omitempty"` // RDF format: "application/rdf+xml", "text/turtle", etc.
	ContentURL            string                 `json:"contentUrl,omitempty"`     // URL to RDF file
	Properties            map[string]interface{} `json:"additionalProperty,omitempty"`
}

// DataCatalog represents a repository container (Schema.org: DataCatalog)
// Used to indicate which repository contains a graph
type DataCatalog struct {
	Type       string                 `json:"@type"` // "DataCatalog"
	Identifier string                 `json:"identifier"`
	URL        string                 `json:"url"`
	Properties map[string]interface{} `json:"additionalProperty,omitempty"`
}

// ============================================================================
// Action Types (Schema.org: Action hierarchy)
// ============================================================================
// Note: Legacy specific action structs have been removed.
// Use SemanticAction with NewSemantic* constructors instead.

// PropertyValue represents generic property values (Schema.org: PropertyValue)
type PropertyValue struct {
	Type       string                 `json:"@type"` // "PropertyValue"
	Name       string                 `json:"name,omitempty"`
	Value      string                 `json:"value,omitempty"`
	Properties map[string]interface{} `json:"-"` // Additional properties (all extra fields)
}

// UnmarshalJSON implements custom unmarshaling that captures all extra fields
func (pv *PropertyValue) UnmarshalJSON(data []byte) error {
	// First unmarshal into a map to capture all fields
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Extract known fields
	if t, ok := raw["@type"].(string); ok {
		pv.Type = t
	}
	if n, ok := raw["name"].(string); ok {
		pv.Name = n
	}
	if v, ok := raw["value"].(string); ok {
		pv.Value = v
	}

	// Put all extra fields into Properties
	pv.Properties = make(map[string]interface{})
	for k, v := range raw {
		if k != "@type" && k != "name" && k != "value" {
			pv.Properties[k] = v
		}
	}

	return nil
}

// MarshalJSON implements custom marshaling that includes all properties
func (pv *PropertyValue) MarshalJSON() ([]byte, error) {
	// Create a map with all fields
	result := make(map[string]interface{})
	result["@type"] = pv.Type
	if pv.Name != "" {
		result["name"] = pv.Name
	}
	if pv.Value != "" {
		result["value"] = pv.Value
	}
	// Include all additional properties
	for k, v := range pv.Properties {
		result[k] = v
	}
	return json.Marshal(result)
}

// ============================================================================
// Helper Functions
// ============================================================================

// ParseGraphDBAction parses a JSON-LD GraphDB action as SemanticAction
func ParseGraphDBAction(data []byte) (*SemanticAction, error) {
	var typeCheck struct {
		Type string `json:"@type"`
	}

	if err := json.Unmarshal(data, &typeCheck); err != nil {
		return nil, fmt.Errorf("failed to parse @type: %w", err)
	}

	switch typeCheck.Type {
	case "TransferAction", "CreateAction", "DeleteAction", "UpdateAction", "UploadAction":
		var action SemanticAction
		if err := json.Unmarshal(data, &action); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", typeCheck.Type, err)
		}
		return &action, nil
	default:
		return nil, fmt.Errorf("unsupported GraphDB action type: %s", typeCheck.Type)
	}
}

// ExtractRepositoryCredentials extracts connection details from a GraphDBRepository
// Returns: serverURL, username, password, repoName, error
func ExtractRepositoryCredentials(repo *GraphDBRepository) (string, string, string, string, error) {
	if repo == nil {
		return "", "", "", "", fmt.Errorf("repository is nil")
	}

	props := repo.Properties
	if props == nil {
		return "", "", "", repo.Identifier, fmt.Errorf("missing additionalProperty in repository")
	}

	serverURL, ok := props["serverUrl"].(string)
	if !ok {
		return "", "", "", repo.Identifier, fmt.Errorf("missing serverUrl in additionalProperty")
	}

	username, _ := props["username"].(string)
	password, _ := props["password"].(string)

	return serverURL, username, password, repo.Identifier, nil
}

// ExtractGraphIdentifier extracts the graph URI/name from a GraphDBGraph
func ExtractGraphIdentifier(graph *GraphDBGraph) string {
	if graph == nil {
		return ""
	}
	return graph.Identifier
}

// NewGraphDBRepository creates a new semantic repository representation
func NewGraphDBRepository(serverURL, repoName, username, password string) *GraphDBRepository {
	return &GraphDBRepository{
		Context:        "https://schema.org",
		Type:           "SoftwareSourceCode",
		Identifier:     repoName,
		CodeRepository: fmt.Sprintf("%s/repositories/%s", serverURL, repoName),
		AccessMode:     "authenticated",
		Properties: map[string]interface{}{
			"serverUrl": serverURL,
			"username":  username,
			"password":  password,
		},
	}
}

// NewGraphDBGraph creates a new semantic graph representation
func NewGraphDBGraph(graphURI, repoURL, repoName string) *GraphDBGraph {
	return &GraphDBGraph{
		Context:    "https://schema.org",
		Type:       "Dataset",
		Identifier: graphURI,
		IncludedInDataCatalog: &DataCatalog{
			Type:       "DataCatalog",
			Identifier: repoName,
			URL:        repoURL,
		},
	}
}

// ActionStatusCompleted marks an action as completed
func ActionStatusCompleted() string {
	return "CompletedActionStatus"
}

// ActionStatusFailed marks an action as failed
func ActionStatusFailed() string {
	return "FailedActionStatus"
}

// ActionStatusActive marks an action as currently running
func ActionStatusActive() string {
	return "ActiveActionStatus"
}

// ActionStatusPotential marks an action as potential (not yet started)
func ActionStatusPotential() string {
	return "PotentialActionStatus"
}

// ActionStatusPaused marks an action as paused (execution suspended)
func ActionStatusPaused() string {
	return "PausedActionStatus"
}

// NewSemanticTransferAction creates a TransferAction using SemanticAction
func NewSemanticTransferAction(id, name string, fromLocation, toLocation, object interface{}) *SemanticAction {
	action := &SemanticAction{
		Context:      "https://schema.org",
		Type:         "TransferAction",
		Identifier:   id,
		Name:         name,
		ActionStatus: "PotentialActionStatus",
		Properties:   make(map[string]interface{}),
	}

	if fromLocation != nil {
		action.Properties["fromLocation"] = fromLocation
	}
	if toLocation != nil {
		action.Properties["toLocation"] = toLocation
	}
	if object != nil {
		action.Properties["object"] = object
	}

	return action
}

// NewSemanticCreateAction creates a CreateAction using SemanticAction
func NewSemanticCreateAction(id, name string, result interface{}) *SemanticAction {
	action := &SemanticAction{
		Context:      "https://schema.org",
		Type:         "CreateAction",
		Identifier:   id,
		Name:         name,
		ActionStatus: "PotentialActionStatus",
		Properties:   make(map[string]interface{}),
	}

	if result != nil {
		action.Properties["result"] = result
	}

	return action
}

// NewSemanticDeleteAction creates a DeleteAction using SemanticAction
func NewSemanticDeleteAction(id, name string, object interface{}) *SemanticAction {
	action := &SemanticAction{
		Context:      "https://schema.org",
		Type:         "DeleteAction",
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

// NewSemanticUpdateAction creates an UpdateAction using SemanticAction
func NewSemanticUpdateAction(id, name, targetName string, object, replacesObject interface{}) *SemanticAction {
	action := &SemanticAction{
		Context:      "https://schema.org",
		Type:         "UpdateAction",
		Identifier:   id,
		Name:         name,
		ActionStatus: "PotentialActionStatus",
		Properties:   make(map[string]interface{}),
	}

	if object != nil {
		action.Properties["object"] = object
	}
	if targetName != "" {
		action.Properties["targetName"] = targetName
	}
	if replacesObject != nil {
		action.Properties["replacesObject"] = replacesObject
	}

	return action
}

// NewSemanticUploadAction creates an UploadAction using SemanticAction
func NewSemanticUploadAction(id, name string, object, target interface{}) *SemanticAction {
	action := &SemanticAction{
		Context:      "https://schema.org",
		Type:         "UploadAction",
		Identifier:   id,
		Name:         name,
		ActionStatus: "PotentialActionStatus",
		Properties:   make(map[string]interface{}),
	}

	if object != nil {
		action.Properties["object"] = object
	}
	if target != nil {
		action.Properties["target"] = target
	}

	return action
}

// GetGraphDBRepositoryFromAction extracts GraphDBRepository from SemanticAction properties
// Can check multiple property keys: fromLocation, toLocation, target, object
func GetGraphDBRepositoryFromAction(action *SemanticAction, propertyKey string) (*GraphDBRepository, error) {
	if action == nil || action.Properties == nil {
		return nil, fmt.Errorf("action or properties is nil")
	}

	prop, ok := action.Properties[propertyKey]
	if !ok {
		return nil, fmt.Errorf("no %s found in action properties", propertyKey)
	}

	switch v := prop.(type) {
	case *GraphDBRepository:
		return v, nil
	case GraphDBRepository:
		return &v, nil
	case map[string]interface{}:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal GraphDBRepository: %w", err)
		}
		var repo GraphDBRepository
		if err := json.Unmarshal(data, &repo); err != nil {
			return nil, fmt.Errorf("failed to unmarshal GraphDBRepository: %w", err)
		}
		return &repo, nil
	default:
		return nil, fmt.Errorf("unexpected %s type: %T", propertyKey, prop)
	}
}

// GetGraphDBGraphFromAction extracts GraphDBGraph from SemanticAction properties
// Can check multiple property keys: object, result, target
func GetGraphDBGraphFromAction(action *SemanticAction, propertyKey string) (*GraphDBGraph, error) {
	if action == nil || action.Properties == nil {
		return nil, fmt.Errorf("action or properties is nil")
	}

	prop, ok := action.Properties[propertyKey]
	if !ok {
		return nil, fmt.Errorf("no %s found in action properties", propertyKey)
	}

	switch v := prop.(type) {
	case *GraphDBGraph:
		return v, nil
	case GraphDBGraph:
		return &v, nil
	case map[string]interface{}:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal GraphDBGraph: %w", err)
		}
		var graph GraphDBGraph
		if err := json.Unmarshal(data, &graph); err != nil {
			return nil, fmt.Errorf("failed to unmarshal GraphDBGraph: %w", err)
		}
		return &graph, nil
	default:
		return nil, fmt.Errorf("unexpected %s type: %T", propertyKey, prop)
	}
}

// GetDataCatalogFromAction extracts DataCatalog from SemanticAction properties
func GetDataCatalogFromAction(action *SemanticAction, propertyKey string) (*DataCatalog, error) {
	if action == nil || action.Properties == nil {
		return nil, fmt.Errorf("action or properties is nil")
	}

	prop, ok := action.Properties[propertyKey]
	if !ok {
		return nil, fmt.Errorf("no %s found in action properties", propertyKey)
	}

	switch v := prop.(type) {
	case *DataCatalog:
		return v, nil
	case DataCatalog:
		return &v, nil
	case map[string]interface{}:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal DataCatalog: %w", err)
		}
		var catalog DataCatalog
		if err := json.Unmarshal(data, &catalog); err != nil {
			return nil, fmt.Errorf("failed to unmarshal DataCatalog: %w", err)
		}
		return &catalog, nil
	default:
		return nil, fmt.Errorf("unexpected %s type: %T", propertyKey, prop)
	}
}

// GetTargetNameFromAction extracts targetName from SemanticAction properties
func GetTargetNameFromAction(action *SemanticAction) string {
	if action == nil || action.Properties == nil {
		return ""
	}

	if name, ok := action.Properties["targetName"].(string); ok {
		return name
	}

	return ""
}
