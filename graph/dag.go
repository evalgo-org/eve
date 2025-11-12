// Package graph provides directed acyclic graph (DAG) utilities for dependency management.
// This package offers cycle detection and topological sorting for workflow dependencies.
package graph

import (
	"fmt"

	"eve.evalgo.org/semantic"
)

// ActionRepository defines the interface for retrieving actions from storage
type ActionRepository interface {
	GetAction(id string) (*semantic.SemanticScheduledAction, error)
	GetAllDependencies(id string) ([]string, error)
	WouldCreateCycle(actionID, dependencyID string) (bool, error)
}

// ValidateDAG checks for circular dependencies in action graph
// Uses repository's native cycle detection if available (e.g., Neo4j)
func ValidateDAG(repo ActionRepository, action *semantic.SemanticScheduledAction) error {
	if len(action.Requires) == 0 {
		return nil // No dependencies, no cycles possible
	}

	// Try using repository's native cycle detection (e.g., Neo4j)
	if repo != nil {
		for _, depID := range action.Requires {
			hasCycle, err := repo.WouldCreateCycle(action.Identifier, depID)
			if err != nil {
				// Repository doesn't support cycle detection - fall back to manual check
				return checkCycleManual(repo, action)
			}
			if hasCycle {
				return fmt.Errorf("circular dependency detected: adding dependency %s to action %s would create a cycle", depID, action.Identifier)
			}
		}
		return nil
	}

	// Fallback to manual check if repository is nil
	return checkCycleManual(repo, action)
}

// checkCycleManual is a fallback for when native cycle detection is not available
// Uses depth-first search with recursion stack detection
func checkCycleManual(repo ActionRepository, action *semantic.SemanticScheduledAction) error {
	visited := make(map[string]bool)
	recursionStack := make(map[string]bool)
	tempActions := map[string]*semantic.SemanticScheduledAction{action.Identifier: action}

	return checkCycleRecursive(repo, action.Identifier, visited, recursionStack, tempActions)
}

func checkCycleRecursive(repo ActionRepository, actionID string, visited, recursionStack map[string]bool, tempActions map[string]*semantic.SemanticScheduledAction) error {
	visited[actionID] = true
	recursionStack[actionID] = true

	// Try to get action from temporary map first, then repository
	var action *semantic.SemanticScheduledAction
	if a, ok := tempActions[actionID]; ok {
		action = a
	} else if repo != nil {
		var err error
		action, err = repo.GetAction(actionID)
		if err != nil {
			// If action doesn't exist yet, it can't have cycles
			return nil
		}
	} else {
		return nil
	}

	for _, depID := range action.Requires {
		if !visited[depID] {
			if err := checkCycleRecursive(repo, depID, visited, recursionStack, tempActions); err != nil {
				return err
			}
		} else if recursionStack[depID] {
			return fmt.Errorf("circular dependency detected: %s -> %s", actionID, depID)
		}
	}

	recursionStack[actionID] = false
	return nil
}

// GetExecutionOrder returns actions in topologically sorted order using Kahn's algorithm
// Actions with no dependencies come first, then actions depending on them, etc.
func GetExecutionOrder(actions []*semantic.SemanticScheduledAction) ([]*semantic.SemanticScheduledAction, error) {
	// Build adjacency list and in-degree map
	graph := make(map[string][]*semantic.SemanticScheduledAction)
	inDegree := make(map[string]int)
	actionMap := make(map[string]*semantic.SemanticScheduledAction)

	for _, action := range actions {
		actionMap[action.Identifier] = action
		inDegree[action.Identifier] = 0
	}

	for _, action := range actions {
		for _, depID := range action.Requires {
			graph[depID] = append(graph[depID], action)
			inDegree[action.Identifier]++
		}
	}

	// Kahn's algorithm for topological sort
	var queue []*semantic.SemanticScheduledAction
	for _, action := range actions {
		if inDegree[action.Identifier] == 0 {
			queue = append(queue, action)
		}
	}

	var result []*semantic.SemanticScheduledAction
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		for _, dependent := range graph[current.Identifier] {
			inDegree[dependent.Identifier]--
			if inDegree[dependent.Identifier] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	// Check if all actions were processed (no cycles)
	if len(result) != len(actions) {
		return nil, fmt.Errorf("circular dependency detected in action graph")
	}

	return result, nil
}

// CheckDependencies checks if all dependencies for an action have completed successfully
// Uses repository to retrieve dependency statuses
func CheckDependencies(repo ActionRepository, action *semantic.SemanticScheduledAction) (bool, error) {
	if len(action.Requires) == 0 {
		return true, nil // No dependencies, ready to run
	}

	// Get all transitive dependencies if repository supports it (e.g., Neo4j)
	var allDeps []string
	if repo != nil {
		deps, err := repo.GetAllDependencies(action.Identifier)
		if err == nil {
			allDeps = deps
		} else {
			// Fallback to direct dependencies
			allDeps = action.Requires
		}
	} else {
		allDeps = action.Requires
	}

	// Check each dependency's status
	for _, depID := range allDeps {
		depAction, err := repo.GetAction(depID)
		if err != nil {
			return false, fmt.Errorf("dependency action %s not found: %w", depID, err)
		}

		// Check if dependency has completed successfully
		if depAction.ActionStatus != "CompletedActionStatus" {
			return false, nil // Dependency not yet successful
		}

		// Check if dependency ran in current cycle (optional freshness check)
		if action.StartTime != nil && depAction.EndTime != nil {
			// Both have run before - check if dependency is fresh
			if depAction.EndTime.Before(*action.StartTime) {
				// Dependency is stale, needs to run again first
				return false, nil
			}
		}
	}

	return true, nil // All dependencies met
}
