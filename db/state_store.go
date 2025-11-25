// Package db provides StateStore for persistent action execution state management.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ActionState represents the state of an action execution in the database.
type ActionState struct {
	ID              string                 `json:"id"`
	WorkflowID      string                 `json:"workflow_id"`
	ActionID        string                 `json:"action_id"`
	Phase           string                 `json:"phase"`
	Status          string                 `json:"status"`
	ProgressPct     int                    `json:"progress_pct"`
	ProgressStage   string                 `json:"progress_stage"`
	ProgressMessage string                 `json:"progress_message"`
	CheckpointID    *string                `json:"checkpoint_id,omitempty"`
	CheckpointData  map[string]interface{} `json:"checkpoint_data,omitempty"`
	Error           *string                `json:"error,omitempty"`
	StartedAt       *time.Time             `json:"started_at,omitempty"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// Phase constants for action execution.
const (
	PhasePending    = "pending"
	PhaseRunning    = "running"
	PhasePausing    = "pausing"
	PhasePaused     = "paused"
	PhaseResuming   = "resuming"
	PhaseCancelling = "cancelling"
	PhaseCancelled  = "cancelled"
	PhaseCompleted  = "completed"
	PhaseFailed     = "failed"
)

// StateStore provides persistent action state management using PostgreSQL.
// All state is stored in the database - no in-memory caching.
type StateStore struct {
	pool    *pgxpool.Pool
	channel string // NOTIFY channel name
}

// NewStateStore creates a new state store.
func NewStateStore(pool *pgxpool.Pool, notifyChannel string) *StateStore {
	return &StateStore{
		pool:    pool,
		channel: notifyChannel,
	}
}

// CreateAction creates a new action execution record.
func (s *StateStore) CreateAction(ctx context.Context, workflowID, actionID string) (*ActionState, error) {
	query := `
		INSERT INTO service_action_executions (workflow_id, action_id, phase, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, workflow_id, action_id, phase, status, progress_pct,
		          COALESCE(progress_stage, ''), COALESCE(progress_message, ''),
		          checkpoint_id, error, started_at, completed_at, created_at, updated_at`

	state := &ActionState{}
	err := s.pool.QueryRow(ctx, query, workflowID, actionID, PhasePending, "pending").Scan(
		&state.ID, &state.WorkflowID, &state.ActionID, &state.Phase, &state.Status,
		&state.ProgressPct, &state.ProgressStage, &state.ProgressMessage,
		&state.CheckpointID, &state.Error, &state.StartedAt, &state.CompletedAt,
		&state.CreatedAt, &state.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create action: %w", err)
	}

	return state, nil
}

// GetAction retrieves an action by workflow and action ID.
func (s *StateStore) GetAction(ctx context.Context, workflowID, actionID string) (*ActionState, error) {
	query := `
		SELECT id, workflow_id, action_id, phase, status, progress_pct,
		       COALESCE(progress_stage, ''), COALESCE(progress_message, ''),
		       checkpoint_id, error, started_at, completed_at, created_at, updated_at
		FROM service_action_executions
		WHERE workflow_id = $1 AND action_id = $2`

	state := &ActionState{}
	err := s.pool.QueryRow(ctx, query, workflowID, actionID).Scan(
		&state.ID, &state.WorkflowID, &state.ActionID, &state.Phase, &state.Status,
		&state.ProgressPct, &state.ProgressStage, &state.ProgressMessage,
		&state.CheckpointID, &state.Error, &state.StartedAt, &state.CompletedAt,
		&state.CreatedAt, &state.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get action: %w", err)
	}

	return state, nil
}

// GetByID retrieves an action by its primary key ID.
func (s *StateStore) GetByID(ctx context.Context, id string) (*ActionState, error) {
	query := `
		SELECT id, workflow_id, action_id, phase, status, progress_pct,
		       COALESCE(progress_stage, ''), COALESCE(progress_message, ''),
		       checkpoint_id, error, started_at, completed_at, created_at, updated_at
		FROM service_action_executions
		WHERE id = $1`

	state := &ActionState{}
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&state.ID, &state.WorkflowID, &state.ActionID, &state.Phase, &state.Status,
		&state.ProgressPct, &state.ProgressStage, &state.ProgressMessage,
		&state.CheckpointID, &state.Error, &state.StartedAt, &state.CompletedAt,
		&state.CreatedAt, &state.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get action by ID: %w", err)
	}

	return state, nil
}

// UpdatePhase updates the phase of an action.
func (s *StateStore) UpdatePhase(ctx context.Context, workflowID, actionID, phase string) error {
	query := `
		UPDATE service_action_executions
		SET phase = $1, updated_at = NOW()
		WHERE workflow_id = $2 AND action_id = $3`

	result, err := s.pool.Exec(ctx, query, phase, workflowID, actionID)
	if err != nil {
		return fmt.Errorf("failed to update phase: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("action not found: workflow=%s, action=%s", workflowID, actionID)
	}

	return nil
}

// UpdateProgress updates the progress of an action.
func (s *StateStore) UpdateProgress(ctx context.Context, workflowID, actionID string, percent int, stage, message string) error {
	query := `
		UPDATE service_action_executions
		SET progress_pct = $1, progress_stage = $2, progress_message = $3, updated_at = NOW()
		WHERE workflow_id = $4 AND action_id = $5`

	result, err := s.pool.Exec(ctx, query, percent, stage, message, workflowID, actionID)
	if err != nil {
		return fmt.Errorf("failed to update progress: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("action not found: workflow=%s, action=%s", workflowID, actionID)
	}

	return nil
}

// Start marks an action as started.
func (s *StateStore) Start(ctx context.Context, workflowID, actionID string) error {
	query := `
		UPDATE service_action_executions
		SET phase = $1, status = 'running', started_at = NOW(), updated_at = NOW()
		WHERE workflow_id = $2 AND action_id = $3`

	result, err := s.pool.Exec(ctx, query, PhaseRunning, workflowID, actionID)
	if err != nil {
		return fmt.Errorf("failed to start action: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("action not found: workflow=%s, action=%s", workflowID, actionID)
	}

	return nil
}

// Complete marks an action as completed.
func (s *StateStore) Complete(ctx context.Context, workflowID, actionID string) error {
	query := `
		UPDATE service_action_executions
		SET phase = $1, status = 'completed', progress_pct = 100, completed_at = NOW(), updated_at = NOW()
		WHERE workflow_id = $2 AND action_id = $3`

	result, err := s.pool.Exec(ctx, query, PhaseCompleted, workflowID, actionID)
	if err != nil {
		return fmt.Errorf("failed to complete action: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("action not found: workflow=%s, action=%s", workflowID, actionID)
	}

	return nil
}

// Fail marks an action as failed.
func (s *StateStore) Fail(ctx context.Context, workflowID, actionID, errorMsg string) error {
	query := `
		UPDATE service_action_executions
		SET phase = $1, status = 'failed', error = $2, completed_at = NOW(), updated_at = NOW()
		WHERE workflow_id = $3 AND action_id = $4`

	result, err := s.pool.Exec(ctx, query, PhaseFailed, errorMsg, workflowID, actionID)
	if err != nil {
		return fmt.Errorf("failed to fail action: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("action not found: workflow=%s, action=%s", workflowID, actionID)
	}

	return nil
}

// RequestPause sets the action to pausing state.
func (s *StateStore) RequestPause(ctx context.Context, workflowID, actionID string) error {
	query := `
		UPDATE service_action_executions
		SET phase = $1, updated_at = NOW()
		WHERE workflow_id = $2 AND action_id = $3 AND phase = $4`

	result, err := s.pool.Exec(ctx, query, PhasePausing, workflowID, actionID, PhaseRunning)
	if err != nil {
		return fmt.Errorf("failed to request pause: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("action not running or not found: workflow=%s, action=%s", workflowID, actionID)
	}

	return nil
}

// CompletePause marks an action as paused with checkpoint.
func (s *StateStore) CompletePause(ctx context.Context, workflowID, actionID, checkpointID string, checkpointData map[string]interface{}) error {
	query := `
		UPDATE service_action_executions
		SET phase = $1, status = 'paused', checkpoint_id = $2, checkpoint_data = $3, updated_at = NOW()
		WHERE workflow_id = $4 AND action_id = $5`

	result, err := s.pool.Exec(ctx, query, PhasePaused, checkpointID, checkpointData, workflowID, actionID)
	if err != nil {
		return fmt.Errorf("failed to complete pause: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("action not found: workflow=%s, action=%s", workflowID, actionID)
	}

	return nil
}

// RequestCancel sets the action to cancelling state.
func (s *StateStore) RequestCancel(ctx context.Context, workflowID, actionID string) error {
	query := `
		UPDATE service_action_executions
		SET phase = $1, updated_at = NOW()
		WHERE workflow_id = $2 AND action_id = $3 AND phase IN ($4, $5)`

	result, err := s.pool.Exec(ctx, query, PhaseCancelling, workflowID, actionID, PhaseRunning, PhasePausing)
	if err != nil {
		return fmt.Errorf("failed to request cancel: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("action not running/pausing or not found: workflow=%s, action=%s", workflowID, actionID)
	}

	return nil
}

// CompleteCancel marks an action as cancelled.
func (s *StateStore) CompleteCancel(ctx context.Context, workflowID, actionID string) error {
	query := `
		UPDATE service_action_executions
		SET phase = $1, status = 'cancelled', completed_at = NOW(), updated_at = NOW()
		WHERE workflow_id = $2 AND action_id = $3`

	result, err := s.pool.Exec(ctx, query, PhaseCancelled, workflowID, actionID)
	if err != nil {
		return fmt.Errorf("failed to complete cancel: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("action not found: workflow=%s, action=%s", workflowID, actionID)
	}

	return nil
}

// IsPausing checks if an action is in pausing state (reads from DB).
func (s *StateStore) IsPausing(ctx context.Context, workflowID, actionID string) (bool, error) {
	query := `SELECT phase FROM service_action_executions WHERE workflow_id = $1 AND action_id = $2`

	var phase string
	err := s.pool.QueryRow(ctx, query, workflowID, actionID).Scan(&phase)
	if err != nil {
		return false, fmt.Errorf("failed to check pausing state: %w", err)
	}

	return phase == PhasePausing, nil
}

// IsCancelling checks if an action is in cancelling state (reads from DB).
func (s *StateStore) IsCancelling(ctx context.Context, workflowID, actionID string) (bool, error) {
	query := `SELECT phase FROM service_action_executions WHERE workflow_id = $1 AND action_id = $2`

	var phase string
	err := s.pool.QueryRow(ctx, query, workflowID, actionID).Scan(&phase)
	if err != nil {
		return false, fmt.Errorf("failed to check cancelling state: %w", err)
	}

	return phase == PhaseCancelling, nil
}

// ShouldStop checks if an action should stop (pausing or cancelling).
func (s *StateStore) ShouldStop(ctx context.Context, workflowID, actionID string) (bool, error) {
	query := `SELECT phase FROM service_action_executions WHERE workflow_id = $1 AND action_id = $2`

	var phase string
	err := s.pool.QueryRow(ctx, query, workflowID, actionID).Scan(&phase)
	if err != nil {
		return false, fmt.Errorf("failed to check stop state: %w", err)
	}

	return phase == PhasePausing || phase == PhaseCancelling, nil
}

// GetActiveByWorkflow returns all active (non-terminal) actions for a workflow.
func (s *StateStore) GetActiveByWorkflow(ctx context.Context, workflowID string) ([]*ActionState, error) {
	query := `
		SELECT id, workflow_id, action_id, phase, status, progress_pct,
		       COALESCE(progress_stage, ''), COALESCE(progress_message, ''),
		       checkpoint_id, error, started_at, completed_at, created_at, updated_at
		FROM service_action_executions
		WHERE workflow_id = $1 AND phase NOT IN ($2, $3, $4)
		ORDER BY created_at`

	rows, err := s.pool.Query(ctx, query, workflowID, PhaseCompleted, PhaseCancelled, PhaseFailed)
	if err != nil {
		return nil, fmt.Errorf("failed to get active actions: %w", err)
	}
	defer rows.Close()

	var states []*ActionState
	for rows.Next() {
		state := &ActionState{}
		err := rows.Scan(
			&state.ID, &state.WorkflowID, &state.ActionID, &state.Phase, &state.Status,
			&state.ProgressPct, &state.ProgressStage, &state.ProgressMessage,
			&state.CheckpointID, &state.Error, &state.StartedAt, &state.CompletedAt,
			&state.CreatedAt, &state.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan action: %w", err)
		}
		states = append(states, state)
	}

	return states, nil
}
