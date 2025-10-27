package db

import (
	"encoding/json"
	"fmt"
)

// Traverse follows relationships between documents starting from a given document.
// This enables graph-like navigation through document references.
//
// Parameters:
//   - opts: TraversalOptions configuring start point, depth, and relationship field
//
// Returns:
//   - []json.RawMessage: Slice of traversed documents as raw JSON
//   - error: Traversal or query errors
//
// Traversal Directions:
//
//	Forward: Follows document references (container → host)
//	- Start at container document
//	- Follow "hostedOn" field to find host
//	- Continue following relationships up to specified depth
//
//	Reverse: Finds documents that reference the start document (host → containers)
//	- Start at host document
//	- Find all documents with "hostedOn" pointing to this host
//	- Continue finding referring documents up to specified depth
//
// Depth Control:
//   - Depth 1: Only immediate neighbors
//   - Depth 2: Neighbors and their neighbors
//   - Depth N: Up to N relationship hops
//
// Example Usage:
//
//	// Find all containers on a host (reverse traversal)
//	opts := TraversalOptions{
//	    StartID:       "host-123",
//	    Depth:         1,
//	    RelationField: "hostedOn",
//	    Direction:     "reverse",
//	}
//	containers, err := service.Traverse(opts)
//
//	// Find host and datacenter for a container (forward traversal)
//	opts = TraversalOptions{
//	    StartID:       "container-456",
//	    Depth:         2,
//	    RelationField: "hostedOn",
//	    Direction:     "forward",
//	}
//	related, err := service.Traverse(opts)
func (c *CouchDBService) Traverse(opts TraversalOptions) ([]json.RawMessage, error) {
	if opts.Direction == "reverse" {
		return c.traverseReverse(opts)
	}
	return c.traverseForward(opts)
}

// traverseForward follows references from the start document.
// Implements forward traversal by following relationship fields.
func (c *CouchDBService) traverseForward(opts TraversalOptions) ([]json.RawMessage, error) {
	visited := make(map[string]bool)
	var results []json.RawMessage

	// Get starting document
	var startDoc map[string]interface{}
	if err := c.GetGenericDocument(opts.StartID, &startDoc); err != nil {
		return nil, fmt.Errorf("failed to get start document: %w", err)
	}

	visited[opts.StartID] = true

	// Process starting document
	startJSON, _ := json.Marshal(startDoc)
	results = append(results, startJSON)

	// Current level of documents to process
	currentLevel := []map[string]interface{}{startDoc}

	// Traverse up to specified depth
	for depth := 0; depth < opts.Depth; depth++ {
		var nextLevel []map[string]interface{}

		for _, doc := range currentLevel {
			// Get the relationship field value
			if relationValue, ok := doc[opts.RelationField]; ok {
				// Handle single reference
				if relationID, ok := relationValue.(string); ok && relationID != "" {
					if !visited[relationID] {
						var relatedDoc map[string]interface{}
						if err := c.GetGenericDocument(relationID, &relatedDoc); err == nil {
							// Apply filter if specified
							if opts.Filter != nil && !matchesFilter(relatedDoc, opts.Filter) {
								continue
							}

							visited[relationID] = true
							docJSON, _ := json.Marshal(relatedDoc)
							results = append(results, docJSON)
							nextLevel = append(nextLevel, relatedDoc)
						}
					}
				}

				// Handle array of references
				if relationArray, ok := relationValue.([]interface{}); ok {
					for _, item := range relationArray {
						if relationID, ok := item.(string); ok && relationID != "" {
							if !visited[relationID] {
								var relatedDoc map[string]interface{}
								if err := c.GetGenericDocument(relationID, &relatedDoc); err == nil {
									// Apply filter if specified
									if opts.Filter != nil && !matchesFilter(relatedDoc, opts.Filter) {
										continue
									}

									visited[relationID] = true
									docJSON, _ := json.Marshal(relatedDoc)
									results = append(results, docJSON)
									nextLevel = append(nextLevel, relatedDoc)
								}
							}
						}
					}
				}
			}
		}

		if len(nextLevel) == 0 {
			break
		}
		currentLevel = nextLevel
	}

	return results, nil
}

