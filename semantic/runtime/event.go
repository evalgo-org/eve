package runtime

import (
	"encoding/json"
	"fmt"
	"time"
)

// Event represents a Schema.org Event for logging significant occurrences
// Used for audit trails and operational logging
type Event struct {
	Context     string    `json:"@context"`
	Type        string    `json:"@type"`
	Identifier  string    `json:"identifier"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	StartDate   time.Time `json:"startDate"`

	// What this event is about (the action/workflow)
	About map[string]interface{} `json:"about,omitempty"`

	// Who/what triggered or organized this event
	Organizer map[string]interface{} `json:"organizer,omitempty"`

	// Where the event occurred (service URL, etc.)
	Location map[string]interface{} `json:"location,omitempty"`

	// Result of the event
	Result map[string]interface{} `json:"result,omitempty"`

	// Additional properties for event-specific metadata
	AdditionalProperty map[string]interface{} `json:"additionalProperty,omitempty"`
}

// EventType constants for common event types
const (
	EventTypeRequestSent       = "request_sent"
	EventTypeResponseReceived  = "response_received"
	EventTypeActionSuccess     = "action_success"
	EventTypeActionFailure     = "action_failure"
	EventTypeWorkflowStarted   = "workflow_started"
	EventTypeWorkflowCompleted = "workflow_completed"
)

// NewEvent creates a new event with basic fields populated
func NewEvent(eventType, name, description string) *Event {
	return &Event{
		Context:     "https://schema.org",
		Type:        "Event",
		Identifier:  generateEventID(),
		Name:        name,
		Description: description,
		StartDate:   time.Now(),
		AdditionalProperty: map[string]interface{}{
			"eventType": eventType,
		},
	}
}

// NewRequestSentEvent creates an event for when a request is sent to a service
func NewRequestSentEvent(action *RuntimeAction, serviceURL string, requestSize int) *Event {
	event := NewEvent(EventTypeRequestSent, "Action Request Sent", "")
	event.Description = fmt.Sprintf("Sent %s to service", action.Type)

	event.About = map[string]interface{}{
		"@type":      "Action",
		"@id":        fmt.Sprintf("/%s/%s", action.IsPartOf, action.Identifier),
		"identifier": action.Identifier,
		"actionType": action.Type,
	}

	event.Location = map[string]interface{}{
		"@type": "VirtualLocation",
		"url":   serviceURL,
	}

	event.AdditionalProperty["requestSize"] = fmt.Sprintf("%d", requestSize)
	event.AdditionalProperty["workflowId"] = action.IsPartOf

	return event
}

// NewResponseReceivedEvent creates an event for when a response is received from a service
func NewResponseReceivedEvent(action *RuntimeAction, serviceURL string, httpStatus, responseSize int, durationMs int64) *Event {
	event := NewEvent(EventTypeResponseReceived, "Action Response Received", "")
	event.Description = fmt.Sprintf("Received response from service (HTTP %d)", httpStatus)

	event.About = map[string]interface{}{
		"@type":      "Action",
		"@id":        fmt.Sprintf("/%s/%s", action.IsPartOf, action.Identifier),
		"identifier": action.Identifier,
	}

	event.Location = map[string]interface{}{
		"@type": "VirtualLocation",
		"url":   serviceURL,
	}

	event.AdditionalProperty["httpStatus"] = httpStatus
	event.AdditionalProperty["responseSize"] = fmt.Sprintf("%d", responseSize)
	event.AdditionalProperty["durationMs"] = durationMs
	event.AdditionalProperty["workflowId"] = action.IsPartOf

	return event
}

// NewActionSuccessEvent creates an event for successful action completion
func NewActionSuccessEvent(action *RuntimeAction, durationMs int64) *Event {
	event := NewEvent(EventTypeActionSuccess, "Action Completed Successfully", "")
	event.Description = fmt.Sprintf("Action completed in %.2f seconds", float64(durationMs)/1000.0)

	event.About = map[string]interface{}{
		"@type":      "Action",
		"@id":        fmt.Sprintf("/%s/%s", action.IsPartOf, action.Identifier),
		"identifier": action.Identifier,
	}

	event.Result = map[string]interface{}{
		"@type":       "Thing",
		"description": event.Description,
	}

	event.AdditionalProperty["durationMs"] = durationMs
	event.AdditionalProperty["workflowId"] = action.IsPartOf

	// Include result details if available
	if action.Result != nil {
		if action.Result.ContentURL != "" {
			event.AdditionalProperty["resultUrl"] = action.Result.ContentURL
		}
		if action.Result.ContentSize != "" {
			event.AdditionalProperty["resultSize"] = action.Result.ContentSize
		}
	}

	return event
}

// NewActionFailureEvent creates an event for failed action
func NewActionFailureEvent(action *RuntimeAction, durationMs int64) *Event {
	event := NewEvent(EventTypeActionFailure, "Action Failed", "")

	errorDesc := "Unknown error"
	errorType := "Error"
	errorCode := ""
	retryable := false

	if action.Error != nil {
		errorDesc = action.Error.Description
		if action.Error.Name != "" {
			errorType = action.Error.Name
		}
		if action.Error.AdditionalProperty != nil {
			if code, ok := action.Error.AdditionalProperty["errorCode"].(string); ok {
				errorCode = code
			}
			if retry, ok := action.Error.AdditionalProperty["retryable"].(bool); ok {
				retryable = retry
			}
		}
	}

	event.Description = errorDesc

	event.About = map[string]interface{}{
		"@type":      "Action",
		"@id":        fmt.Sprintf("/%s/%s", action.IsPartOf, action.Identifier),
		"identifier": action.Identifier,
	}

	event.Result = map[string]interface{}{
		"@type":       "Thing",
		"name":        "Failure",
		"description": errorDesc,
	}

	event.AdditionalProperty["errorType"] = errorType
	if errorCode != "" {
		event.AdditionalProperty["errorCode"] = errorCode
	}
	event.AdditionalProperty["retryable"] = retryable
	event.AdditionalProperty["durationMs"] = durationMs
	event.AdditionalProperty["workflowId"] = action.IsPartOf

	return event
}

// NewWorkflowStartedEvent creates an event for workflow start
func NewWorkflowStartedEvent(workflowID, templateID, templateVersion string, actionCount int, parameters map[string]string) *Event {
	event := NewEvent(EventTypeWorkflowStarted, "Workflow Started", "")
	event.Description = fmt.Sprintf("Started workflow with %d actions", actionCount)

	event.About = map[string]interface{}{
		"@type":      "ItemList",
		"@id":        fmt.Sprintf("/%s", workflowID),
		"identifier": workflowID,
	}

	event.AdditionalProperty["templateId"] = templateID
	if templateVersion != "" {
		event.AdditionalProperty["templateVersion"] = templateVersion
	}
	event.AdditionalProperty["actionCount"] = actionCount
	event.AdditionalProperty["workflowId"] = workflowID
	if len(parameters) > 0 {
		event.AdditionalProperty["parameters"] = parameters
	}

	return event
}

// NewWorkflowCompletedEvent creates an event for workflow completion
func NewWorkflowCompletedEvent(workflowID string, durationMs int64, actionsCompleted, actionsFailed, actionsCancelled int) *Event {
	event := NewEvent(EventTypeWorkflowCompleted, "Workflow Completed", "")

	status := "success"
	if actionsFailed > 0 {
		status = "failed"
	} else if actionsCancelled > 0 {
		status = "cancelled"
	}

	event.Description = fmt.Sprintf("Workflow completed: %d actions completed, %d failed, %d cancelled",
		actionsCompleted, actionsFailed, actionsCancelled)

	event.About = map[string]interface{}{
		"@type":      "ItemList",
		"@id":        fmt.Sprintf("/%s", workflowID),
		"identifier": workflowID,
	}

	event.Result = map[string]interface{}{
		"@type":       "Thing",
		"name":        status,
		"description": event.Description,
	}

	event.AdditionalProperty["status"] = status
	event.AdditionalProperty["durationMs"] = durationMs
	event.AdditionalProperty["actionsCompleted"] = actionsCompleted
	event.AdditionalProperty["actionsFailed"] = actionsFailed
	event.AdditionalProperty["actionsCancelled"] = actionsCancelled
	event.AdditionalProperty["workflowId"] = workflowID

	return event
}

// SetOrganizer sets the organizer (who/what triggered the event)
func (e *Event) SetOrganizer(appName, version, identifier string) {
	e.Organizer = map[string]interface{}{
		"@type":      "SoftwareApplication",
		"name":       appName,
		"version":    version,
		"identifier": identifier,
	}
}

// AddPayloadReference adds a reference to a stored payload file
func (e *Event) AddPayloadReference(key, path string) {
	if e.AdditionalProperty == nil {
		e.AdditionalProperty = make(map[string]interface{})
	}
	e.AdditionalProperty[key] = path
}

// ToJSON marshals the event to JSON
func (e *Event) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// generateEventID generates a unique event identifier
// Format: event-{timestamp}-{random}
func generateEventID() string {
	return fmt.Sprintf("event-%d-%s", time.Now().Unix(), randomString(8))
}

// randomString generates a random alphanumeric string
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
