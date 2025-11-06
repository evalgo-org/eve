package semantic

import (
	"encoding/json"
	"fmt"
)

// S3 Semantic Types for Object Storage Operations
// These types map S3 operations to Schema.org vocabulary for semantic orchestration
// Supports Hetzner S3, AWS S3, and S3-compatible storage

// ============================================================================
// S3 Storage Types (Schema.org: DataCatalog, MediaObject)
// ============================================================================

// S3Bucket represents an S3 bucket as Schema.org DataCatalog
type S3Bucket struct {
	Context    string                 `json:"@context,omitempty"`
	Type       string                 `json:"@type"` // "DataCatalog"
	Identifier string                 `json:"identifier"`
	Name       string                 `json:"name,omitempty"`
	URL        string                 `json:"url,omitempty"` // S3 endpoint URL
	Properties map[string]interface{} `json:"additionalProperty,omitempty"`
}

// S3Object represents an S3 object as Schema.org MediaObject
type S3Object struct {
	Type           string                 `json:"@type"` // "MediaObject" or "DataDownload"
	Identifier     string                 `json:"identifier"`
	Name           string                 `json:"name,omitempty"`
	EncodingFormat string                 `json:"encodingFormat,omitempty"` // MIME type
	ContentUrl     string                 `json:"contentUrl,omitempty"`     // Local file path or S3 key
	ContentSize    int64                  `json:"contentSize,omitempty"`    // File size in bytes
	UploadDate     string                 `json:"uploadDate,omitempty"`     // ISO 8601 timestamp
	Properties     map[string]interface{} `json:"additionalProperty,omitempty"`
}

// ============================================================================
// S3 Action Types
// ============================================================================

// S3UploadAction represents uploading objects to S3
// Maps to Schema.org CreateAction for creating new objects
type S3UploadAction struct {
	Context      string         `json:"@context,omitempty"`
	Type         string         `json:"@type"` // "CreateAction"
	Identifier   string         `json:"identifier"`
	Name         string         `json:"name,omitempty"`
	Description  string         `json:"description,omitempty"`
	Object       *S3Object      `json:"object"`              // Object to upload
	Target       *S3Bucket      `json:"target"`              // Target S3 bucket
	TargetUrl    string         `json:"targetUrl,omitempty"` // Specific S3 key/path
	ActionStatus string         `json:"actionStatus,omitempty"`
	StartTime    string         `json:"startTime,omitempty"`
	EndTime      string         `json:"endTime,omitempty"`
	Result       *S3Object      `json:"result,omitempty"` // Uploaded object metadata
	Error        *PropertyValue `json:"error,omitempty"`
}

// S3DownloadAction represents downloading objects from S3
// Maps to Schema.org DownloadAction for retrieving objects
type S3DownloadAction struct {
	Context      string         `json:"@context,omitempty"`
	Type         string         `json:"@type"` // "DownloadAction"
	Identifier   string         `json:"identifier"`
	Name         string         `json:"name,omitempty"`
	Description  string         `json:"description,omitempty"`
	Object       *S3Object      `json:"object"`           // Object to download (with S3 key)
	Target       *S3Bucket      `json:"target"`           // Source S3 bucket
	Result       *S3Object      `json:"result,omitempty"` // Downloaded object with local path
	ActionStatus string         `json:"actionStatus,omitempty"`
	StartTime    string         `json:"startTime,omitempty"`
	EndTime      string         `json:"endTime,omitempty"`
	Error        *PropertyValue `json:"error,omitempty"`
}

// S3DeleteAction represents deleting objects from S3
// Maps to Schema.org DeleteAction for removing objects
type S3DeleteAction struct {
	Context      string         `json:"@context,omitempty"`
	Type         string         `json:"@type"` // "DeleteAction"
	Identifier   string         `json:"identifier"`
	Name         string         `json:"name,omitempty"`
	Description  string         `json:"description,omitempty"`
	Object       *S3Object      `json:"object"` // Object to delete (with S3 key)
	Target       *S3Bucket      `json:"target"` // S3 bucket containing object
	ActionStatus string         `json:"actionStatus,omitempty"`
	StartTime    string         `json:"startTime,omitempty"`
	EndTime      string         `json:"endTime,omitempty"`
	Error        *PropertyValue `json:"error,omitempty"`
}

