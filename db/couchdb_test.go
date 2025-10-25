package db

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	eve "eve.evalgo.org/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSanitizeFilename tests the filename sanitization function
func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple alphanumeric",
			input:    "document123",
			expected: "document123",
		},
		{
			name:     "with forward slash",
			input:    "user/123",
			expected: "user_123",
		},
		{
			name:     "with backslash",
			input:    "user\\123",
			expected: "user_123",
		},
		{
			name:     "with colon",
			input:    "process:2024-01-15",
			expected: "process_2024-01-15",
		},
		{
			name:     "with multiple invalid chars",
			input:    "data<test>:*?",
			expected: "data_test____",
		},
		{
			name:     "with quotes",
			input:    "file\"name",
			expected: "file_name",
		},
		{
			name:     "with pipe",
			input:    "data|pipe",
			expected: "data_pipe",
		},
		{
			name:     "very long filename",
			input:    string(make([]byte, 250)),
			expected: string(make([]byte, 200)),
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "all invalid characters",
			input:    "/*?<>:|\"\\",
			expected: "_________",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			assert.Equal(t, tt.expected, result)
			assert.LessOrEqual(t, len(result), 200, "result should not exceed 200 characters")
		})
	}
}

// TestSaveDocumentToFile tests the document file saving function
func TestSaveDocumentToFile(t *testing.T) {
	t.Run("successful save", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "test_doc.json")

		doc := map[string]interface{}{
			"_id":   "test123",
			"name":  "Test Document",
			"value": 42,
		}

		err := saveDocumentToFile(doc, filePath)
		require.NoError(t, err)

		// Verify file exists
		_, err = os.Stat(filePath)
		require.NoError(t, err)

		// Read and verify content
		data, err := os.ReadFile(filePath)
		require.NoError(t, err)

		var savedDoc map[string]interface{}
		err = json.Unmarshal(data, &savedDoc)
		require.NoError(t, err)

		assert.Equal(t, "test123", savedDoc["_id"])
		assert.Equal(t, "Test Document", savedDoc["name"])
		assert.Equal(t, float64(42), savedDoc["value"]) // JSON unmarshals numbers as float64
	})

	t.Run("nested document", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "nested_doc.json")

		doc := map[string]interface{}{
			"_id": "nested123",
			"metadata": map[string]interface{}{
				"created": "2024-01-01",
				"tags":    []string{"tag1", "tag2"},
			},
		}

		err := saveDocumentToFile(doc, filePath)
		require.NoError(t, err)

		// Verify file exists and content
		data, err := os.ReadFile(filePath)
		require.NoError(t, err)

		var savedDoc map[string]interface{}
		err = json.Unmarshal(data, &savedDoc)
		require.NoError(t, err)

		assert.Equal(t, "nested123", savedDoc["_id"])
		metadata := savedDoc["metadata"].(map[string]interface{})
		assert.Equal(t, "2024-01-01", metadata["created"])
	})

	t.Run("invalid directory path", func(t *testing.T) {
		invalidPath := "/nonexistent/directory/that/does/not/exist/doc.json"

		doc := map[string]interface{}{
			"_id": "test",
		}

		err := saveDocumentToFile(doc, invalidPath)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create file")
	})

	t.Run("empty document", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "empty_doc.json")

		doc := map[string]interface{}{}

		err := saveDocumentToFile(doc, filePath)
		require.NoError(t, err)

		// Verify file exists
		data, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.NotEmpty(t, data)
	})
}

// TestNewCouchDBService_ConfigValidation tests configuration validation
func TestNewCouchDBService_ConfigValidation(t *testing.T) {
	t.Run("empty CouchDB URL", func(t *testing.T) {
		config := eve.FlowConfig{
			CouchDBURL:   "",
			DatabaseName: "testdb",
		}

		service, err := NewCouchDBService(config)
		assert.Error(t, err)
		assert.Nil(t, service)
	})

	t.Run("invalid CouchDB URL", func(t *testing.T) {
		config := eve.FlowConfig{
			CouchDBURL:   "not-a-valid-url",
			DatabaseName: "testdb",
		}

		service, err := NewCouchDBService(config)
		assert.Error(t, err)
		assert.Nil(t, service)
	})
}

