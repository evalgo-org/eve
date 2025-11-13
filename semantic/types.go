package semantic

import (
	"encoding/json"
	"time"
)

// Common Schema.org base types that all semantic objects share

// SemanticAction represents a Schema.org Action
// Base type for ScheduledAction, CommunicateAction, etc.
type SemanticAction struct {
	Context      string                 `json:"@context"`
	Type         string                 `json:"@type"`
	ID           string                 `json:"@id,omitempty"`
	Identifier   string                 `json:"identifier,omitempty"`
	Name         string                 `json:"name,omitempty"`
	Description  string                 `json:"description,omitempty"`
	ActionStatus string                 `json:"actionStatus"` // PotentialActionStatus, ActiveActionStatus, CompletedActionStatus, FailedActionStatus, PausedActionStatus
	Agent        *SemanticAgent         `json:"agent,omitempty"`
	Object       *SemanticObject        `json:"object,omitempty"`
	Instrument   interface{}            `json:"instrument,omitempty"`
	Target       interface{}            `json:"target,omitempty"`    // Can be EntryPoint, InfisicalProject, DataCatalog, etc.
	TargetUrl    string                 `json:"targetUrl,omitempty"` // Specific target path/key (e.g., S3 object key)
	Query        interface{}            `json:"query,omitempty"`     // For SearchAction (SPARQL query, search parameters)
	StartTime    *time.Time             `json:"startTime,omitempty"`
	EndTime      *time.Time             `json:"endTime,omitempty"`
	Duration     string                 `json:"duration,omitempty"` // ISO 8601 duration
	Result       *SemanticResult        `json:"result,omitempty"`
	Error        *SemanticError         `json:"error,omitempty"`
	Properties   map[string]interface{} `json:"additionalProperty,omitempty"`

	// Semantic relationship fields (Schema.org)
	PartOf     string `json:"isPartOf,omitempty"`      // Parent workflow/action this belongs to
	InstanceOf string `json:"exampleOfWork,omitempty"` // Template this is an instance of (Schema.org uses exampleOfWork for instances)

	// Workflow execution group identifier (unified tracking for all workflow types)
	// For workflow instances: UUID prefix (e.g., "abc123")
	// For MapAction iterations: mapActionRunID (e.g., "mapaction-parent-id-run-12345")
	// For nested workflows: inherited from parent ActivateAction
	WorkflowGroup string `json:"workflowGroup,omitempty"`
}

// SemanticScheduledAction represents a task that runs on a schedule
// Inherits Target, TargetUrl, and Query fields from SemanticAction
type SemanticScheduledAction struct {
	SemanticAction
	Requires []string          `json:"requires,omitempty"` // Dependencies (Action @id references)
	Schedule *SemanticSchedule `json:"schedule,omitempty"`
	Created  time.Time         `json:"dateCreated"`
	Modified time.Time         `json:"dateModified"`
	Meta     *ActionMeta       `json:"controlMetadata,omitempty"` // Control metadata (separate from semantic properties)
}

// ActionMeta contains control metadata for action execution (not semantic properties)
// This is kept separate from additionalProperty to avoid polluting semantic parameters
type ActionMeta struct {
	Enabled         bool              `json:"enabled"`                   // Whether action is enabled
	RetryCount      int               `json:"retryCount"`                // Number of retries on failure
	RetryBackoff    string            `json:"retryBackoff"`              // Backoff strategy (linear, exponential)
	Singleton       bool              `json:"singleton"`                 // Only one instance can run at a time
	URL             string            `json:"url,omitempty"`             // Service endpoint URL (for routing)
	HTTPMethod      string            `json:"httpMethod,omitempty"`      // HTTP method (for routing)
	PotentialAction []PotentialAction `json:"potentialAction,omitempty"` // Available UI controls/actions
}

// PotentialAction represents an available action/control that can be performed on a workflow
// Following Schema.org's potentialAction pattern
type PotentialAction struct {
	Type string `json:"@type"` // RunAction, StopAction, DeleteAction, CreateAction, etc.
	Name string `json:"name"`  // Display name for the action (e.g., "Run Now", "Stop", "Delete")
}

// EntryPoint represents a Schema.org EntryPoint (action target like HTTP endpoint)
type EntryPoint struct {
	Type       string            `json:"@type"` // Must be "EntryPoint"
	URL        string            `json:"url"`
	HTTPMethod string            `json:"httpMethod,omitempty"` // GET, POST, etc.
	Headers    map[string]string `json:"headers,omitempty"`
}

