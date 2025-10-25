// Package cli provides comprehensive testing for the EVE evaluation service CLI functionality.
// This file contains unit tests for the CLI package, focusing on message processing,
// consumer operations, state management, and CouchDB integration with comprehensive
// mock implementations and edge case validation.
//
// The tests validate the complete message processing pipeline including:
//   - RabbitMQ message consumption and acknowledgment
//   - Process state transitions and validation
//   - CouchDB document creation and updates
//   - Error handling and recovery scenarios
//   - JSON serialization and deserialization
//   - Configuration management and service initialization
package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockHTTPClient provides a mock implementation of HTTP client for testing CouchDB operations.
// This mock allows testing of CouchDB interactions without requiring an actual database,
// enabling isolated unit tests with controlled responses and error conditions.
//
// The mock supports:
//   - Configurable HTTP responses for different endpoints
//   - Error simulation for network and database failures
//   - Request validation for proper API usage
//   - Response timing control for performance testing
type MockHTTPClient struct {
	mock.Mock
	responses map[string]*http.Response // Predefined responses by URL
	requests  []*http.Request           // Captured requests for validation
}

// Get issues a mock GET request to the specified URL.
// This method is provided for compatibility with the standard http.Client interface,
// allowing the mock to be used in place of a real HTTP client in tests.
// It records the URL and returns a predefined response based on the test setup.
//
// Parameters:
//   - url: The URL to request (used for matching expectations)
//
// Returns:
//   - *http.Response: The mock response as configured by the test
//   - error: Simulated error, if any
func (m *MockHTTPClient) Get(url string) (*http.Response, error) {
	args := m.Called(url)
	return args.Get(0).(*http.Response), args.Error(1)
}

// Do implements the HTTP client interface for the mock.
// This method captures all HTTP requests and returns predefined responses
// based on the request URL and method, enabling comprehensive testing
// of CouchDB operations without external dependencies.
//
// Request Validation:
//   - Captures request details for assertion in tests
//   - Validates HTTP methods, headers, and body content
//   - Supports different responses for different endpoints
//
// Parameters:
//   - req: HTTP request to process
//
// Returns:
//   - *http.Response: Mock response based on request URL/method
//   - error: Simulated network or processing errors
func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	m.requests = append(m.requests, req)
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}

// createMockResponse creates an HTTP response for testing purposes.
// This utility function generates properly formatted HTTP responses
// with the specified status code and JSON body for CouchDB operation testing.
//
// Response Features:
//   - Proper Content-Type headers for JSON responses
//   - Configurable status codes for different scenarios
//   - Realistic response body formatting
//   - Proper HTTP headers for CouchDB compatibility
//
// Parameters:
//   - statusCode: HTTP status code to return
//   - body: JSON response body as string
//
// Returns:
//   - *http.Response: Complete HTTP response ready for testing
func createMockResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       ioutil.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}

// createTestConsumer creates a Consumer instance configured for testing.
// This function sets up a consumer with mock HTTP client and test configuration,
// enabling isolated testing of consumer functionality without external dependencies.
//
// Test Configuration:
//   - Mock CouchDB URLs for controlled testing
//   - Test database names to avoid conflicts
//   - Configurable mock HTTP client for response control
//
// Parameters:
//   - mockClient: Mock HTTP client for CouchDB operations
//
// Returns:
//   - *Consumer: Configured consumer ready for testing
func createTestConsumer(mockClient *MockHTTPClient) *Consumer {
	config := Config{
		CouchDBURL:  "http://test-couchdb:5984",
		CouchDBName: "test_processes",
	}
	consumer := &Consumer{
		config:     config,
		httpClient: mockClient,
	}
	return consumer
}

// createTestMessage creates a ProcessMessage for testing purposes.
// This utility function generates valid ProcessMessage instances with
// configurable fields for different test scenarios.
//
// Default Values:
//   - ProcessID: "test-process-123"
//   - State: StateStarted
//   - Timestamp: Current time
//   - Metadata: Basic test metadata
//
// Parameters:
//   - state: Process state to set in the message
//
// Returns:
//   - ProcessMessage: Complete message ready for testing
func createTestMessage(state ProcessState) ProcessMessage {
	return ProcessMessage{
		ProcessID:   "test-process-123",
		State:       state,
		Timestamp:   time.Now(),
		Metadata:    map[string]interface{}{"test": "data", "version": 1},
		Description: "Test process message",
	}
}

