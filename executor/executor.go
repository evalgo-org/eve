package executor

import (
	"context"
	"sync"
	"time"

	"eve.evalgo.org/semantic"
)

// Executor is the unified interface for all execution types
type Executor interface {
	// Execute runs an action and returns the result
	Execute(ctx context.Context, action *semantic.SemanticScheduledAction) (*Result, error)

	// CanHandle determines if this executor can process the action
	CanHandle(action *semantic.SemanticScheduledAction) bool

	// Name returns the executor's identifier
	Name() string
}

// Result contains the execution output and metadata
type Result struct {
	// Output is the primary execution result (stdout, response body, etc.)
	Output string

	// Status indicates the execution status
	Status ExecutionStatus

	// Metadata contains additional execution information
	Metadata map[string]interface{}

	// Error contains detailed error information if execution failed
	Error *ExecutionError

	// StartTime when execution began
	StartTime time.Time

	// EndTime when execution completed
	EndTime time.Time

	// Duration of execution
	Duration time.Duration
}

// ExecutionStatus represents the state of execution
type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "pending"
	StatusRunning   ExecutionStatus = "running"
	StatusCompleted ExecutionStatus = "completed"
	StatusFailed    ExecutionStatus = "failed"
	StatusCancelled ExecutionStatus = "cancelled"
)

// ExecutionError provides detailed error information
type ExecutionError struct {
	Message string
	Code    string
	Details map[string]interface{}
}

// Registry manages executor implementations
type Registry struct {
	executors []Executor
	mu        sync.RWMutex
}

// NewRegistry creates a new executor registry
func NewRegistry() *Registry {
	return &Registry{
		executors: make([]Executor, 0),
	}
}

// Register adds an executor to the registry
func (r *Registry) Register(executor Executor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.executors = append(r.executors, executor)
}

// Execute finds the appropriate executor and runs the action
func (r *Registry) Execute(ctx context.Context, action *semantic.SemanticScheduledAction) (*Result, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := &Result{
		StartTime: time.Now(),
		Status:    StatusRunning,
		Metadata:  make(map[string]interface{}),
	}

	// Find matching executor
	var executor Executor
	for _, e := range r.executors {
		if e.CanHandle(action) {
			executor = e
			break
		}
	}

	if executor == nil {
		result.Status = StatusFailed
		result.Error = &ExecutionError{
			Message: "no executor found for action",
			Code:    "NO_EXECUTOR",
			Details: map[string]interface{}{
				"action_type": action.Type,
			},
		}
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result, &ExecutionError{
			Message: "no executor found for action",
			Code:    "NO_EXECUTOR",
		}
	}

	result.Metadata["executor"] = executor.Name()

	// Execute action
	execResult, err := executor.Execute(ctx, action)
	if err != nil {
		result.Status = StatusFailed
		result.Error = &ExecutionError{
			Message: err.Error(),
			Code:    "EXECUTION_ERROR",
		}
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result, err
	}

	// Merge results
	result.Output = execResult.Output
	result.Status = execResult.Status
	if execResult.Error != nil {
		result.Error = execResult.Error
	}
	for k, v := range execResult.Metadata {
		result.Metadata[k] = v
	}
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

// ExecuteWithOptions provides advanced execution control
type ExecuteOptions struct {
	// Context for cancellation/timeout
	Context context.Context

	// RetryPolicy defines retry behavior
	RetryPolicy *RetryPolicy

	// Dependencies are actions that must complete first
	Dependencies []string

	// Storage for persisting results
	Storage Storage

	// Hooks for lifecycle events
	Hooks *ExecutionHooks
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxAttempts int
	Backoff     BackoffStrategy
}

// BackoffStrategy determines delay between retries
type BackoffStrategy string

const (
	BackoffExponential BackoffStrategy = "exponential"
	BackoffLinear      BackoffStrategy = "linear"
	BackoffFixed       BackoffStrategy = "fixed"
)

// Storage interface for persisting execution state
type Storage interface {
	Save(ctx context.Context, actionID string, result *Result) error
	Load(ctx context.Context, actionID string) (*Result, error)
}

// ExecutionHooks allows customization of execution lifecycle
type ExecutionHooks struct {
	BeforeExecute func(ctx context.Context, action *semantic.SemanticScheduledAction) error
	AfterExecute  func(ctx context.Context, action *semantic.SemanticScheduledAction, result *Result) error
	OnError       func(ctx context.Context, action *semantic.SemanticScheduledAction, err error) error
}

// Error implements the error interface for ExecutionError
func (e *ExecutionError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "execution error"
}
