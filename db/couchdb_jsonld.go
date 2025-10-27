package db

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ValidateJSONLD performs basic JSON-LD validation on a document.
// This checks for required JSON-LD fields and structure without full RDF processing.
//
// Parameters:
//   - doc: Document to validate (map or struct)
//   - context: Expected @context value (empty string to skip context check)
//
// Returns:
//   - error: Validation errors if document is invalid
//
// Validation Checks:
//   - Document must have @context field (if context parameter provided)
//   - Document should have @type field for semantic typing
//   - @id field should be present for linked data
//   - Context must match expected value (if specified)
//
// JSON-LD Requirements:
//
//	Minimum valid JSON-LD document:
//	{
//	    "@context": "https://schema.org",
//	    "@type": "Thing",
//	    "@id": "https://example.com/thing/123"
//	}
//
// Example Usage:
//
//	doc := map[string]interface{}{
//	    "@context": "https://schema.org",
//	    "@type":    "SoftwareApplication",
//	    "@id":      "urn:container:nginx-1",
//	    "name":     "nginx",
//	}
//
//	err := ValidateJSONLD(doc, "https://schema.org")
//	if err != nil {
//	    log.Printf("Invalid JSON-LD: %v", err)
//	    return
//	}
func ValidateJSONLD(doc interface{}, context string) error {
	// Convert to map
	jsonData, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	var docMap map[string]interface{}
	if err := json.Unmarshal(jsonData, &docMap); err != nil {
		return fmt.Errorf("failed to unmarshal document: %w", err)
	}

	// Check for @context
	docContext, hasContext := docMap["@context"]
	if !hasContext {
		return fmt.Errorf("document missing @context field")
	}

	// Validate context if specified
	if context != "" {
		contextStr := fmt.Sprintf("%v", docContext)
		if contextStr != context {
			return fmt.Errorf("context mismatch: expected %s, got %s", context, contextStr)
		}
	}

	// Check for @type (recommended but not strictly required)
	if _, hasType := docMap["@type"]; !hasType {
		// Warning but not error
		fmt.Println("Warning: document missing @type field")
	}

	// Check for @id (recommended for linked data)
	if _, hasID := docMap["@id"]; !hasID {
		// Warning but not error
		fmt.Println("Warning: document missing @id field")
	}

	return nil
}

// ExpandJSONLD performs basic JSON-LD expansion.
// This is a simplified implementation that handles common expansion patterns
// without requiring a full JSON-LD processor.
//
// Parameters:
//   - doc: Document to expand
//
// Returns:
//   - map[string]interface{}: Expanded document
//   - error: Expansion errors
//
// Expansion Process:
//
//	JSON-LD expansion converts compact representation to explicit form:
//	- Resolves all terms to full IRIs
//	- Converts values to explicit object format
//	- Expands @type to @type array
//	- Converts single values to arrays
//
// Limitations:
//
//	This is a basic implementation:
//	- Does not fetch remote contexts
//	- Limited to simple context resolution
//	- For full expansion, use a dedicated JSON-LD library
//
// Example Usage:
//
//	doc := map[string]interface{}{
//	    "@context": "https://schema.org",
//	    "@type":    "SoftwareApplication",
//	    "name":     "nginx",
//	}
//
//	expanded, err := ExpandJSONLD(doc)
//	if err != nil {
//	    log.Printf("Expansion failed: %v", err)
//	    return
//	}
//
//	// expanded now has full IRIs
//	fmt.Printf("Expanded: %+v\n", expanded)
func ExpandJSONLD(doc interface{}) (map[string]interface{}, error) {
	// Convert to map
	jsonData, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal document: %w", err)
	}

	var docMap map[string]interface{}
	if err := json.Unmarshal(jsonData, &docMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal document: %w", err)
	}

	// Basic expansion (simplified)
	expanded := make(map[string]interface{})

	// Get context for term expansion
	context := ""
	if ctx, ok := docMap["@context"]; ok {
		context = fmt.Sprintf("%v", ctx)
		context = strings.TrimSuffix(context, "/")
	}

	// Process each field
	for key, value := range docMap {
		// Skip @context in expansion
		if key == "@context" {
			continue
		}

		// Expand @type
		if key == "@type" {
			expanded["@type"] = []interface{}{expandTerm(value, context)}
			continue
		}

		// Expand @id
		if key == "@id" {
			expanded["@id"] = value
			continue
		}

		// Expand regular properties
		expandedKey := expandTerm(key, context)
		expanded[expandedKey] = expandValue(value)
	}

	return expanded, nil
}