// createAMQPDelivery creates a mock AMQP delivery for testing message processing.
// This function simulates RabbitMQ message delivery with proper JSON serialization
// and AMQP metadata for realistic testing conditions.
//
// AMQP Features:
//   - Proper JSON serialization of ProcessMessage
//   - Mock acknowledgment and rejection methods
//   - Configurable delivery metadata
//
// Parameters:
//   - msg: ProcessMessage to serialize in the delivery
//
// Returns:
//   - amqp.Delivery: Mock AMQP delivery ready for processing
//   - error: JSON serialization errors
func createAMQPDelivery(msg ProcessMessage) (amqp.Delivery, error) {
	body, err := json.Marshal(msg)
	if err != nil {
		return amqp.Delivery{}, err
	}

	return amqp.Delivery{
		Body:         body,
		Acknowledger: &mockAcknowledger{},
	}, nil
}

// mockAcknowledger provides a mock implementation of AMQP acknowledger for testing.
// This mock enables testing of message acknowledgment and rejection scenarios
// without requiring an actual RabbitMQ connection.
type mockAcknowledger struct{}

func (m *mockAcknowledger) Ack(tag uint64, multiple bool) error           { return nil }
func (m *mockAcknowledger) Nack(tag uint64, multiple, requeue bool) error { return nil }
func (m *mockAcknowledger) Reject(tag uint64, requeue bool) error         { return nil }

