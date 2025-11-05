// Package executor provides semantic action execution with registry-based service discovery.
// This is the core executor for all EVE-based semantic actions.
package executor

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"eve.evalgo.org/semantic"
)

// Executor defines the interface for semantic action executors
type Executor interface {
	// Execute runs a semantic action and returns output, error
	Execute(action *semantic.SemanticScheduledAction) (string, error)

	// CanHandle returns true if this executor can handle the given action
	CanHandle(action *semantic.SemanticScheduledAction) bool
}

// Registry manages semantic executors with priority-based dispatch
type Registry struct {
	executors []Executor
}

// NewRegistry creates a new executor registry with default executors
func NewRegistry() *Registry {
	registry := &Registry{}

	// Create executors with registry reference for delegation
	scheduledExecutor := &ScheduledActionExecutor{Registry: registry}

	registry.executors = []Executor{
		// URL-based routing (highest priority) - routes any action to /v1/api/semantic/action endpoints
		// This handles ALL semantic services (sparqlservice, s3service, infisicalservice, basexservice, templateservice, workflowstorageservice)
		&URLBasedExecutor{},
		// ScheduledAction wrapper and HTTP-property actions
		scheduledExecutor,
		// Command-based actions (fallback for legacy command property)
		&CommandExecutor{},
	}

	return registry
}

// Register adds a new executor to the registry (prepended for priority)
func (r *Registry) Register(executor Executor) {
	r.executors = append([]Executor{executor}, r.executors...)
}

// Execute finds appropriate executor and runs the action
func (r *Registry) Execute(action *semantic.SemanticScheduledAction) (string, error) {
	// Expand environment variables before execution
	expandedAction, err := expandEnvVars(action)
	if err != nil {
		return "", fmt.Errorf("failed to expand environment variables: %w", err)
	}

	for _, executor := range r.executors {
		if executor.CanHandle(expandedAction) {
			return executor.Execute(expandedAction)
		}
	}

	return "", fmt.Errorf("no executor available for action type: %s", expandedAction.Type)
}

// expandEnvVars expands ${ENV:VARIABLE} placeholders in a semantic action
func expandEnvVars(action *semantic.SemanticScheduledAction) (*semantic.SemanticScheduledAction, error) {
	// Serialize to JSON
	data, err := json.Marshal(action)
	if err != nil {
		return nil, err
	}

	// Expand environment variables using regex
	envVarPattern := regexp.MustCompile(`\$\{ENV:([A-Z_][A-Z0-9_]*)\}`)
	expanded := envVarPattern.ReplaceAllStringFunc(string(data), func(match string) string {
		// Extract variable name
		varName := envVarPattern.FindStringSubmatch(match)[1]
		// Get value from environment
		value := os.Getenv(varName)
		if value == "" {
			// Keep placeholder if variable not set
			return match
		}
		return value
	})

	// Unmarshal back to action
	var expandedAction semantic.SemanticScheduledAction
	if err := json.Unmarshal([]byte(expanded), &expandedAction); err != nil {
		return nil, err
	}

	return &expandedAction, nil
}

// CommandExecutor executes actions with command property (backward compatibility)
type CommandExecutor struct{}

func (e *CommandExecutor) CanHandle(action *semantic.SemanticScheduledAction) bool {
	if action.Properties == nil {
		return false
	}

	cmd, ok := action.Properties["command"].(string)
	return ok && cmd != ""
}

func (e *CommandExecutor) Execute(action *semantic.SemanticScheduledAction) (string, error) {
	cmdStr, ok := action.Properties["command"].(string)
	if !ok {
		return "", fmt.Errorf("command property is not a string")
	}

	// Execute command
	cmd := exec.Command("sh", "-c", cmdStr)
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("command failed: %w", err)
	}

	return string(output), nil
}

// hasHTTPProperties checks if action has HTTP-related properties
func hasHTTPProperties(action *semantic.SemanticScheduledAction) bool {
	if action.Properties == nil {
		return false
	}

	// Check for http-related properties
	_, hasURL := action.Properties["url"]
	_, hasMethod := action.Properties["httpMethod"]

	return hasURL || hasMethod
}