// CompactJSONLD performs basic JSON-LD compaction.
// This converts expanded JSON-LD to a more compact form using a context.
//
// Parameters:
//   - doc: Expanded document to compact
//   - context: Context to use for compaction
//
// Returns:
//   - map[string]interface{}: Compacted document
//   - error: Compaction errors
//
// Compaction Process:
//
//	Converts expanded form back to compact:
//	- Replaces full IRIs with short terms
//	- Uses context for term mapping
//	- Simplifies value objects to simple values
//	- Converts arrays to single values where appropriate
//
// Example Usage:
//
//	expanded := map[string]interface{}{
//	    "@type":                              []interface{}{"https://schema.org/SoftwareApplication"},
//	    "https://schema.org/name":            "nginx",
//	    "https://schema.org/applicationCategory": "web-server",
//	}
//
//	compacted, err := CompactJSONLD(expanded, "https://schema.org")
//	if err != nil {
//	    log.Printf("Compaction failed: %v", err)
//	    return
//	}
//
//	// compacted uses short terms:
//	// {"@context": "https://schema.org", "@type": "SoftwareApplication", "name": "nginx", ...}
func CompactJSONLD(doc interface{}, context string) (map[string]interface{}, error) {
	// Convert to map
	jsonData, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal document: %w", err)
	}

	var docMap map[string]interface{}
	if err := json.Unmarshal(jsonData, &docMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal document: %w", err)
	}

	// Basic compaction (simplified)
	compacted := make(map[string]interface{})
	compacted["@context"] = context

	contextPrefix := strings.TrimSuffix(context, "/") + "/"

	// Process each field
	for key, value := range docMap {
		// Compact @type
		if key == "@type" {
			if typeArray, ok := value.([]interface{}); ok && len(typeArray) > 0 {
				typeStr := fmt.Sprintf("%v", typeArray[0])
				compacted["@type"] = compactTerm(typeStr, contextPrefix)
			} else {
				compacted["@type"] = value
			}
			continue
		}

		// Keep @id as-is
		if key == "@id" {
			compacted["@id"] = value
			continue
		}

		// Compact property names
		compactedKey := compactTerm(key, contextPrefix)
		compacted[compactedKey] = compactValue(value)
	}

	return compacted, nil
}

// NormalizeJSONLD performs basic JSON-LD normalization (canonicalization).
// This produces a deterministic representation for comparison and hashing.
//
// Parameters:
//   - doc: Document to normalize
//
// Returns:
//   - string: Normalized JSON-LD as string
//   - error: Normalization errors
//
// Normalization Process:
//
//	Creates canonical form:
//	- Sorts all keys alphabetically
//	- Removes whitespace
//	- Ensures consistent formatting
//	- Produces deterministic output
//
// Use Cases:
//   - Document comparison and deduplication
//   - Cryptographic signing of JSON-LD
//   - Cache key generation
//   - Content-addressed storage
//
// Example Usage:
//
//	doc := map[string]interface{}{
//	    "@context": "https://schema.org",
//	    "name":     "nginx",
//	    "@type":    "SoftwareApplication",
//	}
//
//	normalized, err := NormalizeJSONLD(doc)
//	if err != nil {
//	    log.Printf("Normalization failed: %v", err)
//	    return
//	}
//
//	// normalized is deterministic string representation
//	fmt.Printf("Hash: %x\n", sha256.Sum256([]byte(normalized)))
func NormalizeJSONLD(doc interface{}) (string, error) {
	// Convert to map
	jsonData, err := json.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("failed to marshal document: %w", err)
	}

	var docMap map[string]interface{}
	if err := json.Unmarshal(jsonData, &docMap); err != nil {
		return "", fmt.Errorf("failed to unmarshal document: %w", err)
	}

	// Marshal with sorted keys (Go's json.Marshal sorts keys by default)
	normalized, err := json.Marshal(docMap)
	if err != nil {
		return "", fmt.Errorf("failed to normalize: %w", err)
	}

	return string(normalized), nil
}

