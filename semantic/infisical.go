package semantic

import (
	"context"
	"encoding/json"
	"fmt"

	infisical "github.com/infisical/go-sdk"
)

// InfisicalRetrieveAction represents a Schema.org RetrieveAction for fetching secrets from Infisical.
// It retrieves secrets from a specific Infisical project and environment.
type InfisicalRetrieveAction struct {
	Context      string           `json:"@context,omitempty"`
	Type         string           `json:"@type"` // "RetrieveAction"
	Identifier   string           `json:"identifier"`
	Name         string           `json:"name,omitempty"`
	Description  string           `json:"description,omitempty"`
	Object       *PropertyValue   `json:"object,omitempty"` // What secret(s) to retrieve (optional - path or name)
	Target       interface{}      `json:"target"`           // EntryPoint for service (Schema.org compliant) or InfisicalProject (backward compat)
	Result       []*PropertyValue `json:"result,omitempty"`
	ActionStatus string           `json:"actionStatus,omitempty"`
	StartTime    string           `json:"startTime,omitempty"`
	EndTime      string           `json:"endTime,omitempty"`
	Error        *PropertyValue   `json:"error,omitempty"`
}

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

// NewInfisicalRetrieveAction creates a new Infisical secret retrieval action.
// DEPRECATED: Use EntryPoint with additionalProperty for Schema.org compliance.
func NewInfisicalRetrieveAction(identifier, name string, project *InfisicalProject) *InfisicalRetrieveAction {
	return &InfisicalRetrieveAction{
		Context:    "https://schema.org",
		Type:       "RetrieveAction",
		Identifier: identifier,
		Name:       name,
		Target:     project,
	}
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

// RetrieveSecrets executes the Infisical secret retrieval using EVE's security/infisical integration.
// It authenticates with Infisical and fetches all secrets from the specified project/environment.
//
// Parameters:
//   - clientID: Infisical client ID for authentication
//   - clientSecret: Infisical client secret for authentication
//
// Returns an error if authentication or retrieval fails.
func (a *InfisicalRetrieveAction) RetrieveSecrets(clientID, clientSecret string) error {
	if a.Target == nil {
		return fmt.Errorf("target is required")
	}

	// Extract configuration from target (EntryPoint with additionalProperty - Schema.org compliant)
	var projectID, environment, url, secretPath string
	var includeImports bool

	if targetMap, ok := a.Target.(map[string]interface{}); ok {
		// EntryPoint format (Schema.org compliant)
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
	}

	if projectID == "" {
		return fmt.Errorf("projectId is required in target.additionalProperty")
	}
	if environment == "" {
		return fmt.Errorf("environment is required in target.additionalProperty")
	}
	if url == "" {
		return fmt.Errorf("service URL is required in target.url")
	}

	// Set secret path default
	if secretPath == "" {
		secretPath = "/"
	}

	// Create Infisical client
	client := infisical.NewInfisicalClient(context.Background(), infisical.Config{
		SiteUrl:          url,
		AutoTokenRefresh: false,
	})

	// Authenticate
	_, err := client.Auth().UniversalAuthLogin(clientID, clientSecret)
	if err != nil {
		a.ActionStatus = "FailedActionStatus"
		a.Error = &PropertyValue{
			Type:  "PropertyValue",
			Name:  "AuthenticationError",
			Value: err.Error(),
		}
		return fmt.Errorf("Infisical authentication failed: %w", err)
	}

	// Retrieve secrets
	secrets, err := client.Secrets().List(infisical.ListSecretsOptions{
		AttachToProcessEnv: false,
		Environment:        environment,
		ProjectID:          projectID,
		SecretPath:         secretPath,
		IncludeImports:     includeImports,
	})
	if err != nil {
		a.ActionStatus = "FailedActionStatus"
		a.Error = &PropertyValue{
			Type:  "PropertyValue",
			Name:  "RetrievalError",
			Value: err.Error(),
		}
		return fmt.Errorf("failed to retrieve secrets: %w", err)
	}

	// Convert to PropertyValue array
	a.Result = make([]*PropertyValue, len(secrets))
	for i, secret := range secrets {
		a.Result[i] = &PropertyValue{
			Type:  "PropertyValue",
			Name:  secret.SecretKey,
			Value: secret.SecretValue,
		}
	}

	a.ActionStatus = "CompletedActionStatus"
	return nil
}

// GetSecretByName retrieves a specific secret by name from the result.
func (a *InfisicalRetrieveAction) GetSecretByName(name string) (string, error) {
	if a.Result == nil {
		return "", fmt.Errorf("no secrets retrieved")
	}

	for _, secret := range a.Result {
		if secret.Name == name {
			return secret.Value, nil
		}
	}

	return "", fmt.Errorf("secret %s not found", name)
}

// GetSecretsAsMap returns all secrets as a map for easy lookup.
func (a *InfisicalRetrieveAction) GetSecretsAsMap() map[string]string {
	secretMap := make(map[string]string)
	if a.Result != nil {
		for _, secret := range a.Result {
			secretMap[secret.Name] = secret.Value
		}
	}
	return secretMap
}

// ============================================================================
// SemanticAction Constructors for Infisical Operations
// ============================================================================

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

// ============================================================================
// SemanticAction Helper Functions for Infisical Operations
// ============================================================================

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
	if action == nil || action.Properties == nil {
		return "", "", "", "", false, fmt.Errorf("action or properties is nil")
	}

	target, ok := action.Properties["target"]
	if !ok {
		return "", "", "", "", false, fmt.Errorf("no target found in action properties")
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
