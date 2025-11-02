package semantic

import (
	"encoding/json"
	"fmt"
)

// BaseX Semantic Types for XML Database Operations
// These types map BaseX operations to Schema.org vocabulary for semantic orchestration

// ============================================================================
// BaseX Database Types (Schema.org: DataCatalog, Dataset)
// ============================================================================

// XMLDatabase represents a BaseX database as Schema.org DataCatalog
type XMLDatabase struct {
	Context    string                 `json:"@context,omitempty"`
	Type       string                 `json:"@type"` // "DataCatalog"
	Identifier string                 `json:"identifier"`
	Name       string                 `json:"name,omitempty"`
	URL        string                 `json:"url,omitempty"` // BaseX REST API URL
	Properties map[string]interface{} `json:"additionalProperty,omitempty"`
}

// XMLDocument represents an XML document as Schema.org Dataset
type XMLDocument struct {
	Type           string                 `json:"@type"` // "Dataset"
	Identifier     string                 `json:"identifier"`
	Name           string                 `json:"name,omitempty"`
	EncodingFormat string                 `json:"encodingFormat,omitempty"` // "application/xml", "text/xsl"
	ContentUrl     string                 `json:"contentUrl,omitempty"`     // Path in BaseX or file URL
	Properties     map[string]interface{} `json:"additionalProperty,omitempty"`
}

// XSLTStylesheet represents an XSLT transformation stylesheet
type XSLTStylesheet struct {
	Type                string                 `json:"@type"` // "SoftwareSourceCode"
	Identifier          string                 `json:"identifier"`
	Name                string                 `json:"name,omitempty"`
	CodeRepository      string                 `json:"codeRepository,omitempty"` // Path to XSLT file
	ContentUrl          string                 `json:"contentUrl,omitempty"`
	ProgrammingLanguage string                 `json:"programmingLanguage,omitempty"` // "XSLT"
	Properties          map[string]interface{} `json:"additionalProperty,omitempty"`
}

// ============================================================================
// BaseX Action Types
// ============================================================================

// TransformAction represents XSLT transformation operations
// Maps to Schema.org UpdateAction for transforming data
type TransformAction struct {
	Context      string          `json:"@context,omitempty"`
	Type         string          `json:"@type"` // "UpdateAction"
	Identifier   string          `json:"identifier"`
	Name         string          `json:"name,omitempty"`
	Description  string          `json:"description,omitempty"`
	Object       *XMLDocument    `json:"object"`           // Source XML document
	Instrument   *XSLTStylesheet `json:"instrument"`       // XSLT stylesheet
	Result       *XMLDocument    `json:"result,omitempty"` // Transformed output
	Target       *XMLDatabase    `json:"target,omitempty"` // Target BaseX database
	ActionStatus string          `json:"actionStatus,omitempty"`
	StartTime    string          `json:"startTime,omitempty"`
	EndTime      string          `json:"endTime,omitempty"`
	Error        *PropertyValue  `json:"error,omitempty"`
}

// QueryAction represents XQuery execution operations
// Maps to Schema.org SearchAction for querying XML databases
type QueryAction struct {
	Context      string         `json:"@context,omitempty"`
	Type         string         `json:"@type"` // "SearchAction"
	Identifier   string         `json:"identifier"`
	Name         string         `json:"name,omitempty"`
	Description  string         `json:"description,omitempty"`
	Query        string         `json:"query"`            // XQuery or XPath expression
	Target       *XMLDatabase   `json:"target"`           // BaseX database to query
	Result       *XMLDocument   `json:"result,omitempty"` // Query result
	ActionStatus string         `json:"actionStatus,omitempty"`
	StartTime    string         `json:"startTime,omitempty"`
	EndTime      string         `json:"endTime,omitempty"`
	Error        *PropertyValue `json:"error,omitempty"`
}

// BaseXUploadAction represents file upload to BaseX operations
type BaseXUploadAction struct {
	Context      string         `json:"@context,omitempty"`
	Type         string         `json:"@type"` // "UploadAction"
	Identifier   string         `json:"identifier"`
	Name         string         `json:"name,omitempty"`
	Description  string         `json:"description,omitempty"`
	Object       *XMLDocument   `json:"object"`              // Document to upload
	Target       *XMLDatabase   `json:"target"`              // Target BaseX database
	TargetUrl    string         `json:"targetUrl,omitempty"` // Specific path in database
	ActionStatus string         `json:"actionStatus,omitempty"`
	StartTime    string         `json:"startTime,omitempty"`
	EndTime      string         `json:"endTime,omitempty"`
	Error        *PropertyValue `json:"error,omitempty"`
}

// CreateAction represents creating a new BaseX database
type CreateDatabaseAction struct {
	Context      string         `json:"@context,omitempty"`
	Type         string         `json:"@type"` // "CreateAction"
	Identifier   string         `json:"identifier"`
	Name         string         `json:"name,omitempty"`
	Description  string         `json:"description,omitempty"`
	Result       *XMLDatabase   `json:"result"` // Database to create
	ActionStatus string         `json:"actionStatus,omitempty"`
	StartTime    string         `json:"startTime,omitempty"`
	EndTime      string         `json:"endTime,omitempty"`
	Error        *PropertyValue `json:"error,omitempty"`
}