// expandTerm expands a term using the context.
// Internal helper for JSON-LD expansion.
func expandTerm(term interface{}, context string) string {
	termStr := fmt.Sprintf("%v", term)

	// If already a full IRI, return as-is
	if strings.HasPrefix(termStr, "http://") || strings.HasPrefix(termStr, "https://") {
		return termStr
	}

	// If context is provided, prepend it
	if context != "" && !strings.HasPrefix(termStr, "@") {
		return context + "/" + termStr
	}

	return termStr
}

// expandValue expands a value to JSON-LD format.
// Internal helper for JSON-LD expansion.
func expandValue(value interface{}) interface{} {
	// Arrays stay as arrays
	if arr, ok := value.([]interface{}); ok {
		expanded := make([]interface{}, len(arr))
		for i, item := range arr {
			expanded[i] = expandValue(item)
		}
		return expanded
	}

	// Objects stay as objects (recursive expansion)
	if obj, ok := value.(map[string]interface{}); ok {
		expanded := make(map[string]interface{})
		for k, v := range obj {
			expanded[k] = expandValue(v)
		}
		return expanded
	}

	// Simple values become value objects
	return []interface{}{
		map[string]interface{}{
			"@value": value,
		},
	}
}

// compactTerm compacts a full IRI to a short term.
// Internal helper for JSON-LD compaction.
func compactTerm(iri string, contextPrefix string) string {
	// Remove context prefix if present
	if strings.HasPrefix(iri, contextPrefix) {
		return strings.TrimPrefix(iri, contextPrefix)
	}
	return iri
}

// compactValue compacts a value from expanded form.
// Internal helper for JSON-LD compaction.
func compactValue(value interface{}) interface{} {
	// Check if it's a value object array
	if arr, ok := value.([]interface{}); ok {
		if len(arr) == 1 {
			if valueObj, ok := arr[0].(map[string]interface{}); ok {
				if val, ok := valueObj["@value"]; ok {
					return val
				}
			}
		}
	}

	return value
}

// ExtractJSONLDType extracts the @type value from a JSON-LD document.
// This is a convenience function for type checking.
//
// Parameters:
//   - doc: JSON-LD document
//
// Returns:
//   - string: The @type value
//   - error: Error if @type not found
//
// Example Usage:
//
//	docType, err := ExtractJSONLDType(doc)
//	if err != nil {
//	    log.Printf("No type found: %v", err)
//	    return
//	}
//
//	switch docType {
//	case "SoftwareApplication":
//	    // Handle container
//	case "ComputerServer":
//	    // Handle host
//	}
func ExtractJSONLDType(doc interface{}) (string, error) {
	jsonData, err := json.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("failed to marshal document: %w", err)
	}

	var docMap map[string]interface{}
	if err := json.Unmarshal(jsonData, &docMap); err != nil {
		return "", fmt.Errorf("failed to unmarshal document: %w", err)
	}

	if typeValue, ok := docMap["@type"]; ok {
		return fmt.Sprintf("%v", typeValue), nil
	}

	return "", fmt.Errorf("document has no @type field")
}

// SetJSONLDContext adds or updates the @context field in a document.
// This is a convenience function for adding JSON-LD context.
//
// Parameters:
//   - doc: Document to modify
//   - context: Context URL or object
//
// Returns:
//   - map[string]interface{}: Document with updated context
//
// Example Usage:
//
//	doc := map[string]interface{}{
//	    "name": "nginx",
//	    "status": "running",
//	}
//
//	doc = SetJSONLDContext(doc, "https://schema.org")
//	// Now doc has @context field
func SetJSONLDContext(doc interface{}, context string) map[string]interface{} {
	jsonData, _ := json.Marshal(doc)
	var docMap map[string]interface{}
	json.Unmarshal(jsonData, &docMap)

	docMap["@context"] = context
	return docMap
}
