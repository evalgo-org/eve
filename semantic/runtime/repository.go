package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	kivik "github.com/go-kivik/kivik/v4"
	_ "github.com/go-kivik/kivik/v4/couchdb"
)

// RuntimeRepository manages RuntimeAction storage in CouchDB.
// This repository uses a single database with path-based document IDs:
//   - Workflows: {workflow-uuid}
//   - Actions: {workflow-uuid}/{action-id}
//
// This pattern enables efficient range queries for all actions in a workflow.
type RuntimeRepository struct {
	client *kivik.Client
	db     *kivik.DB
	ctx    context.Context
}

// NewRuntimeRepository creates a new runtime action repository
func NewRuntimeRepository(url, database, user, password string) (*RuntimeRepository, error) {
	ctx := context.Background()

	// Build connection URL with authentication
	connectionURL := url
	if user != "" && password != "" {
		if !strings.Contains(connectionURL, "@") {
			parts := strings.SplitN(connectionURL, "://", 2)
			if len(parts) == 2 {
				connectionURL = fmt.Sprintf("%s://%s:%s@%s", parts[0], user, password, parts[1])
			}
		}
	}

	// Connect to CouchDB
	client, err := kivik.New("couch", connectionURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create CouchDB client: %w", err)
	}

	// Get or create database
	db := client.DB(database)
	if err := db.Err(); err != nil {
		// Try to create it
		if err := client.CreateDB(ctx, database); err != nil {
			return nil, fmt.Errorf("failed to create database: %w", err)
		}
		db = client.DB(database)
	}

	return &RuntimeRepository{
		client: client,
		db:     db,
		ctx:    ctx,
	}, nil
}

// SaveWorkflow saves a workflow instance
func (r *RuntimeRepository) SaveWorkflow(ctx context.Context, workflow *RuntimeAction) error {
	if workflow == nil {
		return fmt.Errorf("workflow is nil")
	}

	// Document ID is the workflow UUID
	docID := workflow.Identifier

	// Marshal to get complete JSON-LD
	data, err := workflow.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal workflow: %w", err)
	}

	// Parse to map
	var docMap map[string]interface{}
	if err := json.Unmarshal(data, &docMap); err != nil {
		return fmt.Errorf("failed to unmarshal to map: %w", err)
	}

	// Add CouchDB _id
	docMap["_id"] = docID

	// Check if document exists to get revision
	var existing map[string]interface{}
	err = r.db.Get(ctx, docID).ScanDoc(&existing)
	if err == nil {
		// Document exists, preserve _rev
		if rev, ok := existing["_rev"].(string); ok {
			docMap["_rev"] = rev
		}
	}

	_, err = r.db.Put(ctx, docID, docMap)
	if err != nil {
		return fmt.Errorf("failed to save workflow: %w", err)
	}

	return nil
}

// GetWorkflow retrieves a workflow by ID
func (r *RuntimeRepository) GetWorkflow(ctx context.Context, workflowID string) (*RuntimeAction, error) {
	var docMap map[string]interface{}
	err := r.db.Get(ctx, workflowID).ScanDoc(&docMap)
	if err != nil {
		if kivik.HTTPStatus(err) == 404 {
			return nil, fmt.Errorf("workflow not found: %s", workflowID)
		}
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	// Marshal back to JSON
	data, err := json.Marshal(docMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal map: %w", err)
	}

	// Unmarshal to RuntimeAction
	var workflow RuntimeAction
	if err := json.Unmarshal(data, &workflow); err != nil {
		return nil, fmt.Errorf("failed to unmarshal workflow: %w", err)
	}

	return &workflow, nil
}

// SaveAction saves an action
func (r *RuntimeRepository) SaveAction(ctx context.Context, action *RuntimeAction) error {
	if action == nil {
		return fmt.Errorf("action is nil")
	}

	// Document ID: {workflow-uuid}/{action-id}
	docID := fmt.Sprintf("%s/%s", action.IsPartOf, action.Identifier)

	// Marshal to get complete JSON-LD
	data, err := action.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal action: %w", err)
	}

	// Parse to map
	var docMap map[string]interface{}
	if err := json.Unmarshal(data, &docMap); err != nil {
		return fmt.Errorf("failed to unmarshal to map: %w", err)
	}

	// Add CouchDB _id
	docMap["_id"] = docID

	// Check if document exists to get revision
	var existing map[string]interface{}
	err = r.db.Get(ctx, docID).ScanDoc(&existing)
	if err == nil {
		// Document exists, preserve _rev
		if rev, ok := existing["_rev"].(string); ok {
			docMap["_rev"] = rev
		}
	}

	_, err = r.db.Put(ctx, docID, docMap)
	if err != nil {
		return fmt.Errorf("failed to save action: %w", err)
	}

	return nil
}