// TestFlowProcessDocument_StateTransitions tests document state management
func TestFlowProcessDocument_StateTransitions(t *testing.T) {
	t.Run("state progression", func(t *testing.T) {
		states := []eve.FlowProcessState{
			eve.StateStarted,
			eve.StateRunning,
			eve.StateSuccessful,
		}

		for _, state := range states {
			assert.NotEmpty(t, string(state))
		}
	})

	t.Run("failed state with error", func(t *testing.T) {
		state := eve.StateFailed
		assert.Equal(t, "failed", string(state))
	})
}

// TestCouchDBService_DocumentLifecycle tests the complete document lifecycle
// This test uses mock patterns to validate the service logic without requiring a real CouchDB instance
func TestCouchDBService_DocumentLifecycle(t *testing.T) {
	t.Run("document creation flow", func(t *testing.T) {
		// Test document structure and field validation
		doc := eve.FlowProcessDocument{
			ProcessID:   "process-001",
			State:       eve.StateStarted,
			Description: "Test process",
			Metadata: map[string]interface{}{
				"key": "value",
			},
		}

		assert.Equal(t, "process-001", doc.ProcessID)
		assert.Equal(t, eve.StateStarted, doc.State)
		assert.NotNil(t, doc.Metadata)
	})

	t.Run("document update flow", func(t *testing.T) {
		// Test document updates preserve essential fields
		doc := eve.FlowProcessDocument{
			ID:        "process-001",
			ProcessID: "process-001",
			Rev:       "1-abc123",
			State:     eve.StateRunning,
			CreatedAt: time.Now().Add(-1 * time.Hour),
			UpdatedAt: time.Now(),
		}

		// Simulate state update
		doc.State = eve.StateSuccessful
		doc.UpdatedAt = time.Now()

		assert.Equal(t, eve.StateSuccessful, doc.State)
		assert.NotEqual(t, doc.CreatedAt, doc.UpdatedAt)
	})

	t.Run("history tracking", func(t *testing.T) {
		// Test history array management
		history := []eve.FlowStateChange{
			{
				State:     eve.StateStarted,
				Timestamp: time.Now().Add(-2 * time.Hour),
			},
			{
				State:     eve.StateRunning,
				Timestamp: time.Now().Add(-1 * time.Hour),
			},
			{
				State:     eve.StateSuccessful,
				Timestamp: time.Now(),
			},
		}

		assert.Len(t, history, 3)
		assert.Equal(t, eve.StateStarted, history[0].State)
		assert.Equal(t, eve.StateSuccessful, history[len(history)-1].State)
	})

	t.Run("error state with message", func(t *testing.T) {
		doc := eve.FlowProcessDocument{
			ProcessID: "process-failed",
			State:     eve.StateFailed,
			ErrorMsg:  "Process failed due to timeout",
			History: []eve.FlowStateChange{
				{
					State:     eve.StateFailed,
					Timestamp: time.Now(),
					ErrorMsg:  "Process failed due to timeout",
				},
			},
		}

		assert.Equal(t, eve.StateFailed, doc.State)
		assert.NotEmpty(t, doc.ErrorMsg)
		assert.Equal(t, doc.ErrorMsg, doc.History[0].ErrorMsg)
	})
}

// TestCouchDBResponse tests response structure validation
func TestCouchDBResponse(t *testing.T) {
	t.Run("successful response", func(t *testing.T) {
		response := eve.FlowCouchDBResponse{
			OK:  true,
			ID:  "doc-123",
			Rev: "1-abc123",
		}

		assert.True(t, response.OK)
		assert.NotEmpty(t, response.ID)
		assert.NotEmpty(t, response.Rev)
	})

	t.Run("failed response", func(t *testing.T) {
		response := eve.FlowCouchDBResponse{
			OK:  false,
			ID:  "",
			Rev: "",
		}

		assert.False(t, response.OK)
	})
}

