package semantic

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cayleygraph/cayley"
	"github.com/cayleygraph/cayley/graph"
	_ "github.com/cayleygraph/cayley/graph/kv/bolt" // BoltDB backend
	"github.com/cayleygraph/quad"
)

// WorkflowGraph wraps Cayley graph store for workflow metadata
type WorkflowGraph struct {
	store *cayley.Handle
}

// NewWorkflowGraph initializes a Cayley graph with BoltDB backend
func NewWorkflowGraph(dbPath string) (*WorkflowGraph, error) {
	// Use separate path for graph data (alongside BoltDB)
	graphPath := strings.TrimSuffix(dbPath, ".db") + "-graph.db"

	// Initialize or open the graph store
	err := graph.InitQuadStore("bolt", graphPath, nil)
	if err != nil && err != graph.ErrDatabaseExists {
		return nil, fmt.Errorf("failed to initialize graph store: %w", err)
	}

	// Open the store
	store, err := cayley.NewGraph("bolt", graphPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open graph store: %w", err)
	}

	return &WorkflowGraph{store: store}, nil
}

// Close closes the graph store
func (g *WorkflowGraph) Close() error {
	if g.store != nil {
		return g.store.Close()
	}
	return nil
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

	// Create triples for the workflow
	workflowIRI := quad.IRI("workflow:" + workflow.Identifier)
	schemaType := quad.IRI("http://www.w3.org/1999/02/22-rdf-syntax-ns#type")
	schemaItemList := quad.IRI("https://schema.org/ItemList")
	schemaIdentifier := quad.IRI("https://schema.org/identifier")
	schemaName := quad.IRI("https://schema.org/name")
	schemaItemListElement := quad.IRI("https://schema.org/itemListElement")
	schemaItem := quad.IRI("https://schema.org/item")

	quads := []quad.Quad{
		quad.Make(workflowIRI, schemaType, schemaItemList, nil),
		quad.Make(workflowIRI, schemaIdentifier, quad.String(workflow.Identifier), nil),
		quad.Make(workflowIRI, schemaName, quad.String(workflow.Name), nil),
	}

	for _, element := range workflow.ItemListElement {
		if element.Item.Identifier != "" {
			itemIRI := quad.IRI("item:" + element.Item.Identifier)
			quads = append(quads,
				quad.Make(workflowIRI, schemaItemListElement, itemIRI, nil),
				quad.Make(itemIRI, schemaItem, quad.String(element.Item.Identifier), nil),
				quad.Make(itemIRI, schemaIdentifier, quad.String(element.Item.Identifier), nil),
			)
		}
	}

	if err := g.store.AddQuadSet(quads); err != nil {
		return fmt.Errorf("failed to add quads to graph: %w", err)
	}

	return nil
}

// GetWorkflowActions returns all action identifiers for a given workflow
func (g *WorkflowGraph) GetWorkflowActions(workflowID string) ([]string, error) {
	ctx := context.Background()

	workflowIRI := quad.IRI("workflow:" + workflowID)
	p := cayley.StartPath(g.store, workflowIRI).
		Out(quad.IRI("https://schema.org/itemListElement")).
		Out(quad.IRI("https://schema.org/item"))

	var actionIDs []string
	err := p.Iterate(ctx).EachValue(nil, func(value quad.Value) {
		if str, ok := value.(quad.String); ok {
			actionIDs = append(actionIDs, string(str))
		}
	})

	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	return actionIDs, nil
}

// WorkflowInfo contains workflow metadata
type WorkflowInfo struct {
	Identifier  string `json:"identifier"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// GetAllWorkflows returns all workflow identifiers and names
func (g *WorkflowGraph) GetAllWorkflows() ([]WorkflowInfo, error) {
	ctx := context.Background()

	p := cayley.StartPath(g.store).
		Has(quad.IRI("http://www.w3.org/1999/02/22-rdf-syntax-ns#type"), quad.IRI("https://schema.org/ItemList"))

	var workflows []WorkflowInfo
	seen := make(map[string]bool)

	err := p.Iterate(ctx).EachValue(nil, func(value quad.Value) {
		workflowIRI := value.String()
		if seen[workflowIRI] {
			return
		}
		seen[workflowIRI] = true

		info := WorkflowInfo{}

		idPath := cayley.StartPath(g.store, value).
			Out(quad.IRI("https://schema.org/identifier"))
		idPath.Iterate(ctx).EachValue(nil, func(v quad.Value) {
			if str, ok := v.(quad.String); ok {
				info.Identifier = string(str)
			}
		})

		namePath := cayley.StartPath(g.store, value).
			Out(quad.IRI("https://schema.org/name"))
		namePath.Iterate(ctx).EachValue(nil, func(v quad.Value) {
			if str, ok := v.(quad.String); ok {
				info.Name = string(str)
			}
		})

		if info.Identifier != "" {
			workflows = append(workflows, info)
		}
	})

	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	return workflows, nil
}

// GetActionWorkflow returns the workflow identifier for a given action
func (g *WorkflowGraph) GetActionWorkflow(actionID string) (string, error) {
	ctx := context.Background()

	p := cayley.StartPath(g.store).
		Has(quad.IRI("https://schema.org/identifier"), quad.String(actionID)).
		In(quad.IRI("https://schema.org/item")).
		In(quad.IRI("https://schema.org/itemListElement")).
		Out(quad.IRI("https://schema.org/identifier"))

	var workflowID string
	err := p.Iterate(ctx).EachValue(nil, func(value quad.Value) {
		if str, ok := value.(quad.String); ok {
			workflowID = string(str)
		}
	})

	if err != nil {
		return "", fmt.Errorf("query failed: %w", err)
	}

	return workflowID, nil
}

// DumpGraph prints all quads for debugging
func (g *WorkflowGraph) DumpGraph() error {
	ctx := context.Background()
	it := g.store.QuadsAllIterator()
	defer it.Close()

	fmt.Println("=== Graph Contents ===")
	for it.Next(ctx) {
		q := g.store.Quad(it.Result())
		fmt.Printf("S: %v | P: %v | O: %v | L: %v\n",
			q.Subject, q.Predicate, q.Object, q.Label)
	}
	fmt.Println("=== End Graph ===")

	return it.Err()
}
