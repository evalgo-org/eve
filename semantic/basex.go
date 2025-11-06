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

// ============================================================================
// SemanticAction Constructors for BaseX Operations
// ============================================================================

// NewSemanticTransformAction creates a TransformAction using SemanticAction
func NewSemanticTransformAction(id, name string, object *XMLDocument, instrument *XSLTStylesheet, target *XMLDatabase) *SemanticAction {
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
	if instrument != nil {
		action.Properties["instrument"] = instrument
	}
	if target != nil {
		action.Properties["target"] = target
	}

	return action
}

// NewSemanticQueryAction creates a QueryAction using SemanticAction
func NewSemanticQueryAction(id, name, query string, target *XMLDatabase) *SemanticAction {
	action := &SemanticAction{
		Context:      "https://schema.org",
		Type:         "SearchAction",
		Identifier:   id,
		Name:         name,
		ActionStatus: "PotentialActionStatus",
		Properties:   make(map[string]interface{}),
	}

	if query != "" {
		action.Properties["query"] = query
	}
	if target != nil {
		action.Properties["target"] = target
	}

	return action
}

// NewSemanticBaseXUploadAction creates a BaseXUploadAction using SemanticAction
func NewSemanticBaseXUploadAction(id, name string, object *XMLDocument, target *XMLDatabase, targetUrl string) *SemanticAction {
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
	if targetUrl != "" {
		action.Properties["targetUrl"] = targetUrl
	}

	return action
}

// ============================================================================
// SemanticAction Helper Functions for BaseX Operations
// ============================================================================

// GetXMLDocumentFromAction extracts XMLDocument from SemanticAction properties
func GetXMLDocumentFromAction(action *SemanticAction) (*XMLDocument, error) {
	if action == nil || action.Properties == nil {
		return nil, fmt.Errorf("action or properties is nil")
	}

	obj, ok := action.Properties["object"]
	if !ok {
		return nil, fmt.Errorf("no object found in action properties")
	}

	switch v := obj.(type) {
	case *XMLDocument:
		return v, nil
	case XMLDocument:
		return &v, nil
	case map[string]interface{}:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal XMLDocument: %w", err)
		}
		var doc XMLDocument
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("failed to unmarshal XMLDocument: %w", err)
		}
		return &doc, nil
	default:
		return nil, fmt.Errorf("unexpected object type: %T", obj)
	}
}

// GetXSLTStylesheetFromAction extracts XSLTStylesheet from SemanticAction properties
func GetXSLTStylesheetFromAction(action *SemanticAction) (*XSLTStylesheet, error) {
	if action == nil || action.Properties == nil {
		return nil, fmt.Errorf("action or properties is nil")
	}

	instr, ok := action.Properties["instrument"]
	if !ok {
		return nil, fmt.Errorf("no instrument found in action properties")
	}

	switch v := instr.(type) {
	case *XSLTStylesheet:
		return v, nil
	case XSLTStylesheet:
		return &v, nil
	case map[string]interface{}:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal XSLTStylesheet: %w", err)
		}
		var stylesheet XSLTStylesheet
		if err := json.Unmarshal(data, &stylesheet); err != nil {
			return nil, fmt.Errorf("failed to unmarshal XSLTStylesheet: %w", err)
		}
		return &stylesheet, nil
	default:
		return nil, fmt.Errorf("unexpected instrument type: %T", instr)
	}
}

// GetXMLDatabaseFromAction extracts XMLDatabase from SemanticAction properties
func GetXMLDatabaseFromAction(action *SemanticAction) (*XMLDatabase, error) {
	if action == nil || action.Properties == nil {
		return nil, fmt.Errorf("action or properties is nil")
	}

	target, ok := action.Properties["target"]
	if !ok {
		return nil, fmt.Errorf("no target found in action properties")
	}

	switch v := target.(type) {
	case *XMLDatabase:
		return v, nil
	case XMLDatabase:
		return &v, nil
	case map[string]interface{}:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal XMLDatabase: %w", err)
		}
		var db XMLDatabase
		if err := json.Unmarshal(data, &db); err != nil {
			return nil, fmt.Errorf("failed to unmarshal XMLDatabase: %w", err)
		}
		return &db, nil
	default:
		return nil, fmt.Errorf("unexpected target type: %T", target)
	}
}

// GetQueryFromAction extracts query string from SemanticAction properties
func GetQueryFromAction(action *SemanticAction) string {
	if action == nil || action.Properties == nil {
		return ""
	}

	if query, ok := action.Properties["query"].(string); ok {
		return query
	}

	return ""
}

// GetTargetUrlFromAction extracts targetUrl from SemanticAction properties
func GetTargetUrlFromAction(action *SemanticAction) string {
	if action == nil || action.Properties == nil {
		return ""
	}

	if url, ok := action.Properties["targetUrl"].(string); ok {
		return url
	}

	return ""
}
