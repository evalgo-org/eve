package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestURLToFilePath tests URL to filesystem path conversion
func TestURLToFilePath(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "HTTPSWithPath",
			url:      "https://example.com/path/to/resource",
			expected: "example.com_path_to_resource",
		},
		{
			name:     "HTTPWithPath",
			url:      "http://api.service.com/v1/users",
			expected: "api.service.com_v1_users",
		},
		{
			name:     "NoProtocol",
			url:      "example.com/docs/guide.html",
			expected: "example.com_docs_guide.html",
		},
		{
			name:     "HTTPSSimple",
			url:      "https://example.com",
			expected: "example.com",
		},
		{
			name:     "HTTPSimple",
			url:      "http://example.com",
			expected: "example.com",
		},
		{
			name:     "ComplexPath",
			url:      "https://api.example.com/v2/users/123/profile",
			expected: "api.example.com_v2_users_123_profile",
		},
		{
			name:     "WithPort",
			url:      "https://localhost:8080/api/test",
			expected: "localhost:8080_api_test",
		},
		{
			name:     "OtherProtocol",
			url:      "ftp://files.example.com/data",
			expected: "ftp:__files.example.com_data",
		},
		{
			name:     "TrailingSlash",
			url:      "https://example.com/path/",
			expected: "example.com_path_",
		},
		{
			name:     "MultipleSlashes",
			url:      "https://example.com//path//to///resource",
			expected: "example.com__path__to___resource",
		},
		{
			name:     "DomainOnly",
			url:      "https://example.com",
			expected: "example.com",
		},
		{
			name:     "EmptyString",
			url:      "",
			expected: "",
		},
		{
			name:     "QueryParameters",
			url:      "https://example.com/search?q=test&page=1",
			expected: "example.com_search?q=test&page=1",
		},
		{
			name:     "Fragment",
			url:      "https://example.com/docs#section1",
			expected: "example.com_docs#section1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := URLToFilePath(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestFlowConfig tests FlowConfig struct
func TestFlowConfig(t *testing.T) {
	config := FlowConfig{
		RabbitMQURL:  "amqp://localhost:5672",
		QueueName:    "test-queue",
		CouchDBURL:   "http://localhost:5984",
		DatabaseName: "test-db",
		ApiKey:       "test-api-key",
	}

	assert.Equal(t, "amqp://localhost:5672", config.RabbitMQURL)
	assert.Equal(t, "test-queue", config.QueueName)
	assert.Equal(t, "http://localhost:5984", config.CouchDBURL)
	assert.Equal(t, "test-db", config.DatabaseName)
	assert.Equal(t, "test-api-key", config.ApiKey)
}

// TestFlowProcessState tests FlowProcessState type
func TestFlowProcessState(t *testing.T) {
	tests := []struct {
		name  string
		state FlowProcessState
	}{
		{
			name:  "StartedState",
			state: FlowProcessState("started"),
		},
		{
			name:  "RunningState",
			state: FlowProcessState("running"),
		},
		{
			name:  "SuccessfulState",
			state: FlowProcessState("successful"),
		},
		{
			name:  "FailedState",
			state: FlowProcessState("failed"),
		},
		{
			name:  "CustomState",
			state: FlowProcessState("custom"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.state)
			assert.IsType(t, FlowProcessState(""), tt.state)
		})
	}
}

// TestFlowProcessMessage tests FlowProcessMessage struct
func TestFlowProcessMessage(t *testing.T) {
	message := FlowProcessMessage{
		ProcessID:   "proc-123",
		State:       FlowProcessState("running"),
		Description: "Test process",
		Metadata: map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		},
	}

	assert.Equal(t, "proc-123", message.ProcessID)
	assert.Equal(t, FlowProcessState("running"), message.State)
	assert.Equal(t, "Test process", message.Description)
	assert.Contains(t, message.Metadata, "key1")
	assert.Contains(t, message.Metadata, "key2")
	assert.Equal(t, "value1", message.Metadata["key1"])
	assert.Equal(t, 42, message.Metadata["key2"])
}

// TestFlowProcessDocument tests FlowProcessDocument struct
func TestFlowProcessDocument(t *testing.T) {
	doc := FlowProcessDocument{
		ID:        "doc-123",
		Rev:       "1-abc",
		ProcessID: "proc-123",
		State:     FlowProcessState("running"),
	}

	assert.Equal(t, "doc-123", doc.ID)
	assert.Equal(t, "1-abc", doc.Rev)
	assert.Equal(t, "proc-123", doc.ProcessID)
	assert.Equal(t, FlowProcessState("running"), doc.State)
}


// BenchmarkURLToFilePath benchmarks URL conversion
func BenchmarkURLToFilePath(b *testing.B) {
	url := "https://api.example.com/v1/users/123/profile"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = URLToFilePath(url)
	}
}

// BenchmarkURLToFilePath_Short benchmarks short URL conversion
func BenchmarkURLToFilePath_Short(b *testing.B) {
	url := "https://example.com"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = URLToFilePath(url)
	}
}

// BenchmarkURLToFilePath_Long benchmarks long URL conversion
func BenchmarkURLToFilePath_Long(b *testing.B) {
	url := "https://api.example.com/v1/users/123/profile/settings/preferences/notifications/email"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = URLToFilePath(url)
	}
}

// TestURLToFilePath_EdgeCases tests edge cases
func TestURLToFilePath_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "OnlyHTTPS",
			url:      "https://",
			expected: "",
		},
		{
			name:     "OnlyHTTP",
			url:      "http://",
			expected: "",
		},
		{
			name:     "SingleSlash",
			url:      "/",
			expected: "_",
		},
		{
			name:     "MultipleProtocols",
			url:      "https://http://example.com",
			expected: "example.com",
		},
		{
			name:     "MixedCase",
			url:      "HTTPS://EXAMPLE.COM/PATH",
			expected: "HTTPS:__EXAMPLE.COM_PATH",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := URLToFilePath(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestFlowProcessMessage_EmptyMetadata tests message with no metadata
func TestFlowProcessMessage_EmptyMetadata(t *testing.T) {
	message := FlowProcessMessage{
		ProcessID:   "proc-456",
		State:       FlowProcessState("started"),
		Description: "",
		Metadata:    nil,
	}

	assert.Equal(t, "proc-456", message.ProcessID)
	assert.Nil(t, message.Metadata)
	assert.Empty(t, message.Description)
}

// TestFlowProcessMessage_WithError tests message with error
func TestFlowProcessMessage_WithError(t *testing.T) {
	message := FlowProcessMessage{
		ProcessID:   "proc-789",
		State:       FlowProcessState("failed"),
		ErrorMsg:    "Connection timeout",
		Description: "Failed to connect to database",
	}

	assert.Equal(t, "proc-789", message.ProcessID)
	assert.Equal(t, FlowProcessState("failed"), message.State)
	assert.Equal(t, "Connection timeout", message.ErrorMsg)
	assert.Equal(t, "Failed to connect to database", message.Description)
}
