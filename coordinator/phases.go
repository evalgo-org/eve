package coordinator

import (
	"fmt"
	"sync"
	"time"
)

// Phase represents the current phase of a workflow execution.
type Phase string

const (
	PhasePending    Phase = "pending"
	PhasePreFlight  Phase = "pre-flight"
	PhasePlanning   Phase = "planning"
	PhaseExecution  Phase = "execution"
	PhasePausing    Phase = "pausing"
	PhasePaused     Phase = "paused"
	PhaseResuming   Phase = "resuming"
	PhaseCancelling Phase = "cancelling"
	PhaseCancelled  Phase = "cancelled"
	PhaseCompleting Phase = "completing"
	PhaseCompleted  Phase = "completed"
	PhaseFailed     Phase = "failed"
)

// ValidTransitions defines which phase transitions are allowed.
var ValidTransitions = map[Phase][]Phase{
	PhasePending:    {PhasePreFlight, PhaseFailed},
	PhasePreFlight:  {PhasePlanning, PhaseFailed},
	PhasePlanning:   {PhaseExecution, PhaseFailed},
	PhaseExecution:  {PhasePausing, PhaseCancelling, PhaseCompleting, PhaseFailed},
	PhasePausing:    {PhasePaused, PhaseCancelling, PhaseFailed},
	PhasePaused:     {PhaseResuming, PhaseCancelling, PhaseFailed},
	PhaseResuming:   {PhaseExecution, PhaseCancelling, PhaseFailed},
	PhaseCancelling: {PhaseCancelled, PhaseFailed},
	PhaseCompleting: {PhaseCompleted, PhaseFailed},
	// Terminal states: completed, cancelled, failed (no transitions out)
}

// IsTerminal returns true if the phase is a terminal state.
func (p Phase) IsTerminal() bool {
	return p == PhaseCompleted || p == PhaseCancelled || p == PhaseFailed
}

// IsActive returns true if the phase indicates active processing.
func (p Phase) IsActive() bool {
	return p == PhasePreFlight || p == PhasePlanning || p == PhaseExecution ||
		p == PhasePausing || p == PhaseResuming || p == PhaseCancelling || p == PhaseCompleting
}

// IsPausable returns true if the workflow can be paused from this phase.
func (p Phase) IsPausable() bool {
	return p == PhaseExecution
}

// IsResumable returns true if the workflow can be resumed from this phase.
func (p Phase) IsResumable() bool {
	return p == PhasePaused
}

// CanTransitionTo checks if a transition to the target phase is valid.
func (p Phase) CanTransitionTo(target Phase) bool {
	validTargets, ok := ValidTransitions[p]
	if !ok {
		return false
	}
	for _, valid := range validTargets {
		if valid == target {
			return true
		}
	}
	return false
}

// PhaseState represents the state of a single workflow's phase.
type PhaseState struct {
	WorkflowID       string
	Phase            Phase
	PreviousPhase    Phase
	ChangedAt        time.Time
	Reason           string
	CheckpointID     string
	Progress         float64
	CurrentAction    string
	ParentWorkflowID string
	RootWorkflowID   string
}

// PhaseManager manages phase states for multiple workflows.
type PhaseManager struct {
	mu             sync.RWMutex
	workflows      map[string]*PhaseState
	onPhaseChanged func(state *PhaseState)
	onCheckpoint   func(workflowID, checkpointID string, state map[string]interface{})
}

// NewPhaseManager creates a new PhaseManager.
func NewPhaseManager() *PhaseManager {
	return &PhaseManager{
		workflows: make(map[string]*PhaseState),
	}
}

// OnPhaseChanged sets a callback for phase changes.
func (pm *PhaseManager) OnPhaseChanged(fn func(state *PhaseState)) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.onPhaseChanged = fn
}

// OnCheckpoint sets a callback for checkpoint creation.
func (pm *PhaseManager) OnCheckpoint(fn func(workflowID, checkpointID string, state map[string]interface{})) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.onCheckpoint = fn
}