// SemanticHTTPAction represents an HTTP request as an action
type SemanticHTTPAction struct {
	SemanticAction
	URL         string            `json:"url"`
	Method      string            `json:"httpMethod"`
	Headers     map[string]string `json:"httpHeaders,omitempty"`
	Body        interface{}       `json:"text,omitempty"`
	Timeout     int               `json:"temporal,omitempty"`
	CreatedTime string            `json:"dateCreated,omitempty"`
}

// SemanticAgent represents the software/person executing an action
type SemanticAgent struct {
	Type string `json:"@type"` // SoftwareApplication, Person, Organization
	Name string `json:"name"`
}

// SemanticObject represents what an action operates on
type SemanticObject struct {
	Type                string                 `json:"@type"`                // SoftwareSourceCode, DataFeed, CreativeWork, DigitalDocument, MediaObject
	Identifier          string                 `json:"identifier,omitempty"` // Unique identifier
	Name                string                 `json:"name,omitempty"`
	Text                string                 `json:"text,omitempty"`           // Text content or embedded JSON payload
	ContentUrl          string                 `json:"contentUrl,omitempty"`     // URL or file path to content
	EncodingFormat      string                 `json:"encodingFormat,omitempty"` // Format of the text content
	ProgrammingLanguage string                 `json:"programmingLanguage,omitempty"`
	CodeRepository      string                 `json:"codeRepository,omitempty"`
	RuntimePlatform     string                 `json:"runtimePlatform,omitempty"`
	Target              interface{}            `json:"target,omitempty"`             // Target for actions (Project, EntryPoint, etc.)
	Properties          map[string]interface{} `json:"additionalProperty,omitempty"` // Additional properties
}

// SemanticInstrument represents tools used for execution
type SemanticInstrument struct {
	Type           string                 `json:"@type"` // SoftwareApplication, MediaObject
	Name           string                 `json:"name,omitempty"`
	ContentUrl     string                 `json:"contentUrl,omitempty"`         // File path or URL to tool/script
	CodeRepository string                 `json:"codeRepository,omitempty"`     // Repository URL
	EncodingFormat string                 `json:"encodingFormat,omitempty"`     // Format of the content
	Properties     map[string]interface{} `json:"additionalProperty,omitempty"` // Additional properties
}

// SemanticResult represents action execution result following Schema.org Result pattern
type SemanticResult struct {
	Type         string        `json:"@type"`                    // "Result", "Dataset", "DigitalDocument", etc.
	ActionStatus string        `json:"actionStatus,omitempty"`   // CompletedActionStatus, FailedActionStatus
	Output       string        `json:"text,omitempty"`           // Raw text output (JSON, XML, plain text)
	Value        interface{}   `json:"value,omitempty"`          // Structured data (any type: int, map, array, etc.)
	Format       string        `json:"encodingFormat,omitempty"` // MIME type: application/json, text/xml, etc.
	Schema       *ResultSchema `json:"about,omitempty"`          // Describes the structure of Value
}

// ResultSchema describes the structure of result data using Schema.org patterns
type ResultSchema struct {
	Type       string              `json:"@type"`                      // "PropertyValueList", "Dataset", "StructuredValue"
	Properties []PropertyValueSpec `json:"variableMeasured,omitempty"` // For Dataset/PropertyValueList
}

// PropertyValueSpec describes a property in the result schema
type PropertyValueSpec struct {
	Type        string `json:"@type"`                 // "PropertyValue"
	Name        string `json:"name"`                  // Property name (e.g., "username", "secretKey")
	ValueType   string `json:"valueType,omitempty"`   // "Text", "Number", "Boolean", "URL"
	Description string `json:"description,omitempty"` // Human-readable description
}

// SemanticError represents action failure information
type SemanticError struct {
	Type    string `json:"@type"` // Error
	Message string `json:"message"`
}

// SemanticSchedule represents when/how often action runs
type SemanticSchedule struct {
	Type            string   `json:"@type"`                     // Schedule
	RepeatFrequency string   `json:"repeatFrequency,omitempty"` // ISO 8601 duration (PT1H = every hour)
	ByDay           []string `json:"byDay,omitempty"`           // Monday, Tuesday, etc.
	ByMonth         []int    `json:"byMonth,omitempty"`
	StartDate       string   `json:"startDate,omitempty"` // ISO 8601 date
	EndDate         string   `json:"endDate,omitempty"`
}

// ToJSONLD exports any semantic type as JSON-LD
func ToJSONLD(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

// FromJSONLD imports any semantic type from JSON-LD
func FromJSONLD(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// ParseSemanticAction parses JSON-LD bytes into a SemanticAction
func ParseSemanticAction(data []byte) (*SemanticAction, error) {
	var action SemanticAction
	if err := json.Unmarshal(data, &action); err != nil {
		return nil, err
	}
	return &action, nil
}
