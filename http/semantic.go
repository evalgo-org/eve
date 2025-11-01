package http

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

// ToSemanticRequest converts a Request to Schema.org DigitalDocument representation
func (r *Request) ToSemanticRequest() *SemanticRequest {
	sr := &SemanticRequest{
		Context: "https://schema.org",
		Type:    "DigitalDocument",
		URL:     r.URL,
		Method:  r.Method,
		Headers: r.Headers,
		Timeout: r.Timeout,
		Additional: map[string]interface{}{
			"@type": "PropertyValue",
		},
	}

	// Set created time
	sr.CreatedTime = time.Now().Format(time.RFC3339)

	// Handle body based on type
	if r.JSONBody != "" {
		// Parse JSON body to structured data
		var jsonData interface{}
		if err := json.Unmarshal([]byte(r.JSONBody), &jsonData); err == nil {
			sr.Body = jsonData
		} else {
			sr.Body = r.JSONBody
		}
	} else if len(r.FormData) > 0 || len(r.Files) > 0 {
		// Represent form data as structured property values
		sr.Body = map[string]interface{}{
			"@type":    "PropertyValue",
			"formData": r.FormData,
			"files":    r.Files,
		}
	}

	// Add save-to as additional property
	if r.SaveTo != "" {
		sr.Additional["saveTo"] = r.SaveTo
	}

	// Add retry configuration
	if r.RetryCount > 0 {
		sr.Additional["retryCount"] = r.RetryCount
		sr.Additional["retryBackoff"] = r.RetryBackoff
	}

	return sr
}

// ToJSONLD exports the request as JSON-LD for interoperability
func (r *Request) ToJSONLD() (string, error) {
	sr := r.ToSemanticRequest()
	data, err := json.MarshalIndent(sr, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
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

// ToSemanticResponse converts a Response to Schema.org representation
func (r *Response) ToSemanticResponse() *SemanticResponse {
	return &SemanticResponse{
		Context:     "https://schema.org",
		Type:        "DigitalDocument",
		StatusCode:  r.StatusCode,
		Status:      r.Status,
		Headers:     r.Headers,
		Body:        r.BodyString,
		FromCache:   r.FromCache,
		Duration:    r.Duration.String(),
		CreatedTime: time.Now().Format(time.RFC3339),
	}
}

// ToJSONLD exports the response as JSON-LD
func (r *Response) ToJSONLD() (string, error) {
	sr := r.ToSemanticResponse()
	data, err := json.MarshalIndent(sr, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromSemanticRequest creates a Request from Schema.org representation
func FromSemanticRequest(sr *SemanticRequest) *Request {
	req := NewRequest(sr.Method, sr.URL)
	req.Headers = sr.Headers
	req.Timeout = sr.Timeout

	// Extract save-to from additional properties
	if saveTo, ok := sr.Additional["saveTo"].(string); ok {
		req.SaveTo = saveTo
	}

	// Extract retry configuration
	if retryCount, ok := sr.Additional["retryCount"].(float64); ok {
		req.RetryCount = int(retryCount)
	}
	if retryBackoff, ok := sr.Additional["retryBackoff"].(string); ok {
		req.RetryBackoff = retryBackoff
	}

	// Handle body
	if sr.Body != nil {
		switch body := sr.Body.(type) {
		case string:
			req.JSONBody = body
		case map[string]interface{}:
			// Check if it's form data
			if formData, ok := body["formData"].(map[string]string); ok {
				req.FormData = formData
			}
			if files, ok := body["files"].(map[string]string); ok {
				req.Files = files
			}
			// Otherwise treat as JSON
			if _, hasForm := body["formData"]; !hasForm {
				jsonData, _ := json.Marshal(body)
				req.JSONBody = string(jsonData)
			}
		default:
			// Convert to JSON
			jsonData, _ := json.Marshal(body)
			req.JSONBody = string(jsonData)
		}
	}

	return req
}
