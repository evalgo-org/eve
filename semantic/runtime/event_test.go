package runtime

import (
	"encoding/json"
	"testing"
)

func TestNewEvent(t *testing.T) {
	event := NewEvent(EventTypeActionSuccess, "Test Event", "Test description")

	if event.Context != "https://schema.org" {
		t.Errorf("Expected context=https://schema.org, got %s", event.Context)
	}
	if event.Type != "Event" {
		t.Errorf("Expected type=Event, got %s", event.Type)
	}
	if event.Name != "Test Event" {
		t.Errorf("Expected name=Test Event, got %s", event.Name)
	}
	if event.Description != "Test description" {
		t.Errorf("Expected description=Test description, got %s", event.Description)
	}
	if event.AdditionalProperty["eventType"] != EventTypeActionSuccess {
		t.Error("eventType not set in AdditionalProperty")
	}
	if event.Identifier == "" {
		t.Error("Identifier not generated")
	}
	if event.StartDate.IsZero() {
		t.Error("StartDate not set")
	}
}

func TestNewRequestSentEvent(t *testing.T) {
	action := &RuntimeAction{
		Type:       "SearchAction",
		Identifier: "test-action",
		IsPartOf:   "workflow-uuid",
	}

	event := NewRequestSentEvent(action, "https://service.local/api", 1234)

	if event.Name != "Action Request Sent" {
		t.Errorf("Unexpected name: %s", event.Name)
	}

	// Verify about field
	about, ok := event.About["@id"].(string)
	if !ok || about != "/workflow-uuid/test-action" {
		t.Errorf("Expected @id=/workflow-uuid/test-action, got %v", event.About["@id"])
	}

	// Verify location
	location, ok := event.Location["url"].(string)
	if !ok || location != "https://service.local/api" {
		t.Errorf("Expected url=https://service.local/api, got %v", event.Location["url"])
	}

	// Verify additional properties
	if event.AdditionalProperty["requestSize"] != "1234" {
		t.Errorf("requestSize not set correctly: %v", event.AdditionalProperty["requestSize"])
	}
	if event.AdditionalProperty["workflowId"] != "workflow-uuid" {
		t.Error("workflowId not set")
	}
}

func TestNewResponseReceivedEvent(t *testing.T) {
	action := &RuntimeAction{
		Type:       "SearchAction",
		Identifier: "test-action",
		IsPartOf:   "workflow-uuid",
	}

	event := NewResponseReceivedEvent(action, "https://service.local/api", 200, 5678, 3333)

	if event.Name != "Action Response Received" {
		t.Errorf("Unexpected name: %s", event.Name)
	}

	// Verify additional properties
	if event.AdditionalProperty["httpStatus"] != 200 {
		t.Error("httpStatus not set correctly")
	}
	if event.AdditionalProperty["responseSize"] != "5678" {
		t.Error("responseSize not set correctly")
	}
	if event.AdditionalProperty["durationMs"] != int64(3333) {
		t.Error("durationMs not set correctly")
	}
}

func TestNewActionSuccessEvent(t *testing.T) {
	action := &RuntimeAction{
		Type:       "SearchAction",
		Identifier: "test-action",
		IsPartOf:   "workflow-uuid",
		Result: &ActionResult{
			ContentURL:  "/tmp/output.xml",
			ContentSize: "1234",
		},
	}

	event := NewActionSuccessEvent(action, 5000)

	if event.Name != "Action Completed Successfully" {
		t.Errorf("Unexpected name: %s", event.Name)
	}

	if event.Description != "Action completed in 5.00 seconds" {
		t.Errorf("Unexpected description: %s", event.Description)
	}

	// Verify result included
	if event.AdditionalProperty["resultUrl"] != "/tmp/output.xml" {
		t.Error("resultUrl not included")
	}
	if event.AdditionalProperty["resultSize"] != "1234" {
		t.Error("resultSize not included")
	}
	if event.AdditionalProperty["durationMs"] != int64(5000) {
		t.Error("durationMs not set correctly")
	}
}

func TestNewActionFailureEvent(t *testing.T) {
	action := &RuntimeAction{
		Type:       "SearchAction",
		Identifier: "test-action",
		IsPartOf:   "workflow-uuid",
		Error: &ActionError{
			Name:        "ConnectionTimeout",
			Description: "Connection timeout after 30 seconds",
			AdditionalProperty: map[string]interface{}{
				"errorCode": "CONN_TIMEOUT",
				"retryable": true,
			},
		},
	}

	event := NewActionFailureEvent(action, 30000)

	if event.Name != "Action Failed" {
		t.Errorf("Unexpected name: %s", event.Name)
	}

	if event.Description != "Connection timeout after 30 seconds" {
		t.Errorf("Unexpected description: %s", event.Description)
	}

	// Verify error details in additionalProperty
	if event.AdditionalProperty["errorType"] != "ConnectionTimeout" {
		t.Error("errorType not set correctly")
	}
	if event.AdditionalProperty["errorCode"] != "CONN_TIMEOUT" {
		t.Error("errorCode not set correctly")
	}
	if event.AdditionalProperty["retryable"] != true {
		t.Error("retryable not set correctly")
	}
}

