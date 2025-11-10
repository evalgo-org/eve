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
// Helper Functions
// ============================================================================

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

// GetS3BucketFromAction extracts S3Bucket from SemanticAction properties or direct field
// Handles both SemanticAction (target in Properties) and SemanticScheduledAction (target as direct field)
func GetS3BucketFromAction(action *SemanticAction) (*S3Bucket, error) {
	if action == nil {
		return nil, fmt.Errorf("action is nil")
	}

	var target interface{}

	// First try to get from Properties (SemanticAction pattern)
	if action.Properties != nil {
		if t, ok := action.Properties["target"]; ok {
			target = t
		}
	}

	// If not in Properties, try to extract from action JSON (SemanticScheduledAction pattern)
	if target == nil {
		// Re-marshal the action and check for direct target field
		actionJSON, err := json.Marshal(action)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal action: %w", err)
		}

		var rawAction map[string]interface{}
		if err := json.Unmarshal(actionJSON, &rawAction); err != nil {
			return nil, fmt.Errorf("failed to unmarshal action: %w", err)
		}

		if t, ok := rawAction["target"]; ok {
			target = t
		}
	}

	// If still no target found, return error
	if target == nil {
		return nil, fmt.Errorf("no target found in action properties or direct fields")
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

// GetS3TargetUrlFromAction extracts targetUrl from SemanticAction properties or direct field
// Handles both SemanticAction (targetUrl in Properties) and SemanticScheduledAction (targetUrl as direct field)
func GetS3TargetUrlFromAction(action *SemanticAction) string {
	if action == nil {
		return ""
	}

	// First try to get from Properties (SemanticAction pattern)
	if action.Properties != nil {
		if url, ok := action.Properties["targetUrl"].(string); ok {
			return url
		}
	}

	// If not in Properties, try to extract from action JSON (SemanticScheduledAction pattern)
	actionJSON, err := json.Marshal(action)
	if err != nil {
		return ""
	}

	var rawAction map[string]interface{}
	if err := json.Unmarshal(actionJSON, &rawAction); err != nil {
		return ""
	}

	if url, ok := rawAction["targetUrl"].(string); ok {
		return url
	}

	return ""
}

// GetS3QueryFromAction extracts query/prefix from SemanticAction properties or direct field
// Handles both SemanticAction (query in Properties) and SemanticScheduledAction (query as direct field)
func GetS3QueryFromAction(action *SemanticAction) string {
	if action == nil {
		return ""
	}

	// First try to get from Properties (SemanticAction pattern)
	if action.Properties != nil {
		if query, ok := action.Properties["query"].(string); ok {
			return query
		}
	}

	// If not in Properties, try to extract from action JSON (SemanticScheduledAction pattern)
	actionJSON, err := json.Marshal(action)
	if err != nil {
		return ""
	}

	var rawAction map[string]interface{}
	if err := json.Unmarshal(actionJSON, &rawAction); err != nil {
		return ""
	}

	if query, ok := rawAction["query"].(string); ok {
		return query
	}

	return ""
}
