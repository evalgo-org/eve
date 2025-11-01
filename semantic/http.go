package semantic

import (
	"encoding/json"
	"time"
)

// SemanticRequest represents an HTTP Request as a Schema.org DigitalDocument
// This enables semantic understanding and interoperability
type SemanticRequest struct {
	Context     string                 `json:"@context"`
	Type        string                 `json:"@type"`
	Identifier  string                 `json:"identifier,omitempty"`
	Name        string                 `json:"name,omitempty"`
	URL         string                 `json:"url"`
	Method      string                 `json:"httpMethod"`
	Headers     map[string]string      `json:"httpHeaders,omitempty"`
	Body        interface{}            `json:"text,omitempty"`
	Timeout     int                    `json:"temporal,omitempty"`
	CreatedTime string                 `json:"dateCreated,omitempty"`
	Additional  map[string]interface{} `json:"additionalProperty,omitempty"`
}

// NewSemanticRequest creates a new semantic HTTP request
func NewSemanticRequest(method, url string) *SemanticRequest {
	return &SemanticRequest{
		Context:     "https://schema.org",
		Type:        "DigitalDocument",
		URL:         url,
		Method:      method,
		Headers:     make(map[string]string),
		CreatedTime: time.Now().Format(time.RFC3339),
		Additional: map[string]interface{}{
			"@type": "PropertyValue",
		},
	}
}

// SemanticResponse represents an HTTP Response as a Schema.org DigitalDocument
type SemanticResponse struct {
	Context     string            `json:"@context"`
	Type        string            `json:"@type"`
	StatusCode  int               `json:"httpStatusCode"`
	Status      string            `json:"httpStatus"`
	Headers     map[string]string `json:"httpHeaders,omitempty"`
	Body        string            `json:"text,omitempty"`
	FromCache   bool              `json:"fromCache,omitempty"`
	Duration    string            `json:"duration,omitempty"` // ISO 8601 duration
	CreatedTime string            `json:"dateCreated"`
}

// NewSemanticResponse creates a new semantic HTTP response
func NewSemanticResponse(statusCode int, status string) *SemanticResponse {
	return &SemanticResponse{
		Context:     "https://schema.org",
		Type:        "DigitalDocument",
		StatusCode:  statusCode,
		Status:      status,
		Headers:     make(map[string]string),
		CreatedTime: time.Now().Format(time.RFC3339),
	}
}

// ToJSONLD exports the request as JSON-LD
func (sr *SemanticRequest) ToJSONLD() (string, error) {
	data, err := json.MarshalIndent(sr, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ToJSONLD exports the response as JSON-LD
func (sr *SemanticResponse) ToJSONLD() (string, error) {
	data, err := json.MarshalIndent(sr, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
