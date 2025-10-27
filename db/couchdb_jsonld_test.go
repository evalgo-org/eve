package db

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtractJSONLDType tests extracting @type from JSON-LD documents
func TestExtractJSONLDType(t *testing.T) {
	t.Run("document with @type", func(t *testing.T) {
		doc := map[string]interface{}{
			"@context": "https://schema.org",
			"@type":    "SoftwareApplication",
			"name":     "Test App",
		}

		docType, err := ExtractJSONLDType(doc)
		require.NoError(t, err)
		assert.Equal(t, "SoftwareApplication", docType)
	})

	t.Run("document with @type as string in struct", func(t *testing.T) {
		type TestDoc struct {
			Context string `json:"@context"`
			Type    string `json:"@type"`
			Name    string `json:"name"`
		}

		doc := TestDoc{
			Context: "https://schema.org",
			Type:    "Person",
			Name:    "John Doe",
		}

		docType, err := ExtractJSONLDType(doc)
		require.NoError(t, err)
		assert.Equal(t, "Person", docType)
	})

	t.Run("document without @type", func(t *testing.T) {
		doc := map[string]interface{}{
			"name": "Test",
		}

		docType, err := ExtractJSONLDType(doc)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no @type field")
		assert.Empty(t, docType)
	})

	t.Run("document with empty @type", func(t *testing.T) {
		doc := map[string]interface{}{
			"@type": "",
			"name":  "Test",
		}

		docType, err := ExtractJSONLDType(doc)
		require.NoError(t, err)
		assert.Equal(t, "", docType)
	})

	t.Run("document with nil @type", func(t *testing.T) {
		doc := map[string]interface{}{
			"@type": nil,
			"name":  "Test",
		}

		docType, err := ExtractJSONLDType(doc)
		require.NoError(t, err)
		assert.Equal(t, "<nil>", docType)
	})
}

// TestSetJSONLDContext tests setting @context on documents
func TestSetJSONLDContext(t *testing.T) {
	t.Run("set context on map", func(t *testing.T) {
		doc := map[string]interface{}{
			"@type": "Person",
			"name":  "John Doe",
		}

		result := SetJSONLDContext(doc, "https://schema.org")

		assert.Equal(t, "https://schema.org", result["@context"])
		assert.Equal(t, "Person", result["@type"])
		assert.Equal(t, "John Doe", result["name"])
	})

	t.Run("set context on struct", func(t *testing.T) {
		type TestDoc struct {
			Type string `json:"@type"`
			Name string `json:"name"`
		}

		doc := TestDoc{
			Type: "Organization",
			Name: "ACME Corp",
		}

		result := SetJSONLDContext(doc, "https://schema.org")

		assert.Equal(t, "https://schema.org", result["@context"])
		assert.Equal(t, "Organization", result["@type"])
		assert.Equal(t, "ACME Corp", result["name"])
	})

	t.Run("replace existing context", func(t *testing.T) {
		doc := map[string]interface{}{
			"@context": "https://example.com",
			"@type":    "Thing",
			"name":     "Test",
		}

		result := SetJSONLDContext(doc, "https://schema.org")

		assert.Equal(t, "https://schema.org", result["@context"])
	})

	t.Run("handle unmarshal error gracefully", func(t *testing.T) {
		// Create a type that can't be marshaled
		doc := make(chan int)

		result := SetJSONLDContext(doc, "https://schema.org")

		// Should return minimal map with just context
		assert.Equal(t, "https://schema.org", result["@context"])
		assert.Len(t, result, 1)
	})
}

