package workflow

import (
	"fmt"
	"os"
	"time"

	"eve.evalgo.org/semantic"
	"github.com/google/uuid"
)

// Expander converts workflow definitions into executable semantic actions

// ExpandToActions converts a WorkflowDefinition into a list of SemanticScheduledActions
func ExpandToActions(workflow *semantic.WorkflowDefinition) ([]*semantic.SemanticScheduledAction, error) {
	// Generate unique instance ID for this workflow run
	instanceID := uuid.New().String()
	fmt.Fprintf(os.Stderr, "DEBUG expander: Generated workflow instance ID: %s\n", instanceID)

	var actions []*semantic.SemanticScheduledAction

	for _, workflowAction := range workflow.Actions {
		switch workflowAction.Type {
		case "action":
			// Single action → single action (with merged deps)
			action, err := mergeActionDependencies(workflowAction.Action, workflowAction.DependsOn, instanceID)
			if err != nil {
				return nil, fmt.Errorf("failed to process action '%s': %w", workflowAction.Action.Identifier, err)
			}
			actions = append(actions, action)

		case "loop":
			// Loop (ItemList) → multiple actions
			loopActions, err := expandLoop(workflowAction.Loop, workflowAction.DependsOn, instanceID)
			if err != nil {
				return nil, fmt.Errorf("failed to expand loop '%s': %w", workflowAction.Loop.Identifier, err)
			}
			actions = append(actions, loopActions...)

		default:
			return nil, fmt.Errorf("unsupported action type: %s", workflowAction.Type)
		}
	}

	return actions, nil
}

// prefixIdentifier prefixes an identifier with the workflow instance ID
func prefixIdentifier(instanceID, identifier string) string {
	if identifier == "" {
		return ""
	}
	return fmt.Sprintf("%s--%s", instanceID, identifier)
}

// mergeActionDependencies merges workflow-level dependencies with action's own dependencies
func mergeActionDependencies(action *semantic.SemanticScheduledAction, additionalDeps []string, instanceID string) (*semantic.SemanticScheduledAction, error) {
	if action == nil {
		return nil, fmt.Errorf("action is nil")
	}

	// Merge dependencies
	deps := make([]string, 0)
	deps = append(deps, action.Requires...)
	deps = append(deps, additionalDeps...)

	// Remove duplicates
	seen := make(map[string]bool)
	uniqueDeps := make([]string, 0)
	for _, dep := range deps {
		if !seen[dep] {
			seen[dep] = true
			uniqueDeps = append(uniqueDeps, dep)
		}
	}

	// DEBUG: Check before normalization
	fmt.Fprintf(os.Stderr, "DEBUG expander: Before norm: ID='%s', Identifier='%s', Name='%s'\n", action.ID, action.Identifier, action.Name)

	// Normalize @id to identifier for CouchDB storage
	if action.ID != "" && action.Identifier == "" {
		action.Identifier = action.ID
	}

	// Prefix action identifiers with workflow instance ID
	if action.Identifier != "" {
		originalIdentifier := action.Identifier
		action.Identifier = prefixIdentifier(instanceID, originalIdentifier)
		fmt.Fprintf(os.Stderr, "DEBUG expander: Prefixed identifier: '%s' → '%s'\n", originalIdentifier, action.Identifier)
	}
	if action.ID != "" {
		action.ID = prefixIdentifier(instanceID, action.ID)
	}

	// Prefix all dependencies with workflow instance ID
	prefixedDeps := make([]string, 0, len(uniqueDeps))
	for _, dep := range uniqueDeps {
		prefixedDep := prefixIdentifier(instanceID, dep)
		prefixedDeps = append(prefixedDeps, prefixedDep)
		fmt.Fprintf(os.Stderr, "DEBUG expander: Prefixed dependency: '%s' → '%s'\n", dep, prefixedDep)
	}
	action.Requires = prefixedDeps

	// DEBUG: Check after normalization and prefixing
	fmt.Fprintf(os.Stderr, "DEBUG expander: After prefix: ID='%s', Identifier='%s', Name='%s'\n", action.ID, action.Identifier, action.Name)

	if action.Identifier == "" {
		fmt.Fprintf(os.Stderr, "ERROR: Action has no identifier after merge! ID=%s, Name=%s\n", action.ID, action.Name)
	}

	// Extract controlMetadata is already done by JSON unmarshal (controlMetadata -> Meta)
	// Extract target EntryPoint fields into Meta for routing ONLY if Meta.URL is not already set
	// This handles JSON-LD workflows that use "target": {"@type": "EntryPoint", "url": "...", "httpMethod": "..."}
	extractTargetToMeta(action)

	// Ensure Properties map exists (for semantic parameters)
	if action.Properties == nil {
		action.Properties = make(map[string]interface{})
	}

	// Initialize Meta if not present
	if action.Meta == nil {
		action.Meta = &semantic.ActionMeta{}
	}

	// Set default meta properties if not present
	// NOTE: Don't set default for Enabled - respect explicit false values
	// Only set Enabled=true if Meta was just created (meaning no controlMetadata in JSON)
	wasJustCreated := action.Meta.RetryBackoff == "" && action.Meta.URL == ""

	if !action.Meta.Singleton {
		action.Meta.Singleton = true
	}
	if action.Meta.RetryCount == 0 {
		action.Meta.RetryCount = 0
	}
	if action.Meta.RetryBackoff == "" {
		action.Meta.RetryBackoff = "exponential"
	}
	// Only default to enabled=true if Meta was freshly created (no controlMetadata provided)
	if wasJustCreated && !action.Meta.Enabled {
		action.Meta.Enabled = true
	}

	// Set timestamps if not present
	now := time.Now()
	if action.Created.IsZero() {
		action.Created = now
	}
	action.Modified = now

	return action, nil
}