// GetAction retrieves an action by workflow ID and action ID
func (r *RuntimeRepository) GetAction(ctx context.Context, workflowID, actionID string) (*RuntimeAction, error) {
	docID := fmt.Sprintf("%s/%s", workflowID, actionID)

	var docMap map[string]interface{}
	err := r.db.Get(ctx, docID).ScanDoc(&docMap)
	if err != nil {
		if kivik.HTTPStatus(err) == 404 {
			return nil, fmt.Errorf("action not found: %s/%s", workflowID, actionID)
		}
		return nil, fmt.Errorf("failed to get action: %w", err)
	}

	// Marshal back to JSON
	data, err := json.Marshal(docMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal map: %w", err)
	}

	// Unmarshal to RuntimeAction
	var action RuntimeAction
	if err := json.Unmarshal(data, &action); err != nil {
		return nil, fmt.Errorf("failed to unmarshal action: %w", err)
	}

	return &action, nil
}

// ListActionsByWorkflow retrieves all actions for a workflow
func (r *RuntimeRepository) ListActionsByWorkflow(ctx context.Context, workflowID string) ([]*RuntimeAction, error) {
	// Use startkey/endkey range query
	// Pattern: {workflow-uuid}/{action-id}
	startKey := workflowID + "/"
	endKey := workflowID + "/\ufff0" // High unicode character

	rows := r.db.AllDocs(ctx,
		kivik.Param("include_docs", true),
		kivik.Param("startkey", startKey),
		kivik.Param("endkey", endKey),
	)
	defer rows.Close()

	var actions []*RuntimeAction
	for rows.Next() {
		var docMap map[string]interface{}
		if err := rows.ScanDoc(&docMap); err != nil {
			continue
		}

		// Skip if it's the workflow itself (ID without /)
		docID, _ := docMap["_id"].(string)
		if !strings.Contains(docID, "/") || docID == workflowID {
			continue
		}

		// Marshal back to JSON
		data, err := json.Marshal(docMap)
		if err != nil {
			continue
		}

		// Unmarshal to RuntimeAction
		var action RuntimeAction
		if err := json.Unmarshal(data, &action); err != nil {
			continue
		}

		actions = append(actions, &action)
	}

	return actions, rows.Err()
}

// ListWorkflows retrieves all workflow instances
func (r *RuntimeRepository) ListWorkflows(ctx context.Context, limit int) ([]*RuntimeAction, error) {
	params := []kivik.Option{kivik.Param("include_docs", true)}
	if limit > 0 {
		params = append(params, kivik.Param("limit", limit))
	}

	rows := r.db.AllDocs(ctx, params...)
	defer rows.Close()

	var workflows []*RuntimeAction
	for rows.Next() {
		var docMap map[string]interface{}
		if err := rows.ScanDoc(&docMap); err != nil {
			continue
		}

		// Skip design docs and actions (IDs with / or starting with _)
		docID, ok := docMap["_id"].(string)
		if !ok || strings.HasPrefix(docID, "_") || strings.Contains(docID, "/") {
			continue
		}

		// Marshal back to JSON
		data, err := json.Marshal(docMap)
		if err != nil {
			continue
		}

		// Unmarshal to RuntimeAction
		var workflow RuntimeAction
		if err := json.Unmarshal(data, &workflow); err != nil {
			continue
		}

		workflows = append(workflows, &workflow)
	}

	return workflows, rows.Err()
}

// DeleteAction deletes an action
func (r *RuntimeRepository) DeleteAction(ctx context.Context, workflowID, actionID string) error {
	docID := fmt.Sprintf("%s/%s", workflowID, actionID)

	// Get current revision
	var doc map[string]interface{}
	err := r.db.Get(ctx, docID).ScanDoc(&doc)
	if err != nil {
		return fmt.Errorf("failed to get action for deletion: %w", err)
	}

	rev, ok := doc["_rev"].(string)
	if !ok {
		return fmt.Errorf("no revision found for action")
	}

	_, err = r.db.Delete(ctx, docID, rev)
	return err
}

// DeleteWorkflow deletes a workflow and all its actions
func (r *RuntimeRepository) DeleteWorkflow(ctx context.Context, workflowID string) error {
	// Get all actions first
	actions, err := r.ListActionsByWorkflow(ctx, workflowID)
	if err != nil {
		return fmt.Errorf("failed to list actions: %w", err)
	}

	// Delete all actions
	for _, action := range actions {
		if err := r.DeleteAction(ctx, workflowID, action.Identifier); err != nil {
			return fmt.Errorf("failed to delete action %s: %w", action.Identifier, err)
		}
	}

	// Delete workflow itself
	var doc map[string]interface{}
	err = r.db.Get(ctx, workflowID).ScanDoc(&doc)
	if err != nil {
		return fmt.Errorf("failed to get workflow for deletion: %w", err)
	}

	rev, ok := doc["_rev"].(string)
	if !ok {
		return fmt.Errorf("no revision found for workflow")
	}

	_, err = r.db.Delete(ctx, workflowID, rev)
	return err
}

// Close closes the CouchDB connection
func (r *RuntimeRepository) Close() error {
	return r.client.Close()
}