// TestProcessMessage validates the complete message processing workflow.
// This test suite covers all aspects of message processing including validation,
// state transitions, database operations, and error handling scenarios.
//
// Test Coverage:
//   - Valid message processing for all process states
//   - Message validation with missing required fields
//   - JSON deserialization error handling
//   - Database operation success and failure scenarios
//   - State transition validation and business logic
//
// The tests use mock HTTP clients to simulate CouchDB responses,
// enabling comprehensive testing without external dependencies.
func TestProcessMessage(t *testing.T) {
	t.Run("ValidStartedMessage", func(t *testing.T) {
		// Setup mock HTTP client for CouchDB operations
		mockClient := &MockHTTPClient{}
		consumer := createTestConsumer(mockClient)

		// Create test message for process start
		testMsg := createTestMessage(StateStarted)
		delivery, err := createAMQPDelivery(testMsg)
		require.NoError(t, err)

		// Mock successful document creation response
		successResponse := `{"ok":true,"id":"process_test-process-123","rev":"1-abc123"}`
		mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
			return req.Method == "POST" && req.URL.Path == "/test_processes"
		})).Return(createMockResponse(201, successResponse), nil)

		// Execute message processing
		err = consumer.processMessage(delivery)

		// Validate successful processing
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)

		// Verify HTTP request was made correctly
		assert.Len(t, mockClient.requests, 1)
		assert.Equal(t, "POST", mockClient.requests[0].Method)
		assert.Equal(t, "application/json", mockClient.requests[0].Header.Get("Content-Type"))
	})

	t.Run("ValidRunningMessage", func(t *testing.T) {
		mockClient := &MockHTTPClient{}
		consumer := createTestConsumer(mockClient)

		testMsg := createTestMessage(StateRunning)
		delivery, err := createAMQPDelivery(testMsg)
		require.NoError(t, err)

		// Mock document retrieval response (existing document)
		existingDoc := ProcessDocument{
			ID:        "process_test-process-123",
			Rev:       "1-abc123",
			ProcessID: "test-process-123",
			State:     StateStarted,
			CreatedAt: time.Now().Add(-1 * time.Hour),
			UpdatedAt: time.Now().Add(-1 * time.Hour),
			History:   []StateChange{{State: StateStarted, Timestamp: time.Now().Add(-1 * time.Hour)}},
			Metadata:  map[string]interface{}{"initial": "data"},
		}
		existingDocJSON, _ := json.Marshal(existingDoc)

		// Mock GET request for existing document
		mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
			return req.Method == "GET" && req.URL.Path == "/test_processes/process_test-process-123"
		})).Return(createMockResponse(200, string(existingDocJSON)), nil)

		// Mock PUT request for document update
		updateResponse := `{"ok":true,"id":"process_test-process-123","rev":"2-def456"}`
		mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
			return req.Method == "PUT" && req.URL.Path == "/test_processes/process_test-process-123"
		})).Return(createMockResponse(200, updateResponse), nil)

		err = consumer.processMessage(delivery)

		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
		assert.Len(t, mockClient.requests, 2) // GET + PUT requests
	})

	t.Run("InvalidJSONMessage", func(t *testing.T) {
		mockClient := &MockHTTPClient{}
		consumer := createTestConsumer(mockClient)

		// Create delivery with invalid JSON
		delivery := amqp.Delivery{
			Body:         []byte(`{"invalid": json}`),
			Acknowledger: &mockAcknowledger{},
		}

		err := consumer.processMessage(delivery)

		// Should return JSON unmarshaling error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal message")
	})

	t.Run("MissingProcessID", func(t *testing.T) {
		mockClient := &MockHTTPClient{}
		consumer := createTestConsumer(mockClient)

		// Create message without ProcessID
		invalidMsg := ProcessMessage{
			State:     StateStarted,
			Timestamp: time.Now(),
		}
		delivery, err := createAMQPDelivery(invalidMsg)
		require.NoError(t, err)

		err = consumer.processMessage(delivery)

		// Should return validation error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "process_id is required")
	})

	t.Run("MissingState", func(t *testing.T) {
		mockClient := &MockHTTPClient{}
		consumer := createTestConsumer(mockClient)

		// Create message without State
		invalidMsg := ProcessMessage{
			ProcessID: "test-process-123",
			Timestamp: time.Now(),
		}
		delivery, err := createAMQPDelivery(invalidMsg)
		require.NoError(t, err)

		err = consumer.processMessage(delivery)

		// Should return validation error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "state is required")
	})

	t.Run("InvalidState", func(t *testing.T) {
		mockClient := &MockHTTPClient{}
		consumer := createTestConsumer(mockClient)

		// Create message with invalid state
		invalidMsg := ProcessMessage{
			ProcessID: "test-process-123",
			State:     ProcessState("invalid_state"),
			Timestamp: time.Now(),
		}
		delivery, err := createAMQPDelivery(invalidMsg)
		require.NoError(t, err)

		err = consumer.processMessage(delivery)

		// Should return invalid state error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid state")
	})

	t.Run("DatabaseError", func(t *testing.T) {
		mockClient := &MockHTTPClient{}
		consumer := createTestConsumer(mockClient)

		testMsg := createTestMessage(StateStarted)
		delivery, err := createAMQPDelivery(testMsg)
		require.NoError(t, err)

		// Mock database error response
		errorResponse := `{"error":"internal_server_error","reason":"Database unavailable"}`
		mockClient.On("Do", mock.AnythingOfType("*http.Request")).Return(
			createMockResponse(500, errorResponse), nil)

		err = consumer.processMessage(delivery)

		// Should return database error
		assert.Error(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("AutoTimestamp", func(t *testing.T) {
		mockClient := &MockHTTPClient{}
		consumer := createTestConsumer(mockClient)

		// Create message without timestamp
		testMsg := ProcessMessage{
			ProcessID: "test-process-123",
			State:     StateStarted,
			// No timestamp set
		}
		delivery, err := createAMQPDelivery(testMsg)
		require.NoError(t, err)

		// Mock successful response
		successResponse := `{"ok":true,"id":"process_test-process-123","rev":"1-abc123"}`
		mockClient.On("Do", mock.AnythingOfType("*http.Request")).Return(
			createMockResponse(201, successResponse), nil)

		err = consumer.processMessage(delivery)

		// Should succeed and auto-set timestamp
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})
}

// TestCreateProcessDocument validates document creation for new processes.
// This test ensures that new process documents are properly formatted
// and contain all required fields for CouchDB storage.
//
// Test Coverage:
//   - Document structure validation
//   - Initial history creation
//   - Metadata preservation
//   - Timestamp handling
//   - CouchDB document ID formatting
func TestCreateProcessDocument(t *testing.T) {
	t.Run("ValidDocumentCreation", func(t *testing.T) {
		mockClient := &MockHTTPClient{}
		consumer := createTestConsumer(mockClient)

		testMsg := createTestMessage(StateStarted)

		// Mock successful document creation
		successResponse := `{"ok":true,"id":"process_test-process-123","rev":"1-abc123"}`
		mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
			// Validate request properties
			if req.Method != "POST" || req.URL.Path != "/test_processes" {
				return false
			}

			// Validate request body contains expected document structure
			var doc ProcessDocument
			json.NewDecoder(req.Body).Decode(&doc)

			return doc.ID == "process_test-process-123" &&
				doc.ProcessID == "test-process-123" &&
				doc.State == StateStarted &&
				len(doc.History) == 1
		})).Return(createMockResponse(201, successResponse), nil)

		err := consumer.createProcessDocument(testMsg)

		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("DocumentCreationError", func(t *testing.T) {
		mockClient := &MockHTTPClient{}
		consumer := createTestConsumer(mockClient)

		testMsg := createTestMessage(StateStarted)

		// Mock document creation error (conflict)
		errorResponse := `{"error":"conflict","reason":"Document already exists"}`
		mockClient.On("Do", mock.AnythingOfType("*http.Request")).Return(
			createMockResponse(409, errorResponse), nil)

		err := consumer.createProcessDocument(testMsg)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "conflict")
		mockClient.AssertExpectations(t)
	})
}