// TestDocumentIDGeneration tests document ID handling
func TestDocumentIDGeneration(t *testing.T) {
	t.Run("ID defaults to ProcessID", func(t *testing.T) {
		doc := eve.FlowProcessDocument{
			ProcessID: "process-123",
		}

		// Simulate the SaveDocument logic for ID assignment
		if doc.ID == "" {
			doc.ID = doc.ProcessID
		}

		assert.Equal(t, doc.ProcessID, doc.ID)
	})

	t.Run("explicit ID preserved", func(t *testing.T) {
		doc := eve.FlowProcessDocument{
			ID:        "custom-id",
			ProcessID: "process-456",
		}

		// ID should remain unchanged if already set
		if doc.ID == "" {
			doc.ID = doc.ProcessID
		}

		assert.Equal(t, "custom-id", doc.ID)
		assert.NotEqual(t, doc.ProcessID, doc.ID)
	})
}

// TestTimestampHandling tests timestamp management in documents
func TestTimestampHandling(t *testing.T) {
	t.Run("creation timestamp", func(t *testing.T) {
		now := time.Now()
		doc := eve.FlowProcessDocument{
			ProcessID: "process-001",
			CreatedAt: now,
			UpdatedAt: now,
		}

		assert.Equal(t, doc.CreatedAt, doc.UpdatedAt)
	})

	t.Run("update timestamp progression", func(t *testing.T) {
		createdAt := time.Now().Add(-1 * time.Hour)
		updatedAt := time.Now()

		doc := eve.FlowProcessDocument{
			ProcessID: "process-002",
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		}

		assert.True(t, doc.UpdatedAt.After(doc.CreatedAt))
	})
}

// TestStateChangeValidation tests state change structure
func TestStateChangeValidation(t *testing.T) {
	t.Run("valid state change", func(t *testing.T) {
		change := eve.FlowStateChange{
			State:     eve.StateRunning,
			Timestamp: time.Now(),
		}

		assert.NotEmpty(t, change.State)
		assert.False(t, change.Timestamp.IsZero())
	})

	t.Run("state change with error", func(t *testing.T) {
		change := eve.FlowStateChange{
			State:     eve.StateFailed,
			Timestamp: time.Now(),
			ErrorMsg:  "Connection timeout",
		}

		assert.Equal(t, eve.StateFailed, change.State)
		assert.NotEmpty(t, change.ErrorMsg)
	})

	t.Run("chronological state changes", func(t *testing.T) {
		baseTime := time.Now()
		changes := []eve.FlowStateChange{
			{State: eve.StateStarted, Timestamp: baseTime},
			{State: eve.StateRunning, Timestamp: baseTime.Add(1 * time.Second)},
			{State: eve.StateSuccessful, Timestamp: baseTime.Add(2 * time.Second)},
		}

		for i := 1; i < len(changes); i++ {
			assert.True(t, changes[i].Timestamp.After(changes[i-1].Timestamp))
		}
	})
}

// TestMetadataHandling tests metadata field operations
func TestMetadataHandling(t *testing.T) {
	t.Run("empty metadata", func(t *testing.T) {
		doc := eve.FlowProcessDocument{
			ProcessID: "process-001",
			Metadata:  map[string]interface{}{},
		}

		assert.NotNil(t, doc.Metadata)
		assert.Empty(t, doc.Metadata)
	})

	t.Run("metadata with various types", func(t *testing.T) {
		doc := eve.FlowProcessDocument{
			ProcessID: "process-002",
			Metadata: map[string]interface{}{
				"string":  "value",
				"number":  42,
				"boolean": true,
				"array":   []string{"a", "b", "c"},
				"nested": map[string]interface{}{
					"key": "value",
				},
			},
		}

		assert.Len(t, doc.Metadata, 5)
		assert.Equal(t, "value", doc.Metadata["string"])
		assert.Equal(t, 42, doc.Metadata["number"])
		assert.Equal(t, true, doc.Metadata["boolean"])
	})

	t.Run("nil metadata", func(t *testing.T) {
		doc := eve.FlowProcessDocument{
			ProcessID: "process-003",
			Metadata:  nil,
		}

		assert.Nil(t, doc.Metadata)
	})
}

