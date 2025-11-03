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
	Identifier   string                 `json:"identifier,omitempty"`
	Name         string                 `json:"name,omitempty"`
	Description  string                 `json:"description,omitempty"`
	ActionStatus string                 `json:"actionStatus"` // PotentialActionStatus, ActiveActionStatus, CompletedActionStatus, FailedActionStatus
	Agent        *SemanticAgent         `json:"agent,omitempty"`
	Object       *SemanticObject        `json:"object,omitempty"`
	Instrument   *SemanticInstrument    `json:"instrument,omitempty"`
	StartTime    *time.Time             `json:"startTime,omitempty"`
	EndTime      *time.Time             `json:"endTime,omitempty"`
	Duration     string                 `json:"duration,omitempty"` // ISO 8601 duration
	Result       *SemanticResult        `json:"result,omitempty"`
	Error        *SemanticError         `json:"error,omitempty"`
	Properties   map[string]interface{} `json:"additionalProperty,omitempty"`
}

// SemanticScheduledAction represents a task that runs on a schedule
type SemanticScheduledAction struct {
	SemanticAction
	Requires  []string          `json:"requires,omitempty"`  // Dependencies (Action @id references)
	Target    interface{}       `json:"target,omitempty"`    // Can be EntryPoint, InfisicalProject, DataCatalog, etc.
	TargetUrl string            `json:"targetUrl,omitempty"` // Specific target path/key (e.g., S3 object key)
	Query     interface{}       `json:"query,omitempty"`     // For SearchAction (SPARQL query, search parameters)
	Schedule  *SemanticSchedule `json:"schedule,omitempty"`
	Created   time.Time         `json:"dateCreated"`
	Modified  time.Time         `json:"dateModified"`
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
	Properties          map[string]interface{} `json:"additionalProperty,omitempty"` // Additional properties
}

// SemanticInstrument represents tools used for execution
type SemanticInstrument struct {
	Type           string `json:"@type"` // SoftwareApplication, MediaObject
	Name           string `json:"name,omitempty"`
	ContentUrl     string `json:"contentUrl,omitempty"`     // File path or URL to tool/script
	CodeRepository string `json:"codeRepository,omitempty"` // Repository URL
	EncodingFormat string `json:"encodingFormat,omitempty"` // Format of the content
}

// SemanticResult represents action execution result
type SemanticResult struct {
	Type         string `json:"@type"` // Action, PropertyValue
	ActionStatus string `json:"actionStatus,omitempty"`
	Output       string `json:"text,omitempty"`
	Value        int    `json:"value,omitempty"` // Exit code, status code, etc.
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
