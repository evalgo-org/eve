package runtime

import (
	"encoding/json"
	"fmt"
	"time"
)

// RuntimeAction represents a Schema.org Action at runtime with complete JSON-LD preservation.
// It uses custom marshaling to preserve ALL fields from the original JSON-LD while providing
// typed access to commonly used fields for performance.
//
// Key principle: No data loss. Everything from the definition is preserved in AllFields.
type RuntimeAction struct {
	// Core Schema.org fields (typed for fast access)
	Context     interface{} `json:"@context,omitempty"`
	Type        string      `json:"@type"`
	Identifier  string      `json:"identifier"`
	Name        string      `json:"name,omitempty"`
	Description string      `json:"description,omitempty"`

	// Execution state
	ActionStatus string     `json:"actionStatus,omitempty"`
	StartTime    *time.Time `json:"startTime,omitempty"`
	EndTime      *time.Time `json:"endTime,omitempty"`

	// Dependencies
	Requires []string `json:"requires,omitempty"`

	// Semantic relationships
	IsPartOf      string `json:"isPartOf,omitempty"`
	ExampleOfWork string `json:"exampleOfWork,omitempty"`

	// Query specification (can be string or ActionQuery object)
	Query interface{} `json:"query,omitempty"`

	// Target specification
	Target *ActionTarget `json:"target,omitempty"`

	// Result specification/output
	Result *ActionResult `json:"result,omitempty"`

	// Error information
	Error *ActionError `json:"error,omitempty"`

	// Object being acted upon
	Object map[string]interface{} `json:"object,omitempty"`

	// Control metadata (routing and execution control)
	ControlMetadata *ControlMetadata `json:"controlMetadata,omitempty"`

	// Timestamps
	DateCreated  time.Time `json:"dateCreated,omitempty"`
	DateModified time.Time `json:"dateModified,omitempty"`

	// Additional type for control actions, etc.
	AdditionalType string `json:"additionalType,omitempty"`

	// Agent who triggered/executed the action
	Agent map[string]interface{} `json:"agent,omitempty"`

	// THE KEY: Preserve ALL other fields not explicitly typed above
	// This map contains the complete JSON-LD document
	AllFields map[string]interface{} `json:"-"`
}

// ActionQuery represents the query specification of an action
type ActionQuery struct {
	Type       string `json:"@type,omitempty"`
	QueryInput string `json:"queryInput,omitempty"`
}

// ActionTarget represents the target/endpoint of an action
type ActionTarget struct {
	Type               string                 `json:"@type,omitempty"`
	URL                string                 `json:"url,omitempty"`
	URLTemplate        string                 `json:"urlTemplate,omitempty"`
	HTTPMethod         string                 `json:"httpMethod,omitempty"`
	ContentType        string                 `json:"contentType,omitempty"`
	AdditionalProperty map[string]interface{} `json:"additionalProperty,omitempty"`
}

// ActionResult represents the result specification or output of an action
type ActionResult struct {
	Type               string                 `json:"@type,omitempty"`
	Identifier         string                 `json:"identifier,omitempty"`
	ContentURL         string                 `json:"contentUrl,omitempty"`
	EncodingFormat     string                 `json:"encodingFormat,omitempty"`
	Text               string                 `json:"text,omitempty"`
	ContentSize        string                 `json:"contentSize,omitempty"`
	SHA256             string                 `json:"sha256,omitempty"`
	DateCreated        string                 `json:"dateCreated,omitempty"`
	Description        string                 `json:"description,omitempty"`
	AdditionalProperty map[string]interface{} `json:"additionalProperty,omitempty"`
}

// ActionError represents error information for failed actions
type ActionError struct {
	Type               string                 `json:"@type,omitempty"`
	Name               string                 `json:"name,omitempty"`
	Description        string                 `json:"description,omitempty"`
	AdditionalProperty map[string]interface{} `json:"additionalProperty,omitempty"`
}

// ControlMetadata contains execution control and routing information
type ControlMetadata struct {
	URL          string `json:"url,omitempty"`
	HTTPMethod   string `json:"httpMethod,omitempty"`
	Enabled      bool   `json:"enabled,omitempty"`
	RetryCount   int    `json:"retryCount,omitempty"`
	RetryBackoff string `json:"retryBackoff,omitempty"`
	Singleton    bool   `json:"singleton,omitempty"`
}