// TestValidateJSONLD tests JSON-LD validation
func TestValidateJSONLD(t *testing.T) {
	t.Run("valid JSON-LD with context and type", func(t *testing.T) {
		doc := map[string]interface{}{
			"@context": "https://schema.org",
			"@type":    "Person",
			"name":     "John Doe",
		}

		err := ValidateJSONLD(doc, "https://schema.org")
		assert.NoError(t, err)
	})

	t.Run("missing @context", func(t *testing.T) {
		doc := map[string]interface{}{
			"@type": "Person",
			"name":  "John Doe",
		}

		err := ValidateJSONLD(doc, "https://schema.org")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing @context")
	})

	t.Run("missing @type - warning only", func(t *testing.T) {
		doc := map[string]interface{}{
			"@context": "https://schema.org",
			"name":     "John Doe",
		}

		// ValidateJSONLD only warns about missing @type, doesn't error
		err := ValidateJSONLD(doc, "https://schema.org")
		assert.NoError(t, err)
	})

	t.Run("wrong context", func(t *testing.T) {
		doc := map[string]interface{}{
			"@context": "https://wrong-context.com",
			"@type":    "Person",
			"name":     "John Doe",
		}

		err := ValidateJSONLD(doc, "https://schema.org")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context mismatch")
	})

	t.Run("valid struct with JSON-LD tags", func(t *testing.T) {
		type Person struct {
			Context string `json:"@context"`
			Type    string `json:"@type"`
			Name    string `json:"name"`
		}

		doc := Person{
			Context: "https://schema.org",
			Type:    "Person",
			Name:    "Jane Doe",
		}

		err := ValidateJSONLD(doc, "https://schema.org")
		assert.NoError(t, err)
	})
}

// TestExpandJSONLD tests JSON-LD expansion (basic test)
func TestExpandJSONLD(t *testing.T) {
	t.Run("expand simple document", func(t *testing.T) {
		doc := map[string]interface{}{
			"@context": "https://schema.org",
			"@type":    "Person",
			"name":     "John Doe",
		}

		expanded, err := ExpandJSONLD(doc)
		require.NoError(t, err)
		assert.NotNil(t, expanded)
		// Note: Actual expansion would require a JSON-LD processor
		// This is a simplified version for testing structure
	})

	t.Run("handle marshal error", func(t *testing.T) {
		// Invalid doc that can't be marshaled
		doc := make(chan int)

		_, err := ExpandJSONLD(doc)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to marshal document")
	})
}

// TestCompactJSONLD tests JSON-LD compaction (basic test)
func TestCompactJSONLD(t *testing.T) {
	t.Run("compact document", func(t *testing.T) {
		doc := map[string]interface{}{
			"http://schema.org/name": "John Doe",
		}

		compacted, err := CompactJSONLD(doc, "https://schema.org")
		require.NoError(t, err)
		assert.NotNil(t, compacted)
	})

	t.Run("handle marshal error", func(t *testing.T) {
		doc := make(chan int)

		_, err := CompactJSONLD(doc, "https://schema.org")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to marshal document")
	})
}

// TestNormalizeJSONLD tests JSON-LD normalization (basic test)
func TestNormalizeJSONLD(t *testing.T) {
	t.Run("normalize document", func(t *testing.T) {
		doc := map[string]interface{}{
			"@context": "https://schema.org",
			"@type":    "Person",
			"name":     "John Doe",
		}

		normalized, err := NormalizeJSONLD(doc)
		require.NoError(t, err)
		assert.NotEmpty(t, normalized)
	})

	t.Run("handle marshal error", func(t *testing.T) {
		doc := make(chan int)

		_, err := NormalizeJSONLD(doc)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to marshal document")
	})
}

// TestJSONLDDocumentWithID tests documents with @id field
func TestJSONLDDocumentWithID(t *testing.T) {
	t.Run("document with @id", func(t *testing.T) {
		type Host struct {
			Context   string `json:"@context"`
			Type      string `json:"@type"`
			ID        string `json:"@id"`
			Name      string `json:"name"`
			IPAddress string `json:"ipAddress"`
		}

		host := Host{
			Context:   "https://schema.org",
			Type:      "ComputerSystem",
			ID:        "host-001",
			Name:      "web-server",
			IPAddress: "192.168.1.100",
		}

		// Marshal and unmarshal to simulate what happens in SaveGenericDocument
		jsonData, err := json.Marshal(host)
		require.NoError(t, err)

		var docMap map[string]interface{}
		err = json.Unmarshal(jsonData, &docMap)
		require.NoError(t, err)

		// Verify @id is present in JSON
		assert.Equal(t, "host-001", docMap["@id"])
		assert.Equal(t, "ComputerSystem", docMap["@type"])

		// Simulate the @id -> _id mapping
		if id, ok := docMap["@id"]; ok && id != nil && id != "" {
			docMap["_id"] = id
		}

		// Verify _id was added
		assert.Equal(t, "host-001", docMap["_id"])
		assert.Equal(t, "host-001", docMap["@id"]) // @id still present
	})
}
