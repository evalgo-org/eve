// Package registry provides service discovery and registration client utilities.
package registry

import "eve.evalgo.org/semantic"

// ActionCapability describes what a service can do and what it returns
// This enables schema-driven extraction without runtime type guessing
type ActionCapability struct {
	ActionType   string                 `json:"actionType"`             // e.g., "RetrieveAction", "SearchAction"
	ResultSchema *semantic.ResultSchema `json:"resultSchema,omitempty"` // Schema of returned result
	Examples     []CapabilityExample    `json:"examples,omitempty"`     // Example inputs/outputs
	Description  string                 `json:"description,omitempty"`  // Human-readable description
}

// CapabilityExample provides example input/output for documentation
type CapabilityExample struct {
	Name        string                   `json:"name"`                  // Example name
	Description string                   `json:"description,omitempty"` // What this example demonstrates
	Input       map[string]interface{}   `json:"input"`                 // Example input action
	Output      *semantic.SemanticResult `json:"output"`                // Expected output result
}

// ServiceCapabilities wraps multiple action capabilities
type ServiceCapabilities struct {
	Actions []ActionCapability `json:"actions"` // List of supported actions
}