// DeleteAction represents deleting a BaseX database or document
type DeleteDatabaseAction struct {
	Context      string         `json:"@context,omitempty"`
	Type         string         `json:"@type"` // "DeleteAction"
	Identifier   string         `json:"identifier"`
	Name         string         `json:"name,omitempty"`
	Description  string         `json:"description,omitempty"`
	Object       interface{}    `json:"object"` // *XMLDatabase or *XMLDocument
	ActionStatus string         `json:"actionStatus,omitempty"`
	StartTime    string         `json:"startTime,omitempty"`
	EndTime      string         `json:"endTime,omitempty"`
	Error        *PropertyValue `json:"error,omitempty"`
}

// ============================================================================
// Composite Workflow Types
// ============================================================================

// SPARQLTransformWorkflow represents a composite workflow:
// 1. Query SPARQL endpoint
// 2. Transform result with XSLT
// 3. Store in cache or BaseX
type SPARQLTransformWorkflow struct {
	Context      string          `json:"@context,omitempty"`
	Type         string          `json:"@type"` // "Action"
	Identifier   string          `json:"identifier"`
	Name         string          `json:"name,omitempty"`
	Description  string          `json:"description,omitempty"`
	Query        string          `json:"query"`            // SPARQL query
	QueryTarget  *XMLDatabase    `json:"queryTarget"`      // SPARQL endpoint
	Instrument   *XSLTStylesheet `json:"instrument"`       // XSLT transformation
	Result       *XMLDocument    `json:"result,omitempty"` // Final output
	ActionStatus string          `json:"actionStatus,omitempty"`
	Error        *PropertyValue  `json:"error,omitempty"`
}

// ============================================================================
// Helper Functions
// ============================================================================

// ParseBaseXAction parses a JSON-LD BaseX action
func ParseBaseXAction(data []byte) (interface{}, error) {
	var typeCheck struct {
		Type string `json:"@type"`
	}

	if err := json.Unmarshal(data, &typeCheck); err != nil {
		return nil, fmt.Errorf("failed to parse @type: %w", err)
	}

	switch typeCheck.Type {
	case "UpdateAction", "TransformAction": // TransformAction (UpdateAction is Schema.org standard, TransformAction is semantic extension)
		var action TransformAction
		if err := json.Unmarshal(data, &action); err != nil {
			return nil, fmt.Errorf("failed to parse TransformAction: %w", err)
		}
		return &action, nil

	case "SearchAction": // QueryAction
		var action QueryAction
		if err := json.Unmarshal(data, &action); err != nil {
			return nil, fmt.Errorf("failed to parse QueryAction: %w", err)
		}
		return &action, nil

	case "UploadAction":
		var action BaseXUploadAction
		if err := json.Unmarshal(data, &action); err != nil {
			return nil, fmt.Errorf("failed to parse BaseXUploadAction: %w", err)
		}
		return &action, nil

	case "CreateAction": // CreateDatabaseAction
		var action CreateDatabaseAction
		if err := json.Unmarshal(data, &action); err != nil {
			return nil, fmt.Errorf("failed to parse CreateDatabaseAction: %w", err)
		}
		return &action, nil

	case "DeleteAction": // DeleteDatabaseAction
		var action DeleteDatabaseAction
		if err := json.Unmarshal(data, &action); err != nil {
			return nil, fmt.Errorf("failed to parse DeleteDatabaseAction: %w", err)
		}
		return &action, nil

	default:
		return nil, fmt.Errorf("unsupported BaseX action type: %s", typeCheck.Type)
	}
}

// NewXMLDatabase creates a new semantic XML database representation
func NewXMLDatabase(name, baseURL string) *XMLDatabase {
	return &XMLDatabase{
		Context:    "https://schema.org",
		Type:       "DataCatalog",
		Identifier: name,
		Name:       name,
		URL:        baseURL,
		Properties: make(map[string]interface{}),
	}
}

// NewTransformAction creates a new XSLT transformation action
func NewTransformAction(id, name string, source *XMLDocument, stylesheet *XSLTStylesheet, target *XMLDatabase) *TransformAction {
	return &TransformAction{
		Context:      "https://schema.org",
		Type:         "UpdateAction",
		Identifier:   id,
		Name:         name,
		Object:       source,
		Instrument:   stylesheet,
		Target:       target,
		ActionStatus: "PotentialActionStatus",
	}
}

// NewQueryAction creates a new XQuery action
func NewQueryAction(id, name, query string, target *XMLDatabase) *QueryAction {
	return &QueryAction{
		Context:      "https://schema.org",
		Type:         "SearchAction",
		Identifier:   id,
		Name:         name,
		Query:        query,
		Target:       target,
		ActionStatus: "PotentialActionStatus",
	}
}

// NewBaseXUploadAction creates a new file upload action
func NewBaseXUploadAction(id, name string, document *XMLDocument, target *XMLDatabase) *BaseXUploadAction {
	return &BaseXUploadAction{
		Context:      "https://schema.org",
		Type:         "UploadAction",
		Identifier:   id,
		Name:         name,
		Object:       document,
		Target:       target,
		ActionStatus: "PotentialActionStatus",
	}
}

// ExtractDatabaseCredentials extracts connection info from XMLDatabase
func ExtractDatabaseCredentials(db *XMLDatabase) (baseURL, username, password string, err error) {
	if db == nil {
		return "", "", "", fmt.Errorf("database is nil")
	}

	baseURL = db.URL
	if baseURL == "" {
		return "", "", "", fmt.Errorf("database URL is empty")
	}

	if db.Properties != nil {
		if u, ok := db.Properties["username"].(string); ok {
			username = u
		}
		if p, ok := db.Properties["password"].(string); ok {
			password = p
		}
	}

	return baseURL, username, password, nil
}
