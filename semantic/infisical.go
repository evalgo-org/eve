package semantic

import (
	"context"
	"fmt"

	infisical "github.com/infisical/go-sdk"
)

// InfisicalRetrieveAction represents a Schema.org RetrieveAction for fetching secrets from Infisical.
// It retrieves secrets from a specific Infisical project and environment.
type InfisicalRetrieveAction struct {
	Context      string            `json:"@context,omitempty"`
	Type         string            `json:"@type"` // "RetrieveAction"
	Identifier   string            `json:"identifier"`
	Name         string            `json:"name,omitempty"`
	Description  string            `json:"description,omitempty"`
	Target       *InfisicalProject `json:"target"` // Which project/environment to retrieve from
	Result       []*PropertyValue  `json:"result,omitempty"`
	ActionStatus string            `json:"actionStatus,omitempty"`
	StartTime    string            `json:"startTime,omitempty"`
	EndTime      string            `json:"endTime,omitempty"`
	Error        *PropertyValue    `json:"error,omitempty"`
}

// InfisicalProject represents an Infisical project with environment configuration.
// Maps to Schema.org Project type.
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
		return fmt.Errorf("target project is required")
	}

	// Extract host from URL
	host := a.Target.Url
	if len(host) > 8 && host[:8] == "https://" {
		host = host[8:]
	} else if len(host) > 7 && host[:7] == "http://" {
		host = host[7:]
	}

	// Create Infisical client
	client := infisical.NewInfisicalClient(context.Background(), infisical.Config{
		SiteUrl:          a.Target.Url,
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

	// Set secret path default
	secretPath := a.Target.SecretPath
	if secretPath == "" {
		secretPath = "/"
	}

	// Retrieve secrets
	secrets, err := client.Secrets().List(infisical.ListSecretsOptions{
		AttachToProcessEnv: false,
		Environment:        a.Target.Environment,
		ProjectID:          a.Target.Identifier,
		SecretPath:         secretPath,
		IncludeImports:     a.Target.IncludeImports,
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