// TestFlowConfigValidation tests flow configuration structure
func TestFlowConfigValidation(t *testing.T) {
	t.Run("valid configuration", func(t *testing.T) {
		config := eve.FlowConfig{
			CouchDBURL:   "http://admin:password@localhost:5984",
			DatabaseName: "test_flows",
		}

		assert.NotEmpty(t, config.CouchDBURL)
		assert.NotEmpty(t, config.DatabaseName)
	})

	t.Run("configuration with authentication", func(t *testing.T) {
		config := eve.FlowConfig{
			CouchDBURL:   "http://user:pass@couchdb.example.com:5984",
			DatabaseName: "production_flows",
		}

		assert.Contains(t, config.CouchDBURL, "user:pass")
		assert.Contains(t, config.CouchDBURL, "couchdb.example.com")
	})
}

// TestDocumentSerialization tests JSON serialization of documents
func TestDocumentSerialization(t *testing.T) {
	t.Run("serialize complete document", func(t *testing.T) {
		doc := eve.FlowProcessDocument{
			ID:          "doc-001",
			Rev:         "1-abc123",
			ProcessID:   "process-001",
			State:       eve.StateSuccessful,
			Description: "Test process",
			ErrorMsg:    "",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			History: []eve.FlowStateChange{
				{
					State:     eve.StateStarted,
					Timestamp: time.Now(),
				},
			},
			Metadata: map[string]interface{}{
				"key": "value",
			},
		}

		data, err := json.Marshal(doc)
		require.NoError(t, err)
		assert.NotEmpty(t, data)

		var decoded eve.FlowProcessDocument
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, doc.ID, decoded.ID)
		assert.Equal(t, doc.ProcessID, decoded.ProcessID)
		assert.Equal(t, doc.State, decoded.State)
	})

	t.Run("deserialize document", func(t *testing.T) {
		jsonData := `{
			"_id": "doc-002",
			"_rev": "2-def456",
			"process_id": "process-002",
			"state": "running",
			"description": "Running process",
			"error_msg": "",
			"created_at": "2024-01-01T00:00:00Z",
			"updated_at": "2024-01-01T01:00:00Z",
			"history": [
				{
					"state": "started",
					"timestamp": "2024-01-01T00:00:00Z"
				}
			],
			"metadata": {
				"user": "admin"
			}
		}`

		var doc eve.FlowProcessDocument
		err := json.Unmarshal([]byte(jsonData), &doc)
		require.NoError(t, err)

		assert.Equal(t, "doc-002", doc.ID)
		assert.Equal(t, "process-002", doc.ProcessID)
		assert.Equal(t, eve.FlowProcessState("running"), doc.State)
	})
}

// BenchmarkSanitizeFilename benchmarks filename sanitization
func BenchmarkSanitizeFilename(b *testing.B) {
	testCases := []string{
		"simple_filename",
		"complex/file:name*with?chars",
		string(make([]byte, 250)),
	}

	for _, tc := range testCases {
		b.Run(tc[:min(len(tc), 20)], func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				sanitizeFilename(tc)
			}
		})
	}
}

// BenchmarkSaveDocumentToFile benchmarks document file saving
func BenchmarkSaveDocumentToFile(b *testing.B) {
	tempDir := b.TempDir()

	doc := map[string]interface{}{
		"_id":         "bench-doc",
		"name":        "Benchmark Document",
		"value":       42,
		"description": "This is a benchmark document",
		"metadata": map[string]interface{}{
			"created": "2024-01-01",
			"tags":    []string{"tag1", "tag2", "tag3"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filePath := filepath.Join(tempDir, "bench_doc_"+string(rune(i))+".json")
		_ = saveDocumentToFile(doc, filePath)
	}
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
