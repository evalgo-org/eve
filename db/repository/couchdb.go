package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	kivik "github.com/go-kivik/kivik/v4"
	_ "github.com/go-kivik/kivik/v4/couchdb"

	"eve.evalgo.org/semantic"
)

// CouchDBRepository implements DocumentRepository using CouchDB
type CouchDBRepository struct {
	client      *kivik.Client
	workflowsDB *kivik.DB
	actionsDB   *kivik.DB
	ctx         context.Context
}

// NewCouchDBRepository creates a new CouchDB document repository
func NewCouchDBRepository(url, user, password string) (*CouchDBRepository, error) {
	ctx := context.Background()

	// Build connection URL with authentication
	connectionURL := url
	if user != "" && password != "" {
		// Parse URL to inject credentials if not already present
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

	// Get or create workflows database
	workflowsDB := client.DB("when_workflows")
	if err := workflowsDB.Err(); err != nil {
		// Try to create it
		if err := client.CreateDB(ctx, "when_workflows"); err != nil {
			return nil, fmt.Errorf("failed to create workflows database: %w", err)
		}
		workflowsDB = client.DB("when_workflows")
	}

	// Get or create actions database
	actionsDB := client.DB("when_actions")
	if err := actionsDB.Err(); err != nil {
		// Try to create it
		if err := client.CreateDB(ctx, "when_actions"); err != nil {
			return nil, fmt.Errorf("failed to create actions database: %w", err)
		}
		actionsDB = client.DB("when_actions")
	}

	return &CouchDBRepository{
		client:      client,
		workflowsDB: workflowsDB,
		actionsDB:   actionsDB,
		ctx:         ctx,
	}, nil
}

// Workflow operations

func (r *CouchDBRepository) SaveWorkflow(ctx context.Context, workflowID string, workflow map[string]interface{}) error {
	// Add _id if not present
	workflow["_id"] = workflowID

	// Check if document exists to get revision
	var existing map[string]interface{}
	err := r.workflowsDB.Get(ctx, workflowID).ScanDoc(&existing)
	if err == nil {
		// Document exists, preserve _rev
		if rev, ok := existing["_rev"].(string); ok {
			workflow["_rev"] = rev
		}
	}

	_, err = r.workflowsDB.Put(ctx, workflowID, workflow)
	return err
}

func (r *CouchDBRepository) GetWorkflow(ctx context.Context, workflowID string) (map[string]interface{}, error) {
	var workflow map[string]interface{}
	err := r.workflowsDB.Get(ctx, workflowID).ScanDoc(&workflow)
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %w", err)
	}

	return workflow, nil
}

func (r *CouchDBRepository) ListWorkflows(ctx context.Context) ([]map[string]interface{}, error) {
	rows := r.workflowsDB.AllDocs(ctx, kivik.Param("include_docs", true))
	defer rows.Close()

	var workflows []map[string]interface{}
	for rows.Next() {
		var workflow map[string]interface{}
		if err := rows.ScanDoc(&workflow); err != nil {
			continue
		}
		workflows = append(workflows, workflow)
	}

	return workflows, rows.Err()
}

func (r *CouchDBRepository) DeleteWorkflow(ctx context.Context, workflowID string) error {
	// Get current revision
	var doc map[string]interface{}
	err := r.workflowsDB.Get(ctx, workflowID).ScanDoc(&doc)
	if err != nil {
		return err
	}

	rev, ok := doc["_rev"].(string)
	if !ok {
		return fmt.Errorf("no revision found for workflow")
	}

	_, err = r.workflowsDB.Delete(ctx, workflowID, rev)
	return err
}

// Action operations

func (r *CouchDBRepository) SaveAction(ctx context.Context, actionID string, action *semantic.SemanticScheduledAction, workflowID string) error {
	// Convert action to map
	data, err := json.Marshal(action)
	if err != nil {
		return err
	}

	var actionMap map[string]interface{}
	if err := json.Unmarshal(data, &actionMap); err != nil {
		return err
	}

	// Add _id
	actionMap["_id"] = actionID

	// Add partOf field if workflowID is specified
	if workflowID != "" {
		actionMap["partOf"] = workflowID
	}

	// Check if document exists to get revision
	var existing map[string]interface{}
	err = r.actionsDB.Get(ctx, actionID).ScanDoc(&existing)
	if err == nil {
		// Document exists, preserve _rev
		if rev, ok := existing["_rev"].(string); ok {
			actionMap["_rev"] = rev
		}
	}

	_, err = r.actionsDB.Put(ctx, actionID, actionMap)
	return err
}

func (r *CouchDBRepository) GetAction(ctx context.Context, actionID string) (*semantic.SemanticScheduledAction, error) {
	var actionMap map[string]interface{}
	err := r.actionsDB.Get(ctx, actionID).ScanDoc(&actionMap)
	if err != nil {
		return nil, fmt.Errorf("action not found: %w", err)
	}

	// DEBUG: Check if controlMetadata exists in the map
	if controlMeta, ok := actionMap["controlMetadata"]; ok {
		fmt.Fprintf(os.Stderr, "DEBUG GetAction: Found controlMetadata in DB for action '%s': %+v\n", actionID, controlMeta)
	} else {
		fmt.Fprintf(os.Stderr, "DEBUG GetAction: NO controlMetadata found in DB for action '%s'\n", actionID)
	}

	// Convert map back to action
	data, err := json.Marshal(actionMap)
	if err != nil {
		return nil, err
	}

	var action semantic.SemanticScheduledAction
	if err := json.Unmarshal(data, &action); err != nil {
		return nil, err
	}

	// DEBUG: Check if Meta is populated after unmarshal
	if action.Meta != nil {
		fmt.Fprintf(os.Stderr, "DEBUG GetAction: After unmarshal, Meta is set - URL='%s', HTTPMethod='%s'\n", action.Meta.URL, action.Meta.HTTPMethod)
	} else {
		fmt.Fprintf(os.Stderr, "DEBUG GetAction: After unmarshal, Meta is NIL for action '%s'\n", actionID)
	}

	return &action, nil
}