// traverseReverse finds documents that reference the start document.
// Implements reverse traversal by querying for documents with matching relationship field.
func (c *CouchDBService) traverseReverse(opts TraversalOptions) ([]json.RawMessage, error) {
	visited := make(map[string]bool)
	var results []json.RawMessage

	// Get starting document
	var startDoc map[string]interface{}
	if err := c.GetGenericDocument(opts.StartID, &startDoc); err != nil {
		return nil, fmt.Errorf("failed to get start document: %w", err)
	}

	visited[opts.StartID] = true

	// Process starting document
	startJSON, _ := json.Marshal(startDoc)
	results = append(results, startJSON)

	// Current level document IDs to find references to
	currentLevel := []string{opts.StartID}

	// Traverse up to specified depth
	for depth := 0; depth < opts.Depth; depth++ {
		var nextLevel []string

		for _, docID := range currentLevel {
			// Find all documents that reference this document
			dependents, err := c.GetDependents(docID, opts.RelationField)
			if err != nil {
				continue
			}

			for _, dependent := range dependents {
				var depDoc map[string]interface{}
				if err := json.Unmarshal(dependent, &depDoc); err != nil {
					continue
				}

				// Get document ID
				depID, ok := depDoc["_id"].(string)
				if !ok || visited[depID] {
					continue
				}

				// Apply filter if specified
				if opts.Filter != nil && !matchesFilter(depDoc, opts.Filter) {
					continue
				}

				visited[depID] = true
				results = append(results, dependent)
				nextLevel = append(nextLevel, depID)
			}
		}

		if len(nextLevel) == 0 {
			break
		}
		currentLevel = nextLevel
	}

	return results, nil
}

// TraverseTyped performs typed graph traversal using generics.
// This provides compile-time type safety for traversal results.
//
// Type Parameter:
//   - T: Expected document type
//
// Parameters:
//   - opts: TraversalOptions configuration
//
// Returns:
//   - []T: Slice of traversed documents of type T
//   - error: Traversal or parsing errors
//
// Example Usage:
//
//	type Container struct {
//	    ID       string `json:"_id"`
//	    Name     string `json:"name"`
//	    HostedOn string `json:"hostedOn"`
//	}
//
//	opts := TraversalOptions{
//	    StartID:       "host-123",
//	    Depth:         1,
//	    RelationField: "hostedOn",
//	    Direction:     "reverse",
//	}
//
//	containers, err := TraverseTyped[Container](service, opts)
//	for _, container := range containers {
//	    fmt.Printf("Container: %s on %s\n", container.Name, container.HostedOn)
//	}
func TraverseTyped[T any](c *CouchDBService, opts TraversalOptions) ([]T, error) {
	rawResults, err := c.Traverse(opts)
	if err != nil {
		return nil, err
	}

	var results []T
	for _, raw := range rawResults {
		var doc T
		if err := json.Unmarshal(raw, &doc); err != nil {
			continue
		}
		results = append(results, doc)
	}

	return results, nil
}

// GetDependents finds all documents that reference the given document ID.
// This performs a reverse lookup to find documents with the specified relationship.
//
// Parameters:
//   - id: Document ID to find dependents for
//   - relationField: Field name containing the reference
//
// Returns:
//   - []json.RawMessage: Slice of dependent documents as raw JSON
//   - error: Query or execution errors
//
// Query Strategy:
//
//	Uses Mango query to find documents where relationField equals the given ID:
//	- Efficient with proper indexing on relationField
//	- Returns all matching documents regardless of type
//	- Suitable for finding all containers on a host
//
// Example Usage:
//
//	// Find all containers running on host-123
//	containers, err := service.GetDependents("host-123", "hostedOn")
//	if err != nil {
//	    log.Printf("Failed to get dependents: %v", err)
//	    return
//	}
//
//	fmt.Printf("Found %d containers on this host\n", len(containers))
//	for _, container := range containers {
//	    var c map[string]interface{}
//	    json.Unmarshal(container, &c)
//	    fmt.Printf("  - %s\n", c["name"])
//	}
func (c *CouchDBService) GetDependents(id string, relationField string) ([]json.RawMessage, error) {
	// Query for documents where relationField equals the given ID
	selector := map[string]interface{}{
		relationField: id,
	}

	query := MangoQuery{
		Selector: selector,
	}

	results, err := c.Find(query)
	if err != nil {
		return nil, fmt.Errorf("failed to find dependents: %w", err)
	}

	return results, nil
}

