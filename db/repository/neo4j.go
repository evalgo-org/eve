package repository

import (
	"context"
	"fmt"

	"eve.evalgo.org/semantic"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Neo4jRepository implements GraphRepository using Neo4j
type Neo4jRepository struct {
	driver neo4j.DriverWithContext
	ctx    context.Context
}

// NewNeo4jRepository creates a new Neo4j graph repository
func NewNeo4jRepository(uri, username, password string) (*Neo4jRepository, error) {
	ctx := context.Background()

	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		return nil, fmt.Errorf("failed to create Neo4j driver: %w", err)
	}

	// Verify connectivity
	if err := driver.VerifyConnectivity(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to Neo4j: %w", err)
	}

	return &Neo4jRepository{
		driver: driver,
		ctx:    ctx,
	}, nil
}

// StoreActionGraph stores an action and its dependencies in the graph
func (r *Neo4jRepository) StoreActionGraph(ctx context.Context, action *semantic.SemanticScheduledAction) error {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	// Create action node
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		// Create/update action node
		query := `
			MERGE (a:Action {id: $id})
			SET a.name = $name,
			    a.type = $type,
			    a.description = $description,
			    a.status = $status
			RETURN a
		`
		params := map[string]interface{}{
			"id":          action.Identifier,
			"name":        action.Name,
			"type":        action.Type,
			"description": action.Description,
			"status":      action.ActionStatus,
		}

		if _, err := tx.Run(ctx, query, params); err != nil {
			return nil, err
		}

		// Create dependency relationships
		for _, depID := range action.Requires {
			depQuery := `
				MATCH (a:Action {id: $actionId})
				MERGE (dep:Action {id: $depId})
				MERGE (a)-[:REQUIRES]->(dep)
			`
			depParams := map[string]interface{}{
				"actionId": action.Identifier,
				"depId":    depID,
			}

			if _, err := tx.Run(ctx, depQuery, depParams); err != nil {
				return nil, err
			}
		}

		return nil, nil
	})

	return err
}

// GetDependencies gets direct dependencies (immediate requires)
func (r *Neo4jRepository) GetDependencies(ctx context.Context, actionID string) ([]string, error) {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		query := `
			MATCH (a:Action {id: $id})-[:REQUIRES]->(dep:Action)
			RETURN dep.id as depId
		`
		params := map[string]interface{}{"id": actionID}

		result, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}

		var deps []string
		for result.Next(ctx) {
			record := result.Record()
			if depID, ok := record.Get("depId"); ok {
				deps = append(deps, depID.(string))
			}
		}

		return deps, result.Err()
	})

	if err != nil {
		return nil, err
	}

	return result.([]string), nil
}

// GetAllDependencies gets all transitive dependencies (recursive)
func (r *Neo4jRepository) GetAllDependencies(ctx context.Context, actionID string) ([]string, error) {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		// Use Cypher path expression for transitive closure
		query := `
			MATCH (a:Action {id: $id})-[:REQUIRES*]->(dep:Action)
			RETURN DISTINCT dep.id as depId
		`
		params := map[string]interface{}{"id": actionID}

		result, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}

		var deps []string
		for result.Next(ctx) {
			record := result.Record()
			if depID, ok := record.Get("depId"); ok {
				deps = append(deps, depID.(string))
			}
		}

		return deps, result.Err()
	})

	if err != nil {
		return nil, err
	}

	return result.([]string), nil
}

// GetDependents gets actions that depend on this action
func (r *Neo4jRepository) GetDependents(ctx context.Context, actionID string) ([]string, error) {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		// Reverse direction - find actions that require this one
		query := `
			MATCH (dependent:Action)-[:REQUIRES]->(a:Action {id: $id})
			RETURN dependent.id as dependentId
		`
		params := map[string]interface{}{"id": actionID}

		result, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}

		var dependents []string
		for result.Next(ctx) {
			record := result.Record()
			if depID, ok := record.Get("dependentId"); ok {
				dependents = append(dependents, depID.(string))
			}
		}

		return dependents, result.Err()
	})

	if err != nil {
		return nil, err
	}

	return result.([]string), nil
}

