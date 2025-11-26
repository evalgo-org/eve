// Package coordinator provides WebSocket-based coordination between services and when-v3.
// It handles phase management, health tracking, and real-time communication for
// distributed workflow execution.
package coordinator

import (
	"encoding/json"
	"time"
)

// MessageType defines the types of WebSocket messages exchanged between services and when-v3.
type MessageType string

const (
	// Service → when-v3 messages
	MessageTypeRegister        MessageType = "register"
	MessageTypeWorkflowCreated MessageType = "workflow_created"
	MessageTypePhaseChanged    MessageType = "phase_changed"
	MessageTypeCheckpoint      MessageType = "checkpoint"
	MessageTypeError           MessageType = "error"
	MessageTypeStatusResponse  MessageType = "status_response"
	MessageTypePong            MessageType = "pong"
	MessageTypeProgress        MessageType = "progress"

	// when-v3 → Service messages
	MessageTypeRegistered MessageType = "registered"
	MessageTypePause      MessageType = "pause"
	MessageTypeResume     MessageType = "resume"
	MessageTypeCancel     MessageType = "cancel"
	MessageTypeStatus     MessageType = "status"
	MessageTypePing       MessageType = "ping"
)

// WSMessage is the base message structure for all WebSocket communication.
type WSMessage struct {
	ID         string                 `json:"id"`                    // For request/response correlation
	Type       MessageType            `json:"type"`                  // Message type
	WorkflowID string                 `json:"workflow_id,omitempty"` // Associated workflow
	Timestamp  time.Time              `json:"timestamp"`             // Message timestamp
	Payload    map[string]interface{} `json:"payload,omitempty"`     // Message-specific data
}

// NewMessage creates a new WSMessage with the given type.
func NewMessage(msgType MessageType) *WSMessage {
	return &WSMessage{
		ID:        generateMessageID(),
		Type:      msgType,
		Timestamp: time.Now(),
		Payload:   make(map[string]interface{}),
	}
}

// NewMessageWithWorkflow creates a new WSMessage for a specific workflow.
func NewMessageWithWorkflow(msgType MessageType, workflowID string) *WSMessage {
	msg := NewMessage(msgType)
	msg.WorkflowID = workflowID
	return msg
}

// JSON serializes the message to JSON bytes.
func (m *WSMessage) JSON() ([]byte, error) {
	return json.Marshal(m)
}

