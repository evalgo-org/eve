//go:build integration
// +build integration

package db

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	eve "eve.evalgo.org/common"
)

// setupCouchDBContainer starts a CouchDB container for testing
func setupCouchDBContainer(t *testing.T) (string, func()) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "couchdb:3.3",
		ExposedPorts: []string{"5984/tcp"},
		Env: map[string]string{
			"COUCHDB_USER":     "admin",
			"COUCHDB_PASSWORD": "testpass",
		},
		WaitingFor: wait.ForHTTP("/_up").WithPort("5984/tcp").WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "Failed to start CouchDB container")

	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "5984")
	require.NoError(t, err)

	url := fmt.Sprintf("http://admin:testpass@%s:%s", host, port.Port())

	cleanup := func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}

	return url, cleanup
}

// TestCouchDBService_Integration_SaveDocument tests saving a document
func TestCouchDBService_Integration_SaveDocument(t *testing.T) {
	url, cleanup := setupCouchDBContainer(t)
	defer cleanup()

	config := eve.FlowConfig{
		CouchDBURL:   url,
		DatabaseName: "test_flows",
	}

	service, err := NewCouchDBService(config)
	require.NoError(t, err, "Failed to create CouchDB service")
	defer service.Close()

	t.Run("save new document", func(t *testing.T) {
		doc := eve.FlowProcessDocument{
			ProcessID:   "test-001",
			State:       eve.StateStarted,
			Description: "Integration test document",
			Metadata: map[string]interface{}{
				"test": "value",
			},
		}

		resp, err := service.SaveDocument(doc)
		require.NoError(t, err, "Failed to save document")
		assert.True(t, resp.OK)
		assert.Equal(t, "test-001", resp.ID)
		assert.NotEmpty(t, resp.Rev)
	})

	t.Run("update existing document", func(t *testing.T) {
		// First save
		doc := eve.FlowProcessDocument{
			ProcessID:   "test-002",
			State:       eve.StateStarted,
			Description: "Initial state",
		}

		resp1, err := service.SaveDocument(doc)
		require.NoError(t, err)

		// Update the document
		doc.State = eve.StateRunning
		doc.Description = "Updated state"
		doc.Rev = resp1.Rev

		resp2, err := service.SaveDocument(doc)
		require.NoError(t, err)
		assert.True(t, resp2.OK)
		assert.NotEqual(t, resp1.Rev, resp2.Rev, "Revision should change on update")
	})
}

// TestCouchDBService_Integration_GetDocument tests retrieving a document
func TestCouchDBService_Integration_GetDocument(t *testing.T) {
	url, cleanup := setupCouchDBContainer(t)
	defer cleanup()

	config := eve.FlowConfig{
		CouchDBURL:   url,
		DatabaseName: "test_flows",
	}

	service, err := NewCouchDBService(config)
	require.NoError(t, err)
	defer service.Close()

	t.Run("get existing document", func(t *testing.T) {
		// Create a document first
		doc := eve.FlowProcessDocument{
			ProcessID:   "test-get-001",
			State:       eve.StateSuccessful,
			Description: "Test document for retrieval",
			Metadata: map[string]interface{}{
				"key1": "value1",
				"key2": 42,
			},
		}

		_, err := service.SaveDocument(doc)
		require.NoError(t, err)

		// Retrieve the document
		retrieved, err := service.GetDocument("test-get-001")
		require.NoError(t, err)
		assert.Equal(t, "test-get-001", retrieved.ProcessID)
		assert.Equal(t, eve.StateSuccessful, retrieved.State)
		assert.Equal(t, "Test document for retrieval", retrieved.Description)
		assert.Equal(t, "value1", retrieved.Metadata["key1"])
		assert.Equal(t, float64(42), retrieved.Metadata["key2"]) // JSON numbers are float64
	})

	t.Run("get non-existent document", func(t *testing.T) {
		_, err := service.GetDocument("non-existent-id")
		assert.Error(t, err, "Should return error for non-existent document")
		assert.Contains(t, err.Error(), "not found")
	})
}

// TestCouchDBService_Integration_DeleteDocument tests deleting a document
func TestCouchDBService_Integration_DeleteDocument(t *testing.T) {
	url, cleanup := setupCouchDBContainer(t)
	defer cleanup()

	config := eve.FlowConfig{
		CouchDBURL:   url,
		DatabaseName: "test_flows",
	}

	service, err := NewCouchDBService(config)
	require.NoError(t, err)
	defer service.Close()

	t.Run("delete existing document", func(t *testing.T) {
		// Create a document
		doc := eve.FlowProcessDocument{
			ProcessID:   "test-delete-001",
			State:       eve.StateStarted,
			Description: "Document to be deleted",
		}

		resp, err := service.SaveDocument(doc)
		require.NoError(t, err)

		// Delete the document
		err = service.DeleteDocument("test-delete-001", resp.Rev)
		require.NoError(t, err)

		// Verify it's deleted
		_, err = service.GetDocument("test-delete-001")
		assert.Error(t, err, "Document should not exist after deletion")
	})

	t.Run("delete with wrong revision", func(t *testing.T) {
		// Create a document
		doc := eve.FlowProcessDocument{
			ProcessID:   "test-delete-002",
			State:       eve.StateStarted,
			Description: "Document with wrong revision",
		}

		_, err := service.SaveDocument(doc)
		require.NoError(t, err)

		// Try to delete with wrong revision
		err = service.DeleteDocument("test-delete-002", "wrong-revision")
		assert.Error(t, err, "Should fail with wrong revision")
	})
}

