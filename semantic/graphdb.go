package semantic

import (
	"encoding/json"
	"fmt"
	"time"
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

// TransferAction represents repository or graph migration operations
// Maps to Schema.org TransferAction for moving data from source to target
type TransferAction struct {
	Context      string         `json:"@context,omitempty"`
	Type         string         `json:"@type"` // "TransferAction"
	Identifier   string         `json:"identifier"`
	Name         string         `json:"name,omitempty"`
	Description  string         `json:"description,omitempty"`
	FromLocation interface{}    `json:"fromLocation"`     // *GraphDBRepository or *GraphDBGraph
	ToLocation   interface{}    `json:"toLocation"`       // *GraphDBRepository or *GraphDBGraph
	Object       interface{}    `json:"object,omitempty"` // For graph migrations: *GraphDBGraph
	ActionStatus string         `json:"actionStatus,omitempty"`
	StartTime    *time.Time     `json:"startTime,omitempty"`
	EndTime      *time.Time     `json:"endTime,omitempty"`
	Error        *PropertyValue `json:"error,omitempty"`
}

// CreateAction represents repository or graph creation operations
// Maps to Schema.org CreateAction for creating new resources
type CreateAction struct {
	Context      string         `json:"@context,omitempty"`
	Type         string         `json:"@type"` // "CreateAction"
	Identifier   string         `json:"identifier"`
	Name         string         `json:"name,omitempty"`
	Description  string         `json:"description,omitempty"`
	Result       interface{}    `json:"result"` // *GraphDBRepository or *GraphDBGraph to create
	ActionStatus string         `json:"actionStatus,omitempty"`
	StartTime    *time.Time     `json:"startTime,omitempty"`
	EndTime      *time.Time     `json:"endTime,omitempty"`
	Error        *PropertyValue `json:"error,omitempty"`
}

// DeleteAction represents repository or graph deletion operations
// Maps to Schema.org DeleteAction for removing resources
type DeleteAction struct {
	Context      string         `json:"@context,omitempty"`
	Type         string         `json:"@type"` // "DeleteAction"
	Identifier   string         `json:"identifier"`
	Name         string         `json:"name,omitempty"`
	Description  string         `json:"description,omitempty"`
	Object       interface{}    `json:"object"` // *GraphDBRepository or *GraphDBGraph to delete
	ActionStatus string         `json:"actionStatus,omitempty"`
	StartTime    *time.Time     `json:"startTime,omitempty"`
	EndTime      *time.Time     `json:"endTime,omitempty"`
	Error        *PropertyValue `json:"error,omitempty"`
}

// UpdateAction represents repository or graph rename operations
// Maps to Schema.org UpdateAction for modifying resources
type UpdateAction struct {
	Context        string         `json:"@context,omitempty"`
	Type           string         `json:"@type"` // "UpdateAction"
	Identifier     string         `json:"identifier"`
	Name           string         `json:"name,omitempty"`
	Description    string         `json:"description,omitempty"`
	Object         interface{}    `json:"object"`                   // Resource to update
	TargetName     string         `json:"targetName"`               // New name
	ReplacesObject interface{}    `json:"replacesObject,omitempty"` // Old resource
	ActionStatus   string         `json:"actionStatus,omitempty"`
	StartTime      *time.Time     `json:"startTime,omitempty"`
	EndTime        *time.Time     `json:"endTime,omitempty"`
	Error          *PropertyValue `json:"error,omitempty"`
}

// UploadAction represents graph or repository import operations
// Maps to Schema.org UploadAction for importing RDF data
type UploadAction struct {
	Context      string         `json:"@context,omitempty"`
	Type         string         `json:"@type"` // "UploadAction"
	Identifier   string         `json:"identifier"`
	Name         string         `json:"name,omitempty"`
	Description  string         `json:"description,omitempty"`
	Object       interface{}    `json:"object"` // *GraphDBGraph with data to upload
	Target       interface{}    `json:"target"` // *DataCatalog or *GraphDBRepository
	ActionStatus string         `json:"actionStatus,omitempty"`
	StartTime    *time.Time     `json:"startTime,omitempty"`
	EndTime      *time.Time     `json:"endTime,omitempty"`
	Error        *PropertyValue `json:"error,omitempty"`
}

// PropertyValue represents generic property values (Schema.org: PropertyValue)
type PropertyValue struct {
	Type  string `json:"@type"` // "PropertyValue"
	Name  string `json:"name,omitempty"`
	Value string `json:"value"`
}

// ============================================================================
// Helper Functions
// ============================================================================

// ParseGraphDBAction parses a JSON-LD GraphDB action and returns the appropriate type
func ParseGraphDBAction(data []byte) (interface{}, error) {
	// First, determine the action type
	var typeCheck struct {
		Type string `json:"@type"`
	}

	if err := json.Unmarshal(data, &typeCheck); err != nil {
		return nil, fmt.Errorf("failed to parse @type: %w", err)
	}

	// Parse based on type
	switch typeCheck.Type {
	case "TransferAction":
		var action TransferAction
		if err := json.Unmarshal(data, &action); err != nil {
			return nil, fmt.Errorf("failed to parse TransferAction: %w", err)
		}
		return &action, nil

	case "CreateAction":
		var action CreateAction
		if err := json.Unmarshal(data, &action); err != nil {
			return nil, fmt.Errorf("failed to parse CreateAction: %w", err)
		}
		return &action, nil

	case "DeleteAction":
		var action DeleteAction
		if err := json.Unmarshal(data, &action); err != nil {
			return nil, fmt.Errorf("failed to parse DeleteAction: %w", err)
		}
		return &action, nil

	case "UpdateAction":
		var action UpdateAction
		if err := json.Unmarshal(data, &action); err != nil {
			return nil, fmt.Errorf("failed to parse UpdateAction: %w", err)
		}
		return &action, nil

	case "UploadAction":
		var action UploadAction
		if err := json.Unmarshal(data, &action); err != nil {
			return nil, fmt.Errorf("failed to parse UploadAction: %w", err)
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

// NewTransferAction creates a new semantic TransferAction for migrations
func NewTransferAction(id, name string, from, to *GraphDBRepository) *TransferAction {
	return &TransferAction{
		Context:      "https://schema.org",
		Type:         "TransferAction",
		Identifier:   id,
		Name:         name,
		FromLocation: from,
		ToLocation:   to,
		ActionStatus: "PotentialActionStatus",
	}
}

// NewCreateAction creates a new semantic CreateAction for repository/graph creation
func NewCreateAction(id, name string, result interface{}) *CreateAction {
	return &CreateAction{
		Context:      "https://schema.org",
		Type:         "CreateAction",
		Identifier:   id,
		Name:         name,
		Result:       result,
		ActionStatus: "PotentialActionStatus",
	}
}

// NewDeleteAction creates a new semantic DeleteAction for deletion operations
func NewDeleteAction(id, name string, object interface{}) *DeleteAction {
	return &DeleteAction{
		Context:      "https://schema.org",
		Type:         "DeleteAction",
		Identifier:   id,
		Name:         name,
		Object:       object,
		ActionStatus: "PotentialActionStatus",
	}
}

// NewUploadAction creates a new semantic UploadAction for import operations
func NewUploadAction(id, name string, object, target interface{}) *UploadAction {
	return &UploadAction{
		Context:      "https://schema.org",
		Type:         "UploadAction",
		Identifier:   id,
		Name:         name,
		Object:       object,
		Target:       target,
		ActionStatus: "PotentialActionStatus",
	}
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
