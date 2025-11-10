package semantic

import (
	"encoding/json"
	"fmt"
)

// ============================================================================
// Infisical Action Types
// ============================================================================
// Note: Legacy InfisicalRetrieveAction struct has been removed.
// Use SemanticAction with NewSemanticInfisicalRetrieveAction instead.

// InfisicalProject represents an Infisical project with environment configuration.
// DEPRECATED: Use EntryPoint with additionalProperty instead for Schema.org compliance.
// This type is kept for reference only.
type InfisicalProject struct {
	Type               string                 `json:"@type"`      // "Project"
	Identifier         string                 `json:"identifier"` // project_id
	Name               string                 `json:"name,omitempty"`
	Environment        string                 `json:"environment"`                  // "dev", "prod", "staging", etc.
	Url                string                 `json:"url"`                          // Infisical instance URL (e.g. "https://app.infisical.com")
	SecretPath         string                 `json:"secretPath,omitempty"`         // default "/"
	IncludeImports     bool                   `json:"includeImports,omitempty"`     // Include imported secrets
	AdditionalProperty map[string]interface{} `json:"additionalProperty,omitempty"` // For auth credentials
}

// NewInfisicalProject creates a new Infisical project reference.
// DEPRECATED: Use EntryPoint with additionalProperty for Schema.org compliance.
func NewInfisicalProject(projectID, environment, url string) *InfisicalProject {
	return &InfisicalProject{
		Type:           "Project",
		Identifier:     projectID,
		Environment:    environment,
		Url:            url,
		SecretPath:     "/",
		IncludeImports: true,
	}
}

// NewSemanticInfisicalRetrieveAction creates an InfisicalRetrieveAction using SemanticAction
// target should be an EntryPoint with additionalProperty containing projectId, environment, etc.
func NewSemanticInfisicalRetrieveAction(id, name string, object *PropertyValue, target interface{}) *SemanticAction {
	action := &SemanticAction{
		Context:      "https://schema.org",
		Type:         "RetrieveAction",
		Identifier:   id,
		Name:         name,
		ActionStatus: "PotentialActionStatus",
		Properties:   make(map[string]interface{}),
	}

	if object != nil {
		action.Properties["object"] = object
	}
	if target != nil {
		action.Properties["target"] = target
	}

	return action
}

// GetPropertyValueFromAction extracts PropertyValue from SemanticAction properties
func GetPropertyValueFromAction(action *SemanticAction, propertyKey string) (*PropertyValue, error) {
	if action == nil || action.Properties == nil {
		return nil, fmt.Errorf("action or properties is nil")
	}

	prop, ok := action.Properties[propertyKey]
	if !ok {
		return nil, nil // PropertyValue is optional
	}

	switch v := prop.(type) {
	case *PropertyValue:
		return v, nil
	case PropertyValue:
		return &v, nil
	case map[string]interface{}:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal PropertyValue: %w", err)
		}
		var pv PropertyValue
		if err := json.Unmarshal(data, &pv); err != nil {
			return nil, fmt.Errorf("failed to unmarshal PropertyValue: %w", err)
		}
		return &pv, nil
	default:
		return nil, fmt.Errorf("unexpected %s type: %T", propertyKey, prop)
	}
}

// GetInfisicalTargetFromAction extracts Infisical target configuration from SemanticAction properties
// Returns: url, projectID, environment, secretPath, includeImports, error
func GetInfisicalTargetFromAction(action *SemanticAction) (string, string, string, string, bool, error) {
	if action == nil {
		return "", "", "", "", false, fmt.Errorf("action is nil")
	}

	// Target is in the action.Target field, not in Properties
	target := action.Target
	if target == nil {
		return "", "", "", "", false, fmt.Errorf("no target found in action")
	}

	// Handle EntryPoint format (Schema.org compliant)
	if targetMap, ok := target.(map[string]interface{}); ok {
		var url, projectID, environment, secretPath string
		var includeImports bool

		// Extract URL from EntryPoint
		if urlStr, ok := targetMap["url"].(string); ok {
			url = urlStr
		}

		// Extract configuration from additionalProperty
		if additionalProp, ok := targetMap["additionalProperty"].(map[string]interface{}); ok {
			if pid, ok := additionalProp["projectId"].(string); ok {
				projectID = pid
			}
			if env, ok := additionalProp["environment"].(string); ok {
				environment = env
			}
			if sp, ok := additionalProp["secretPath"].(string); ok {
				secretPath = sp
			}
			if ii, ok := additionalProp["includeImports"].(bool); ok {
				includeImports = ii
			}
		}

		// Set defaults
		if secretPath == "" {
			secretPath = "/"
		}

		if projectID == "" {
			return "", "", "", "", false, fmt.Errorf("projectId is required in target.additionalProperty")
		}
		if environment == "" {
			return "", "", "", "", false, fmt.Errorf("environment is required in target.additionalProperty")
		}
		if url == "" {
			return "", "", "", "", false, fmt.Errorf("service URL is required in target.url")
		}

		return url, projectID, environment, secretPath, includeImports, nil
	}

	// Handle legacy InfisicalProject format (backward compatibility)
	if project, ok := target.(*InfisicalProject); ok {
		secretPath := project.SecretPath
		if secretPath == "" {
			secretPath = "/"
		}
		return project.Url, project.Identifier, project.Environment, secretPath, project.IncludeImports, nil
	}

	return "", "", "", "", false, fmt.Errorf("unexpected target type: %T", target)
}

// GetPropertyValuesFromAction extracts array of PropertyValues from SemanticAction properties
// Used to extract result secrets
func GetPropertyValuesFromAction(action *SemanticAction, propertyKey string) ([]*PropertyValue, error) {
	if action == nil || action.Properties == nil {
		return nil, fmt.Errorf("action or properties is nil")
	}

	prop, ok := action.Properties[propertyKey]
	if !ok {
		return nil, nil // Results are optional
	}

	switch v := prop.(type) {
	case []*PropertyValue:
		return v, nil
	case []PropertyValue:
		result := make([]*PropertyValue, len(v))
		for i := range v {
			result[i] = &v[i]
		}
		return result, nil
	case []interface{}:
		result := make([]*PropertyValue, len(v))
		for i, item := range v {
			if pvMap, ok := item.(map[string]interface{}); ok {
				data, err := json.Marshal(pvMap)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal PropertyValue at index %d: %w", i, err)
				}
				var pv PropertyValue
				if err := json.Unmarshal(data, &pv); err != nil {
					return nil, fmt.Errorf("failed to unmarshal PropertyValue at index %d: %w", i, err)
				}
				result[i] = &pv
			} else {
				return nil, fmt.Errorf("unexpected item type at index %d: %T", i, item)
			}
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unexpected %s type: %T", propertyKey, prop)
	}
}