// TestCouchDBService_Integration_GetDocumentsByState tests filtering by state
func TestCouchDBService_Integration_GetDocumentsByState(t *testing.T) {
	url, cleanup := setupCouchDBContainer(t)
	defer cleanup()

	config := eve.FlowConfig{
		CouchDBURL:   url,
		DatabaseName: "test_flows",
	}

	service, err := NewCouchDBService(config)
	require.NoError(t, err)
	defer service.Close()

	// Create multiple documents with different states
	states := []eve.FlowProcessState{
		eve.StateStarted,
		eve.StateRunning,
		eve.StateRunning,
		eve.StateSuccessful,
		eve.StateFailed,
	}

	for i, state := range states {
		doc := eve.FlowProcessDocument{
			ProcessID:   fmt.Sprintf("test-state-%d", i),
			State:       state,
			Description: fmt.Sprintf("Document in state %s", state),
		}
		_, err := service.SaveDocument(doc)
		require.NoError(t, err)
	}

	// Give CouchDB a moment to index
	time.Sleep(100 * time.Millisecond)

	t.Run("filter by running state", func(t *testing.T) {
		docs, err := service.GetDocumentsByState(eve.StateRunning)
		require.NoError(t, err)
		assert.Len(t, docs, 2, "Should find 2 running documents")

		for _, doc := range docs {
			assert.Equal(t, eve.StateRunning, doc.State)
		}
	})

	t.Run("filter by successful state", func(t *testing.T) {
		docs, err := service.GetDocumentsByState(eve.StateSuccessful)
		require.NoError(t, err)
		assert.Len(t, docs, 1, "Should find 1 successful document")
		assert.Equal(t, eve.StateSuccessful, docs[0].State)
	})
}

// TestCouchDBService_Integration_GetAllDocuments tests retrieving all documents
func TestCouchDBService_Integration_GetAllDocuments(t *testing.T) {
	url, cleanup := setupCouchDBContainer(t)
	defer cleanup()

	config := eve.FlowConfig{
		CouchDBURL:   url,
		DatabaseName: "test_flows",
	}

	service, err := NewCouchDBService(config)
	require.NoError(t, err)
	defer service.Close()

	// Create several documents
	for i := 0; i < 5; i++ {
		doc := eve.FlowProcessDocument{
			ProcessID:   fmt.Sprintf("test-all-%d", i),
			State:       eve.StateRunning,
			Description: fmt.Sprintf("Document %d", i),
		}
		_, err := service.SaveDocument(doc)
		require.NoError(t, err)
	}

	// Give CouchDB a moment to index
	time.Sleep(100 * time.Millisecond)

	// Get all documents
	docs, err := service.GetAllDocuments()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(docs), 5, "Should find at least 5 documents")
}

// TestCouchDBService_Integration_DocumentHistory tests audit trail functionality
func TestCouchDBService_Integration_DocumentHistory(t *testing.T) {
	url, cleanup := setupCouchDBContainer(t)
	defer cleanup()

	config := eve.FlowConfig{
		CouchDBURL:   url,
		DatabaseName: "test_flows",
	}

	service, err := NewCouchDBService(config)
	require.NoError(t, err)
	defer service.Close()

	// Create document and update it through multiple states
	doc := eve.FlowProcessDocument{
		ProcessID:   "test-history-001",
		State:       eve.StateStarted,
		Description: "Initial state",
	}

	resp1, err := service.SaveDocument(doc)
	require.NoError(t, err)

	// Update to running
	doc.State = eve.StateRunning
	doc.Description = "Processing"
	doc.Rev = resp1.Rev
	resp2, err := service.SaveDocument(doc)
	require.NoError(t, err)

	// Update to successful
	doc.State = eve.StateSuccessful
	doc.Description = "Completed"
	doc.Rev = resp2.Rev
	_, err = service.SaveDocument(doc)
	require.NoError(t, err)

	// Retrieve and check history
	retrieved, err := service.GetDocument("test-history-001")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(retrieved.History), 3, "Should have at least 3 history entries")

	// Verify state progression in history
	assert.Equal(t, eve.StateSuccessful, retrieved.State, "Final state should be successful")
}
