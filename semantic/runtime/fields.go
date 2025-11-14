package runtime

import (
	"fmt"
	"strings"
)

// getNestedField retrieves a nested field from a map using dot notation
// Example: "result.contentUrl" navigates through map["result"]["contentUrl"]
func getNestedField(data map[string]interface{}, path string) (interface{}, error) {
	if data == nil {
		return nil, fmt.Errorf("data is nil")
	}

	parts := strings.Split(path, ".")
	current := data

	for i, key := range parts {
		value, ok := current[key]
		if !ok {
			return nil, fmt.Errorf("field not found: %s", key)
		}

		// Last element - return value
		if i == len(parts)-1 {
			return value, nil
		}

		// Navigate deeper
		current, ok = value.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("field %s is not an object, cannot navigate further", key)
		}
	}

	return current, nil
}

// setNestedField sets a nested field in a map using dot notation
// Creates intermediate maps if they don't exist
func setNestedField(data map[string]interface{}, path string, value interface{}) error {
	if data == nil {
		return fmt.Errorf("data is nil")
	}

	parts := strings.Split(path, ".")
	current := data

	for i, key := range parts {
		// Last element - set value
		if i == len(parts)-1 {
			current[key] = value
			return nil
		}

		// Navigate or create intermediate maps
		if existing, ok := current[key]; ok {
			// Field exists, try to navigate
			if nextMap, ok := existing.(map[string]interface{}); ok {
				current = nextMap
			} else {
				return fmt.Errorf("field %s exists but is not an object", key)
			}
		} else {
			// Field doesn't exist, create it
			newMap := make(map[string]interface{})
			current[key] = newMap
			current = newMap
		}
	}

	return nil
}

// WalkJSON walks through a JSON structure and applies a function to all string values
// This is used for variable substitution
func WalkJSON(data interface{}, fn func(string) (string, error)) (interface{}, error) {
	switch v := data.(type) {
	case string:
		// Apply function to string values
		return fn(v)

	case map[string]interface{}:
		// Recursively process maps
		result := make(map[string]interface{})
		for key, value := range v {
			processed, err := WalkJSON(value, fn)
			if err != nil {
				return nil, err
			}
			result[key] = processed
		}
		return result, nil

	case []interface{}:
		// Recursively process arrays
		result := make([]interface{}, len(v))
		for i, value := range v {
			processed, err := WalkJSON(value, fn)
			if err != nil {
				return nil, err
			}
			result[i] = processed
		}
		return result, nil

	default:
		// Return other types as-is (numbers, booleans, null)
		return v, nil
	}
}

// MergeFields merges source fields into destination
// Source values take precedence over destination values
func MergeFields(dest, source map[string]interface{}) {
	if dest == nil || source == nil {
		return
	}

	for k, v := range source {
		dest[k] = v
	}
}
