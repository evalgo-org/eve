package statemanager

import (
	"sync"
	"time"
)

// Manager handles state tracking for operations
type Manager struct {
	mu            sync.RWMutex
	operations    map[string]*OperationState
	maxOperations int
	serviceName   string
}

// Config for creating a new Manager
type Config struct {
	ServiceName   string
	MaxOperations int // Keep last N operations, default 1000
}

// New creates a new state manager
func New(cfg Config) *Manager {
	if cfg.MaxOperations == 0 {
		cfg.MaxOperations = 1000
	}
	return &Manager{
		operations:    make(map[string]*OperationState),
		maxOperations: cfg.MaxOperations,
		serviceName:   cfg.ServiceName,
	}
}

// StartOperation creates a new operation in running state
func (m *Manager) StartOperation(id, operation string, metadata map[string]interface{}) *OperationState {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Evict oldest if at capacity
	if len(m.operations) >= m.maxOperations {
		m.evictOldest()
	}

	op := &OperationState{
		ID:          id,
		ServiceName: m.serviceName,
		Operation:   operation,
		Status:      StatusRunning,
		StartedAt:   time.Now(),
		Metadata:    metadata,
	}

	m.operations[id] = op
	return op
}

// CompleteOperation marks an operation as completed or failed
func (m *Manager) CompleteOperation(id string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if op, exists := m.operations[id]; exists {
		now := time.Now()
		op.CompletedAt = &now
		op.Duration = now.Sub(op.StartedAt).String()

		if err != nil {
			op.Status = StatusFailed
			op.Error = err.Error()
		} else {
			op.Status = StatusCompleted
		}
	}
}

// UpdateMetadata adds/updates metadata for an operation
func (m *Manager) UpdateMetadata(id string, key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if op, exists := m.operations[id]; exists {
		if op.Metadata == nil {
			op.Metadata = make(map[string]interface{})
		}
		op.Metadata[key] = value
	}
}

// GetOperation retrieves an operation by ID
func (m *Manager) GetOperation(id string) *OperationState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if op, exists := m.operations[id]; exists {
		// Return a copy to prevent external modification
		opCopy := *op
		return &opCopy
	}
	return nil
}

// ListOperations returns all tracked operations
func (m *Manager) ListOperations() []*OperationState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ops := make([]*OperationState, 0, len(m.operations))
	for _, op := range m.operations {
		// Return copies to prevent external modification
		opCopy := *op
		ops = append(ops, &opCopy)
	}
	return ops
}

// GetStats returns aggregated statistics
func (m *Manager) GetStats() *OperationStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := &OperationStats{
		TotalOperations: len(m.operations),
		ByStatus:        make(map[Status]int),
		ByOperation:     make(map[string]int),
	}

	var totalDuration time.Duration
	var completedCount int

	for _, op := range m.operations {
		stats.ByStatus[op.Status]++
		stats.ByOperation[op.Operation]++

		if op.CompletedAt != nil {
			totalDuration += op.CompletedAt.Sub(op.StartedAt)
			completedCount++
		}
	}

	if completedCount > 0 {
		avgDuration := totalDuration / time.Duration(completedCount)
		stats.AverageDuration = avgDuration.String()
	}

	return stats
}

// evictOldest removes the oldest operation (must be called with lock held)
func (m *Manager) evictOldest() {
	var oldestID string
	var oldestTime time.Time

	for id, op := range m.operations {
		if oldestID == "" || op.StartedAt.Before(oldestTime) {
			oldestID = id
			oldestTime = op.StartedAt
		}
	}

	if oldestID != "" {
		delete(m.operations, oldestID)
	}
}