func TestNewWorkflowStartedEvent(t *testing.T) {
	parameters := map[string]string{
		"CONCEPT_SCHEME": "https://data.zeiss.com/IQS/4",
	}

	event := NewWorkflowStartedEvent("workflow-uuid", "iqs-cache-empolis-jsons", "2.1.0", 10, parameters)

	if event.Name != "Workflow Started" {
		t.Errorf("Unexpected name: %s", event.Name)
	}

	if event.Description != "Started workflow with 10 actions" {
		t.Errorf("Unexpected description: %s", event.Description)
	}

	// Verify workflow reference
	about, ok := event.About["@id"].(string)
	if !ok || about != "/workflow-uuid" {
		t.Error("workflow @id not set correctly")
	}

	// Verify additional properties
	if event.AdditionalProperty["templateId"] != "iqs-cache-empolis-jsons" {
		t.Error("templateId not set")
	}
	if event.AdditionalProperty["templateVersion"] != "2.1.0" {
		t.Error("templateVersion not set")
	}
	if event.AdditionalProperty["actionCount"] != 10 {
		t.Error("actionCount not set")
	}

	// Verify parameters
	params, ok := event.AdditionalProperty["parameters"].(map[string]string)
	if !ok || params["CONCEPT_SCHEME"] != "https://data.zeiss.com/IQS/4" {
		t.Error("parameters not set correctly")
	}
}

func TestNewWorkflowCompletedEvent(t *testing.T) {
	tests := []struct {
		name           string
		completed      int
		failed         int
		cancelled      int
		expectedStatus string
		expectedDesc   string
	}{
		{
			name:           "all success",
			completed:      10,
			failed:         0,
			cancelled:      0,
			expectedStatus: "success",
			expectedDesc:   "Workflow completed: 10 actions completed, 0 failed, 0 cancelled",
		},
		{
			name:           "with failures",
			completed:      8,
			failed:         2,
			cancelled:      0,
			expectedStatus: "failed",
			expectedDesc:   "Workflow completed: 8 actions completed, 2 failed, 0 cancelled",
		},
		{
			name:           "with cancellations",
			completed:      7,
			failed:         0,
			cancelled:      3,
			expectedStatus: "cancelled",
			expectedDesc:   "Workflow completed: 7 actions completed, 0 failed, 3 cancelled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := NewWorkflowCompletedEvent("workflow-uuid", 900000, tt.completed, tt.failed, tt.cancelled)

			if event.Name != "Workflow Completed" {
				t.Errorf("Unexpected name: %s", event.Name)
			}

			if event.Description != tt.expectedDesc {
				t.Errorf("Expected description=%s, got %s", tt.expectedDesc, event.Description)
			}

			if event.AdditionalProperty["status"] != tt.expectedStatus {
				t.Errorf("Expected status=%s, got %v", tt.expectedStatus, event.AdditionalProperty["status"])
			}

			if event.AdditionalProperty["durationMs"] != int64(900000) {
				t.Error("durationMs not set correctly")
			}
		})
	}
}

func TestEvent_SetOrganizer(t *testing.T) {
	event := NewEvent(EventTypeActionSuccess, "Test", "")
	event.SetOrganizer("when-ng-executor", "1.0.0", "executor-pod-123")

	if event.Organizer == nil {
		t.Fatal("Organizer not set")
	}

	if event.Organizer["name"] != "when-ng-executor" {
		t.Error("Organizer name not set correctly")
	}
	if event.Organizer["version"] != "1.0.0" {
		t.Error("Organizer version not set correctly")
	}
	if event.Organizer["identifier"] != "executor-pod-123" {
		t.Error("Organizer identifier not set correctly")
	}
}

func TestEvent_AddPayloadReference(t *testing.T) {
	event := NewEvent(EventTypeRequestSent, "Test", "")
	event.AddPayloadReference("requestPayloadRef", "/var/log/requests/req-123.json")

	if event.AdditionalProperty["requestPayloadRef"] != "/var/log/requests/req-123.json" {
		t.Error("Payload reference not added correctly")
	}
}

func TestEvent_ToJSON(t *testing.T) {
	event := NewEvent(EventTypeActionSuccess, "Test Event", "Test description")
	event.SetOrganizer("test-app", "1.0.0", "test-id")

	data, err := event.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Unmarshal to verify structure
	var unmarshaled map[string]interface{}
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify key fields
	if unmarshaled["@context"] != "https://schema.org" {
		t.Error("@context not preserved")
	}
	if unmarshaled["@type"] != "Event" {
		t.Error("@type not preserved")
	}
	if unmarshaled["name"] != "Test Event" {
		t.Error("name not preserved")
	}
}

func TestGenerateEventID(t *testing.T) {
	id1 := generateEventID()
	id2 := generateEventID()

	if id1 == "" {
		t.Error("Event ID is empty")
	}
	if id1 == id2 {
		t.Error("Event IDs should be unique")
	}

	// Verify format (event-timestamp-random)
	if len(id1) < len("event-1234567890-a") {
		t.Error("Event ID too short")
	}
}