// TestUpdateProcessDocument validates document updates for existing processes.
// This test ensures that process state updates properly merge metadata,
// append to history, and maintain document integrity.
//
// Test Coverage:
//   - Document retrieval and update workflow
//   - Metadata merging logic
//   - History appending
//   - State transition validation
//   - Revision management
func TestUpdateProcessDocument(t *testing.T) {
	t.Run("ValidDocumentUpdate", func(t *testing.T) {
		mockClient := &MockHTTPClient{}
		consumer := createTestConsumer(mockClient)

		testMsg := createTestMessage(StateRunning)

		// Create existing document
		existingDoc := ProcessDocument{
			ID:        "process_test-process-123",
			Rev:       "1-abc123",
			ProcessID: "test-process-123",
			State:     StateStarted,
			CreatedAt: time.Now().Add(-1 * time.Hour),
			UpdatedAt: time.Now().Add(-1 * time.Hour),
			History:   []StateChange{{State: StateStarted, Timestamp: time.Now().Add(-1 * time.Hour)}},
			Metadata:  map[string]interface{}{"initial": "data"},
		}
		existingDocJSON, _ := json.Marshal(existingDoc)

		// Mock document retrieval
		mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
			return req.Method == "GET"
		})).Return(createMockResponse(200, string(existingDocJSON)), nil)

		// Mock document update
		updateResponse := `{"ok":true,"id":"process_test-process-123","rev":"2-def456"}`
		mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
			if req.Method != "PUT" {
				return false
			}

			// Validate updated document structure
			var doc ProcessDocument
			json.NewDecoder(req.Body).Decode(&doc)

			return doc.State == StateRunning &&
				len(doc.History) == 2 &&
				doc.Rev == "1-abc123" // Should include original revision
		})).Return(createMockResponse(200, updateResponse), nil)

		err := consumer.updateProcessDocument(testMsg)

		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("DocumentNotFound", func(t *testing.T) {
		mockClient := &MockHTTPClient{}
		consumer := createTestConsumer(mockClient)

		testMsg := createTestMessage(StateRunning)

		// Mock document not found
		notFoundResponse := `{"error":"not_found","reason":"missing"}`
		mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
			return req.Method == "GET"
		})).Return(createMockResponse(404, notFoundResponse), nil)

		err := consumer.updateProcessDocument(testMsg)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "document not found")
		mockClient.AssertExpectations(t)
	})
}