// RegisterWorkflow registers a new workflow with initial pending state.
func (pm *PhaseManager) RegisterWorkflow(workflowID, parentWorkflowID, rootWorkflowID string) *PhaseState {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	state := &PhaseState{
		WorkflowID:       workflowID,
		Phase:            PhasePending,
		ChangedAt:        time.Now(),
		ParentWorkflowID: parentWorkflowID,
		RootWorkflowID:   rootWorkflowID,
	}

	pm.workflows[workflowID] = state
	return state
}

// GetState returns the current state of a workflow.
func (pm *PhaseManager) GetState(workflowID string) (*PhaseState, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	state, ok := pm.workflows[workflowID]
	if !ok {
		return nil, false
	}

	// Return a copy to prevent data races
	copy := *state
	return &copy, true
}

// GetPhase returns just the current phase of a workflow.
func (pm *PhaseManager) GetPhase(workflowID string) (Phase, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	state, ok := pm.workflows[workflowID]
	if !ok {
		return "", false
	}
	return state.Phase, true
}

// TransitionTo attempts to transition a workflow to a new phase.
func (pm *PhaseManager) TransitionTo(workflowID string, newPhase Phase, reason string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	state, ok := pm.workflows[workflowID]
	if !ok {
		return fmt.Errorf("workflow not found: %s", workflowID)
	}

	if !state.Phase.CanTransitionTo(newPhase) {
		return fmt.Errorf("invalid transition from %s to %s for workflow %s",
			state.Phase, newPhase, workflowID)
	}

	state.PreviousPhase = state.Phase
	state.Phase = newPhase
	state.ChangedAt = time.Now()
	state.Reason = reason

	// Notify callback
	if pm.onPhaseChanged != nil {
		go pm.onPhaseChanged(state)
	}

	return nil
}

// SetProgress updates the progress of a workflow.
func (pm *PhaseManager) SetProgress(workflowID string, progress float64, currentAction string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	state, ok := pm.workflows[workflowID]
	if !ok {
		return fmt.Errorf("workflow not found: %s", workflowID)
	}

	state.Progress = progress
	state.CurrentAction = currentAction
	return nil
}

// Pause initiates pausing of a workflow.
func (pm *PhaseManager) Pause(workflowID, reason string) error {
	pm.mu.Lock()
	state, ok := pm.workflows[workflowID]
	pm.mu.Unlock()

	if !ok {
		return fmt.Errorf("workflow not found: %s", workflowID)
	}

	if !state.Phase.IsPausable() {
		return fmt.Errorf("workflow %s cannot be paused from phase %s", workflowID, state.Phase)
	}

	return pm.TransitionTo(workflowID, PhasePausing, reason)
}

// CompletePause finishes the pause transition.
func (pm *PhaseManager) CompletePause(workflowID, checkpointID string) error {
	pm.mu.Lock()
	state, ok := pm.workflows[workflowID]
	if !ok {
		pm.mu.Unlock()
		return fmt.Errorf("workflow not found: %s", workflowID)
	}

	if state.Phase != PhasePausing {
		pm.mu.Unlock()
		return fmt.Errorf("workflow %s is not pausing (current: %s)", workflowID, state.Phase)
	}

	state.PreviousPhase = state.Phase
	state.Phase = PhasePaused
	state.ChangedAt = time.Now()
	state.CheckpointID = checkpointID

	if pm.onPhaseChanged != nil {
		go pm.onPhaseChanged(state)
	}
	pm.mu.Unlock()

	return nil
}