// S3ListAction represents listing objects in S3 bucket
// Maps to Schema.org SearchAction for discovering objects
type S3ListAction struct {
	Context      string         `json:"@context,omitempty"`
	Type         string         `json:"@type"` // "SearchAction"
	Identifier   string         `json:"identifier"`
	Name         string         `json:"name,omitempty"`
	Description  string         `json:"description,omitempty"`
	Query        string         `json:"query,omitempty"`  // Prefix filter
	Target       *S3Bucket      `json:"target"`           // S3 bucket to list
	Result       []*S3Object    `json:"result,omitempty"` // List of objects
	ActionStatus string         `json:"actionStatus,omitempty"`
	StartTime    string         `json:"startTime,omitempty"`
	EndTime      string         `json:"endTime,omitempty"`
	Error        *PropertyValue `json:"error,omitempty"`
}

// ============================================================================
// Helper Functions
// ============================================================================

// ParseS3Action parses a JSON-LD S3 action
func ParseS3Action(data []byte) (interface{}, error) {
	var typeCheck struct {
		Type string `json:"@type"`
	}

	if err := json.Unmarshal(data, &typeCheck); err != nil {
		return nil, fmt.Errorf("failed to parse @type: %w", err)
	}

	switch typeCheck.Type {
	case "CreateAction":
		var action S3UploadAction
		if err := json.Unmarshal(data, &action); err != nil {
			return nil, fmt.Errorf("failed to parse S3UploadAction: %w", err)
		}
		return &action, nil

	case "DownloadAction":
		var action S3DownloadAction
		if err := json.Unmarshal(data, &action); err != nil {
			return nil, fmt.Errorf("failed to parse S3DownloadAction: %w", err)
		}
		return &action, nil

	case "DeleteAction":
		var action S3DeleteAction
		if err := json.Unmarshal(data, &action); err != nil {
			return nil, fmt.Errorf("failed to parse S3DeleteAction: %w", err)
		}
		return &action, nil

	case "SearchAction":
		var action S3ListAction
		if err := json.Unmarshal(data, &action); err != nil {
			return nil, fmt.Errorf("failed to parse S3ListAction: %w", err)
		}
		return &action, nil

	default:
		return nil, fmt.Errorf("unsupported S3 action type: %s", typeCheck.Type)
	}
}

// NewS3Bucket creates a new semantic S3 bucket representation
func NewS3Bucket(name, url, region, accessKey, secretKey string) *S3Bucket {
	bucket := &S3Bucket{
		Context:    "https://schema.org",
		Type:       "DataCatalog",
		Identifier: name,
		Name:       name,
		URL:        url,
		Properties: make(map[string]interface{}),
	}

	if region != "" {
		bucket.Properties["region"] = region
	}
	if accessKey != "" {
		bucket.Properties["accessKey"] = accessKey
	}
	if secretKey != "" {
		bucket.Properties["secretKey"] = secretKey
	}

	return bucket
}

// NewS3Object creates a new S3 object representation
func NewS3Object(key, contentType string) *S3Object {
	return &S3Object{
		Type:           "MediaObject",
		Identifier:     key,
		Name:           key,
		EncodingFormat: contentType,
	}
}

// NewS3UploadAction creates a new S3 upload action
func NewS3UploadAction(id, name string, object *S3Object, target *S3Bucket) *S3UploadAction {
	return &S3UploadAction{
		Context:      "https://schema.org",
		Type:         "CreateAction",
		Identifier:   id,
		Name:         name,
		Object:       object,
		Target:       target,
		ActionStatus: "PotentialActionStatus",
	}
}