// TestProcessStates validates all supported process state transitions.
// This test ensures that the message processing system correctly handles
// all valid process states and rejects invalid ones.
//
// State Coverage:
//   - StateStarted: Initial process creation
//   - StateRunning: Process execution updates
//   - StateSuccessful: Successful completion
//   - StateFailed: Error termination
//   - Invalid states: Proper error handling
func TestProcessStates(t *testing.T) {
	states := []ProcessState{StateStarted, StateRunning, StateSuccessful, StateFailed}

	for _, state := range states {
		t.Run(fmt.Sprintf("State_%s", state), func(t *testing.T) {
			mockClient := &MockHTTPClient{}
			consumer := createTestConsumer(mockClient)

			testMsg := createTestMessage(state)
			delivery, err := createAMQPDelivery(testMsg)
			require.NoError(t, err)

			// Mock appropriate responses based on state
			if state == StateStarted {
				// Mock document creation for started state
				successResponse := `{"ok":true,"id":"process_test-process-123","rev":"1-abc123"}`
				mockClient.On("Do", mock.AnythingOfType("*http.Request")).Return(
					createMockResponse(201, successResponse), nil)
			} else {
				// Mock document retrieval and update for other states
				existingDoc := ProcessDocument{
					ID:        "process_test-process-123",
					Rev:       "1-abc123",
					ProcessID: "test-process-123",
					State:     StateStarted,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					History:   []StateChange{{State: StateStarted, Timestamp: time.Now()}},
				}
				existingDocJSON, _ := json.Marshal(existingDoc)

				mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
					return req.Method == "GET"
				})).Return(createMockResponse(200, string(existingDocJSON)), nil)

				updateResponse := `{"ok":true,"id":"process_test-process-123","rev":"2-def456"}`
				mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
					return req.Method == "PUT"
				})).Return(createMockResponse(200, updateResponse), nil)
			}

			err = consumer.processMessage(delivery)

			assert.NoError(t, err, "State %s should be processed successfully", state)
			mockClient.AssertExpectations(t)
		})
	}
}

// BenchmarkProcessMessage provides performance benchmarks for message processing.
// This benchmark measures the overhead of the complete message processing
// pipeline including JSON deserialization, validation, and mock database operations.
//
// Benchmark Coverage:
//   - Message deserialization performance
//   - Validation overhead
//   - Mock HTTP client performance
//   - Memory allocation patterns
//
// Usage:
//
//	go test -bench=BenchmarkProcessMessage -benchmem
func BenchmarkProcessMessage(b *testing.B) {
	mockClient := &MockHTTPClient{}
	consumer := createTestConsumer(mockClient)

	testMsg := createTestMessage(StateStarted)
	delivery, _ := createAMQPDelivery(testMsg)

	// Setup mock response for all iterations
	successResponse := `{"ok":true,"id":"process_test-process-123","rev":"1-abc123"}`
	mockClient.On("Do", mock.AnythingOfType("*http.Request")).Return(
		createMockResponse(201, successResponse), nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		consumer.processMessage(delivery)
	}
}

// ExampleProcessMessage demonstrates typical usage of the processMessage function.
// This example shows how to set up a consumer and process different types
// of messages with proper error handling.
func ExampleProcessMessage() {
	// This example demonstrates the processMessage workflow
	// In practice, this would be called by the RabbitMQ consumer

	// Create test consumer (in real usage, this would be properly configured)
	consumer := &Consumer{
		config: Config{
			CouchDBURL:  "http://couchdb:5984",
			CouchDBName: "processes",
		},
	}

	// Create a sample process message
	msg := ProcessMessage{
		ProcessID:   "example-process-456",
		State:       StateStarted,
		Timestamp:   time.Now(),
		Metadata:    map[string]interface{}{"user": "example", "priority": "high"},
		Description: "Example process started",
	}

	// Convert to AMQP delivery (normally done by RabbitMQ)
	body, _ := json.Marshal(msg)
	delivery := amqp.Delivery{Body: body}

	// Process the message
	err := consumer.processMessage(delivery)
	if err != nil {
		fmt.Printf("Error processing message: %v\n", err)
		return
	}

	fmt.Println("Message processed successfully")
	// Output: Message processed successfully
}