// Resume initiates resuming of a workflow.
func (pm *PhaseManager) Resume(workflowID, fromCheckpoint string) error {
	pm.mu.Lock()
	state, ok := pm.workflows[workflowID]
	pm.mu.Unlock()

	if !ok {
		return fmt.Errorf("workflow not found: %s", workflowID)
	}

	if !state.Phase.IsResumable() {
		return fmt.Errorf("workflow %s cannot be resumed from phase %s", workflowID, state.Phase)
	}

	pm.mu.Lock()
	if fromCheckpoint != "" {
		state.CheckpointID = fromCheckpoint
	}
	pm.mu.Unlock()

	return pm.TransitionTo(workflowID, PhaseResuming, "resume requested")
}

// CompleteResume finishes the resume transition.
func (pm *PhaseManager) CompleteResume(workflowID string) error {
	return pm.TransitionTo(workflowID, PhaseExecution, "resumed")
}

// Cancel initiates cancellation of a workflow.
func (pm *PhaseManager) Cancel(workflowID, reason string) error {
	pm.mu.Lock()
	state, ok := pm.workflows[workflowID]
	pm.mu.Unlock()

	if !ok {
		return fmt.Errorf("workflow not found: %s", workflowID)
	}

	// Can cancel from most non-terminal states
	if state.Phase.IsTerminal() {
		return fmt.Errorf("workflow %s is already in terminal state %s", workflowID, state.Phase)
	}

	return pm.TransitionTo(workflowID, PhaseCancelling, reason)
}

// CompleteCancellation finishes the cancellation.
func (pm *PhaseManager) CompleteCancellation(workflowID string) error {
	return pm.TransitionTo(workflowID, PhaseCancelled, "cancelled")
}

// Fail marks a workflow as failed.
func (pm *PhaseManager) Fail(workflowID, reason string) error {
	pm.mu.Lock()
	state, ok := pm.workflows[workflowID]
	if !ok {
		pm.mu.Unlock()
		return fmt.Errorf("workflow not found: %s", workflowID)
	}

	if state.Phase.IsTerminal() {
		pm.mu.Unlock()
		return fmt.Errorf("workflow %s is already in terminal state %s", workflowID, state.Phase)
	}

	state.PreviousPhase = state.Phase
	state.Phase = PhaseFailed
	state.ChangedAt = time.Now()
	state.Reason = reason

	if pm.onPhaseChanged != nil {
		go pm.onPhaseChanged(state)
	}
	pm.mu.Unlock()

	return nil
}

// Complete marks a workflow as completed.
func (pm *PhaseManager) Complete(workflowID string) error {
	// First transition to completing
	if err := pm.TransitionTo(workflowID, PhaseCompleting, "completing"); err != nil {
		return err
	}
	// Then to completed
	return pm.TransitionTo(workflowID, PhaseCompleted, "completed successfully")
}

// RemoveWorkflow removes a workflow from tracking.
func (pm *PhaseManager) RemoveWorkflow(workflowID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	delete(pm.workflows, workflowID)
}

// GetActiveWorkflows returns all workflows that are not in terminal states.
func (pm *PhaseManager) GetActiveWorkflows() []*PhaseState {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var active []*PhaseState
	for _, state := range pm.workflows {
		if !state.Phase.IsTerminal() {
			copy := *state
			active = append(active, &copy)
		}
	}
	return active
}

// GetAllWorkflows returns all tracked workflows.
func (pm *PhaseManager) GetAllWorkflows() []*PhaseState {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var all []*PhaseState
	for _, state := range pm.workflows {
		copy := *state
		all = append(all, &copy)
	}
	return all
}

// CreateCheckpoint creates a checkpoint for a workflow.
func (pm *PhaseManager) CreateCheckpoint(workflowID, checkpointID, reason string, state map[string]interface{}) error {
	pm.mu.Lock()
	ws, ok := pm.workflows[workflowID]
	if !ok {
		pm.mu.Unlock()
		return fmt.Errorf("workflow not found: %s", workflowID)
	}
	ws.CheckpointID = checkpointID
	callback := pm.onCheckpoint
	pm.mu.Unlock()

	if callback != nil {
		callback(workflowID, checkpointID, state)
	}
	return nil
}
