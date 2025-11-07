package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"eve.evalgo.org/db"
)

// PostgresMetricsRepository implements MetricsRepository using PostgreSQL
type PostgresMetricsRepository struct {
	db  *db.PostgresDB
	ctx context.Context
}

// NewPostgresMetricsRepository creates a new PostgreSQL metrics repository
func NewPostgresMetricsRepository(pg *db.PostgresDB) *PostgresMetricsRepository {
	return &PostgresMetricsRepository{
		db:  pg,
		ctx: context.Background(),
	}
}

// SaveRun saves an action execution result
func (r *PostgresMetricsRepository) SaveRun(ctx context.Context, run *ActionRun) error {
	runData := map[string]interface{}{
		"runId":      run.RunID,
		"actionId":   run.ActionID,
		"workflowId": run.WorkflowID,
		"startTime":  run.StartTime.Format(time.RFC3339),
		"endTime":    run.EndTime.Format(time.RFC3339),
		"duration":   run.Duration.Milliseconds(),
		"status":     run.Status,
		"error":      run.Error,
		"result":     run.Result,
		"attempt":    run.Attempt,
	}

	jsonData, err := json.Marshal(runData)
	if err != nil {
		return fmt.Errorf("failed to marshal run data: %w", err)
	}

	err = r.db.Exec(ctx, `
		INSERT INTO action_runs (run_id, action_id, run_data, created_at)
		VALUES ($1, $2, $3, $4)
	`, run.RunID, run.ActionID, jsonData, run.StartTime)

	return err
}

