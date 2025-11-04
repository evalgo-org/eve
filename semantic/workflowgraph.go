package semantic

import (
	"encoding/json"
	"fmt"
	"strings"

	"go.etcd.io/bbolt"
)

// WorkflowGraph wraps bbolt database for workflow metadata
type WorkflowGraph struct {
	db *bbolt.DB
}

var (
	workflowsBucket       = []byte("workflows")
	workflowActionsBucket = []byte("workflow_actions")
	actionWorkflowBucket  = []byte("action_workflow")
)

// NewWorkflowGraph initializes a bbolt database for workflow graph
func NewWorkflowGraph(dbPath string) (*WorkflowGraph, error) {
	// Use separate path for graph data
	graphPath := strings.TrimSuffix(dbPath, ".db") + "-graph.db"

	// Open bbolt database
	db, err := bbolt.Open(graphPath, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open graph database: %w", err)
	}

	// Create buckets
	err = db.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(workflowsBucket); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(workflowActionsBucket); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(actionWorkflowBucket); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create buckets: %w", err)
	}

	return &WorkflowGraph{db: db}, nil
}

// Close closes the database
func (g *WorkflowGraph) Close() error {
	if g.db != nil {
		return g.db.Close()
	}
	return nil
}

// WorkflowInfo contains workflow metadata
type WorkflowInfo struct {
	Identifier  string `json:"identifier"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// ImportJSONLD imports a JSON-LD workflow document into the graph
func (g *WorkflowGraph) ImportJSONLD(workflowJSON []byte) error {
	var workflow struct {
		Context         string `json:"@context"`
		Type            string `json:"@type"`
		Identifier      string `json:"identifier"`
		Name            string `json:"name"`
		Description     string `json:"description"`
		ItemListElement []struct {
			Type     string `json:"@type"`
			Position int    `json:"position"`
			Item     struct {
				Type       string `json:"@type"`
				Identifier string `json:"identifier"`
				Name       string `json:"name"`
			} `json:"item"`
		} `json:"itemListElement"`
	}

	if err := json.Unmarshal(workflowJSON, &workflow); err != nil {
		return fmt.Errorf("failed to parse workflow JSON: %w", err)
	}

	// Extract action IDs
	var actionIDs []string
	for _, element := range workflow.ItemListElement {
		if element.Item.Identifier != "" {
			actionIDs = append(actionIDs, element.Item.Identifier)
		}
	}

	// Store in database
	return g.db.Update(func(tx *bbolt.Tx) error {
		workflowsBkt := tx.Bucket(workflowsBucket)
		actionsBkt := tx.Bucket(workflowActionsBucket)
		reverseIdxBkt := tx.Bucket(actionWorkflowBucket)

		// Store workflow info
		info := WorkflowInfo{
			Identifier:  workflow.Identifier,
			Name:        workflow.Name,
			Description: workflow.Description,
		}
		infoJSON, err := json.Marshal(info)
		if err != nil {
			return fmt.Errorf("failed to marshal workflow info: %w", err)
		}
		if err := workflowsBkt.Put([]byte(workflow.Identifier), infoJSON); err != nil {
			return err
		}

		// Store workflow actions
		actionsJSON, err := json.Marshal(actionIDs)
		if err != nil {
			return fmt.Errorf("failed to marshal action IDs: %w", err)
		}
		if err := actionsBkt.Put([]byte(workflow.Identifier), actionsJSON); err != nil {
			return err
		}

		// Store reverse index (action -> workflow)
		for _, actionID := range actionIDs {
			if err := reverseIdxBkt.Put([]byte(actionID), []byte(workflow.Identifier)); err != nil {
				return err
			}
		}

		return nil
	})
}

// GetWorkflowActions returns all action identifiers for a given workflow
func (g *WorkflowGraph) GetWorkflowActions(workflowID string) ([]string, error) {
	var actionIDs []string

	err := g.db.View(func(tx *bbolt.Tx) error {
		bkt := tx.Bucket(workflowActionsBucket)
		data := bkt.Get([]byte(workflowID))
		if data == nil {
			return nil // No actions for this workflow
		}

		return json.Unmarshal(data, &actionIDs)
	})

	return actionIDs, err
}

// GetAllWorkflows returns all workflow identifiers and names
func (g *WorkflowGraph) GetAllWorkflows() ([]WorkflowInfo, error) {
	var workflows []WorkflowInfo

	err := g.db.View(func(tx *bbolt.Tx) error {
		bkt := tx.Bucket(workflowsBucket)
		return bkt.ForEach(func(k, v []byte) error {
			var info WorkflowInfo
			if err := json.Unmarshal(v, &info); err != nil {
				return err
			}
			workflows = append(workflows, info)
			return nil
		})
	})

	return workflows, err
}

// GetActionWorkflow returns the workflow identifier for a given action
func (g *WorkflowGraph) GetActionWorkflow(actionID string) (string, error) {
	var workflowID string

	err := g.db.View(func(tx *bbolt.Tx) error {
		bkt := tx.Bucket(actionWorkflowBucket)
		data := bkt.Get([]byte(actionID))
		if data != nil {
			workflowID = string(data)
		}
		return nil
	})

	return workflowID, err
}

// DumpGraph prints all data for debugging
func (g *WorkflowGraph) DumpGraph() error {
	return g.db.View(func(tx *bbolt.Tx) error {
		fmt.Println("=== Graph Contents ===")

		fmt.Println("\n--- Workflows ---")
		tx.Bucket(workflowsBucket).ForEach(func(k, v []byte) error {
			fmt.Printf("Workflow %s: %s\n", k, v)
			return nil
		})

		fmt.Println("\n--- Workflow Actions ---")
		tx.Bucket(workflowActionsBucket).ForEach(func(k, v []byte) error {
			fmt.Printf("Workflow %s -> Actions: %s\n", k, v)
			return nil
		})

		fmt.Println("\n--- Action to Workflow Index ---")
		tx.Bucket(actionWorkflowBucket).ForEach(func(k, v []byte) error {
			fmt.Printf("Action %s -> Workflow: %s\n", k, v)
			return nil
		})

		fmt.Println("=== End Graph ===")
		return nil
	})
}