// ParseMessage deserializes a JSON message.
func ParseMessage(data []byte) (*WSMessage, error) {
	var msg WSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// RegisterPayload is the payload for a register message.
type RegisterPayload struct {
	ServiceName     string   `json:"service_name"`
	ServiceID       string   `json:"service_id,omitempty"`
	InstanceID      string   `json:"instance_id,omitempty"`
	Capabilities    []string `json:"capabilities"`
	Version         string   `json:"version,omitempty"`          // Service software version
	ProtocolVersion string   `json:"protocol_version,omitempty"` // Coordination protocol version (e.g., "1.0")
	SchemaVersion   int      `json:"schema_version,omitempty"`   // Database schema version service expects
}

// RegisteredPayload is the payload for a registered response.
type RegisteredPayload struct {
	ServiceID          string `json:"service_id"`
	InstanceID         string `json:"instance_id,omitempty"`
	Message            string `json:"message,omitempty"`
	ProtocolVersion    string `json:"protocol_version,omitempty"`     // Negotiated protocol version
	HubProtocolVersion string `json:"hub_protocol_version,omitempty"` // Hub's protocol version
}

// WorkflowCreatedPayload is the payload for workflow_created message.
type WorkflowCreatedPayload struct {
	WorkflowID       string `json:"workflow_id"`
	ParentWorkflowID string `json:"parent_workflow_id,omitempty"`
	RootWorkflowID   string `json:"root_workflow_id,omitempty"`
	ActionID         string `json:"action_id,omitempty"`
	ActionType       string `json:"action_type,omitempty"`
}

// PhaseChangedPayload is the payload for phase_changed message.
type PhaseChangedPayload struct {
	WorkflowID   string `json:"workflow_id"`
	FromPhase    Phase  `json:"from"`
	ToPhase      Phase  `json:"to"`
	CheckpointID string `json:"checkpoint_id,omitempty"`
	Reason       string `json:"reason,omitempty"`
}

// CheckpointPayload is the payload for checkpoint message.
type CheckpointPayload struct {
	WorkflowID   string                 `json:"workflow_id"`
	CheckpointID string                 `json:"checkpoint_id"`
	Reason       string                 `json:"reason"`
	State        map[string]interface{} `json:"state,omitempty"`
}

// ErrorPayload is the payload for error message.
type ErrorPayload struct {
	WorkflowID  string `json:"workflow_id"`
	Error       string `json:"error"`
	Recoverable bool   `json:"recoverable"`
	ActionID    string `json:"action_id,omitempty"`
}

// StatusResponsePayload is the payload for status_response message.
type StatusResponsePayload struct {
	WorkflowID    string  `json:"workflow_id"`
	Phase         Phase   `json:"phase"`
	Progress      float64 `json:"progress"`
	CurrentAction string  `json:"current_action,omitempty"`
	Message       string  `json:"message,omitempty"`
}

// ProgressPayload is the payload for progress message.
type ProgressPayload struct {
	WorkflowID  string  `json:"workflow_id"`
	ActionID    string  `json:"action_id,omitempty"`
	Percent     float64 `json:"percent"`
	Stage       string  `json:"stage,omitempty"`
	Message     string  `json:"message,omitempty"`
	CurrentItem int     `json:"current_item,omitempty"`
	TotalItems  int     `json:"total_items,omitempty"`
}

// PausePayload is the payload for pause command.
type PausePayload struct {
	WorkflowID string `json:"workflow_id"`
	Reason     string `json:"reason,omitempty"`
}

// ResumePayload is the payload for resume command.
type ResumePayload struct {
	WorkflowID     string `json:"workflow_id"`
	FromCheckpoint string `json:"from_checkpoint,omitempty"`
}

// CancelPayload is the payload for cancel command.
type CancelPayload struct {
	WorkflowID string `json:"workflow_id"`
	Reason     string `json:"reason,omitempty"`
	Force      bool   `json:"force,omitempty"`
}

// StatusPayload is the payload for status request.
type StatusPayload struct {
	WorkflowID string `json:"workflow_id"`
}

// Helper functions to extract typed payloads from messages

// GetRegisterPayload extracts RegisterPayload from message.
func (m *WSMessage) GetRegisterPayload() (*RegisterPayload, error) {
	data, err := json.Marshal(m.Payload)
	if err != nil {
		return nil, err
	}
	var payload RegisterPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// GetPausePayload extracts PausePayload from message.
func (m *WSMessage) GetPausePayload() (*PausePayload, error) {
	data, err := json.Marshal(m.Payload)
	if err != nil {
		return nil, err
	}
	var payload PausePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	// Also check workflow_id at message level
	if payload.WorkflowID == "" {
		payload.WorkflowID = m.WorkflowID
	}
	return &payload, nil
}

// GetResumePayload extracts ResumePayload from message.
func (m *WSMessage) GetResumePayload() (*ResumePayload, error) {
	data, err := json.Marshal(m.Payload)
	if err != nil {
		return nil, err
	}
	var payload ResumePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	if payload.WorkflowID == "" {
		payload.WorkflowID = m.WorkflowID
	}
	return &payload, nil
}

// GetCancelPayload extracts CancelPayload from message.
func (m *WSMessage) GetCancelPayload() (*CancelPayload, error) {
	data, err := json.Marshal(m.Payload)
	if err != nil {
		return nil, err
	}
	var payload CancelPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	if payload.WorkflowID == "" {
		payload.WorkflowID = m.WorkflowID
	}
	return &payload, nil
}

// SetPayload sets the payload from a typed struct.
func (m *WSMessage) SetPayload(payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &m.Payload)
}
