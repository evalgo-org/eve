package http

import (
	"encoding/json"
	"time"

	"eve.evalgo.org/semantic"
)

// ToSemanticRequest converts a Request to Schema.org DigitalDocument representation
func (r *Request) ToSemanticRequest() *semantic.SemanticRequest {
	sr := semantic.NewSemanticRequest(r.Method, r.URL)
	sr.Headers = r.Headers
	sr.Timeout = r.Timeout

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
	return sr.ToJSONLD()
}

// ToSemanticResponse converts a Response to Schema.org representation
func (r *Response) ToSemanticResponse() *semantic.SemanticResponse {
	sr := semantic.NewSemanticResponse(r.StatusCode, r.Status)
	sr.Headers = r.Headers
	sr.Body = r.BodyString
	sr.FromCache = r.FromCache
	sr.Duration = r.Duration.String()
	sr.CreatedTime = time.Now().Format(time.RFC3339)
	return sr
}

// ToJSONLD exports the response as JSON-LD
func (r *Response) ToJSONLD() (string, error) {
	sr := r.ToSemanticResponse()
	return sr.ToJSONLD()
}

// FromSemanticRequest creates a Request from Schema.org representation
func FromSemanticRequest(sr *semantic.SemanticRequest) *Request {
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