func (r *CouchDBRepository) ListActions(ctx context.Context, workflowID string) ([]*semantic.SemanticScheduledAction, error) {
	var rows *kivik.ResultSet

	if workflowID != "" {
		// Query by workflow using Mango query
		selector := map[string]interface{}{
			"partOf": workflowID,
		}
		rows = r.actionsDB.Find(ctx, selector)
	} else {
		// Get all actions
		rows = r.actionsDB.AllDocs(ctx, kivik.Param("include_docs", true))
	}
	defer rows.Close()

	var actions []*semantic.SemanticScheduledAction
	actionCount := 0
	for rows.Next() {
		var actionMap map[string]interface{}
		if err := rows.ScanDoc(&actionMap); err != nil {
			fmt.Fprintf(os.Stderr, "DEBUG ListActions: Failed to scan doc: %v\n", err)
			continue
		}

		actionCount++
		actionID := actionMap["_id"]
		identifier := actionMap["identifier"]

		// Convert to action
		data, err := json.Marshal(actionMap)
		if err != nil {
			fmt.Fprintf(os.Stderr, "DEBUG ListActions: Failed to marshal action %v: %v\n", actionID, err)
			continue
		}

		var action semantic.SemanticScheduledAction
		if err := json.Unmarshal(data, &action); err != nil {
			fmt.Fprintf(os.Stderr, "DEBUG ListActions: Failed to unmarshal action %v: %v\n", actionID, err)
			continue
		}

		// DEBUG: First action only
		if actionCount <= 3 {
			fmt.Fprintf(os.Stderr, "DEBUG ListActions [%d]: _id='%v', identifier field='%v', action.Identifier='%s'\n", actionCount, actionID, identifier, action.Identifier)
			if controlMeta, ok := actionMap["controlMetadata"]; ok {
				fmt.Fprintf(os.Stderr, "DEBUG ListActions [%d]: Has controlMetadata: %+v\n", actionCount, controlMeta)
			} else {
				fmt.Fprintf(os.Stderr, "DEBUG ListActions [%d]: NO controlMetadata in DB\n", actionCount)
			}
			if action.Meta != nil {
				fmt.Fprintf(os.Stderr, "DEBUG ListActions [%d]: After unmarshal, Meta.URL='%s', Meta.HTTPMethod='%s'\n", actionCount, action.Meta.URL, action.Meta.HTTPMethod)
			} else {
				fmt.Fprintf(os.Stderr, "DEBUG ListActions [%d]: After unmarshal, Meta is NIL\n", actionCount)
			}
		}

		actions = append(actions, &action)
	}

	return actions, rows.Err()
}

func (r *CouchDBRepository) DeleteAction(ctx context.Context, actionID string) error {
	// Get current revision
	var doc map[string]interface{}
	err := r.actionsDB.Get(ctx, actionID).ScanDoc(&doc)
	if err != nil {
		return err
	}

	rev, ok := doc["_rev"].(string)
	if !ok {
		return fmt.Errorf("no revision found for action")
	}

	_, err = r.actionsDB.Delete(ctx, actionID, rev)
	return err
}

// Bulk operations

func (r *CouchDBRepository) BulkSaveActions(ctx context.Context, actions []*semantic.SemanticScheduledAction) error {
	docs := make([]interface{}, len(actions))
	for i, action := range actions {
		data, err := json.Marshal(action)
		if err != nil {
			return err
		}

		var actionMap map[string]interface{}
		if err := json.Unmarshal(data, &actionMap); err != nil {
			return err
		}

		actionMap["_id"] = action.Identifier
		docs[i] = actionMap
	}

	_, err := r.actionsDB.BulkDocs(ctx, docs)
	return err
}

// WatchChanges watches for document changes (real-time updates)
func (r *CouchDBRepository) WatchChanges(ctx context.Context) (<-chan ChangeEvent, error) {
	out := make(chan ChangeEvent)

	// Watch both workflows and actions
	go func() {
		defer close(out)

		// Start watching workflows
		workflowChanges := r.workflowsDB.Changes(ctx,
			kivik.Param("feed", "continuous"),
			kivik.Param("include_docs", true),
		)

		for workflowChanges.Next() {
			var doc map[string]interface{}
			if err := workflowChanges.ScanDoc(&doc); err != nil {
				continue
			}

			operation := "updated"
			if deleted, ok := doc["_deleted"].(bool); ok && deleted {
				operation = "deleted"
			}

			out <- ChangeEvent{
				Type:      "workflow",
				Operation: operation,
				ID:        workflowChanges.ID(),
				Document:  doc,
			}
		}
	}()

	// Watch actions in separate goroutine
	go func() {
		actionChanges := r.actionsDB.Changes(ctx,
			kivik.Param("feed", "continuous"),
			kivik.Param("include_docs", true),
		)

		for actionChanges.Next() {
			var doc map[string]interface{}
			if err := actionChanges.ScanDoc(&doc); err != nil {
				continue
			}

			operation := "updated"
			if deleted, ok := doc["_deleted"].(bool); ok && deleted {
				operation = "deleted"
			}

			out <- ChangeEvent{
				Type:      "action",
				Operation: operation,
				ID:        actionChanges.ID(),
				Document:  doc,
			}
		}
	}()

	return out, nil
}

// Close closes the CouchDB connection
func (r *CouchDBRepository) Close() error {
	return r.client.Close()
}