// WouldCreateCycle detects if adding a dependency would create a cycle
func (r *Neo4jRepository) WouldCreateCycle(ctx context.Context, actionID, dependencyID string) (bool, error) {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		// Check if there's a path from dependency back to action
		// If yes, adding actionID->dependencyID would create a cycle
		query := `
			MATCH path = (dep:Action {id: $depId})-[:REQUIRES*]->(a:Action {id: $actionId})
			RETURN count(path) > 0 as hasCycle
		`
		params := map[string]interface{}{
			"actionId": actionID,
			"depId":    dependencyID,
		}

		result, err := tx.Run(ctx, query, params)
		if err != nil {
			return false, err
		}

		if result.Next(ctx) {
			record := result.Record()
			if hasCycle, ok := record.Get("hasCycle"); ok {
				return hasCycle.(bool), nil
			}
		}

		return false, result.Err()
	})

	if err != nil {
		return false, err
	}

	return result.(bool), nil
}

// FindPath finds the shortest path between two actions
func (r *Neo4jRepository) FindPath(ctx context.Context, fromAction, toAction string) ([]string, error) {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		query := `
			MATCH path = shortestPath((from:Action {id: $fromId})-[:REQUIRES*]->(to:Action {id: $toId}))
			RETURN [node in nodes(path) | node.id] as path
		`
		params := map[string]interface{}{
			"fromId": fromAction,
			"toId":   toAction,
		}

		result, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}

		if result.Next(ctx) {
			record := result.Record()
			if pathValue, ok := record.Get("path"); ok {
				pathList := pathValue.([]interface{})
				path := make([]string, len(pathList))
				for i, v := range pathList {
					path[i] = v.(string)
				}
				return path, nil
			}
		}

		return []string{}, result.Err()
	})

	if err != nil {
		return nil, err
	}

	return result.([]string), nil
}

// GetWorkflowActions gets all actions in a workflow
func (r *Neo4jRepository) GetWorkflowActions(ctx context.Context, workflowID string) ([]string, error) {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		query := `
			MATCH (a:Action)-[:PART_OF]->(w:Workflow {id: $workflowId})
			RETURN a.id as actionId
		`
		params := map[string]interface{}{"workflowId": workflowID}

		result, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}

		var actions []string
		for result.Next(ctx) {
			record := result.Record()
			if actionID, ok := record.Get("actionId"); ok {
				actions = append(actions, actionID.(string))
			}
		}

		return actions, result.Err()
	})

	if err != nil {
		return nil, err
	}

	return result.([]string), nil
}

// LinkActionToWorkflow creates workflow -> action relationship
func (r *Neo4jRepository) LinkActionToWorkflow(ctx context.Context, actionID, workflowID string) error {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		query := `
			MERGE (w:Workflow {id: $workflowId})
			MERGE (a:Action {id: $actionId})
			MERGE (a)-[:PART_OF]->(w)
		`
		params := map[string]interface{}{
			"workflowId": workflowID,
			"actionId":   actionID,
		}

		_, err := tx.Run(ctx, query, params)
		return nil, err
	})

	return err
}

// DeleteActionGraph deletes an action from the graph
func (r *Neo4jRepository) DeleteActionGraph(ctx context.Context, actionID string) error {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		query := `
			MATCH (a:Action {id: $id})
			DETACH DELETE a
		`
		params := map[string]interface{}{"id": actionID}

		_, err := tx.Run(ctx, query, params)
		return nil, err
	})

	return err
}

// DeleteWorkflowGraph deletes a workflow from the graph
func (r *Neo4jRepository) DeleteWorkflowGraph(ctx context.Context, workflowID string) error {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		query := `
			MATCH (w:Workflow {id: $id})
			DETACH DELETE w
		`
		params := map[string]interface{}{"id": workflowID}

		_, err := tx.Run(ctx, query, params)
		return nil, err
	})

	return err
}

// Close closes the Neo4j driver
func (r *Neo4jRepository) Close() error {
	return r.driver.Close(r.ctx)
}
