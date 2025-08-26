package common

import (
	"net/http"
	"time"

	"github.com/streadway/amqp"
)

// ProcessState represents the possible states of a process
type FlowProcessState string

const (
	StateStarted    FlowProcessState = "started"
	StateRunning    FlowProcessState = "running"
	StateSuccessful FlowProcessState = "successful"
	StateFailed     FlowProcessState = "failed"
)

// ProcessMessage represents the incoming RabbitMQ message
type FlowProcessMessage struct {
	ProcessID   string                 `json:"process_id"`
	State       FlowProcessState       `json:"state"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ErrorMsg    string                 `json:"error_message,omitempty"`
	Description string                 `json:"description,omitempty"`
}

// ProcessDocument represents the CouchDB document structure
type FlowProcessDocument struct {
	ID          string                 `json:"_id"`
	Rev         string                 `json:"_rev,omitempty"`
	ProcessID   string                 `json:"process_id"`
	State       FlowProcessState       `json:"state"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	History     []FlowStateChange      `json:"history"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ErrorMsg    string                 `json:"error_message,omitempty"`
	Description string                 `json:"description,omitempty"`
}

// StateChange represents a state transition in the history
type FlowStateChange struct {
	State     FlowProcessState `json:"state"`
	Timestamp time.Time        `json:"timestamp"`
	ErrorMsg  string           `json:"error_message,omitempty"`
}

// CouchDBResponse represents the response from CouchDB operations
type FlowCouchDBResponse struct {
	OK  bool   `json:"ok"`
	ID  string `json:"id"`
	Rev string `json:"rev"`
}

// CouchDBError represents an error response from CouchDB
type FlowCouchDBError struct {
	Error  string `json:"error"`
	Reason string `json:"reason"`
}

// Config holds the application configuration
type FlowConfig struct {
	RabbitMQURL  string
	QueueName    string
	CouchDBURL   string
	DatabaseName string
	ApiKey       string
}

// Consumer handles the RabbitMQ consumer logic
type FlowConsumer struct {
	config     FlowConfig
	connection *amqp.Connection
	channel    *amqp.Channel
	httpClient *http.Client
}
