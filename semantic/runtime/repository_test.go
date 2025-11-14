package runtime

import (
	"context"
	"testing"
	"time"
)

// Test requires CouchDB running at localhost:5984
// Skip if not available

func TestRuntimeRepository_WorkflowOperations(t *testing.T) {
	t.Skip("Integration test - requires CouchDB")

	repo, err := NewRuntimeRepository("http://localhost:5984", "when_test", "", "")
	if err != nil {
		t.Skipf("CouchDB not available: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()

	// Create workflow
	workflow := &RuntimeAction{
		Context:      "https://schema.org",
		Type:         "ItemList",
		Identifier:   "test-workflow-123",
		Name:         "Test Workflow",
		ActionStatus: "PotentialActionStatus",
		DateCreated:  time.Now(),
		DateModified: time.Now(),
		AllFields: map[string]interface{}{
			"identifier": "test-workflow-123",
			"name":       "Test Workflow",
		},
	}

	// Save workflow
	err = repo.SaveWorkflow(ctx, workflow)
	if err != nil {
		t.Fatalf("SaveWorkflow failed: %v", err)
	}

	// Get workflow
	retrieved, err := repo.GetWorkflow(ctx, "test-workflow-123")
	if err != nil {
		t.Fatalf("GetWorkflow failed: %v", err)
	}

	if retrieved.Identifier != "test-workflow-123" {
		t.Errorf("Expected identifier=test-workflow-123, got %s", retrieved.Identifier)
	}

	// List workflows
	workflows, err := repo.ListWorkflows(ctx, 10)
	if err != nil {
		t.Fatalf("ListWorkflows failed: %v", err)
	}

	if len(workflows) == 0 {
		t.Error("Expected at least one workflow")
	}

	// Delete workflow
	err = repo.DeleteWorkflow(ctx, "test-workflow-123")
	if err != nil {
		t.Fatalf("DeleteWorkflow failed: %v", err)
	}
}

func TestRuntimeRepository_ActionOperations(t *testing.T) {
	t.Skip("Integration test - requires CouchDB")

	repo, err := NewRuntimeRepository("http://localhost:5984", "when_test", "", "")
	if err != nil {
		t.Skipf("CouchDB not available: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()

	// Create workflow first
	workflow := &RuntimeAction{
		Context:    "https://schema.org",
		Type:       "ItemList",
		Identifier: "test-workflow-456",
		AllFields:  map[string]interface{}{"identifier": "test-workflow-456"},
	}
	repo.SaveWorkflow(ctx, workflow)
	defer repo.DeleteWorkflow(ctx, "test-workflow-456")

	// Create action
	action := &RuntimeAction{
		Context:      "https://schema.org",
		Type:         "SearchAction",
		Identifier:   "action1",
		Name:         "Test Action",
		IsPartOf:     "test-workflow-456",
		ActionStatus: "PotentialActionStatus",
		AllFields: map[string]interface{}{
			"identifier": "action1",
			"name":       "Test Action",
			"isPartOf":   "test-workflow-456",
		},
		ControlMetadata: &ControlMetadata{
			URL:        "https://service.local/api",
			HTTPMethod: "POST",
			Enabled:    true,
		},
	}

	// Save action
	err = repo.SaveAction(ctx, action)
	if err != nil {
		t.Fatalf("SaveAction failed: %v", err)
	}

	// Get action
	retrieved, err := repo.GetAction(ctx, "test-workflow-456", "action1")
	if err != nil {
		t.Fatalf("GetAction failed: %v", err)
	}

	if retrieved.Identifier != "action1" {
		t.Errorf("Expected identifier=action1, got %s", retrieved.Identifier)
	}

	if retrieved.IsPartOf != "test-workflow-456" {
		t.Errorf("Expected isPartOf=test-workflow-456, got %s", retrieved.IsPartOf)
	}

	// List actions by workflow
	actions, err := repo.ListActionsByWorkflow(ctx, "test-workflow-456")
	if err != nil {
		t.Fatalf("ListActionsByWorkflow failed: %v", err)
	}

	if len(actions) != 1 {
		t.Errorf("Expected 1 action, got %d", len(actions))
	}

	// Delete action
	err = repo.DeleteAction(ctx, "test-workflow-456", "action1")
	if err != nil {
		t.Fatalf("DeleteAction failed: %v", err)
	}
}
