package statemanager

import "time"

// OperationState represents a tracked operation
type OperationState struct {
	ID          string                 `json:"id"`
	ServiceName string                 `json:"service_name"`
	Operation   string                 `json:"operation"` // e.g., "xquery", "s3-upload", "template-render"
	Status      Status                 `json:"status"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Duration    string                 `json:"duration,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"` // Service-specific data
}

// Status represents the state of an operation
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusTimeout   Status = "timeout"
)

// OperationStats provides aggregated statistics
type OperationStats struct {
	TotalOperations int            `json:"total_operations"`
	ByStatus        map[Status]int `json:"by_status"`
	ByOperation     map[string]int `json:"by_operation"`
	AverageDuration string         `json:"average_duration,omitempty"`
}