// NewSemanticS3UploadAction creates an S3 upload action using SemanticAction
func NewSemanticS3UploadAction(id, name string, object *S3Object, target *S3Bucket, targetUrl string) *SemanticAction {
	action := &SemanticAction{
		Context:      "https://schema.org",
		Type:         "CreateAction",
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

// NewS3DownloadAction creates a new S3 download action
func NewS3DownloadAction(id, name string, object *S3Object, target *S3Bucket) *S3DownloadAction {
	return &S3DownloadAction{
		Context:      "https://schema.org",
		Type:         "DownloadAction",
		Identifier:   id,
		Name:         name,
		Object:       object,
		Target:       target,
		ActionStatus: "PotentialActionStatus",
	}
}

// NewSemanticS3DownloadAction creates an S3 download action using SemanticAction
func NewSemanticS3DownloadAction(id, name string, object *S3Object, target *S3Bucket) *SemanticAction {
	action := &SemanticAction{
		Context:      "https://schema.org",
		Type:         "DownloadAction",
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

// NewS3DeleteAction creates a new S3 delete action
func NewS3DeleteAction(id, name string, object *S3Object, target *S3Bucket) *S3DeleteAction {
	return &S3DeleteAction{
		Context:      "https://schema.org",
		Type:         "DeleteAction",
		Identifier:   id,
		Name:         name,
		Object:       object,
		Target:       target,
		ActionStatus: "PotentialActionStatus",
	}
}

// NewSemanticS3DeleteAction creates an S3 delete action using SemanticAction
func NewSemanticS3DeleteAction(id, name string, object *S3Object, target *S3Bucket) *SemanticAction {
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
	if target != nil {
		action.Properties["target"] = target
	}

	return action
}

// NewS3ListAction creates a new S3 list action
func NewS3ListAction(id, name, prefix string, target *S3Bucket) *S3ListAction {
	return &S3ListAction{
		Context:      "https://schema.org",
		Type:         "SearchAction",
		Identifier:   id,
		Name:         name,
		Query:        prefix,
		Target:       target,
		ActionStatus: "PotentialActionStatus",
	}
}

// NewSemanticS3ListAction creates an S3 list action using SemanticAction
func NewSemanticS3ListAction(id, name, prefix string, target *S3Bucket) *SemanticAction {
	action := &SemanticAction{
		Context:      "https://schema.org",
		Type:         "SearchAction",
		Identifier:   id,
		Name:         name,
		ActionStatus: "PotentialActionStatus",
		Properties:   make(map[string]interface{}),
	}

	if prefix != "" {
		action.Properties["query"] = prefix
	}
	if target != nil {
		action.Properties["target"] = target
	}

	return action
}

// ExtractS3Credentials extracts connection info from S3Bucket
func ExtractS3Credentials(bucket *S3Bucket) (url, region, accessKey, secretKey, bucketName string, err error) {
	if bucket == nil {
		return "", "", "", "", "", fmt.Errorf("bucket is nil")
	}

	url = bucket.URL
	bucketName = bucket.Identifier

	if url == "" {
		return "", "", "", "", "", fmt.Errorf("bucket URL is empty")
	}

	if bucket.Properties != nil {
		if r, ok := bucket.Properties["region"].(string); ok {
			region = r
		}
		if a, ok := bucket.Properties["accessKey"].(string); ok {
			accessKey = a
		}
		if s, ok := bucket.Properties["secretKey"].(string); ok {
			secretKey = s
		}
	}

	return url, region, accessKey, secretKey, bucketName, nil
}

// ============================================================================
// SemanticAction Helper Functions for S3 Operations
// ============================================================================

// GetS3ObjectFromAction extracts S3Object from SemanticAction properties
func GetS3ObjectFromAction(action *SemanticAction) (*S3Object, error) {
	if action == nil || action.Properties == nil {
		return nil, fmt.Errorf("action or properties is nil")
	}

	obj, ok := action.Properties["object"]
	if !ok {
		return nil, fmt.Errorf("no object found in action properties")
	}

	switch v := obj.(type) {
	case *S3Object:
		return v, nil
	case S3Object:
		return &v, nil
	case map[string]interface{}:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal S3Object: %w", err)
		}
		var s3obj S3Object
		if err := json.Unmarshal(data, &s3obj); err != nil {
			return nil, fmt.Errorf("failed to unmarshal S3Object: %w", err)
		}
		return &s3obj, nil
	default:
		return nil, fmt.Errorf("unexpected object type: %T", obj)
	}
}

// GetS3BucketFromAction extracts S3Bucket from SemanticAction properties
func GetS3BucketFromAction(action *SemanticAction) (*S3Bucket, error) {
	if action == nil || action.Properties == nil {
		return nil, fmt.Errorf("action or properties is nil")
	}

	target, ok := action.Properties["target"]
	if !ok {
		return nil, fmt.Errorf("no target found in action properties")
	}

	switch v := target.(type) {
	case *S3Bucket:
		return v, nil
	case S3Bucket:
		return &v, nil
	case map[string]interface{}:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal S3Bucket: %w", err)
		}
		var bucket S3Bucket
		if err := json.Unmarshal(data, &bucket); err != nil {
			return nil, fmt.Errorf("failed to unmarshal S3Bucket: %w", err)
		}
		return &bucket, nil
	default:
		return nil, fmt.Errorf("unexpected target type: %T", target)
	}
}

// GetS3TargetUrlFromAction extracts targetUrl from SemanticAction properties
func GetS3TargetUrlFromAction(action *SemanticAction) string {
	if action == nil || action.Properties == nil {
		return ""
	}

	if url, ok := action.Properties["targetUrl"].(string); ok {
		return url
	}

	return ""
}

// GetS3QueryFromAction extracts query/prefix from SemanticAction properties
func GetS3QueryFromAction(action *SemanticAction) string {
	if action == nil || action.Properties == nil {
		return ""
	}

	if query, ok := action.Properties["query"].(string); ok {
		return query
	}

	return ""
}
