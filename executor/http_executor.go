package executor

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"eve.evalgo.org/semantic"
)

// HTTPExecutor executes HTTP-based semantic actions
type HTTPExecutor struct {
	Client *http.Client
}

// NewHTTPExecutor creates a new HTTP executor with default settings
func NewHTTPExecutor() *HTTPExecutor {
	return &HTTPExecutor{
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the executor's identifier
func (e *HTTPExecutor) Name() string {
	return "http"
}

// CanHandle determines if this executor can process the action
func (e *HTTPExecutor) CanHandle(action *semantic.SemanticScheduledAction) bool {
	if action == nil || action.Object == nil {
		return false
	}

	// Check if it's an HTTP-based action type
	switch action.Type {
	case "SearchAction", "RetrieveAction", "SendAction", "CreateAction",
		"UpdateAction", "DeleteAction", "ReplaceAction":
		// Check if object has a URL
		if action.Object.ContentUrl != "" {
			return strings.HasPrefix(action.Object.ContentUrl, "http://") ||
				strings.HasPrefix(action.Object.ContentUrl, "https://")
		}
	}

	return false
}

// Execute runs the HTTP action and returns the result
func (e *HTTPExecutor) Execute(ctx context.Context, action *semantic.SemanticScheduledAction) (*Result, error) {
	result := &Result{
		StartTime: time.Now(),
		Status:    StatusRunning,
		Metadata:  make(map[string]interface{}),
	}

	if action == nil || action.Object == nil {
		result.Status = StatusFailed
		result.Error = &ExecutionError{
			Message: "action or action.Object is nil",
			Code:    "INVALID_ACTION",
		}
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result, result.Error
	}

	// Determine HTTP method
	method := e.getHTTPMethod(action.Type)
	result.Metadata["http_method"] = method

	// Build request
	var body io.Reader
	if action.Object.Text != "" {
		body = strings.NewReader(action.Object.Text)
		result.Metadata["request_body_length"] = len(action.Object.Text)
	}

	req, err := http.NewRequestWithContext(ctx, method, action.Object.ContentUrl, body)
	if err != nil {
		result.Status = StatusFailed
		result.Error = &ExecutionError{
			Message: fmt.Sprintf("failed to create HTTP request: %v", err),
			Code:    "REQUEST_ERROR",
		}
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result, result.Error
	}

	// Set content type if specified
	if action.Object.EncodingFormat != "" {
		req.Header.Set("Content-Type", action.Object.EncodingFormat)
	} else if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Add custom headers if present
	if action.Object.Properties != nil {
		for key, value := range action.Object.Properties {
			if strValue, ok := value.(string); ok {
				req.Header.Set(key, strValue)
			}
		}
	}

	result.Metadata["url"] = action.Object.ContentUrl

	// Execute request
	resp, err := e.Client.Do(req)
	if err != nil {
		result.Status = StatusFailed
		result.Error = &ExecutionError{
			Message: fmt.Sprintf("HTTP request failed: %v", err),
			Code:    "HTTP_ERROR",
		}
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result, result.Error
	}
	defer resp.Body.Close()

	result.Metadata["http_status"] = resp.StatusCode
	result.Metadata["http_status_text"] = resp.Status

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Status = StatusFailed
		result.Error = &ExecutionError{
			Message: fmt.Sprintf("failed to read response body: %v", err),
			Code:    "RESPONSE_ERROR",
		}
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result, result.Error
	}

	result.Output = string(respBody)
	result.Metadata["response_body_length"] = len(respBody)
	result.Metadata["content_type"] = resp.Header.Get("Content-Type")

	// Determine success/failure based on status code
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Status = StatusCompleted
	} else {
		result.Status = StatusFailed
		result.Error = &ExecutionError{
			Message: fmt.Sprintf("HTTP request failed with status %d", resp.StatusCode),
			Code:    fmt.Sprintf("HTTP_%d", resp.StatusCode),
			Details: map[string]interface{}{
				"status_code": resp.StatusCode,
				"status":      resp.Status,
				"body":        string(respBody),
			},
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

// getHTTPMethod maps semantic action types to HTTP methods
func (e *HTTPExecutor) getHTTPMethod(actionType string) string {
	switch actionType {
	case "SearchAction", "RetrieveAction":
		return http.MethodGet
	case "SendAction", "CreateAction":
		return http.MethodPost
	case "UpdateAction", "ReplaceAction":
		return http.MethodPut
	case "DeleteAction":
		return http.MethodDelete
	default:
		return http.MethodGet
	}
}
