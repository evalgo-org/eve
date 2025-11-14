package runtime

import (
	"fmt"
	"regexp"
	"strings"
)

// Variable reference pattern: ${variable-name} or ${action-id.field.path}
var variablePattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// VariableResolver is an interface for resolving variable references
type VariableResolver interface {
	// Resolve resolves a variable reference to its value
	// Example: "action-id.result.contentUrl" or "PARAMETER_NAME"
	Resolve(reference string) (string, error)
}

// SubstituteVariables walks through an action's AllFields and substitutes all ${...} references
// Returns a new action with substituted values (does not modify original)
func SubstituteVariables(action *RuntimeAction, resolver VariableResolver) (*RuntimeAction, error) {
	if action == nil {
		return nil, fmt.Errorf("action is nil")
	}
	if resolver == nil {
		return nil, fmt.Errorf("resolver is nil")
	}

	// Deep copy to avoid modifying original
	substituted := action.DeepCopy()
	if substituted == nil {
		return nil, fmt.Errorf("failed to deep copy action")
	}

	// Walk through AllFields and substitute variables
	result, err := WalkJSON(substituted.AllFields, func(value string) (string, error) {
		return substituteString(value, resolver)
	})

	if err != nil {
		return nil, fmt.Errorf("variable substitution failed: %w", err)
	}

	// Update AllFields with substituted values
	if resultMap, ok := result.(map[string]interface{}); ok {
		substituted.AllFields = resultMap
		return substituted, nil
	}

	return nil, fmt.Errorf("WalkJSON did not return a map")
}

// substituteString substitutes all ${...} references in a string
func substituteString(value string, resolver VariableResolver) (string, error) {
	// Quick check - if no ${, nothing to do
	if !strings.Contains(value, "${") {
		return value, nil
	}

	// Find all ${...} patterns
	matches := variablePattern.FindAllStringSubmatch(value, -1)
	if len(matches) == 0 {
		return value, nil
	}

	result := value
	for _, match := range matches {
		placeholder := match[0] // "${foo.bar}"
		reference := match[1]   // "foo.bar"

		// Resolve the variable
		resolvedValue, err := resolver.Resolve(reference)
		if err != nil {
			return "", fmt.Errorf("failed to resolve %s: %w", placeholder, err)
		}

		// Replace in string
		result = strings.ReplaceAll(result, placeholder, resolvedValue)
	}

	return result, nil
}

// MapVariableResolver implements VariableResolver using a simple map
// Useful for testing and simple parameter substitution
type MapVariableResolver struct {
	Variables map[string]string
}

// Resolve implements VariableResolver
func (m *MapVariableResolver) Resolve(reference string) (string, error) {
	value, ok := m.Variables[reference]
	if !ok {
		return "", fmt.Errorf("variable not found: %s", reference)
	}
	return value, nil
}

// ChainVariableResolver chains multiple resolvers
// Tries each resolver in order until one succeeds
type ChainVariableResolver struct {
	Resolvers []VariableResolver
}

// Resolve implements VariableResolver
func (c *ChainVariableResolver) Resolve(reference string) (string, error) {
	var lastErr error

	for _, resolver := range c.Resolvers {
		value, err := resolver.Resolve(reference)
		if err == nil {
			return value, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return "", fmt.Errorf("no resolver could resolve %s: %w", reference, lastErr)
	}

	return "", fmt.Errorf("no resolvers configured")
}

// ActionResultResolver resolves references to action results
// Example: "action-id.result.contentUrl"
type ActionResultResolver struct {
	// GetAction retrieves an action by ID
	GetAction func(actionID string) (*RuntimeAction, error)
}

// Resolve implements VariableResolver
func (a *ActionResultResolver) Resolve(reference string) (string, error) {
	if a.GetAction == nil {
		return "", fmt.Errorf("GetAction function not set")
	}

	parts := strings.Split(reference, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid action reference format: %s (expected action-id.field.path)", reference)
	}

	actionID := parts[0]
	fieldPath := strings.Join(parts[1:], ".")

	// Load referenced action
	action, err := a.GetAction(actionID)
	if err != nil {
		return "", fmt.Errorf("action not found: %s: %w", actionID, err)
	}

	// Check that action has completed
	if action.ActionStatus != "CompletedActionStatus" {
		return "", fmt.Errorf("action %s has not completed yet (status: %s)", actionID, action.ActionStatus)
	}

	// Get field value
	value, err := action.GetField(fieldPath)
	if err != nil {
		return "", fmt.Errorf("field not found in action %s: %s: %w", actionID, fieldPath, err)
	}

	// Convert to string
	return fmt.Sprintf("%v", value), nil
}

// ExtractVariableReferences extracts all variable references from a string
// Returns the references without ${} wrappers
func ExtractVariableReferences(value string) []string {
	matches := variablePattern.FindAllStringSubmatch(value, -1)
	if len(matches) == 0 {
		return nil
	}

	references := make([]string, len(matches))
	for i, match := range matches {
		references[i] = match[1] // The reference without ${}
	}

	return references
}

// HasVariableReferences checks if a string contains any ${...} references
func HasVariableReferences(value string) bool {
	return strings.Contains(value, "${")
}