// GetDependencies finds all documents referenced by the given document.
// This performs forward lookup to find documents referenced in relationship fields.
//
// Parameters:
//   - id: Document ID to find dependencies for
//   - relationFields: Slice of field names to check for references
//
// Returns:
//   - map[string]json.RawMessage: Map of field name to referenced document
//   - error: Query or execution errors
//
// Multi-Field Support:
//
//	Checks multiple relationship fields in the document:
//	- "hostedOn": Find the host this container runs on
//	- "dependsOn": Find services this service depends on
//	- "network": Find the network configuration
//
// Example Usage:
//
//	// Find all dependencies for a container
//	dependencies, err := service.GetDependencies("container-456", []string{
//	    "hostedOn",
//	    "dependsOn",
//	    "network",
//	})
//
//	if err != nil {
//	    log.Printf("Failed to get dependencies: %v", err)
//	    return
//	}
//
//	if hostDoc, ok := dependencies["hostedOn"]; ok {
//	    var host map[string]interface{}
//	    json.Unmarshal(hostDoc, &host)
//	    fmt.Printf("Hosted on: %s\n", host["name"])
//	}
//
//	if netDoc, ok := dependencies["network"]; ok {
//	    var network map[string]interface{}
//	    json.Unmarshal(netDoc, &network)
//	    fmt.Printf("Network: %s\n", network["name"])
//	}
func (c *CouchDBService) GetDependencies(id string, relationFields []string) (map[string]json.RawMessage, error) {
	// Get the source document
	var doc map[string]interface{}
	if err := c.GetGenericDocument(id, &doc); err != nil {
		return nil, fmt.Errorf("failed to get source document: %w", err)
	}

	dependencies := make(map[string]json.RawMessage)

	// Check each relationship field
	for _, field := range relationFields {
		if relationValue, ok := doc[field]; ok {
			// Handle single reference
			if relationID, ok := relationValue.(string); ok && relationID != "" {
				var relatedDoc map[string]interface{}
				if err := c.GetGenericDocument(relationID, &relatedDoc); err == nil {
					docJSON, _ := json.Marshal(relatedDoc)
					dependencies[field] = docJSON
				}
			}

			// Handle array of references
			if relationArray, ok := relationValue.([]interface{}); ok {
				var relatedDocs []map[string]interface{}
				for _, item := range relationArray {
					if relationID, ok := item.(string); ok && relationID != "" {
						var relatedDoc map[string]interface{}
						if err := c.GetGenericDocument(relationID, &relatedDoc); err == nil {
							relatedDocs = append(relatedDocs, relatedDoc)
						}
					}
				}
				if len(relatedDocs) > 0 {
					docsJSON, _ := json.Marshal(relatedDocs)
					dependencies[field] = docsJSON
				}
			}
		}
	}

	return dependencies, nil
}

// matchesFilter checks if a document matches the filter criteria.
// This helper function applies filter conditions to documents during traversal.
func matchesFilter(doc map[string]interface{}, filter map[string]interface{}) bool {
	for key, value := range filter {
		docValue, ok := doc[key]
		if !ok {
			return false
		}

		// Simple equality check
		if docValue != value {
			return false
		}
	}
	return true
}

// GetRelationshipGraph builds a complete relationship graph for a document.
// This discovers all related documents in both forward and reverse directions.
//
// Parameters:
//   - startID: Starting document ID
//   - relationField: Relationship field to follow
//   - maxDepth: Maximum depth to traverse
//
// Returns:
//   - *RelationshipGraph: Complete graph structure
//   - error: Traversal or query errors
//
// Graph Structure:
//
//	The graph includes:
//	- Nodes: All discovered documents
//	- Edges: Relationships between documents
//	- Metadata: Relationship types and directions
//
// Example Usage:
//
//	graph, err := service.GetRelationshipGraph("container-456", "hostedOn", 3)
//	if err != nil {
//	    log.Printf("Failed to build graph: %v", err)
//	    return
//	}
//
//	fmt.Printf("Graph has %d nodes and %d edges\n",
//	    len(graph.Nodes), len(graph.Edges))
func (c *CouchDBService) GetRelationshipGraph(startID string, relationField string, maxDepth int) (*RelationshipGraph, error) {
	graph := &RelationshipGraph{
		Nodes: make(map[string]json.RawMessage),
		Edges: []RelationshipEdge{},
	}

	// Forward traversal
	forwardOpts := TraversalOptions{
		StartID:       startID,
		Depth:         maxDepth,
		RelationField: relationField,
		Direction:     "forward",
	}
	forwardDocs, _ := c.Traverse(forwardOpts)

	// Reverse traversal
	reverseOpts := TraversalOptions{
		StartID:       startID,
		Depth:         maxDepth,
		RelationField: relationField,
		Direction:     "reverse",
	}
	reverseDocs, _ := c.Traverse(reverseOpts)

	// Add all documents to nodes
	for _, doc := range forwardDocs {
		var docMap map[string]interface{}
		if err := json.Unmarshal(doc, &docMap); err == nil {
			if id, ok := docMap["_id"].(string); ok {
				graph.Nodes[id] = doc
			}
		}
	}

	for _, doc := range reverseDocs {
		var docMap map[string]interface{}
		if err := json.Unmarshal(doc, &docMap); err == nil {
			if id, ok := docMap["_id"].(string); ok {
				graph.Nodes[id] = doc

				// Create edges for reverse relationships
				if targetID, ok := docMap[relationField].(string); ok {
					edge := RelationshipEdge{
						From:  id,
						To:    targetID,
						Type:  relationField,
					}
					graph.Edges = append(graph.Edges, edge)
				}
			}
		}
	}

	return graph, nil
}

// RelationshipGraph represents a graph of related documents.
// Contains nodes (documents) and edges (relationships) for visualization and analysis.
type RelationshipGraph struct {
	Nodes map[string]json.RawMessage // Map of document ID to document
	Edges []RelationshipEdge          // Relationships between documents
}

// RelationshipEdge represents a relationship between two documents.
// Describes the direction and type of the relationship.
type RelationshipEdge struct {
	From string // Source document ID
	To   string // Target document ID
	Type string // Relationship type (field name)
}
