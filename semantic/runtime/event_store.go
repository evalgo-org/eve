package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"eve.evalgo.org/db"
)

// EventStore manages Event storage in PostgreSQL for audit trails
type EventStore struct {
	db *db.PostgresDB
}

// NewEventStore creates a new event store
func NewEventStore(pg *db.PostgresDB) *EventStore {
	return &EventStore{
		db: pg,
	}
}

// SaveEvent saves an event to PostgreSQL
func (s *EventStore) SaveEvent(ctx context.Context, event *Event) error {
	// Marshal event to JSON
	eventJSON, err := event.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Extract workflow ID from additionalProperty
	workflowID, _ := event.AdditionalProperty["workflowId"].(string)

	// Extract action ID from about field if present
	actionID := ""
	if event.About != nil {
		if id, ok := event.About["identifier"].(string); ok {
			actionID = id
		}
	}

	// Extract event type
	eventType, _ := event.AdditionalProperty["eventType"].(string)

	// Insert into database
	err = s.db.Exec(ctx, `
		INSERT INTO workflow_events (
			event_id, workflow_id, action_id, event_type, event_data, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`, event.Identifier, workflowID, actionID, eventType, eventJSON, event.StartDate)

	if err != nil {
		return fmt.Errorf("failed to save event: %w", err)
	}

	return nil
}

// GetEventsByWorkflow retrieves all events for a workflow
func (s *EventStore) GetEventsByWorkflow(ctx context.Context, workflowID string, limit, offset int) ([]*Event, error) {
	query := `
		SELECT event_data FROM workflow_events
		WHERE workflow_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.Query(ctx, query, workflowID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		var eventJSON []byte
		if err := rows.Scan(&eventJSON); err != nil {
			continue
		}

		var event Event
		if err := json.Unmarshal(eventJSON, &event); err != nil {
			continue
		}

		events = append(events, &event)
	}

	return events, rows.Err()
}

// GetEventsByAction retrieves all events for a specific action
func (s *EventStore) GetEventsByAction(ctx context.Context, workflowID, actionID string, limit, offset int) ([]*Event, error) {
	query := `
		SELECT event_data FROM workflow_events
		WHERE workflow_id = $1 AND action_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := s.db.Query(ctx, query, workflowID, actionID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		var eventJSON []byte
		if err := rows.Scan(&eventJSON); err != nil {
			continue
		}

		var event Event
		if err := json.Unmarshal(eventJSON, &event); err != nil {
			continue
		}

		events = append(events, &event)
	}

	return events, rows.Err()
}

// GetEventsByType retrieves events by type
func (s *EventStore) GetEventsByType(ctx context.Context, eventType string, limit, offset int) ([]*Event, error) {
	query := `
		SELECT event_data FROM workflow_events
		WHERE event_type = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.Query(ctx, query, eventType, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		var eventJSON []byte
		if err := rows.Scan(&eventJSON); err != nil {
			continue
		}

		var event Event
		if err := json.Unmarshal(eventJSON, &event); err != nil {
			continue
		}

		events = append(events, &event)
	}

	return events, rows.Err()
}

// DeleteEventsByWorkflow deletes all events for a workflow
func (s *EventStore) DeleteEventsByWorkflow(ctx context.Context, workflowID string) error {
	err := s.db.Exec(ctx, `
		DELETE FROM workflow_events WHERE workflow_id = $1
	`, workflowID)

	if err != nil {
		return fmt.Errorf("failed to delete events: %w", err)
	}

	return nil
}

// GetEventStats retrieves statistics about events for a workflow
func (s *EventStore) GetEventStats(ctx context.Context, workflowID string, from, to time.Time) (map[string]int, error) {
	query := `
		SELECT event_type, COUNT(*) as count
		FROM workflow_events
		WHERE workflow_id = $1 AND created_at BETWEEN $2 AND $3
		GROUP BY event_type
	`

	rows, err := s.db.Query(ctx, query, workflowID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to query event stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var eventType string
		var count int
		if err := rows.Scan(&eventType, &count); err != nil {
			continue
		}
		stats[eventType] = count
	}

	return stats, rows.Err()
}

// CreateTables creates the necessary database tables if they don't exist
func (s *EventStore) CreateTables(ctx context.Context) error {
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS workflow_events (
		id BIGSERIAL PRIMARY KEY,
		event_id VARCHAR(255) NOT NULL,
		workflow_id VARCHAR(255),
		action_id VARCHAR(255),
		event_type VARCHAR(100),
		event_data JSONB NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL,
		UNIQUE(event_id)
	);

	CREATE INDEX IF NOT EXISTS idx_workflow_events_workflow_id ON workflow_events(workflow_id);
	CREATE INDEX IF NOT EXISTS idx_workflow_events_action_id ON workflow_events(action_id);
	CREATE INDEX IF NOT EXISTS idx_workflow_events_event_type ON workflow_events(event_type);
	CREATE INDEX IF NOT EXISTS idx_workflow_events_created_at ON workflow_events(created_at);
	`

	err := s.db.Exec(ctx, createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	return nil
}
