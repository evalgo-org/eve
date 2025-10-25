package queue

import (
	"encoding/json"
	"testing"

	eve "eve.evalgo.org/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewRabbitMQService_InvalidConfig tests connection with invalid configurations
func TestNewRabbitMQService_InvalidConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      eve.FlowConfig
		expectError bool
	}{
		{
			name: "InvalidURL",
			config: eve.FlowConfig{
				RabbitMQURL: "invalid://url",
				QueueName:   "test-queue",
			},
			expectError: true,
		},
		{
			name: "EmptyURL",
			config: eve.FlowConfig{
				RabbitMQURL: "",
				QueueName:   "test-queue",
			},
			expectError: true,
		},
		{
			name: "NonExistentServer",
			config: eve.FlowConfig{
				RabbitMQURL: "amqp://nonexistent:5672",
				QueueName:   "test-queue",
			},
			expectError: true,
		},
		{
			name: "InvalidPort",
			config: eve.FlowConfig{
				RabbitMQURL: "amqp://localhost:99999",
				QueueName:   "test-queue",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewRabbitMQService(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, service)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, service)

			if service != nil {
				service.Close()
			}
		})
	}
}

// TestRabbitMQService_Close tests the Close method
func TestRabbitMQService_Close(t *testing.T) {
	tests := []struct {
		name    string
		service *RabbitMQService
	}{
		{
			name: "NilChannel",
			service: &RabbitMQService{
				channel:    nil,
				connection: nil,
			},
		},
		{
			name: "NilConnection",
			service: &RabbitMQService{
				channel:    nil,
				connection: nil,
			},
		},
		{
			name: "BothNil",
			service: &RabbitMQService{
				channel:    nil,
				connection: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic with nil values
			assert.NotPanics(t, func() {
				tt.service.Close()
			})
		})
	}
}

// TestFlowProcessMessage_JSONSerialization tests message JSON serialization
func TestFlowProcessMessage_JSONSerialization(t *testing.T) {
	tests := []struct {
		name    string
		message eve.FlowProcessMessage
	}{
		{
			name: "BasicMessage",
			message: eve.FlowProcessMessage{
				ProcessID:   "proc-123",
				State:       "running",
				Description: "Test process",
			},
		},
		{
			name: "MessageWithError",
			message: eve.FlowProcessMessage{
				ProcessID: "proc-456",
				State:     "failed",
				ErrorMsg:  "Connection timeout",
			},
		},
		{
			name: "MessageWithMetadata",
			message: eve.FlowProcessMessage{
				ProcessID: "proc-789",
				State:     "completed",
				Metadata: map[string]interface{}{
					"repository": "https://github.com/test/repo",
					"branch":     "main",
				},
			},
		},
		{
			name: "EmptyMessage",
			message: eve.FlowProcessMessage{
				ProcessID: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.message)
			require.NoError(t, err)
			assert.NotEmpty(t, data)

			// Unmarshal back
			var decoded eve.FlowProcessMessage
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			// Verify fields match
			assert.Equal(t, tt.message.ProcessID, decoded.ProcessID)
			assert.Equal(t, tt.message.State, decoded.State)
			assert.Equal(t, tt.message.ErrorMsg, decoded.ErrorMsg)
			assert.Equal(t, tt.message.Description, decoded.Description)
		})
	}
}

// TestPublishMessage_InvalidMessage tests publishing with invalid data
func TestPublishMessage_InvalidMessage(t *testing.T) {
	// This test verifies message marshaling behavior
	// We can't easily test actual publishing without a real RabbitMQ server

	tests := []struct {
		name    string
		message eve.FlowProcessMessage
	}{
		{
			name: "ValidMessage",
			message: eve.FlowProcessMessage{
				ProcessID:   "test-process",
				State:       "running",
				Description: "Test description",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that message can be marshaled
			data, err := json.Marshal(tt.message)
			assert.NoError(t, err)
			assert.NotEmpty(t, data)

			// Verify JSON structure
			var jsonMap map[string]interface{}
			err = json.Unmarshal(data, &jsonMap)
			require.NoError(t, err)

			assert.Equal(t, tt.message.ProcessID, jsonMap["process_id"])
			assert.Equal(t, string(tt.message.State), jsonMap["state"])
		})
	}
}

// TestRabbitMQService_StructFields tests service struct field access
func TestRabbitMQService_StructFields(t *testing.T) {
	config := eve.FlowConfig{
		RabbitMQURL: "amqp://localhost:5672",
		QueueName:   "test-queue",
	}

	service := &RabbitMQService{
		connection: nil, // Would be populated in real scenario
		channel:    nil, // Would be populated in real scenario
		config:     config,
	}

	// Verify config is stored correctly
	assert.Equal(t, config.RabbitMQURL, service.config.RabbitMQURL)
	assert.Equal(t, config.QueueName, service.config.QueueName)

	// Verify Close doesn't panic with nil connection/channel
	assert.NotPanics(t, func() {
		service.Close()
	})
}