// expandLoop expands an ItemList loop into multiple semantic actions
func expandLoop(loop *semantic.SemanticItemList, additionalDeps []string, instanceID string) ([]*semantic.SemanticScheduledAction, error) {
	if loop == nil {
		return nil, fmt.Errorf("loop is nil")
	}

	// Check max iterations safety limit
	maxIter := loop.MaxIterations
	if maxIter == 0 {
		maxIter = 1000 // default safety limit
	}

	if len(loop.ItemListElement) > maxIter {
		return nil, fmt.Errorf("loop exceeds max iterations limit (%d > %d)", len(loop.ItemListElement), maxIter)
	}

	actions := make([]*semantic.SemanticScheduledAction, 0, len(loop.ItemListElement))

	// Merge loop-level dependencies with additional deps
	loopDeps := make([]string, 0)
	loopDeps = append(loopDeps, loop.DependsOn...)
	loopDeps = append(loopDeps, additionalDeps...)

	for _, listItem := range loop.ItemListElement {
		if listItem.Item == nil {
			return nil, fmt.Errorf("list item at position %d has no action", listItem.Position)
		}

		action, err := mergeActionDependencies(listItem.Item, loopDeps, instanceID)
		if err != nil {
			return nil, fmt.Errorf("failed to process loop item at position %d: %w", listItem.Position, err)
		}

		actions = append(actions, action)
	}

	return actions, nil
}

// extractTargetToMeta extracts "target" EntryPoint fields into Meta for routing
// This enables URL-based routing while keeping semantic properties separate
// NOTE: controlMetadata.url takes precedence - only set Meta.URL from target if not already set
func extractTargetToMeta(action *semantic.SemanticScheduledAction) {
	// Initialize Meta if not present
	if action.Meta == nil {
		action.Meta = &semantic.ActionMeta{}
	}

	// Check if action.Target exists (Schema.org EntryPoint or other target types)
	if action.Target != nil {
		// Type-assert to map to handle different target types (EntryPoint, InfisicalProject, DataCatalog, etc.)
		if targetMap, ok := action.Target.(map[string]interface{}); ok {
			// Extract url from target ONLY if Meta.URL is not already set (from controlMetadata)
			if action.Meta.URL == "" {
				if url, hasURL := targetMap["url"].(string); hasURL && url != "" {
					action.Meta.URL = url
				}
			}
			// Extract httpMethod from target if it exists
			if httpMethod, hasMethod := targetMap["httpMethod"].(string); hasMethod && httpMethod != "" {
				action.Meta.HTTPMethod = httpMethod
			}
			// Note: Headers are not part of ActionMeta, keep them in target if needed
		}
	}
}