// UnmarshalJSON implements custom unmarshaling to preserve all fields
func (a *RuntimeAction) UnmarshalJSON(data []byte) error {
	// First, unmarshal into AllFields to capture everything
	a.AllFields = make(map[string]interface{})
	if err := json.Unmarshal(data, &a.AllFields); err != nil {
		return fmt.Errorf("failed to unmarshal into AllFields: %w", err)
	}

	// Then unmarshal into typed fields using an alias to avoid recursion
	type Alias RuntimeAction
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(a),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return fmt.Errorf("failed to unmarshal typed fields: %w", err)
	}

	return nil
}

// MarshalJSON implements custom marshaling to output complete JSON-LD
// AllFields is the source of truth - typed fields are only used if not in AllFields
func (a *RuntimeAction) MarshalJSON() ([]byte, error) {
	// Start with AllFields (source of truth)
	output := make(map[string]interface{})
	for k, v := range a.AllFields {
		output[k] = v
	}

	// Add typed field values ONLY if not already in AllFields
	// This allows variable substitution and SetField to work correctly
	if a.Context != nil && output["@context"] == nil {
		output["@context"] = a.Context
	}
	if a.Type != "" && output["@type"] == nil {
		output["@type"] = a.Type
	}
	if a.Identifier != "" && output["identifier"] == nil {
		output["identifier"] = a.Identifier
	}
	if a.Name != "" && output["name"] == nil {
		output["name"] = a.Name
	}
	if a.Description != "" && output["description"] == nil {
		output["description"] = a.Description
	}
	if a.ActionStatus != "" && output["actionStatus"] == nil {
		output["actionStatus"] = a.ActionStatus
	}
	if a.StartTime != nil && output["startTime"] == nil {
		output["startTime"] = a.StartTime
	}
	if a.EndTime != nil && output["endTime"] == nil {
		output["endTime"] = a.EndTime
	}
	if len(a.Requires) > 0 && output["requires"] == nil {
		output["requires"] = a.Requires
	}
	if a.IsPartOf != "" && output["isPartOf"] == nil {
		output["isPartOf"] = a.IsPartOf
	}
	if a.ExampleOfWork != "" && output["exampleOfWork"] == nil {
		output["exampleOfWork"] = a.ExampleOfWork
	}
	if a.Query != nil && output["query"] == nil {
		output["query"] = a.Query
	}
	if a.Target != nil && output["target"] == nil {
		output["target"] = a.Target
	}
	if a.Result != nil && output["result"] == nil {
		output["result"] = a.Result
	}
	if a.Error != nil && output["error"] == nil {
		output["error"] = a.Error
	}
	if len(a.Object) > 0 && output["object"] == nil {
		output["object"] = a.Object
	}
	if a.ControlMetadata != nil && output["controlMetadata"] == nil {
		output["controlMetadata"] = a.ControlMetadata
	}
	if !a.DateCreated.IsZero() && output["dateCreated"] == nil {
		output["dateCreated"] = a.DateCreated
	}
	if !a.DateModified.IsZero() && output["dateModified"] == nil {
		output["dateModified"] = a.DateModified
	}
	if a.AdditionalType != "" && output["additionalType"] == nil {
		output["additionalType"] = a.AdditionalType
	}
	if len(a.Agent) > 0 && output["agent"] == nil {
		output["agent"] = a.Agent
	}

	return json.Marshal(output)
}

// DeepCopy creates a deep copy of the RuntimeAction
func (a *RuntimeAction) DeepCopy() *RuntimeAction {
	// Marshal and unmarshal to create a complete deep copy
	data, err := json.Marshal(a)
	if err != nil {
		return nil
	}

	copy := &RuntimeAction{}
	if err := json.Unmarshal(data, copy); err != nil {
		return nil
	}

	return copy
}

// GetField retrieves a nested field from AllFields using dot notation
// Example: action.GetField("result.contentUrl")
func (a *RuntimeAction) GetField(path string) (interface{}, error) {
	return getNestedField(a.AllFields, path)
}

// SetField sets a nested field in AllFields using dot notation
func (a *RuntimeAction) SetField(path string, value interface{}) error {
	return setNestedField(a.AllFields, path, value)
}