// ScheduledActionExecutor handles ScheduledAction wrapper type and HTTP-property actions
// This executor is for backward compatibility with non-semantic action definitions.
type ScheduledActionExecutor struct {
	FetcherPath string
	Registry    *Registry // Reference to registry for delegating embedded actions
}

func (e *ScheduledActionExecutor) CanHandle(action *semantic.SemanticScheduledAction) bool {
	// Handle ScheduledAction type
	if action.Type == "ScheduledAction" {
		return true
	}

	// Also handle actions with HTTP target properties
	if action.Object != nil && action.Object.CodeRepository != "" {
		return true
	}

	return hasHTTPProperties(action)
}

func (e *ScheduledActionExecutor) Execute(action *semantic.SemanticScheduledAction) (string, error) {
	// Check if this is a wrapper ScheduledAction with embedded action in object.text
	if action.Type == "ScheduledAction" && action.Object != nil && action.Object.Text != "" {
		// Extract the @type from embedded JSON to determine routing
		var typeDetector struct {
			Type string `json:"@type"`
		}
		if err := json.Unmarshal([]byte(action.Object.Text), &typeDetector); err == nil {
			// For embedded actions, try to parse and delegate to registry
			var embeddedAction semantic.SemanticScheduledAction
			if err := json.Unmarshal([]byte(action.Object.Text), &embeddedAction); err == nil {
				// Delegate to registry which will use URLBasedExecutor if it has semantic endpoint
				if e.Registry != nil {
					return e.Registry.Execute(&embeddedAction)
				}
			}
		}
		// If delegation failed, fall through to fetcher
	}

	// Serialize action to JSON-LD
	jsonldData, err := json.Marshal(action)
	if err != nil {
		return "", fmt.Errorf("failed to serialize action to JSON-LD: %w", err)
	}

	// Get fetcher path
	fetcherPath := e.FetcherPath
	if fetcherPath == "" {
		fetcherPath = "fetcher"
	}

	// Execute: fetcher semantic --inline '<jsonld>'
	cmd := exec.Command(fetcherPath, "semantic", "--inline", string(jsonldData))
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("fetcher execution failed: %w", err)
	}

	return string(output), nil
}

// Helper functions for semantic introspection

// IntrospectAction analyzes what an action will do (semantic introspection)
func IntrospectAction(action *semantic.SemanticScheduledAction) map[string]interface{} {
	result := map[string]interface{}{
		"id":   action.Identifier,
		"name": action.Name,
		"type": action.Type,
	}

	// Extract semantic information
	if action.Object != nil {
		result["object"] = action.Object
	}

	if action.Target != nil {
		result["target"] = action.Target
	}

	if action.Properties != nil {
		if cmd, ok := action.Properties["command"].(string); ok {
			result["command"] = cmd
		}
	}

	if action.Schedule != nil {
		result["schedule"] = action.Schedule.RepeatFrequency
	}

	result["dependencies"] = action.Requires
	result["status"] = action.ActionStatus

	return result
}

// QueryActionsByType returns all actions of a specific semantic type
func QueryActionsByType(actions []*semantic.SemanticScheduledAction, actionType string) []*semantic.SemanticScheduledAction {
	var result []*semantic.SemanticScheduledAction

	for _, action := range actions {
		if action.Type == actionType {
			result = append(result, action)
		}
	}

	return result
}

// QueryActionsByURL returns all actions that interact with a specific URL
func QueryActionsByURL(actions []*semantic.SemanticScheduledAction, urlPattern string) []*semantic.SemanticScheduledAction {
	var result []*semantic.SemanticScheduledAction

	for _, action := range actions {
		// Check target URL
		if action.Target != nil {
			if entryPoint, ok := action.Target.(*semantic.EntryPoint); ok {
				if strings.Contains(entryPoint.URL, urlPattern) {
					result = append(result, action)
					continue
				}
			}
		}

		// Check properties for URL
		if action.Properties != nil {
			if url, ok := action.Properties["url"].(string); ok {
				if strings.Contains(url, urlPattern) {
					result = append(result, action)
				}
			}
		}
	}

	return result
}

// ExportAsJSONLD exports an action as JSON-LD
func ExportAsJSONLD(action *semantic.SemanticScheduledAction) ([]byte, error) {
	return json.MarshalIndent(action, "", "  ")
}