// GetRunHistory retrieves execution history for an action
func (r *PostgresMetricsRepository) GetRunHistory(ctx context.Context, actionID string, limit int) ([]*ActionRun, error) {
	query := `
		SELECT run_data FROM action_runs
		WHERE action_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, actionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []*ActionRun
	for rows.Next() {
		var jsonData []byte
		if err := rows.Scan(&jsonData); err != nil {
			continue
		}

		var runData map[string]interface{}
		if err := json.Unmarshal(jsonData, &runData); err != nil {
			continue
		}

		run := &ActionRun{
			RunID:      getString(runData, "runId"),
			ActionID:   getString(runData, "actionId"),
			WorkflowID: getString(runData, "workflowId"),
			Status:     getString(runData, "status"),
			Error:      getString(runData, "error"),
			Result:     getMap(runData, "result"),
			Attempt:    getInt(runData, "attempt"),
		}

		// Parse timestamps
		if startStr := getString(runData, "startTime"); startStr != "" {
			if t, err := time.Parse(time.RFC3339, startStr); err == nil {
				run.StartTime = t
			}
		}
		if endStr := getString(runData, "endTime"); endStr != "" {
			if t, err := time.Parse(time.RFC3339, endStr); err == nil {
				run.EndTime = t
			}
		}
		if durationMs := getInt64(runData, "duration"); durationMs > 0 {
			run.Duration = time.Duration(durationMs) * time.Millisecond
		}

		runs = append(runs, run)
	}

	return runs, rows.Err()
}

// GetMetrics retrieves metrics for an action over a time window
func (r *PostgresMetricsRepository) GetMetrics(ctx context.Context, actionID string, from, to time.Time) (*ActionMetrics, error) {
	query := `
		SELECT
			COUNT(*) as total_runs,
			SUM(CASE WHEN (run_data->>'status') = 'CompletedActionStatus' THEN 1 ELSE 0 END) as successful,
			SUM(CASE WHEN (run_data->>'status') = 'FailedActionStatus' THEN 1 ELSE 0 END) as failed,
			AVG((run_data->>'duration')::bigint) as avg_duration,
			MIN((run_data->>'duration')::bigint) as min_duration,
			MAX((run_data->>'duration')::bigint) as max_duration,
			MAX(created_at) as last_run
		FROM action_runs
		WHERE action_id = $1 AND created_at BETWEEN $2 AND $3
	`

	var (
		totalRuns     int64
		successful    int64
		failed        int64
		avgDurationMs *int64
		minDurationMs *int64
		maxDurationMs *int64
		lastRun       *time.Time
	)

	err := r.db.QueryRow(ctx, query, actionID, from, to).Scan(
		&totalRuns,
		&successful,
		&failed,
		&avgDurationMs,
		&minDurationMs,
		&maxDurationMs,
		&lastRun,
	)
	if err != nil {
		return nil, err
	}

	metrics := &ActionMetrics{
		ActionID:       actionID,
		TotalRuns:      totalRuns,
		SuccessfulRuns: successful,
		FailedRuns:     failed,
	}

	if avgDurationMs != nil {
		metrics.AvgDuration = time.Duration(*avgDurationMs) * time.Millisecond
	}
	if minDurationMs != nil {
		metrics.MinDuration = time.Duration(*minDurationMs) * time.Millisecond
	}
	if maxDurationMs != nil {
		metrics.MaxDuration = time.Duration(*maxDurationMs) * time.Millisecond
	}
	if lastRun != nil {
		metrics.LastRun = *lastRun
	}

	return metrics, nil
}

// GetAggregatedMetrics retrieves aggregated metrics over time buckets
func (r *PostgresMetricsRepository) GetAggregatedMetrics(ctx context.Context, actionID string, window time.Duration, aggregation string) ([]DataPoint, error) {
	// Note: window parameter could be used for dynamic bucketing in future
	_ = window

	query := `
		SELECT
			date_trunc('hour', created_at) as bucket,
			AVG((run_data->>'duration')::bigint) as value
		FROM action_runs
		WHERE action_id = $1
		GROUP BY bucket
		ORDER BY bucket DESC
		LIMIT 100
	`

	// Adjust aggregation function
	if aggregation == "sum" {
		query = `
			SELECT
				date_trunc('hour', created_at) as bucket,
				SUM((run_data->>'duration')::bigint) as value
			FROM action_runs
			WHERE action_id = $1
			GROUP BY bucket
			ORDER BY bucket DESC
			LIMIT 100
		`
	}

	rows, err := r.db.Query(ctx, query, actionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dataPoints []DataPoint
	for rows.Next() {
		var (
			bucket time.Time
			value  *float64
		)
		if err := rows.Scan(&bucket, &value); err != nil {
			continue
		}

		if value != nil {
			dataPoints = append(dataPoints, DataPoint{
				Timestamp: bucket,
				Value:     *value,
			})
		}
	}

	return dataPoints, rows.Err()
}

// DeleteOldRuns deletes runs older than the specified time
func (r *PostgresMetricsRepository) DeleteOldRuns(ctx context.Context, before time.Time) (int64, error) {
	err := r.db.Exec(ctx, `
		DELETE FROM action_runs WHERE created_at < $1
	`, before)
	if err != nil {
		return 0, err
	}

	// Note: We can't get rows affected with the current interface
	// This would need to be enhanced if row counts are needed
	return 0, nil
}

// GetActionWorkflowID retrieves the workflow ID for an action from the semantic_actions table
func (r *PostgresMetricsRepository) GetActionWorkflowID(ctx context.Context, actionID string) (string, error) {
	var workflowID string
	err := r.db.QueryRow(ctx, `SELECT workflow_id FROM semantic_actions WHERE action_id = $1`, actionID).Scan(&workflowID)
	if err != nil {
		return "", fmt.Errorf("failed to get workflow_id for action %s: %w", actionID, err)
	}
	return workflowID, nil
}

// SaveWorkflowMetadata saves workflow metadata to PostgreSQL for foreign key relationships
func (r *PostgresMetricsRepository) SaveWorkflowMetadata(ctx context.Context, workflowID, name, description, workflowType string, jsonLD []byte) error {
	return r.db.Exec(ctx, `
		INSERT INTO workflows (workflow_id, name, description, workflow_type, json_ld, active)
		VALUES ($1, $2, $3, $4, $5, true)
		ON CONFLICT (workflow_id) DO UPDATE
		SET name = EXCLUDED.name,
		    description = EXCLUDED.description,
		    workflow_type = EXCLUDED.workflow_type,
		    json_ld = EXCLUDED.json_ld,
		    updated_at = NOW()
	`, workflowID, name, description, workflowType, jsonLD)
}

// SaveActionMetadata saves action metadata to PostgreSQL for foreign key relationships
func (r *PostgresMetricsRepository) SaveActionMetadata(ctx context.Context, actionID, workflowID, actionType, name, description string, jsonLD []byte) error {
	return r.db.Exec(ctx, `
		INSERT INTO semantic_actions (action_id, workflow_id, action_type, name, description, json_ld)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (action_id) DO UPDATE
		SET workflow_id = EXCLUDED.workflow_id,
		    action_type = EXCLUDED.action_type,
		    name = EXCLUDED.name,
		    description = EXCLUDED.description,
		    json_ld = EXCLUDED.json_ld,
		    updated_at = NOW()
	`, actionID, workflowID, actionType, name, description, jsonLD)
}

// DeleteWorkflowMetadata soft-deletes a workflow in PostgreSQL
func (r *PostgresMetricsRepository) DeleteWorkflowMetadata(ctx context.Context, workflowID string) error {
	return r.db.Exec(ctx, `
		UPDATE workflows SET active = false, updated_at = NOW()
		WHERE workflow_id = $1
	`, workflowID)
}

// DeleteActionMetadata deletes action metadata from PostgreSQL
func (r *PostgresMetricsRepository) DeleteActionMetadata(ctx context.Context, actionID string) error {
	return r.db.Exec(ctx, `
		DELETE FROM semantic_actions WHERE action_id = $1
	`, actionID)
}

// Helper functions to extract values from map

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

func getInt64(m map[string]interface{}, key string) int64 {
	if v, ok := m[key].(float64); ok {
		return int64(v)
	}
	return 0
}

func getMap(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key].(map[string]interface{}); ok {
		return v
	}
	return nil
}