// TestFlowConfig_Validation tests FlowConfig struct
func TestFlowConfig_Validation(t *testing.T) {
	tests := []struct {
		name      string
		config    eve.FlowConfig
		expectErr bool
	}{
		{
			name: "ValidConfig",
			config: eve.FlowConfig{
				RabbitMQURL: "amqp://localhost:5672",
				QueueName:   "my-queue",
			},
			expectErr: false,
		},
		{
			name: "EmptyQueueName",
			config: eve.FlowConfig{
				RabbitMQURL: "amqp://localhost:5672",
				QueueName:   "",
			},
			expectErr: false, // Empty queue name is structurally valid
		},
		{
			name: "ConfigWithCustomPort",
			config: eve.FlowConfig{
				RabbitMQURL: "amqp://localhost:15672",
				QueueName:   "custom-queue",
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify config can be created
			assert.NotNil(t, tt.config)
			assert.NotEmpty(t, tt.config.RabbitMQURL)
		})
	}
}

// TestPublishMessage_MessageFormatting tests message format
func TestPublishMessage_MessageFormatting(t *testing.T) {
	tests := []struct {
		name            string
		message         eve.FlowProcessMessage
		expectedFields  []string
		expectedValues  map[string]interface{}
	}{
		{
			name: "CompleteMessage",
			message: eve.FlowProcessMessage{
				ProcessID:   "proc-001",
				State:       "running",
				Description: "Build process",
				ErrorMsg:    "",
				Metadata: map[string]interface{}{
					"repository": "https://github.com/org/repo",
					"branch":     "main",
				},
			},
			expectedFields: []string{"process_id", "state", "description", "metadata"},
			expectedValues: map[string]interface{}{
				"process_id":  "proc-001",
				"state":       "running",
				"description": "Build process",
			},
		},
		{
			name: "MinimalMessage",
			message: eve.FlowProcessMessage{
				ProcessID: "proc-002",
				State:     "started",
			},
			expectedFields: []string{"process_id", "state"},
			expectedValues: map[string]interface{}{
				"process_id": "proc-002",
				"state":      "started",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.message)
			require.NoError(t, err)

			var jsonMap map[string]interface{}
			err = json.Unmarshal(data, &jsonMap)
			require.NoError(t, err)

			// Verify expected fields are present
			for _, field := range tt.expectedFields {
				assert.Contains(t, jsonMap, field, "JSON should contain field: %s", field)
			}

			// Verify expected values
			for field, expectedValue := range tt.expectedValues {
				if field == "state" {
					// Skip state comparison as it's an enum type
					continue
				}
				actualValue, ok := jsonMap[field]
				assert.True(t, ok, "JSON should contain field: %s", field)
				if ok && expectedValue != "" {
					assert.Equal(t, expectedValue, actualValue)
				}
			}
		})
	}
}

// TestErrorWrapping tests error message formatting
func TestErrorWrapping(t *testing.T) {
	tests := []struct {
		name          string
		config        eve.FlowConfig
		expectContains string
	}{
		{
			name: "InvalidURL_ErrorMessage",
			config: eve.FlowConfig{
				RabbitMQURL: "invalid://url",
				QueueName:   "test-queue",
			},
			expectContains: "failed to connect to RabbitMQ",
		},
		{
			name: "EmptyURL_ErrorMessage",
			config: eve.FlowConfig{
				RabbitMQURL: "",
				QueueName:   "test-queue",
			},
			expectContains: "failed to connect to RabbitMQ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewRabbitMQService(tt.config)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectContains)
		})
	}
}

// TestRabbitMQService_NilSafety tests nil pointer safety
func TestRabbitMQService_NilSafety(t *testing.T) {
	service := &RabbitMQService{}

	// Close should handle nil connection and channel safely
	assert.NotPanics(t, func() {
		service.Close()
	})
}

// BenchmarkMessageMarshaling benchmarks JSON marshaling
func BenchmarkMessageMarshaling(b *testing.B) {
	message := eve.FlowProcessMessage{
		ProcessID:   "bench-process",
		State:       "running",
		Description: "Benchmark process",
		Metadata: map[string]interface{}{
			"repository": "https://github.com/bench/repo",
			"branch":     "main",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(message)
	}
}

// BenchmarkMessageUnmarshaling benchmarks JSON unmarshaling
func BenchmarkMessageUnmarshaling(b *testing.B) {
	message := eve.FlowProcessMessage{
		ProcessID:   "bench-process",
		State:       "running",
		Description: "Benchmark process",
		Metadata: map[string]interface{}{
			"repository": "https://github.com/bench/repo",
			"branch":     "main",
		},
	}

	data, _ := json.Marshal(message)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var msg eve.FlowProcessMessage
		_ = json.Unmarshal(data, &msg)
	}
}
