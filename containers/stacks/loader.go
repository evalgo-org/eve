package stacks

import (
	"encoding/json"
	"fmt"
	"os"
)

// LoadStackFromFile loads a stack definition from a JSON-LD file.
//
// The file should contain a schema.org ItemList with SoftwareApplication elements.
//
// Example:
//
//	stack, err := LoadStackFromFile("definitions/infisical.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
func LoadStackFromFile(path string) (*Stack, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read stack file: %w", err)
	}

	return LoadStackFromJSON(data)
}

// LoadStackFromJSON loads a stack definition from JSON-LD bytes.
func LoadStackFromJSON(data []byte) (*Stack, error) {
	var stack Stack
	if err := json.Unmarshal(data, &stack); err != nil {
		return nil, fmt.Errorf("failed to parse stack JSON: %w", err)
	}

	// Set defaults for schema.org fields if not specified
	if stack.Context == "" {
		stack.Context = "https://schema.org"
	}
	if stack.Type == "" {
		stack.Type = "ItemList"
	}

	// Set defaults for container types
	for i := range stack.ItemListElement {
		if stack.ItemListElement[i].Type == "" {
			stack.ItemListElement[i].Type = "SoftwareApplication"
		}
		// Set default protocol for ports
		for j := range stack.ItemListElement[i].Ports {
			if stack.ItemListElement[i].Ports[j].Protocol == "" {
				stack.ItemListElement[i].Ports[j].Protocol = "tcp"
			}
		}
		// Set default action type
		for j := range stack.ItemListElement[i].PotentialAction {
			if stack.ItemListElement[i].PotentialAction[j].Type == "" {
				stack.ItemListElement[i].PotentialAction[j].Type = "Action"
			}
		}
		// Set default health check values
		if stack.ItemListElement[i].HealthCheck.Interval == 0 && stack.ItemListElement[i].HealthCheck.Type != "" {
			stack.ItemListElement[i].HealthCheck.Interval = 10
		}
		if stack.ItemListElement[i].HealthCheck.Timeout == 0 && stack.ItemListElement[i].HealthCheck.Type != "" {
			stack.ItemListElement[i].HealthCheck.Timeout = 5
		}
		if stack.ItemListElement[i].HealthCheck.Retries == 0 && stack.ItemListElement[i].HealthCheck.Type != "" {
			stack.ItemListElement[i].HealthCheck.Retries = 3
		}
		if stack.ItemListElement[i].HealthCheck.StartPeriod == 0 && stack.ItemListElement[i].HealthCheck.Type != "" {
			stack.ItemListElement[i].HealthCheck.StartPeriod = 10
		}
		// Set default volume mount type
		for j := range stack.ItemListElement[i].Volumes {
			if stack.ItemListElement[i].Volumes[j].Type == "" {
				stack.ItemListElement[i].Volumes[j].Type = "volume"
			}
		}
		// Set default software requirement type
		for j := range stack.ItemListElement[i].SoftwareRequirements {
			if stack.ItemListElement[i].SoftwareRequirements[j].Type == "" {
				stack.ItemListElement[i].SoftwareRequirements[j].Type = "SoftwareApplication"
			}
		}
	}

	// Set default network driver
	if stack.Network.Driver == "" && stack.Network.Name != "" {
		stack.Network.Driver = "bridge"
	}

	// Validate the stack
	if err := stack.Validate(); err != nil {
		return nil, fmt.Errorf("invalid stack: %w", err)
	}

	return &stack, nil
}

// SaveStackToFile saves a stack definition to a JSON-LD file.
func SaveStackToFile(stack *Stack, path string) error {
	data, err := json.MarshalIndent(stack, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal stack: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write stack file: %w", err)
	}

	return nil
}

// StackToJSON converts a stack to JSON-LD bytes.
func StackToJSON(stack *Stack) ([]byte, error) {
	return json.MarshalIndent(stack, "", "  ")
}
